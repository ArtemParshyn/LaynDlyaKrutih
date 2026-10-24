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

type ForecastRequest struct {
	AsOf           string             `json:"as_of"`
	OpeningBalance int64              `json:"opening_balance"`
	History        []HistoryDay       `json:"history"`
	Scheduled      []ScheduledPayment `json:"scheduled"`
}

type ForecastDay struct {
	Date            string `json:"date"`
	ExpectedBalance int64  `json:"expected_balance"`
	BalanceLow      int64  `json:"balance_low"`
	BalanceHigh     int64  `json:"balance_high"`
	Inflow          int64  `json:"inflow"`
	Outflow         int64  `json:"outflow"`
}

type Alert struct {
	Date      string  `json:"date"`
	Shortfall int64   `json:"shortfall"`
	Chance    float64 `json:"chance"`
	Reason    string  `json:"reason"`
}

// RarePayment is a large transfer seen more than once, but not often enough
// to be part of the daily flow. It is not stored: ledger derives it on read.
type RarePayment struct {
	Amount            int64 `json:"amount"`
	IsIncome          bool  `json:"is_income"`
	Seen              int   `json:"seen"`
	RoughlyEveryDays  int   `json:"roughly_every_days"`
}

type Operation struct {
	Date        string `json:"date"`
	Amount      int64  `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsIncome    bool   `json:"is_income"`
	Manual      bool   `json:"manual,omitempty"`
}

// ManualOperation is entered by the user, not parsed from a statement.
// Amount is a positive number of kopecks. The gateway decides where it goes:
// on or before the statement end it corrects the actual balance, after that
// it is a plan and only moves the forecast.
type ManualOperation struct {
	Date        string `json:"date" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	IsIncome    bool   `json:"is_income"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Recurring   bool   `json:"recurring"`
}

type BalancePoint struct {
	Date    string `json:"date"`
	Balance int64  `json:"balance"`
}

// Training is the Prophet job started by a CSV upload. The forecast itself
// is requested later, when the user asks to process the statement.
type Training struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ForecastResponse struct {
	Days              []ForecastDay      `json:"days"`
	Alerts            []Alert            `json:"alerts"`
	Rough             bool               `json:"rough"`
	RecurringPayments []ScheduledPayment `json:"recurring_payments"`
	Operations        []Operation        `json:"operations"`
	History           []BalancePoint     `json:"history"`
	RarePayments      []RarePayment      `json:"rare_payments"`
	Training          *Training          `json:"training,omitempty"`
	Summary           string             `json:"summary,omitempty"`
}

// LedgerData is received from the Ledger (выписка) service
type LedgerData struct {
	AsOf              string             `json:"as_of"`
	OpeningBalance    int64              `json:"opening_balance"`
	History           []HistoryDay       `json:"history"`
	RecurringPayments []ScheduledPayment `json:"recurring_payments,omitempty"`
	Scheduled         []ScheduledPayment `json:"scheduled,omitempty"`
}

// PlanEvent represents a planned event sent to/from Planning service
type PlanEvent struct {
	ID          uint64 `json:"id,omitempty"`
	Date        string `json:"date" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	IsIncome    bool   `json:"is_income"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description,omitempty"`
	Recurring   bool   `json:"recurring"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// ScenarioRequest represents "what if" events to calculate a second forecast line
type ScenarioRequest struct {
	Events []ScheduledPayment `json:"events" binding:"required"`
}
