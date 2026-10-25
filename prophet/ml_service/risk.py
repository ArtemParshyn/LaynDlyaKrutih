"""Риск кассового разрыва: дата, сумма, вероятность.

Разрыв — это не «остаток ниже нуля», а «остатка не хватает на обязательный
платёж». Для каждого дня горизонта прогноз остатка сравнивается с платежом
из календаря.

Вероятность берётся из 80%-го интервала Prophet:
``sigma = (upper - lower) / (2 * 1.2816)``, затем
``P(баланс < платёж)`` по нормальному распределению. Это не «уверенность
модели» в бытовом смысле, а доля распределения ниже требуемой суммы.
"""

from __future__ import annotations

import numpy as np
import pandas as pd
from scipy.stats import norm

from . import analytics, calendar_v2, recurring

# 1.2816 — z-квантиль для 80% интервала (10-й и 90-й перцентили).
Z80 = 1.2815515655446004
RISK_THRESHOLD = 0.5


def _streams(company_id: str) -> list[dict]:
    """Потоки компании, обогащённые слотами календаря v2."""
    try:
        operations = analytics.load_operations(company_id)
    except FileNotFoundError:
        return []

    operations = operations.copy()
    operations["operation_date"] = pd.to_datetime(operations["operation_date"])
    streams = recurring.detect_streams(operations)
    return calendar_v2.enrich_streams(operations, streams)


def payments_for(company_id: str, fc: pd.DataFrame, horizon_days: int) -> pd.Series:
    """Сумма обязательных платежей по дням прогноза."""
    index = pd.DatetimeIndex(pd.to_datetime(fc["ds"]))
    payments = pd.Series(0.0, index=index)
    if len(index) == 0:
        return payments

    streams = _streams(company_id)
    active = [item for item in streams if item["status"] == "active" and item["mandatory"]]
    if not active:
        return payments

    calendar = calendar_v2.build_calendar(
        active,
        start=index.min().date(),
        days=horizon_days,
    )
    for payment in calendar["payments"]:
        stamp = pd.Timestamp(payment["date"])
        if stamp in payments.index:
            payments.loc[stamp] += float(payment["amount"])
    return payments


def assess(
    fc: pd.DataFrame,
    company_id: str,
    horizon_days: int,
    threshold: float = RISK_THRESHOLD,
) -> dict:
    """Дата, сумма и вероятность разрыва на горизонте.

    ``fc`` — прогноз Prophet с колонками ``ds``, ``yhat``, ``yhat_lower``,
    ``yhat_upper``. Уровень остатка уже включает регулярные платежи.
    """
    frame = fc.copy().reset_index(drop=True)
    index = pd.DatetimeIndex(pd.to_datetime(frame["ds"]))
    payments = payments_for(company_id, frame, horizon_days).reindex(index).fillna(0.0)

    frame["payment"] = payments.to_numpy(dtype=float)
    sigma = (frame["yhat_upper"] - frame["yhat_lower"]) / (2 * Z80)
    safe_sigma = sigma.where(sigma > 1e-6, 1e-6)

    probability = norm.cdf((frame["payment"] - frame["yhat"]) / safe_sigma)
    deficit = np.maximum(frame["payment"] - frame["yhat"], 0.0)
    stressed = np.maximum(frame["payment"] - frame["yhat_lower"], 0.0)

    frame["probability"] = probability
    frame["deficit"] = deficit
    frame["deficit_stressed"] = stressed

    risky = frame[(frame["deficit"] > 0) & (frame["probability"] >= threshold)]
    first = risky.iloc[0] if not risky.empty else None
    worst = frame.loc[frame["deficit"].idxmax()] if (frame["deficit"] > 0).any() else None

    reason = None
    if first is not None:
        reason = "negative_balance" if float(first["payment"]) == 0 else "payment_shortfall"

    negative = frame[frame["yhat"] < 0]
    return {
        "risk": bool(not risky.empty),
        "reason": reason,
        "date": None if first is None else pd.Timestamp(first["ds"]).date().isoformat(),
        "shortage": None if worst is None else round(float(worst["deficit"]), 2),
        "shortage_stressed": None if worst is None else round(float(worst["deficit_stressed"]), 2),
        "probability": None if first is None else round(float(first["probability"]), 4),
        "required_payment": None if first is None else round(float(first["payment"]), 2),
        "balance_forecast": None if first is None else round(float(first["yhat"]), 2),
        "worst_date": None if worst is None else pd.Timestamp(worst["ds"]).date().isoformat(),
        "max_probability": round(float(frame["probability"].max()), 4),
        "negative_balance_days": int(len(negative)),
        "payment_shortfall_days": int((frame["deficit"] > 0).sum()),
        "horizon_days": horizon_days,
    }
