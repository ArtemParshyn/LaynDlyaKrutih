"""Thirty-day cash balance forecast. All amounts are integer kopecks."""

from collections import defaultdict
from datetime import date, timedelta
from random import Random

from contract import Alert, ForecastDay, ForecastRequest, HistoryDay, RarePayment, ScheduledPayment

HORIZON_DAYS = 30
LOOKBACK_DAYS = 84
SIMULATIONS = 512
# A month-over-month drift is applied only when both halves have real flow,
# and never steeper than this share per week. One lucky week must not tilt
# the whole month.
MAX_WEEKLY_DRIFT = 0.05
# Drift is applied only once the slope is clearly more than noise.
MIN_WEEKLY_DRIFT = 0.01


def _median_amount(amounts: list[int]) -> int:
    # Avoid floats for large balances; median of two integers rounds down.
    values = sorted(amounts)
    if not values:
        return 0
    middle = len(values) // 2
    return values[middle] if len(values) % 2 else (values[middle - 1] + values[middle]) // 2


def _typical_daily(rows: list[HistoryDay], field: str) -> int:
    """Ordinary flow on this weekday, scaled by how often it actually happens.

    A payment that lands on most of these days stays at its median. A payment
    that lands on only some of them is counted as its share of the days, so two
    companies with different turnover get different lines. A handful of huge
    transfers is left out: those stay a note, not a daily expense.
    """
    observed = len(rows)
    amounts = sorted(getattr(row, field) for row in rows if getattr(row, field) > 0)
    if observed == 0 or not amounts:
        return 0
    # A transfer at least five times the typical one on these same days is rare.
    # It stays a note and must not set, or inflate, the daily line.
    typical = _median_amount(amounts)
    spikes = [amount for amount in amounts if typical > 0 and amount >= typical * 5 and amount >= 100_000]
    regular = [amount for amount in amounts if amount not in spikes]
    if not regular:
        return 0
    # At least half the days: keep the full median, same as before.
    if len(regular) * 2 >= observed:
        return _median_amount(regular)
    # A couple of days is not a habit, unless that is the whole history.
    if len(regular) < 3 and observed >= 7:
        return 0
    return _median_amount(regular) * len(regular) // observed


def _daily_baselines(history: list[HistoryDay]) -> dict[int, tuple[int, int]]:
    recent = history[-LOOKBACK_DAYS:]
    grouped: dict[int, list[HistoryDay]] = defaultdict(list)
    for record in recent:
        grouped[record.date.weekday()].append(record)
    # With fewer than three observations for a weekday, pooling the other
    # days is more stable than trusting a single unusual transaction.
    return {
        weekday: (
            _typical_daily(grouped[weekday] if len(grouped[weekday]) >= 3 else recent, "irregular_in"),
            _typical_daily(grouped[weekday] if len(grouped[weekday]) >= 3 else recent, "irregular_out"),
        )
        for weekday in range(7)
    }


def _percentile(values: list[int], fraction: float) -> int:
    ordered = sorted(values)
    return ordered[int((len(ordered) - 1) * fraction)]


def _band_floor(history: list[HistoryDay]) -> int:
    """Smallest visible gap between the expected line and either boundary."""
    flows = [row.irregular_in + row.irregular_out for row in history[-LOOKBACK_DAYS:] if row.irregular_in or row.irregular_out]
    if len(flows) < 2:
        return 0
    ordered = sorted(flows)
    # Ignore the rare huge transfers: they belong in history, not in the daily band.
    typical = ordered[len(ordered) // 2]
    return max(typical, 1)


def _ordinary_errors(errors: list[int]) -> list[int]:
    """Drop rare transfers that would otherwise blow the chart scale apart.

    A payment of a few million that happens a couple of times is not the daily
    uncertainty. Leaving it in the bootstrap stretches the invisible chart edge
    by tens of millions while the expected balance barely moves. Keep a value
    only when it shows up on at least a tenth of the observed days.
    """
    if len(errors) < 8:
        return errors
    counts: dict[int, int] = defaultdict(int)
    for value in errors:
        counts[value] += 1
    minimum = max(2, len(errors) // 10)
    ordinary = [value for value in errors if counts[value] >= minimum]
    return ordinary or errors


def _error_bands(history: list[HistoryDay], baselines: dict[int, tuple[int, int]], rough: bool) -> tuple[list[tuple[int, int]], list[list[int]]]:
    """Block bootstrap of historical net-flow errors, retaining short streaks.

    Each simulated path samples consecutive seven-day blocks. Using cumulative
    errors, rather than a fixed daily percentage, makes the band grow with the
    forecast horizon. A fixed seed ensures identical input produces identical
    results. These are empirical scenarios, not calibrated probabilities.
    """
    raw_errors = [row.irregular_in - row.irregular_out -
                  (baselines[row.date.weekday()][0] - baselines[row.date.weekday()][1])
                  for row in history[-LOOKBACK_DAYS:]]
    errors = _ordinary_errors(raw_errors)
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
        # A one-sided error leaves the other edge glued to the expected line.
        # Keep a narrow visible gap, but do not let it grow with the horizon:
        # the chart should stay close to the expected balance.
        floor = _band_floor(history)
        low_width = min(max(low_width, floor), floor * 2)
        high_width = min(max(high_width, floor), floor * 2)
        bands.append((-low_width, high_width))
    return bands, paths


def _level(rows: list[HistoryDay], field: str) -> int:
    """Median of a flow that shows up on at least half of these days.

    The same rule as the daily baseline: two large transfers must not set the
    level that the trend is measured against.
    """
    amounts = [getattr(row, field) for row in rows if getattr(row, field) > 0]
    if len(amounts) * 2 < len(rows):
        return 0
    return _median_amount(amounts)


def _weekly_drift(history: list[HistoryDay]) -> tuple[float, float]:
    """Gentle slope from the last four weeks versus the four before them.

    Returns a fraction of the weekday baseline added each week ahead, separately
    for inflow and outflow. Flat, short, or tiny changes stay at zero: one
    quiet week must not tilt the whole month. Compared are median payment
    sizes, not totals, so two missing weekends do not look like a trend.
    """
    recent = history[-LOOKBACK_DAYS:]
    if len(recent) < 56:
        return 0.0, 0.0
    earlier, later = recent[-56:-28], recent[-28:]
    before_in, before_out = _level(earlier, "irregular_in"), _level(earlier, "irregular_out")
    after_in, after_out = _level(later, "irregular_in"), _level(later, "irregular_out")

    def slope(before: int, after: int) -> float:
        if before <= 0 or after <= 0:
            return 0.0
        # Four weeks separate the two windows, so the change is spread per week.
        per_week = (after - before) / before / 4
        if abs(per_week) < MIN_WEEKLY_DRIFT:
            return 0.0
        return max(-MAX_WEEKLY_DRIFT, min(MAX_WEEKLY_DRIFT, per_week))

    return slope(before_in, after_in), slope(before_out, after_out)


def _rare_payments(history: list[HistoryDay]) -> list[RarePayment]:
    """Large transfers that showed up more than once, but on few days.

    They stay out of the daily baseline and out of the chart band. The note
    only says the amount and how often it was seen, without placing it on a date.
    """
    recent = history[-LOOKBACK_DAYS:]
    if len(recent) < 14:
        return []
    notes: list[RarePayment] = []
    for field, is_income in (("irregular_in", True), ("irregular_out", False)):
        amounts = sorted(getattr(row, field) for row in recent if getattr(row, field) > 0)
        if len(amounts) < 2:
            continue
        other = "irregular_out" if is_income else "irregular_in"
        other_amounts = [getattr(row, other) for row in recent if getattr(row, other) > 0]
        # Ordinary scale is the lower half of this flow, or the other direction
        # when this one is only a couple of identical transfers. Two spikes must
        # not set the level they are compared with.
        pool = other_amounts if len(amounts) <= 3 and other_amounts else amounts
        ordinary = _median_amount(sorted(pool)[: max(1, len(pool) // 2)])
        if ordinary <= 0:
            continue
        # Anything at least five times a normal day, and not itself the
        # everyday flow (those already sit in the baseline).
        spikes = [amount for amount in amounts if amount >= ordinary * 5 and amount >= 100_000]
        if len(spikes) < 2 or len(spikes) * 2 >= len(recent):
            continue
        typical = _median_amount(spikes)
        close = [amount for amount in spikes if abs(amount - typical) * 5 <= typical]
        if len(close) < 2:
            continue
        notes.append(RarePayment(
            amount=_median_amount(close),
            is_income=is_income,
            seen=len(close),
            roughly_every_days=max(1, len(recent) // len(close)),
        ))
    return notes


def _scheduled_by_day(payments: list[ScheduledPayment]) -> dict[date, tuple[int, int]]:
    by_day: dict[date, list[int]] = defaultdict(lambda: [0, 0])
    for payment in payments:
        if payment.is_income:
            by_day[payment.date][0] += payment.amount
        else:
            by_day[payment.date][1] += payment.amount
    return {day: (inflow, outflow) for day, (inflow, outflow) in by_day.items()}


def build(request: ForecastRequest, rough: bool) -> tuple[list[ForecastDay], list[Alert], list[RarePayment]]:
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
    drift_in, drift_out = _weekly_drift(history)
    rare = _rare_payments(history)
    scheduled = _scheduled_by_day(request.scheduled)
    balance = request.opening_balance
    days: list[ForecastDay] = []
    alerts: list[Alert] = []

    for offset in range(1, HORIZON_DAYS + 1):
        day = request.as_of + timedelta(days=offset)
        extra_in, extra_out = baselines[day.weekday()]
        # Tilt only the ordinary weekday flow. Planned rent and payroll stay
        # at the amount the user entered.
        weeks_ahead = offset / 7
        extra_in = int(extra_in * (1 + drift_in * weeks_ahead))
        extra_out = int(extra_out * (1 + drift_out * weeks_ahead))
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

    return days, alerts, rare
