package service

import (
	"errors"
	"fmt"
	"planning/dto"
	"planning/model"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrEventNotFound = errors.New("event not found")

type PlanningService struct {
	db *gorm.DB
}

func NewPlanningService(db *gorm.DB) *PlanningService {
	return &PlanningService{db: db}
}

// CreateEvent persists a planned event for a user
func (s *PlanningService) CreateEvent(userID uint64, req dto.CreateEventRequest) (*dto.EventResponse, error) {
	if s.db == nil {
		return nil, errors.New("database not available")
	}

	parsedDate, err := parseDate(req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Прочее"
	}

	event := model.PlanEvent{
		UserID:      userID,
		Date:        parsedDate,
		Amount:      req.Amount,
		IsIncome:    req.IsIncome,
		Category:    category,
		Description: strings.TrimSpace(req.Description),
		Recurring:   req.Recurring,
	}

	if err := s.db.Create(&event).Error; err != nil {
		return nil, fmt.Errorf("failed to save event: %w", err)
	}

	return &dto.EventResponse{
		ID:          event.ID,
		Date:        event.Date.Format("2006-01-02"),
		Amount:      event.Amount,
		IsIncome:    event.IsIncome,
		Category:    event.Category,
		Description: event.Description,
		Recurring:   event.Recurring,
		CreatedAt:   event.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetScheduledPayments converts stored planned events into scheduled payments for the forecast engine
// For recurring events, projects occurrences monthly for the next 60 days
func (s *PlanningService) GetScheduledPayments(userID uint64) ([]dto.ScheduledPayment, error) {
	if s.db == nil {
		return []dto.ScheduledPayment{}, nil
	}

	var events []model.PlanEvent
	if err := s.db.Where("user_id = ?", userID).Find(&events).Error; err != nil {
		return nil, err
	}

	var payments []dto.ScheduledPayment
	now := time.Now()
	horizon := now.AddDate(0, 0, 60)

	for _, e := range events {
		if !e.Recurring {
			payments = append(payments, dto.ScheduledPayment{
				Date:     e.Date.Format("2006-01-02"),
				Amount:   e.Amount,
				IsIncome: e.IsIncome,
				Category: e.Category,
			})
			continue
		}

		// Recurring event: project monthly steps from event date up to 60 days ahead
		currDate := e.Date
		maxDate := horizon
		if currDate.After(maxDate) {
			maxDate = currDate.AddDate(0, 2, 0)
		}

		for !currDate.After(maxDate) {
			payments = append(payments, dto.ScheduledPayment{
				Date:     currDate.Format("2006-01-02"),
				Amount:   e.Amount,
				IsIncome: e.IsIncome,
				Category: e.Category,
			})
			currDate = currDate.AddDate(0, 1, 0) // +1 month
		}
	}

	sort.Slice(payments, func(i, j int) bool {
		return payments[i].Date < payments[j].Date
	})

	if payments == nil {
		payments = []dto.ScheduledPayment{}
	}

	return payments, nil
}

// GetEventsList returns raw list of user-created planned events
func (s *PlanningService) GetEventsList(userID uint64) ([]dto.EventResponse, error) {
	if s.db == nil {
		return []dto.EventResponse{}, nil
	}

	var events []model.PlanEvent
	if err := s.db.Where("user_id = ?", userID).Order("date ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}

	res := make([]dto.EventResponse, len(events))
	for i, e := range events {
		res[i] = dto.EventResponse{
			ID:          e.ID,
			Date:        e.Date.Format("2006-01-02"),
			Amount:      e.Amount,
			IsIncome:    e.IsIncome,
			Category:    e.Category,
			Description: e.Description,
			Recurring:   e.Recurring,
			CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		}
	}

	return res, nil
}

// DeleteEvent deletes an event owned by user
func (s *PlanningService) DeleteEvent(userID, eventID uint64) error {
	if s.db == nil {
		return errors.New("database not available")
	}

	result := s.db.Where("id = ? AND user_id = ?", eventID, userID).Delete(&model.PlanEvent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"2006/01/02",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format '%s'", s)
}
