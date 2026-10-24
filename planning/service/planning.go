package service

import (
	"fmt"
	"planning/dto"
	"planning/model"
	"time"

	"gorm.io/gorm"
)

type PlanningService struct {
	db *gorm.DB
}

func New(db *gorm.DB) *PlanningService {
	return &PlanningService{db: db}
}

func (s *PlanningService) Create(userID uint64, event dto.Event) error {
	date, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return fmt.Errorf("дата должна быть в формате YYYY-MM-DD")
	}

	row := model.Event{
		UserID:   userID,
		Date:     date,
		Amount:   event.Amount,
		IsIncome: event.IsIncome,
		Category: event.Category,
	}
	return s.db.Create(&row).Error
}

func (s *PlanningService) ListFuture(userID uint64) ([]dto.Event, error) {
	today := time.Now().Format("2006-01-02")

	var rows []model.Event
	err := s.db.
		Where("user_id = ? AND date >= ?", userID, today).
		Order("date ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	events := make([]dto.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, dto.Event{
			Date:     row.Date.Format("2006-01-02"),
			Amount:   row.Amount,
			IsIncome: row.IsIncome,
			Category: row.Category,
		})
	}
	return events, nil
}
