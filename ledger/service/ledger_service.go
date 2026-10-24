package service

import (
	"errors"
	"fmt"
	"ledger/dto"
	"ledger/model"
	"time"

	"gorm.io/gorm"
)

var ErrNoData = errors.New("no statements or operations found")

type LedgerService struct {
	db *gorm.DB
}

func NewLedgerService(db *gorm.DB) *LedgerService {
	return &LedgerService{db: db}
}

// ProcessAndSaveStatement parses CSV, flags recurring operations, and saves to database atomically
func (s *LedgerService) ProcessAndSaveStatement(userID uint64, filename string, csvBytes []byte) error {
	ops, err := ParseCSV(csvBytes, userID)
	if err != nil {
		return fmt.Errorf("CSV parse error: %w", err)
	}

	if len(ops) == 0 {
		return errors.New("no valid transaction rows in statement")
	}

	// Find as_of (max date)
	var asOf time.Time
	for _, op := range ops {
		if op.Date.After(asOf) {
			asOf = op.Date
		}
	}

	// Identify recurring chains and flag Recurring=true in-place
	DetectAndProjectRecurring(ops, asOf)

	// Execute within database transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Remove previous operations and current forecast for this user
		if err := tx.Where("user_id = ?", userID).Delete(&model.Operation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastDay{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastAlert{}).Error; err != nil {
			return err
		}

		// Save statement upload log
		statement := model.Statement{
			UserID:     userID,
			Filename:   filename,
			UploadedAt: time.Now(),
		}
		if err := tx.Create(&statement).Error; err != nil {
			return err
		}

		// Batch insert parsed operations
		if err := tx.CreateInBatches(ops, 500).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetData computes opening balance, continuous history, and recurring payments
func (s *LedgerService) GetData(userID uint64) (*dto.DataResponse, error) {
	var ops []model.Operation
	err := s.db.Where("user_id = ?", userID).Order("date ASC, id ASC").Find(&ops).Error
	if err != nil {
		return nil, err
	}

	if len(ops) == 0 {
		return nil, ErrNoData
	}

	minDate := ops[0].Date
	asOf := ops[0].Date
	var openingBalance int64 = 0

	for _, op := range ops {
		openingBalance += op.Amount
		if op.Date.Before(minDate) {
			minDate = op.Date
		}
		if op.Date.After(asOf) {
			asOf = op.Date
		}
	}

	recurringPayments := DetectAndProjectRecurring(ops, asOf)
	history := BuildContinuousHistory(ops, minDate, asOf)

	return &dto.DataResponse{
		AsOf:              asOf.Format("2006-01-02"),
		OpeningBalance:    openingBalance,
		History:           history,
		RecurringPayments: recurringPayments,
	}, nil
}

// ArchiveForecast copies current forecast days to forecast_history and clears current tables
func (s *LedgerService) ArchiveForecast(userID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var currentDays []model.ForecastDay
		if err := tx.Where("user_id = ?", userID).Find(&currentDays).Error; err != nil {
			return err
		}

		if len(currentDays) > 0 {
			now := time.Now()
			historyItems := make([]model.ForecastHistory, len(currentDays))
			for i, d := range currentDays {
				historyItems[i] = model.ForecastHistory{
					UserID:          d.UserID,
					Date:            d.Date,
					ExpectedBalance: d.ExpectedBalance,
					BalanceLow:      d.BalanceLow,
					BalanceHigh:     d.BalanceHigh,
					Inflow:          d.Inflow,
					Outflow:         d.Outflow,
					Rough:           d.Rough,
					ArchivedAt:      now,
				}
			}

			if err := tx.CreateInBatches(historyItems, 500).Error; err != nil {
				return err
			}
		}

		// Delete from current tables
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastDay{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastAlert{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// SaveForecast clears previous current forecast and persists the new 30-day forecast and alerts
func (s *LedgerService) SaveForecast(userID uint64, payload dto.ForecastPayload) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing current forecast
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastDay{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.ForecastAlert{}).Error; err != nil {
			return err
		}

		// Insert new days
		if len(payload.Days) > 0 {
			days := make([]model.ForecastDay, len(payload.Days))
			for i, d := range payload.Days {
				parsedDate, err := time.Parse("2006-01-02", d.Date)
				if err != nil {
					parsedDate, _ = parseDate(d.Date)
				}
				days[i] = model.ForecastDay{
					UserID:          userID,
					Date:            parsedDate,
					ExpectedBalance: d.ExpectedBalance,
					BalanceLow:      d.BalanceLow,
					BalanceHigh:     d.BalanceHigh,
					Inflow:          d.Inflow,
					Outflow:         d.Outflow,
					Rough:           payload.Rough,
				}
			}
			if err := tx.CreateInBatches(days, 500).Error; err != nil {
				return err
			}
		}

		// Insert alerts
		if len(payload.Alerts) > 0 {
			alerts := make([]model.ForecastAlert, len(payload.Alerts))
			for i, a := range payload.Alerts {
				parsedDate, err := time.Parse("2006-01-02", a.Date)
				if err != nil {
					parsedDate, _ = parseDate(a.Date)
				}
				alerts[i] = model.ForecastAlert{
					UserID:    userID,
					Date:      parsedDate,
					Shortfall: a.Shortfall,
					Chance:    a.Chance,
					Reason:    a.Reason,
				}
			}
			if err := tx.CreateInBatches(alerts, 500).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetForecast retrieves saved forecast days and alerts or returns empty slice if not calculated yet
func (s *LedgerService) GetForecast(userID uint64) (*dto.ForecastPayload, error) {
	var days []model.ForecastDay
	if err := s.db.Where("user_id = ?", userID).Order("date ASC").Find(&days).Error; err != nil {
		return nil, err
	}

	var alerts []model.ForecastAlert
	if err := s.db.Where("user_id = ?", userID).Order("date ASC").Find(&alerts).Error; err != nil {
		return nil, err
	}

	if len(days) == 0 {
		return &dto.ForecastPayload{
			Days:   []dto.ForecastDayDTO{},
			Alerts: []dto.AlertDTO{},
			Rough:  false,
		}, nil
	}

	dayDTOs := make([]dto.ForecastDayDTO, len(days))
	rough := false
	for i, d := range days {
		dayDTOs[i] = dto.ForecastDayDTO{
			Date:            d.Date.Format("2006-01-02"),
			ExpectedBalance: d.ExpectedBalance,
			BalanceLow:      d.BalanceLow,
			BalanceHigh:     d.BalanceHigh,
			Inflow:          d.Inflow,
			Outflow:         d.Outflow,
		}
		if d.Rough {
			rough = true
		}
	}

	alertDTOs := make([]dto.AlertDTO, len(alerts))
	for i, a := range alerts {
		alertDTOs[i] = dto.AlertDTO{
			Date:      a.Date.Format("2006-01-02"),
			Shortfall: a.Shortfall,
			Chance:    a.Chance,
			Reason:    a.Reason,
		}
	}

	return &dto.ForecastPayload{
		Days:   dayDTOs,
		Alerts: alertDTOs,
		Rough:  rough,
	}, nil
}
