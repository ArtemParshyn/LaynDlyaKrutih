package model

import "time"

type PlanEvent struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"index;not null" json:"user_id"`
	Date        time.Time `gorm:"type:date;not null" json:"date"`
	Amount      int64     `gorm:"not null" json:"amount"` // in kopecks
	IsIncome    bool      `gorm:"not null" json:"is_income"`
	Category    string    `gorm:"type:varchar(255);not null" json:"category"`
	Description string    `gorm:"type:text" json:"description"`
	Recurring   bool      `gorm:"default:false;not null" json:"recurring"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PlanEvent) TableName() string {
	return "plan_events"
}
