package entity

import "time"

// Pulse is one statement and the forecast built from it.
// LedgerKey is the id ledger, planning and Prophet use. The first pulse of an
// old account keeps the user id, so the statement already stored stays put.
// Later pulses use a key that cannot collide with a real user.
type Pulse struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"type:varchar(120);not null" json:"name"`
	LedgerKey int64     `gorm:"uniqueIndex;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (Pulse) TableName() string {
	return "pulses"
}
