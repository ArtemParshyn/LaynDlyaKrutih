package entity

import "time"

type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Email       string    `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	CompanyName string    `gorm:"not null;type:varchar(255)" json:"company_name"`
	Password    string    `gorm:"not null;type:varchar(255)" json:"-"`
	OneCToken   string    `gorm:"column:onec_token;type:varchar(128)" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
