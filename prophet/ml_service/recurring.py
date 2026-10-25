"""Регулярные списания и календарь обязательных платежей.

Детерминированный детектор: группируем списания по контрагенту и категории,
ищем устойчивый интервал между платежами, оцениваем сумму и строим календарь.

Никакой вероятности поступлений: только регулярные расходы.
"""

from __future__ import annotations

from datetime import date, timedelta

import holidays
import numpy as np
import pandas as pd
from dateutil.relativedelta import relativedelta

# Категории, которые считаем обязательными платежами
MANDATORY_CATEGORIES = {
    "Аренда", "Зарплата", "Налоги", "Коммунальные", "Кредиты и лизинг",
}

# Регулярные, но необязательные: подписки, бухгалтерия, охрана, клининг и т.д.
OPTIONAL_CATEGORIES = {
    "Связь и IT", "Административные", "Логистика", "Маркетинг", "Себестоимость",
}

EXCLUDED_CATEGORIES = {"Внутренние и владелец", "Возвраты", "Оборудование", "Прочее"}

CANDIDATE_PERIODS = [7, 14, 15, 30, 60, 91, 182, 365]
PERIOD_LABELS = {
    7: "недельный", 14: "двухнедельный", 15: "дважды в месяц",
    30: "месячный", 60: "двухмесячный", 91: "квартальный",
    182: "полугодовой", 365: "годовой",
}

MIN_OCCURRENCES = 4
MAX_INTERVAL_CV = 0.40
MAX_PERIOD_DEVIATION = 0.25
MIN_COVERAGE = 0.50

RU_HOLIDAYS = holidays.Russia(years=range(2023, 2032))


def is_business_day(value: date) -> bool:
    return value.weekday() < 5 and value not in RU_HOLIDAYS


def shift_to_business_day(value: date, direction: int = 1) -> date:
    """Сдвигает дату на ближайший рабочий день.

    Налоги двигаются вперёд (срок переносится на следующий рабочий день),
    зарплата чаще платится вперёд пятницей, поэтому назад.
    """
    result = value
    for _ in range(10):
        if is_business_day(result):
            return result
        result += timedelta(days=direction)
    return value


def _shift_direction(category: str) -> int:
    return -1 if category == "Зарплата" else 1


def _anchors(dates: pd.Series, period: int) -> tuple[list[int], int | None]:
    """Возвращает дни-якоря месяца и день недели для недельных потоков."""
    if period <= 14:
        weekday = int(dates.dt.dayofweek.mode().iloc[0])
        return [], weekday

    days = sorted(int(day) for day in dates.dt.day)
    if period == 15:
        # Два платежа в месяц: делим дни на две группы по наибольшему разрыву
        gaps = [(days[i + 1] - days[i], i) for i in range(len(days) - 1)]
        if gaps:
            _, index = max(gaps)
            left, right = days[: index + 1], days[index + 1:]
            first = int(round(float(np.median(left))))
            second = int(round(float(np.median(right))))
            if first != second:
                return [first, second], None
        return [int(round(float(np.median(days))))], None

    return [int(round(float(np.median(days))))], None


def _project_dates(stream: dict, start: date, end: date) -> list[date]:
    """Проекция будущих платежей потока на горизонт [start, end]."""
    result: list[date] = []
    last = date.fromisoformat(stream["last_date"])
    next_date = date.fromisoformat(stream["next_date"])
    period = stream["period_days"]
    anchors = stream.get("anchors") or []
    direction = _shift_direction(stream["category"])

    if next_date < start:
        # Платёж уже должен был пройти к началу горизонта — показываем как
        # просроченный в первый день.
        result.append(start)

    if period <= 14:
        current = next_date
        while current < start:
            current += timedelta(days=period)
        while current <= end:
            adjusted = shift_to_business_day(current, direction)
            if start <= adjusted <= end:
                result.append(adjusted)
            current += timedelta(days=period)
    elif period == 15 and len(anchors) == 2:
        cursor = last.replace(day=1)
        while cursor <= end + relativedelta(months=1):
            for anchor in anchors:
                candidate = cursor + relativedelta(day=anchor)
                if candidate <= last:
                    continue
                adjusted = shift_to_business_day(candidate, direction)
                if start <= adjusted <= end:
                    result.append(adjusted)
            cursor += relativedelta(months=1)
    else:
        months = {30: 1, 60: 2, 91: 3, 182: 6, 365: 12}.get(period, 1)
        anchor = anchors[0] if anchors else last.day
        cursor = last.replace(day=1)
        while cursor <= end + relativedelta(months=months):
            candidate = cursor + relativedelta(day=anchor)
            if candidate > last:
                adjusted = shift_to_business_day(candidate, direction)
                if start <= adjusted <= end:
                    result.append(adjusted)
            cursor += relativedelta(months=months)

    return sorted(set(result))


def detect_streams(categorized: pd.DataFrame) -> list[dict]:
    """Ищет регулярные списания по размеченным операциям одной или всех компаний."""
    frame = categorized.copy()
    frame["operation_date"] = pd.to_datetime(frame["operation_date"])
    company_last = frame.groupby("company_id")["operation_date"].max()

    frame = frame[frame["direction"] == "Списание"]
    frame = frame[~frame["category"].isin(EXCLUDED_CATEGORIES)]
    frame = frame[frame["subcategory"] != "свои счета"]

    if frame.empty:
        return []

    frame["counterparty_key"] = frame["counterparty_inn"].fillna(frame["role"])

    streams: list[dict] = []

    for (company, category, subcategory), part in frame.groupby(
        ["company_id", "category", "subcategory"]
    ):
        if category not in MANDATORY_CATEGORIES | OPTIONAL_CATEGORIES:
            continue

        if category == "Зарплата":
            groups = part.groupby("role")
        elif category == "Налоги":
            groups = [("налоги", part)]
        else:
            groups = part.groupby("counterparty_key")

        for counterparty, stream in groups:
            daily = stream.groupby("operation_date")["amount"].sum().sort_index()
            if len(daily) < MIN_OCCURRENCES:
                continue

            dates = daily.index
            gaps = np.diff(dates.values).astype("timedelta64[D]").astype(int)
            median_gap = float(np.median(gaps))
            if median_gap <= 0:
                continue

            period = min(CANDIDATE_PERIODS, key=lambda value: abs(value - median_gap))
            if abs(period - median_gap) > period * MAX_PERIOD_DEVIATION:
                continue

            interval_cv = float(np.std(gaps) / median_gap)
            if interval_cv > MAX_INTERVAL_CV:
                continue

            span = (dates[-1] - dates[0]).days
            coverage = len(dates) / (span / period + 1)
            if coverage < MIN_COVERAGE:
                continue

            amounts = daily.values.astype(float)
            median_amount = float(np.median(amounts))
            low, _, high = np.percentile(amounts, [25, 50, 75])
            amount_cv = float(np.std(amounts) / np.mean(amounts)) if np.mean(amounts) else 1.0

            anchors, weekday = _anchors(pd.Series(dates), period)
            if period <= 14:
                next_nominal = dates[-1] + timedelta(days=period)
            elif period == 15 and len(anchors) == 2:
                next_nominal = _next_anchor_date(dates[-1], anchors)
            else:
                months = {30: 1, 60: 2, 91: 3, 182: 6, 365: 12}.get(period, 1)
                anchor = anchors[0] if anchors else dates[-1].day
                next_nominal = _next_month_date(dates[-1], months, anchor)
            next_date = shift_to_business_day(
                next_nominal.date(), _shift_direction(category)
            )

            silence = (company_last[company] - dates[-1]).days
            if silence <= period * 1.5:
                status = "active"
            elif silence <= period * 3:
                status = "paused"
            else:
                status = "ended"

            confidence = (
                0.35 * min(len(dates) / 12, 1)
                + 0.30 * max(0, 1 - interval_cv / 0.4)
                + 0.20 * max(0, 1 - amount_cv / 0.6)
                + 0.15 * min(coverage, 1)
            )

            stream_id = f"{company}:{category}:{subcategory}:{counterparty}"
            streams.append(
                {
                    "stream_id": stream_id,
                    "company_id": company,
                    "direction": "Списание",
                    "category": category,
                    "subcategory": subcategory,
                    "counterparty": str(counterparty),
                    "counterparty_name": str(
                        stream["counterparty_name"].mode().iloc[0]
                    ) if "counterparty_name" in stream and not stream["counterparty_name"].mode().empty else str(counterparty),
                    "period_days": period,
                    "period": PERIOD_LABELS.get(period, f"{period} дней"),
                    "anchor_weekday": weekday,
                    "anchors": anchors,
                    "occurrences": int(len(dates)),
                    "first_date": dates[0].date().isoformat(),
                    "last_date": dates[-1].date().isoformat(),
                    "next_date": next_date.isoformat(),
                    "window_days": 2,
                    "amount_median": round(median_amount, 2),
                    "amount_low": round(float(low), 2),
                    "amount_high": round(float(high), 2),
                    "amount_cv": round(amount_cv, 4),
                    "interval_cv": round(interval_cv, 4),
                    "coverage": round(float(coverage), 4),
                    "status": status,
                    "mandatory": category in MANDATORY_CATEGORIES,
                    "confidence": round(float(confidence), 4),
                }
            )

    return sorted(streams, key=lambda item: item["amount_median"], reverse=True)


def _next_anchor_date(last: pd.Timestamp, anchors: list[int]) -> pd.Timestamp:
    """Ближайший якорь после последней даты для «дважды в месяц»."""
    candidates = []
    for month_offset in (0, 1):
        base = last.replace(day=1) + relativedelta(months=month_offset)
        for anchor in anchors:
            candidates.append(base + relativedelta(day=anchor))
    future = [candidate for candidate in candidates if candidate > last]
    return min(future) if future else last + relativedelta(months=1)


def _next_month_date(last: pd.Timestamp, months: int, anchor: int) -> pd.Timestamp:
    base = last.replace(day=1) + relativedelta(months=months)
    candidate = base + relativedelta(day=anchor)
    if candidate <= last:
        candidate = base + relativedelta(months=months) + relativedelta(day=anchor)
    return candidate


def build_calendar(
    streams: list[dict],
    start: date,
    days: int = 30,
    include_optional: bool = False,
) -> dict:
    """Строит календарь платежей на горизонт от start."""
    end = start + timedelta(days=days)
    payments = []
    total = 0.0

    for stream in streams:
        if stream["status"] != "active":
            continue
        if not include_optional and not stream["mandatory"]:
            continue

        for payment_date in _project_dates(stream, start, end):
            overdue = payment_date == start and date.fromisoformat(stream["next_date"]) < start
            payments.append(
                {
                    "date": payment_date.isoformat(),
                    "category": stream["category"],
                    "subcategory": stream["subcategory"],
                    "counterparty": stream["counterparty_name"],
                    "amount": stream["amount_median"],
                    "amount_low": stream["amount_low"],
                    "amount_high": stream["amount_high"],
                    "mandatory": stream["mandatory"],
                    "confidence": stream["confidence"],
                    "overdue": overdue,
                }
            )
            total += stream["amount_median"]

    payments.sort(key=lambda item: (item["date"], item["category"]))
    return {
        "start": start.isoformat(),
        "end": end.isoformat(),
        "days": days,
        "total": round(total, 2),
        "payments": payments,
    }
