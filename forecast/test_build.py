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
        days, alerts = build(request, rough=not check(request))
        self.assertEqual(len(days), 30)
        self.assertEqual((days[0].inflow, days[0].outflow, days[0].expected_balance),
                         (1000, 600, 100_400))
        self.assertEqual(days[0].balance_low <= days[0].expected_balance <= days[0].balance_high, True)
        self.assertEqual(alerts, [])
        self.assertEqual(build(request, rough=False), (days, alerts))

    def test_short_history_rough_and_low_edge_alert(self):
        as_of = date(2026, 9, 30)
        history = [HistoryDay(date=as_of, irregular_in=0, irregular_out=1000)]
        request = ForecastRequest(as_of=as_of, opening_balance=500, history=history, scheduled=[])
        self.assertFalse(check(request))
        days, alerts = build(request, rough=True)
        self.assertEqual(days[0].expected_balance, -500)
        self.assertEqual(alerts[0].shortfall, -days[0].balance_low)
        self.assertTrue(all(days[i].balance_low <= days[i].expected_balance <= days[i].balance_high
                            for i in range(30)))

    def test_uncertainty_grows_from_historical_errors(self):
        as_of = date(2026, 9, 30)
        history = [HistoryDay(date=as_of - timedelta(days=55 - i),
                              irregular_in=10_000 if i % 14 == 0 else 0,
                              irregular_out=0) for i in range(56)]
        request = ForecastRequest(as_of=as_of, opening_balance=100_000,
                                  history=history, scheduled=[])
        days, _ = build(request, rough=False)
        widths = [day.balance_high - day.balance_low for day in days]
        self.assertGreater(widths[-1], 0)
        self.assertEqual(widths, sorted(widths))

    def test_warning_when_only_lower_bound_is_negative(self):
        as_of = date(2026, 9, 30)
        history = [HistoryDay(date=as_of - timedelta(days=55 - i),
                              irregular_in=0,
                              irregular_out=10_000 if i % 14 == 0 else 0)
                   for i in range(56)]
        request = ForecastRequest(as_of=as_of, opening_balance=12_000,
                                  history=history, scheduled=[])
        days, alerts = build(request, rough=False)
        self.assertGreaterEqual(days[0].expected_balance, 0)
        self.assertTrue(any(day.expected_balance >= 0 and day.balance_low < 0 for day in days))
        self.assertTrue(any(alert.reason == "возможен кассовый разрыв" for alert in alerts))


if __name__ == "__main__":
    unittest.main()
