"""Выписка -> дневной ряд остатка.

Логика повторяет ``build_daily`` из EDA.ipynb: дубли по operation_id
выкидываются, пропущенные календарные дни достраиваются, остаток
переносится вперёд (в день без операций баланс не меняется).
"""

from __future__ import annotations

from io import StringIO
from pathlib import Path

import numpy as np
import pandas as pd

REQUIRED_COLUMNS = {
    "operation_id",
    "operation_date",
    "amount",
    "direction",
    "balance_eod",
}


def read_statement(source: str | Path | bytes) -> pd.DataFrame:
    """Читает выписку так же, как в ноутбуке: ``;``, utf-8-sig, ИНН строкой."""
    kwargs = dict(
        sep=";",
        encoding="utf-8-sig",
        dtype={"counterparty_inn": str, "company_id": str},
    )
    if isinstance(source, bytes):
        text = source.decode("utf-8-sig")
        return pd.read_csv(StringIO(text), **kwargs)
    return pd.read_csv(source, **kwargs)


def build_daily(df: pd.DataFrame) -> pd.DataFrame:
    """Одна строка на календарный день: ds, y (= balance_eod), потоки."""
    missing = REQUIRED_COLUMNS - set(df.columns)
    if missing:
        raise ValueError(
            "В выписке нет обязательных колонок: " + ", ".join(sorted(missing))
        )

    df = df.copy()
    df["operation_date"] = pd.to_datetime(df["operation_date"], errors="coerce")
    df = df.dropna(subset=["operation_date"])
    if df.empty:
        raise ValueError("В выписке нет ни одной операции с корректной датой")

    df = df.drop_duplicates(subset=["operation_id"])
    df["amount"] = pd.to_numeric(df["amount"], errors="coerce").fillna(0.0)
    df["balance_eod"] = pd.to_numeric(df["balance_eod"], errors="coerce")

    df["is_inflow"] = df["direction"].eq("Поступление")
    df["inflow"] = np.where(df["is_inflow"], df["amount"], 0.0)
    df["outflow"] = np.where(df["is_inflow"], 0.0, df["amount"])

    daily = (
        df.groupby("operation_date")
        .agg(
            inflow=("inflow", "sum"),
            outflow=("outflow", "sum"),
            n_ops=("operation_id", "size"),
            balance_eod=("balance_eod", "last"),
        )
        .reset_index()
        .rename(columns={"operation_date": "ds"})
        .sort_values("ds")
        .reset_index(drop=True)
    )
    daily["net_flow"] = daily["inflow"] - daily["outflow"]

    full_range = pd.date_range(daily["ds"].min(), daily["ds"].max(), freq="D")
    daily = (
        daily.set_index("ds").reindex(full_range).rename_axis("ds").reset_index()
    )
    daily["balance_eod"] = daily["balance_eod"].ffill()
    for col in ("inflow", "outflow", "net_flow", "n_ops"):
        daily[col] = daily[col].fillna(0.0)

    if daily["balance_eod"].isna().all():
        raise ValueError("Не удалось восстановить остаток: balance_eod пустой")

    daily = daily.dropna(subset=["balance_eod"]).reset_index(drop=True)
    daily["y"] = daily["balance_eod"].astype(float)
    return daily[["ds", "y", "inflow", "outflow", "net_flow", "n_ops"]]


def to_prophet_frame(daily: pd.DataFrame) -> pd.DataFrame:
    return daily[["ds", "y"]].reset_index(drop=True)
