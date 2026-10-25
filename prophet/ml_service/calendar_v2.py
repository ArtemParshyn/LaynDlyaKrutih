"""Календарь регулярных платежей v2: скользящий якорь и слоты.

Отличия от ``recurring``:

1. День платежа берётся по последним платежам, а не по всей истории:
   у налогов и коммуналки день дрейфует.
2. Внутри месяца может быть несколько слотов: зарплата дважды в месяц,
   крупный платёж плюс occasional мелкий.
3. Сумма — за месяц, а не за отдельный платёж; для налогов — среднее
   за 12 месяцев, потому что они квартальные и годовые.
4. Потоки с десятком выплат в месяц сворачиваются в одну месячную сумму.

Используется оценкой риска кассового разрыва.
"""

from __future__ import annotations

from datetime import date, timedelta

import numpy as np
import pandas as pd
from dateutil.relativedelta import relativedelta

from . import recurring

RECENT_PAYMENTS = 8
SMALL_SHARE = 0.4
SMALL_MIN_SHARE = 0.10
MAX_PAYMENTS_PER_MONTH = 4


def _stream_ops(categorized: pd.DataFrame, stream: dict) -> pd.DataFrame:
    """Операции, из которых собран поток. Повторяет группировку v1."""
    ops = categorized[
        (categorized["direction"] == "Списание")
        & (categorized["category"] == stream["category"])
        & (categorized["subcategory"] == stream["subcategory"])
    ]
    if ops.empty:
        return ops

    if stream["category"] == "Зарплата":
        return ops[ops["role"].astype(str) == str(stream["counterparty"])]
    if stream["category"] == "Налоги":
        return ops

    key = ops["counterparty_inn"].astype("string").fillna(
        ops["role"].astype("string")
    )
    return ops[key.astype(str) == str(stream["counterparty"])]


def _slot(rows: list[tuple[pd.Timestamp, float]]) -> dict:
    days = [stamp.day for stamp, _ in rows]
    amounts = [amount for _, amount in rows]
    return {
        "day": int(round(float(np.median(days)))),
        "amount": round(float(np.median(amounts)), 2),
        "count": len(rows),
    }


def _two_clusters(rows: list[tuple[pd.Timestamp, float]]) -> list[dict] | None:
    """Два платежа в месяц: делим дни по наибольшему разрыву."""
    days = sorted({stamp.day for stamp, _ in rows})
    if len(days) < 4:
        return None
    gaps = [(days[i + 1] - days[i], i) for i in range(len(days) - 1)]
    _, index = max(gaps)
    left, right = days[: index + 1], days[index + 1:]

    def pick(bounds: list[int]) -> list[tuple[pd.Timestamp, float]]:
        return [(stamp, amount) for stamp, amount in rows if stamp.day in bounds]

    first, second = pick(left), pick(right)
    if len(first) < 3 or len(second) < 3:
        return None
    return [_slot(first), _slot(second)]


def _slots(stream: dict, ops: pd.DataFrame) -> list[dict]:
    if ops.empty:
        return [{"day": 1, "amount": 0.0, "count": 0}]

    ops = ops.sort_values("operation_date")
    rows = list(zip(ops["operation_date"], ops["amount"].astype(float)))
    recent = rows[-RECENT_PAYMENTS:]

    months = ops["operation_date"].dt.to_period("M")
    monthly = ops.groupby(months)["amount"].sum()
    # Сумма за месяц, а не за отдельный платёж. Для налогов берём год:
    # они квартальные и годовые, последние 3 месяца дают случайный выброс.
    if stream["category"] == "Налоги":
        month_amount = float(monthly.tail(12).mean())
    else:
        month_amount = float(np.median(monthly.tail(3)))
    per_month = len(ops) / max(monthly.nunique(), 1)

    main_day = int(round(float(np.median([stamp.day for stamp, _ in recent]))))

    # Недельные и двухнедельные: месячная сумма делится на число платежей.
    if stream["period_days"] <= 14:
        share = stream["period_days"] / 30.0
        return [{
            "day": main_day,
            "amount": round(month_amount * share, 2),
            "count": len(ops),
        }]

    # Десяток выплат в месяц — не слот, а фон. Ставим одну месячную сумму.
    if per_month > MAX_PAYMENTS_PER_MONTH:
        return [{"day": main_day, "amount": round(month_amount, 2), "count": len(ops)}]

    # Зарплата дважды в месяц: две устойчивые группы дней.
    if stream["period_days"] == 15:
        clusters = _two_clusters(recent)
        if clusters:
            return clusters

    # Крупный платёж и occasional мелкий (налоги, комиссии).
    amounts = [amount for _, amount in recent]
    base = float(np.median(amounts))
    small = [
        (stamp, amount) for stamp, amount in recent
        if amount < base * SMALL_SHARE
    ]
    if (
        base > 0
        and len(small) >= 2
        and len(small) / len(recent) >= SMALL_MIN_SHARE
    ):
        small_amount = float(np.median([amount for _, amount in small]))
        main_amount = max(month_amount - small_amount, 0.0)
        return [
            {"day": main_day, "amount": round(main_amount, 2), "count": len(recent) - len(small)},
            {
                "day": int(round(float(np.median([stamp.day for stamp, _ in small])))),
                "amount": round(small_amount, 2),
                "count": len(small),
            },
        ]

    return [{"day": main_day, "amount": round(month_amount, 2), "count": len(ops)}]


def enrich_streams(categorized: pd.DataFrame, streams: list[dict]) -> list[dict]:
    """Добавляет слоты и пересчитывает сумму по последним платежам."""
    enriched = []
    for stream in streams:
        ops = _stream_ops(categorized, stream)
        if ops.empty:
            continue

        months = ops["operation_date"].dt.to_period("M").nunique()
        per_month = len(ops) / max(months, 1)
        slots = _slots(stream, ops)
        total = round(sum(slot["amount"] for slot in slots), 2)
        enriched.append(
            {
                **stream,
                "slots": slots,
                "amount_median": total,
                "amount_low": round(min(slot["amount"] for slot in slots), 2),
                "amount_high": round(max(slot["amount"] for slot in slots), 2),
                "payments_per_month": round(per_month, 2),
            }
        )
    return enriched


def _project(stream: dict, start: date, end: date) -> list[tuple[date, float]]:
    """Даты и суммы платежей потока на горизонте."""
    result: list[tuple[date, float]] = []
    last = date.fromisoformat(stream["last_date"])
    period = stream["period_days"]
    direction = recurring._shift_direction(stream["category"])

    if period <= 14:
        current = date.fromisoformat(stream["next_date"])
        while current < start:
            current += timedelta(days=period)
        while current <= end:
            adjusted = recurring.shift_to_business_day(current, direction)
            if start <= adjusted <= end:
                result.append((adjusted, stream["amount_median"]))
            current += timedelta(days=period)
        return result

    step = {15: 1, 30: 1, 60: 2, 91: 3, 182: 6, 365: 12}.get(period, 1)
    cursor = start.replace(day=1)
    while cursor <= end + relativedelta(months=step):
        for slot in stream.get("slots") or [{"day": last.day, "amount": stream["amount_median"]}]:
            candidate = cursor + relativedelta(day=slot["day"])
            if candidate <= last or not (start <= candidate <= end):
                continue
            adjusted = recurring.shift_to_business_day(candidate, direction)
            if start <= adjusted <= end:
                result.append((adjusted, slot["amount"]))
        cursor += relativedelta(months=step)
    return result


def build_calendar(
    streams: list[dict],
    start: date,
    days: int = 30,
    include_optional: bool = False,
) -> dict:
    """Тот же контракт, что у ``recurring.build_calendar``."""
    end = start + timedelta(days=days)
    payments = []
    total = 0.0

    for stream in streams:
        if stream["status"] != "active":
            continue
        if not include_optional and not stream["mandatory"]:
            continue
        for payment_date, amount in _project(stream, start, end):
            payments.append(
                {
                    "date": payment_date.isoformat(),
                    "category": stream["category"],
                    "subcategory": stream["subcategory"],
                    "counterparty": stream["counterparty_name"],
                    "amount": round(float(amount), 2),
                    "amount_low": stream["amount_low"],
                    "amount_high": stream["amount_high"],
                    "mandatory": stream["mandatory"],
                    "confidence": stream["confidence"],
                    "overdue": False,
                }
            )
            total += amount

    payments.sort(key=lambda item: (item["date"], item["category"]))
    return {
        "start": start.isoformat(),
        "end": end.isoformat(),
        "days": days,
        "total": round(total, 2),
        "payments": payments,
    }


def calendar_series(
    streams: list[dict],
    index: pd.DatetimeIndex,
    *,
    include_optional: bool = False,
) -> pd.Series:
    """Сумма платежей календаря на каждую дату индекса."""
    amounts = pd.Series(0.0, index=index)
    if len(index) == 0:
        return amounts

    start = index.min().date()
    end = index.max().date()
    cursor = start
    while cursor <= end:
        chunk_end = min(cursor + timedelta(days=119), end)
        calendar = build_calendar(
            streams,
            start=cursor,
            days=(chunk_end - cursor).days,
            include_optional=include_optional,
        )
        for payment in calendar["payments"]:
            stamp = pd.Timestamp(payment["date"])
            if stamp in amounts.index:
                amounts.loc[stamp] += float(payment["amount"])
        cursor = chunk_end + timedelta(days=1)
    return amounts
