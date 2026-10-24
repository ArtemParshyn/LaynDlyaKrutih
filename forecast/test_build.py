import unittest
from datetime import date, timedelta

from build import build
from check import check
from contract import ForecastRequest, HistoryDay, ScheduledPayment


class ForecastTests(unittest.TestCase):
    def test_weekday_medians_and_scheduled_payment_in_kopecks(self):
        as_of = date(2026, 9, 30)
        history = []
        for offset in range(56):
            day = as_of - timedelta(days=55 - offset)
            income = 1000 if day.weekday() == 3 else 0
            if offset == 6:
                income = 1_000_000  # outlier must not affect median
            history.append(HistoryDay(date=day, irregular_in=income, irregular_out=100))
        request = ForecastRequest(as_of=as_of, opening_balance=100_000, history=history,
                                  scheduled=[ScheduledPayment(date=date(2026, 10, 1), amount=500,
                                                              is_income=False, category="аренда")])
        days, alerts, rare = build(request, rough=not check(request))
        self.assertEqual(len(days), 30)
        self.assertEqual((days[0].inflow, days[0].outflow, days[0].expected_balance),
                         (1000, 600, 100_400))
        self.assertTrue(all(day.inflow < 1_000_000 for day in days))
        self.assertEqual(days[0].balance_low <= days[0].expected_balance <= days[0].balance_high, True)
        self.assertEqual(alerts, [])
        self.assertEqual(rare, [])
        self.assertEqual(build(request, rough=False), (days, alerts, rare))

    def test_rare_large_payments_do_not_become_daily(self):
        as_of = date(2026, 10, 10)
        start = as_of - timedelta(days=83)
        spikes = {date(2026, 9, 7), date(2026, 9, 21)}  # two Mondays, 2_000_000 each
        history = []
        day = start
        while day <= as_of:
            history.append(HistoryDay(
                date=day,
                irregular_in=15_000 if day.weekday() < 5 else 0,
                irregular_out=2_000_000 if day in spikes else 0,
            ))
            day += timedelta(days=1)
        request = ForecastRequest(
            as_of=as_of, opening_balance=5_000_000, history=history,
            scheduled=[ScheduledPayment(date=date(2026, 10, 15), amount=40_000,
                                        is_income=False, category="аренда")],
        )
        days, _, rare = build(request, rough=False)
        self.assertTrue(all(day.outflow <= 40_000 for day in days))
        self.assertEqual(sum(day.outflow for day in days), 40_000)
        self.assertEqual(len(rare), 1)
        self.assertFalse(rare[0].is_income)
        self.assertEqual(rare[0].amount, 2_000_000)
        self.assertEqual(rare[0].seen, 2)

    def test_short_history_rough_and_low_edge_alert(self):
        as_of = date(2026, 9, 30)
        history = [HistoryDay(date=as_of, irregular_in=0, irregular_out=1000)]
        request = ForecastRequest(as_of=as_of, opening_balance=500, history=history, scheduled=[])
        self.assertFalse(check(request))
        days, alerts, _ = build(request, rough=True)
        self.assertEqual(days[0].expected_balance, -500)
        self.assertEqual(alerts[0].shortfall, -days[0].balance_low)
        self.assertTrue(all(days[i].balance_low <= days[i].expected_balance <= days[i].balance_high
                            for i in range(30)))

    def test_uncertainty_grows_from_historical_errors(self):
        as_of = date(2026, 9, 30)
        # Quiet and busy days both repeat often enough to stay in the band.
        history = [HistoryDay(date=as_of - timedelta(days=55 - i),
                              irregular_in=10_000 if i % 2 == 0 else 0,
                              irregular_out=0) for i in range(56)]
        request = ForecastRequest(as_of=as_of, opening_balance=100_000,
                                  history=history, scheduled=[])
        days, _, _ = build(request, rough=False)
        widths = [day.balance_high - day.balance_low for day in days]
        self.assertGreater(widths[-1], 0)
        self.assertEqual(widths, sorted(widths))

    def test_warning_when_only_lower_bound_is_negative(self):
        as_of = date(2026, 9, 30)
        history = [HistoryDay(date=as_of - timedelta(days=55 - i),
                              irregular_in=0,
                              irregular_out=20_000 if i % 2 == 0 else 0)
                   for i in range(56)]
        request = ForecastRequest(as_of=as_of, opening_balance=40_000,
                                  history=history, scheduled=[])
        days, alerts, _ = build(request, rough=False)
        self.assertGreaterEqual(days[0].expected_balance, 0)
        self.assertTrue(any(day.balance_low < 0 for day in days))
        self.assertTrue(alerts)

    def test_rare_spikes_do_not_stretch_the_chart(self):
        as_of = date(2026, 10, 10)
        start = as_of - timedelta(days=83)
        spikes = {date(2026, 9, 7), date(2026, 9, 21)}
        history = []
        day = start
        while day <= as_of:
            history.append(HistoryDay(
                date=day,
                irregular_in=15_000 if day.weekday() < 5 else 0,
                irregular_out=200_000_000 if day in spikes else (5_000 if day.weekday() < 5 else 0),
            ))
            day += timedelta(days=1)
        request = ForecastRequest(as_of=as_of, opening_balance=50_000_000,
                                  history=history, scheduled=[])
        days, _, rare = build(request, rough=False)
        span = max(day.balance_high for day in days) - min(day.balance_low for day in days)
        self.assertEqual(len(rare), 1)
        self.assertEqual(rare[0].amount, 200_000_000)
        self.assertLess(span, 500_000)
        self.assertGreater(days[-1].balance_high, days[-1].expected_balance)
        self.assertLess(days[-1].balance_low, days[-1].expected_balance)
        self.assertLessEqual(days[-1].balance_high - days[-1].balance_low, 80_000)

    def test_recent_growth_tilts_the_expected_line_slightly(self):
        as_of = date(2026, 10, 8)  # Thursday, so the next day is a weekday
        history = []
        for offset in range(84):
            day = as_of - timedelta(days=83 - offset)
            # Weekday income doubles in the latest month. Weekend days stay at
            # zero and must not be read as a falling trend.
            income = 20_000 if offset >= 56 else 10_000
            if day.weekday() >= 5:
                income = 0
            history.append(HistoryDay(date=day, irregular_in=income, irregular_out=0))
        request = ForecastRequest(as_of=as_of, opening_balance=100_000, history=history, scheduled=[])
        days, _, rare = build(request, rough=False)
        self.assertEqual(rare, [])
        flat = 10_000  # older weekday level; the forecast should start above it
        self.assertGreater(days[0].inflow, flat)
        # One month of growth must not double the daily income.
        self.assertLess(days[-1].inflow, flat * 2)
        # Day 30 can be a weekend, so compare the last weekday with the first.
        last_weekday = next(day for day in reversed(days) if day.inflow > 0)
        self.assertGreater(last_weekday.inflow, days[0].inflow)

    def test_flat_history_has_no_drift(self):
        as_of = date(2026, 10, 8)
        history = [HistoryDay(date=as_of - timedelta(days=83 - offset),
                              irregular_in=10_000 if (as_of - timedelta(days=83 - offset)).weekday() < 5 else 0,
                              irregular_out=0) for offset in range(84)]
        request = ForecastRequest(as_of=as_of, opening_balance=100_000, history=history, scheduled=[])
        days, _, _ = build(request, rough=False)
        self.assertEqual(days[0].inflow, 10_000)
        self.assertTrue(all(day.inflow in (0, 10_000) for day in days))

    def test_sparse_turnover_is_not_the_same_flat_line(self):
        as_of = date(2026, 10, 8)
        quiet, busy = [], []
        for offset in range(84):
            day = as_of - timedelta(days=83 - offset)
            quiet.append(HistoryDay(date=day, irregular_in=30_000 if day.weekday() == 0 else 0, irregular_out=0))
            busy.append(HistoryDay(date=day, irregular_in=80_000 if day.weekday() < 5 else 0, irregular_out=20_000))
        quiet_days, _, _ = build(ForecastRequest(as_of=as_of, opening_balance=100_000, history=quiet, scheduled=[]), rough=False)
        busy_days, _, _ = build(ForecastRequest(as_of=as_of, opening_balance=100_000, history=busy, scheduled=[]), rough=False)
        self.assertGreater(busy_days[0].inflow, quiet_days[0].inflow)
        self.assertNotEqual(quiet_days[-1].expected_balance, busy_days[-1].expected_balance)
        # Income that lands every Monday stays on Monday, and nowhere else.
        self.assertEqual(quiet_days[3].inflow, 30_000)
        self.assertTrue(all(day.inflow == 0 for day in quiet_days if day.date.weekday() != 0))


if __name__ == "__main__":
    unittest.main()
