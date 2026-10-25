package model

import "time"

// Operation represents a single transaction from the bank statement
type Operation struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      uint64    `gorm:"index;not null" json:"user_id"`
	Date        time.Time `gorm:"type:date;index;not null" json:"date"`
	Amount      int64     `gorm:"not null" json:"amount"` // in kopecks (signed: income +, expense -)
	Category    string    `gorm:"type:varchar(255);not null" json:"category"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Recurring   bool      `gorm:"default:false;not null" json:"recurring"`
}

func (Operation) TableName() string {
	return "operations"
}

// ManualOperation is a user-entered past payment. It is not part of the bank
// statement: a new CSV replaces operations, but leaves these rows in place.
// Amount is signed kopecks. These rows move the actual balance and must not
// feed the weekday baseline or recurring detection.
type ManualOperation struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UserID      uint64    `gorm:"index;not null" json:"user_id"`
	Date        time.Time `gorm:"type:date;index;not null" json:"date"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Category    string    `gorm:"type:varchar(255);not null" json:"category"`
	Description string    `gorm:"type:text;not null" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (ManualOperation) TableName() string {
	return "manual_operations"
}

// Statement tracks statement file upload events
type Statement struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"index;not null" json:"user_id"`
	Filename   string    `gorm:"type:varchar(255);not null" json:"filename"`
	UploadedAt time.Time `gorm:"autoCreateTime" json:"uploaded_at"`
}

func (Statement) TableName() string {
	return "statements"
}

// ForecastDay represents a single projected day in the 30-day forecast
type ForecastDay struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	UserID          uint64    `gorm:"index;not null" json:"user_id"`
	Date            time.Time `gorm:"type:date;not null" json:"date"`
	ExpectedBalance int64     `gorm:"not null" json:"expected_balance"`
	BalanceLow      int64     `gorm:"not null" json:"balance_low"`
	BalanceHigh     int64     `gorm:"not null" json:"balance_high"`
	Inflow          int64     `gorm:"not null" json:"inflow"`
	Outflow         int64     `gorm:"not null" json:"outflow"`
	Rough           bool      `gorm:"default:false;not null" json:"rough"`
}

func (ForecastDay) TableName() string {
	return "forecast_days"
}

// ForecastAlert represents a financial warning or risk alert
type ForecastAlert struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"index;not null" json:"user_id"`
	Date      time.Time `gorm:"type:date;not null" json:"date"`
	Shortfall int64     `gorm:"not null" json:"shortfall"`
	Chance    float64   `gorm:"not null" json:"chance"`
	Reason    string    `gorm:"type:text;not null" json:"reason"`
}

func (ForecastAlert) TableName() string {
	return "forecast_alerts"
}

// ForecastHistory stores archived past forecast lines
type ForecastHistory struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	UserID          uint64    `gorm:"index;not null" json:"user_id"`
	Date            time.Time `gorm:"type:date;not null" json:"date"`
	ExpectedBalance int64     `gorm:"not null" json:"expected_balance"`
	BalanceLow      int64     `gorm:"not null" json:"balance_low"`
	BalanceHigh     int64     `gorm:"not null" json:"balance_high"`
	Inflow          int64     `gorm:"not null" json:"inflow"`
	Outflow         int64     `gorm:"not null" json:"outflow"`
	Rough           bool      `gorm:"default:false;not null" json:"rough"`
	ArchivedAt      time.Time `gorm:"autoCreateTime;index" json:"archived_at"`
}

func (ForecastHistory) TableName() string {
	return "forecast_history"
}
