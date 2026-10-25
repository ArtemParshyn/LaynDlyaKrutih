"""Подбор гиперпараметров Prophet через Optuna и прогноз остатка.

Двухступенчатая схема из EDA.ipynb:

1. Screen. TPE перебирает changepoint_prior_scale, seasonality_prior_scale,
   режим сезонности и недельный флаг. Цель — MAE на последних
   ``HOLDOUT_DAYS`` днях.
2. Confirm. Топ-``TOP_K`` конфигов прогоняются через rolling-origin
   cross_validation, победитель переобучается на всей истории.

Годовая сезонность не тюнится: Prophet сам предупреждает, что она
нестабильна на истории короче двух лет, поэтому включается только
при ``span_days >= 730``.
"""

from __future__ import annotations

import logging
from typing import Any

import numpy as np
import optuna
import pandas as pd
from prophet import Prophet
from prophet.diagnostics import cross_validation

from . import config

logger = logging.getLogger("ml_service")
optuna.logging.set_verbosity(optuna.logging.WARNING)


def _metrics(y: np.ndarray, yhat: np.ndarray, min_abs_y: float = 1000.0) -> dict[str, float]:
    y, yhat = np.asarray(y, float), np.asarray(yhat, float)
    mae = float(np.mean(np.abs(y - yhat)))

    denom = np.abs(y)
    mask = denom >= min_abs_y
    mape = (
        float(np.mean(np.abs(y[mask] - yhat[mask]) / denom[mask]) * 100)
        if mask.any()
        else float("nan")
    )

    denom_s = np.abs(y) + np.abs(yhat)
    ms = denom_s > 1e-9
    smape = (
        float(np.mean(2 * np.abs(y[ms] - yhat[ms]) / denom_s[ms]) * 100)
        if ms.any()
        else float("nan")
    )
    return {"mae": mae, "mape": mape, "smape": smape}


def _make_model(params: dict[str, Any], yearly: bool) -> Prophet:
    model = Prophet(
        growth="linear",
        changepoint_prior_scale=params["cps"],
        seasonality_prior_scale=params["sps"],
        seasonality_mode=params["mode"],
        yearly_seasonality=yearly,
        weekly_seasonality=params["weekly"],
        n_changepoints=15,
        daily_seasonality=False,
    )
    model.add_country_holidays(country_name="RU")
    return model


def tune(history: pd.DataFrame, n_trials: int, top_k: int) -> dict[str, Any]:
    """Возвращает лучшие параметры и метрики качества подбора.

    ``history`` — полный ряд ``ds``/``y``. Последние ``HOLDOUT_DAYS``
    откладываются и в обучении финальной модели не участвуют: по ним
    считается честный holdout.
    """
    if len(history) <= config.HOLDOUT_DAYS + 30:
        raise ValueError(
            f"Слишком короткая история для подбора: {len(history)} дней "
            f"(нужно больше {config.HOLDOUT_DAYS + 30})"
        )

    train_df = history.iloc[:-config.HOLDOUT_DAYS].reset_index(drop=True)
    test_df = history.iloc[-config.HOLDOUT_DAYS:].reset_index(drop=True)

    span_days = int((train_df["ds"].max() - train_df["ds"].min()).days + 1)
    yearly = span_days >= config.MIN_DAYS_YEARLY

    def objective(trial: optuna.Trial) -> float:
        params = {
            "cps": trial.suggest_float("cps", 0.001, 0.5, log=True),
            "sps": trial.suggest_float("sps", 0.01, 50.0, log=True),
            "mode": trial.suggest_categorical("mode", ["additive", "multiplicative"]),
            "weekly": trial.suggest_categorical("weekly", [True, False]),
        }
        model = _make_model(params, yearly)
        model.fit(train_df)
        forecast = model.predict(
            model.make_future_dataframe(periods=len(test_df), freq="D")
        ).tail(len(test_df))
        scores = _metrics(test_df["y"].values, forecast["yhat"].values)
        trial.set_user_attr("mape", scores["mape"])
        trial.set_user_attr("smape", scores["smape"])
        return scores["mae"]

    study = optuna.create_study(
        direction="minimize",
        sampler=optuna.samplers.TPESampler(seed=config.RANDOM_SEED),
    )
    study.optimize(objective, n_trials=n_trials, show_progress_bar=False)

    finished = sorted(
        (t for t in study.trials if t.value is not None), key=lambda t: t.value
    )
    if not finished:
        raise RuntimeError("Optuna не смогла завершить ни одного trial")
    top = finished[:top_k]

    best: dict[str, Any] | None = None
    for trial in top:
        mae_cv, mape_cv = _cv_score(trial.params, train_df, yearly, span_days)
        if best is None or mae_cv < best["mae_cv"]:
            best = {
                "params": trial.params,
                "mae_screen": float(trial.value),
                "mae_cv": mae_cv,
                "mape_cv": mape_cv,
            }

    assert best is not None
    best["params"]["yearly"] = yearly

    final = _make_model(best["params"], yearly)
    final.fit(train_df)
    holdout_fc = final.predict(
        final.make_future_dataframe(periods=len(test_df), freq="D")
    ).tail(len(test_df))
    holdout = _metrics(test_df["y"].values, holdout_fc["yhat"].values)

    return {
        "params": best["params"],
        "metrics": {
            "mae_screen": best["mae_screen"],
            "mae_cv": best["mae_cv"],
            "mape_cv": best["mape_cv"],
            "mae_holdout": holdout["mae"],
            "mape_holdout": holdout["mape"],
            "smape_holdout": holdout["smape"],
            "n_days_train": int(len(train_df)),
            "span_days": span_days,
            "n_trials": n_trials,
        },
    }


def _cv_score(
    params: dict[str, Any], train_df: pd.DataFrame, yearly: bool, span_days: int
) -> tuple[float, float]:
    """MAE и MAPE rolling-origin CV. inf, если ряд слишком короткий.

    ``initial`` как в ноутбуке — 365 дней, чтобы CV давал несколько окон.
    На коротких рядах ужимаем, но оставляем минимум два окна прогноза.
    """
    initial_days = min(365, max(120, span_days - 2 * config.HOLDOUT_DAYS))
    if span_days - initial_days < config.HOLDOUT_DAYS:
        return float("inf"), float("nan")

    model = _make_model(params, yearly)
    model.fit(train_df)
    try:
        df_cv = cross_validation(
            model,
            horizon=f"{config.HOLDOUT_DAYS} days",
            period=f"{config.HOLDOUT_DAYS} days",
            initial=f"{initial_days} days",
        )
    except Exception as exc:  # noqa: BLE001 — CV падает на коротких рядах, это не фатально
        logger.warning("cross_validation пропущен: %s", exc)
        return float("inf"), float("nan")

    if len(df_cv) == 0:
        return float("inf"), float("nan")
    scores = _metrics(df_cv["y"].values, df_cv["yhat"].values)
    return scores["mae"], scores["mape"]


def fit_final(history: pd.DataFrame, params: dict[str, Any]) -> Prophet:
    """Финальная модель на всей доступной истории — с неё строится прогноз."""
    model = _make_model(params, bool(params.get("yearly", False)))
    model.fit(history[["ds", "y"]])
    return model


def forecast(model: Prophet, periods: int) -> pd.DataFrame:
    future = model.make_future_dataframe(periods=periods, freq="D", include_history=False)
    fc = model.predict(future)
    return fc[["ds", "yhat", "yhat_lower", "yhat_upper"]].reset_index(drop=True)


def cash_gap(fc: pd.DataFrame, threshold: float) -> dict[str, Any] | None:
    """Первый день, где прогноз остатка уходит ниже нуля.

    Вероятность оценивается по доле 80%-го интервала Prophet, лежащей
    ниже нуля: интервал трактуется как равномерный. Это грубая, но
    объяснимая предпринимателю оценка «насколько уверен модель».
    """
    for row in fc.itertuples(index=False):
        lower, upper = float(row.yhat_lower), float(row.yhat_upper)
        if upper <= 0:
            probability = 1.0
        elif lower >= 0:
            probability = 0.0
        else:
            probability = (0.0 - lower) / (upper - lower)

        if float(row.yhat) < 0 and probability >= threshold:
            return {
                "date": pd.Timestamp(row.ds).date().isoformat(),
                "shortage": round(max(0.0, -float(row.yhat)), 2),
                "probability": round(probability, 3),
                "balance_forecast": round(float(row.yhat), 2),
            }
    return None
