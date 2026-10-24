package dto

type HistoryDay struct {
	Date         string `json:"date"`
	IrregularIn  int64  `json:"irregular_in"`
	IrregularOut int64  `json:"irregular_out"`
}

type ScheduledPayment struct {
	Date     string `json:"date"`
	Amount   int64  `json:"amount"`
	IsIncome bool   `json:"is_income"`
	Category string `json:"category"`
}

type DataResponse struct {
	AsOf              string             `json:"as_of"`
	OpeningBalance    int64              `json:"opening_balance"`
	History           []HistoryDay       `json:"history"`
	RecurringPayments []ScheduledPayment `json:"recurring_payments"`
}

type ForecastDayDTO struct {
	Date            string `json:"date"`
	ExpectedBalance int64  `json:"expected_balance"`
	BalanceLow      int64  `json:"balance_low"`
	BalanceHigh     int64  `json:"balance_high"`
	Inflow          int64  `json:"inflow"`
	Outflow         int64  `json:"outflow"`
}

type AlertDTO struct {
	Date      string  `json:"date"`
	Shortfall int64   `json:"shortfall"`
	Chance    float64 `json:"chance"`
	Reason    string  `json:"reason"`
}

type ForecastPayload struct {
	Days   []ForecastDayDTO `json:"days"`
	Alerts []AlertDTO       `json:"alerts"`
	Rough  bool             `json:"rough"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
