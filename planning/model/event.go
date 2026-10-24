package model

import "time"

type Event struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"index;not null"`
	Date      time.Time `gorm:"type:date;not null"`
	Amount    int64     `gorm:"not null"`
	IsIncome  bool      `gorm:"not null"`
	Category  string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time
}

func (Event) TableName() string {
	return "events"
}
