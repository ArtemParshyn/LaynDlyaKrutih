"""Thirty-day cash balance forecast. All amounts are integer kopecks."""

from collections import defaultdict
from datetime import date, timedelta
from random import Random

from contract import Alert, ForecastDay, ForecastRequest, HistoryDay, ScheduledPayment

HORIZON_DAYS = 30
LOOKBACK_DAYS = 84
SIMULATIONS = 512


def _median_amount(amounts: list[int]) -> int:
    # Avoid floats for large balances; median of two integers rounds down.
    values = sorted(amounts)
    if not values:
        return 0
    middle = len(values) // 2
    return values[middle] if len(values) % 2 else (values[middle - 1] + values[middle]) // 2


def _daily_baselines(history: list[HistoryDay]) -> dict[int, tuple[int, int]]:
    recent = history[-LOOKBACK_DAYS:]
    grouped: dict[int, list[HistoryDay]] = defaultdict(list)
    for record in recent:
        grouped[record.date.weekday()].append(record)
    # With fewer than three observations for a weekday, pooling the other
    # days is more stable than trusting a single unusual transaction.
    return {
        weekday: (
            _median_amount([row.irregular_in for row in grouped[weekday]]
                           if len(grouped[weekday]) >= 3 else [row.irregular_in for row in recent]),
            _median_amount([row.irregular_out for row in grouped[weekday]]
                           if len(grouped[weekday]) >= 3 else [row.irregular_out for row in recent]),
        )
        for weekday in range(7)
    }


def _percentile(values: list[int], fraction: float) -> int:
    ordered = sorted(values)
    return ordered[int((len(ordered) - 1) * fraction)]


def _error_bands(history: list[HistoryDay], baselines: dict[int, tuple[int, int]], rough: bool) -> tuple[list[tuple[int, int]], list[list[int]]]:
    """Block bootstrap of historical net-flow errors, retaining short streaks.

    Each simulated path samples consecutive seven-day blocks. Using cumulative
    errors, rather than a fixed daily percentage, makes the band grow with the
    forecast horizon. A fixed seed ensures identical input produces identical
    results. These are empirical scenarios, not calibrated probabilities.
    """
    errors = [row.irregular_in - row.irregular_out -
              (baselines[row.date.weekday()][0] - baselines[row.date.weekday()][1])
              for row in history[-LOOKBACK_DAYS:]]
    if not errors:
        return [(0, 0)] * HORIZON_DAYS, []

    rng = Random(0)
    paths = [[0] * HORIZON_DAYS for _ in range(SIMULATIONS)]
    for path in paths:
        accumulated = 0
        for start in range(0, HORIZON_DAYS, 7):
            index = rng.randrange(len(errors))
            for offset in range(start, min(start + 7, HORIZON_DAYS)):
                accumulated += errors[(index + offset - start) % len(errors)]
                path[offset] = accumulated

    bands = []
    low_width = high_width = 0
    for offset in range(HORIZON_DAYS):
        deviations = [path[offset] for path in paths]
        lower = _percentile(deviations, 0.10)
        upper = _percentile(deviations, 0.90)
        # Below 30 days the empirical tails are poorly sampled. Widen them,
        # and never allow uncertainty to shrink merely due to sampling noise.
        multiplier = 2 if rough else 1
        low_width = max(low_width, max(0, -lower) * multiplier)
        high_width = max(high_width, max(0, upper) * multiplier)
        bands.append((-low_width, high_width))
    return bands, paths


def _scheduled_by_day(payments: list[ScheduledPayment]) -> dict[date, tuple[int, int]]:
    by_day: dict[date, list[int]] = defaultdict(lambda: [0, 0])
    for payment in payments:
        if payment.is_income:
            by_day[payment.date][0] += payment.amount
        else:
            by_day[payment.date][1] += payment.amount
    return {day: (inflow, outflow) for day, (inflow, outflow) in by_day.items()}


def build(request: ForecastRequest, rough: bool) -> tuple[list[ForecastDay], list[Alert]]:
    # A duplicate date in history must not count twice. Ledger normally sends
    # consecutive zero-filled days; fill gaps defensively for direct API calls.
    by_date = {row.date: row for row in request.history if row.date <= request.as_of}
    history: list[HistoryDay] = []
    if by_date:
        day = max(min(by_date), request.as_of - timedelta(days=LOOKBACK_DAYS - 1))
        while day <= request.as_of:
            history.append(by_date.get(day, HistoryDay(date=day, irregular_in=0, irregular_out=0)))
            day += timedelta(days=1)

    baselines = _daily_baselines(history)
    bands, error_paths = _error_bands(history, baselines, rough)
    scheduled = _scheduled_by_day(request.scheduled)
    balance = request.opening_balance
    days: list[ForecastDay] = []
    alerts: list[Alert] = []

    for offset in range(1, HORIZON_DAYS + 1):
        day = request.as_of + timedelta(days=offset)
        extra_in, extra_out = baselines[day.weekday()]
        planned_in, planned_out = scheduled.get(day, (0, 0))
        inflow = extra_in + planned_in
        outflow = extra_out + planned_out
        balance += inflow - outflow
        lower, upper = bands[offset - 1]
        low, high = balance + lower, balance + upper
        days.append(ForecastDay(date=day, expected_balance=balance,
                                balance_low=low, balance_high=high,
                                inflow=inflow, outflow=outflow))
        if low < 0:
            # Empirical share of simulated paths; not a calibrated probability.
            chance = (sum(balance + path[offset - 1] < 0 for path in error_paths) / len(error_paths)
                      if error_paths else float(balance < 0))
            alerts.append(Alert(date=day, shortfall=-low,
                                chance=chance,
                                reason="возможен кассовый разрыв" if balance >= 0 else "ожидается кассовый разрыв"))

    return days, alerts
