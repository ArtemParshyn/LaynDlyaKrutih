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

// RarePayment is derived from stored operations. It is not a forecast day and
// is not written to the forecast tables.
type RarePayment struct {
	Amount           int64 `json:"amount"`
	IsIncome         bool  `json:"is_income"`
	Seen             int   `json:"seen"`
	RoughlyEveryDays int   `json:"roughly_every_days"`
}

type OperationDTO struct {
	Date        string `json:"date"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsIncome    bool   `json:"is_income"`
}

type BalancePoint struct {
	Date    string `json:"date"`
	Balance int64  `json:"balance"`
}

type ForecastPayload struct {
	Days       []ForecastDayDTO `json:"days"`
	Alerts     []AlertDTO       `json:"alerts"`
	Rough      bool             `json:"rough"`
	Operations   []OperationDTO   `json:"operations,omitempty"`
	History      []BalancePoint   `json:"history,omitempty"`
	RarePayments []RarePayment    `json:"rare_payments,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
