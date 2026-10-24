package dto

type CreateEventRequest struct {
	Date        string `json:"date" binding:"required"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	IsIncome    bool   `json:"is_income"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description"`
	Recurring   bool   `json:"recurring"`
}

type EventResponse struct {
	ID          uint64 `json:"id"`
	Date        string `json:"date"`
	Amount      int64  `json:"amount"`
	IsIncome    bool   `json:"is_income"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Recurring   bool   `json:"recurring"`
	CreatedAt   string `json:"created_at"`
}

type ScheduledPayment struct {
	Date     string `json:"date"`
	Amount   int64  `json:"amount"`
	IsIncome bool   `json:"is_income"`
	Category string `json:"category"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
