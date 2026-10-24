package dto

type Event struct {
	Date     string `json:"date"`
	Amount   int64  `json:"amount"`
	IsIncome bool   `json:"is_income"`
	Category string `json:"category"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
