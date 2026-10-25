package service

import (
	"errors"
	"fmt"
	"ledger/dto"
	"ledger/model"
	"sort"
	"strconv"
	"strings"
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

	return s.replaceStatement(userID, filename, ops, true)
}

// AppendStatement adds another CSV to the statement already open.
// Rows that repeat the same day, amount and description are skipped, so
// uploading the same file twice does not double the balance.
func (s *LedgerService) AppendStatement(userID uint64, filename string, csvBytes []byte) error {
	incoming, err := ParseCSV(csvBytes, userID)
	if err != nil {
		return fmt.Errorf("CSV parse error: %w", err)
	}
	if len(incoming) == 0 {
		return errors.New("no valid transaction rows in statement")
	}

	var stored []model.Operation
	if err := s.db.Where("user_id = ?", userID).Find(&stored).Error; err != nil {
		return err
	}
	if len(stored) == 0 {
		return s.ProcessAndSaveStatement(userID, filename, csvBytes)
	}

	seen := make(map[string]int, len(stored))
	for _, op := range stored {
		seen[operationKey(op)]++
	}
	fresh := make([]model.Operation, 0, len(incoming))
	for _, op := range incoming {
		key := operationKey(op)
		if seen[key] > 0 {
			seen[key]--
			continue
		}
		fresh = append(fresh, op)
	}

	combined := append(stored, fresh...)
	var asOf time.Time
	for _, op := range combined {
		op.Recurring = false
		if op.Date.After(asOf) {
			asOf = op.Date
		}
	}
	for i := range combined {
		combined[i].Recurring = false
	}
	DetectAndProjectRecurring(combined, asOf)
	return s.replaceStatement(userID, filename, combined, false)
}

func operationKey(op model.Operation) string {
	return op.Date.Format("2006-01-02") + "|" + strconv.FormatInt(op.Amount, 10) + "|" + strings.TrimSpace(op.Description)
}

func (s *LedgerService) replaceStatement(userID uint64, filename string, ops []model.Operation, resetManual bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Remove previous operations and current forecast for this user
		if err := tx.Where("user_id = ?", userID).Delete(&model.Operation{}).Error; err != nil {
			return err
		}
		// A replacement file drops hand-entered payments. Adding a second CSV
		// keeps them: they belong to the dataset that is still open.
		if resetManual {
			if err := tx.Where("user_id = ?", userID).Delete(&model.ManualOperation{}).Error; err != nil {
				return err
			}
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

// Wipe drops every row that belongs to one pulse. The gateway calls it with
// the pulse key, not with the account id, so other graphs stay.
func (s *LedgerService) Wipe(userID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tables := []any{
			&model.Operation{},
			&model.ManualOperation{},
			&model.ForecastDay{},
			&model.ForecastAlert{},
			&model.ForecastHistory{},
			&model.Statement{},
		}
		for _, table := range tables {
			if err := tx.Where("user_id = ?", userID).Delete(table).Error; err != nil {
				return err
			}
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

	// A hand-entered past payment moves the balance the forecast starts from.
	// It stays out of history: that series is the ordinary daily flow.
	manuals, err := s.manualOperations(userID)
	if err != nil {
		return nil, err
	}
	for _, op := range manuals {
		if op.Date.After(asOf) {
			continue
		}
		openingBalance += op.Amount
	}

	return &dto.DataResponse{
		AsOf:              asOf.Format("2006-01-02"),
		OpeningBalance:    openingBalance,
		History:           history,
		RecurringPayments: recurringPayments,
	}, nil
}

// AddManualOperation stores a payment the bank statement does not contain.
// The date must not be after the statement end: later dates are plans, and a
// plan must not change the actual balance.
func (s *LedgerService) AddManualOperation(userID uint64, req dto.ManualOperationRequest) error {
	asOf, err := s.statementEnd(userID)
	if err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.Date))
	if err != nil {
		parsed, err = parseDate(req.Date)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
	}
	if parsed.After(asOf) {
		return errors.New("manual operation date is after the statement")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be positive")
	}

	amount := req.Amount
	if !req.IsIncome {
		amount = -amount
	}
	description := strings.TrimSpace(req.Description)
	category := strings.TrimSpace(req.Category)
	if category == "" || strings.EqualFold(category, "прочее") {
		category = ClassifyCategory(description, req.IsIncome)
	}

	return s.db.Create(&model.ManualOperation{
		UserID:      userID,
		Date:        parsed,
		Amount:      amount,
		Category:    category,
		Description: description,
	}).Error
}

func (s *LedgerService) statementEnd(userID uint64) (time.Time, error) {
	var op model.Operation
	err := s.db.Where("user_id = ?", userID).Order("date DESC").First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, ErrNoData
	}
	if err != nil {
		return time.Time{}, err
	}
	return op.Date, nil
}

func (s *LedgerService) manualOperations(userID uint64) ([]model.ManualOperation, error) {
	var ops []model.ManualOperation
	if err := s.db.Where("user_id = ?", userID).Order("date ASC, id ASC").Find(&ops).Error; err != nil {
		return nil, err
	}
	return ops, nil
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
			Days:       []dto.ForecastDayDTO{},
			Alerts:     []dto.AlertDTO{},
			Rough:      false,
			Operations: s.recentOperations(userID, 10),
			History:    s.balanceHistory(userID),
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
		Days:         dayDTOs,
		Alerts:       alertDTOs,
		Rough:        rough,
		Operations:   s.recentOperations(userID, 10),
		History:      s.balanceHistory(userID),
		RarePayments: s.rarePayments(userID),
	}, nil
}

func (s *LedgerService) rarePayments(userID uint64) []dto.RarePayment {
	var ops []model.Operation
	if err := s.db.Where("user_id = ?", userID).Find(&ops).Error; err != nil || len(ops) == 0 {
		return []dto.RarePayment{}
	}
	return rarePayments(ops)
}

func (s *LedgerService) balanceHistory(userID uint64) []dto.BalancePoint {
	var ops []model.Operation
	if err := s.db.Where("user_id = ?", userID).Order("date ASC, id ASC").Find(&ops).Error; err != nil || len(ops) == 0 {
		return []dto.BalancePoint{}
	}
	manuals, _ := s.manualOperations(userID)
	type movement struct {
		date   time.Time
		amount int64
		balanceEOD *int64
	}
	rows := make([]movement, 0, len(ops)+len(manuals))
	asOf := ops[len(ops)-1].Date
	for _, op := range ops {
		// A bank statement may carry the actual balance at end of day. Keep
		// that anchor: summing all movements from zero produces a fake balance
		// and makes the historical graph diverge from the source CSV.
		rows = append(rows, movement{date: op.Date, amount: op.Amount, balanceEOD: op.BalanceEOD})
		if op.Date.After(asOf) {
			asOf = op.Date
		}
	}
	for _, op := range manuals {
		if op.Date.After(asOf) {
			continue
		}
		rows = append(rows, movement{date: op.Date, amount: op.Amount})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].date.Before(rows[j].date)
	})

	var balance int64
	byDate := make([]dto.BalancePoint, 0)
	var current string
	for _, op := range rows {
		balance += op.amount
		if op.balanceEOD != nil {
			balance = *op.balanceEOD
		}
		day := op.date.Format("2006-01-02")
		if day != current {
			byDate = append(byDate, dto.BalancePoint{Date: day, Balance: balance})
			current = day
			continue
		}
		byDate[len(byDate)-1].Balance = balance
	}
	if len(byDate) > 90 {
		byDate = byDate[len(byDate)-90:]
	}
	return byDate
}

// Operations returns every stored movement, oldest first. The gateway turns
// this into the CSV Prophet trains on. Manual rows are included because they
// already moved the actual balance.
func (s *LedgerService) Operations(userID uint64) ([]dto.OperationDTO, error) {
	var ops []model.Operation
	if err := s.db.Where("user_id = ?", userID).Order("date ASC, id ASC").Find(&ops).Error; err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, ErrNoData
	}
	manuals, err := s.manualOperations(userID)
	if err != nil {
		return nil, err
	}
	asOf := ops[len(ops)-1].Date
	out := make([]dto.OperationDTO, 0, len(ops)+len(manuals))
	for _, op := range ops {
		out = append(out, dto.OperationDTO{
			ID:          op.ID,
			Date:        op.Date.Format("2006-01-02"),
			Amount:      op.Amount,
			Category:    op.Category,
			Description: op.Description,
			IsIncome:    op.Amount >= 0,
			BalanceEOD:  op.BalanceEOD,
		})
	}
	for _, op := range manuals {
		if op.Date.After(asOf) {
			continue
		}
		out = append(out, dto.OperationDTO{
			ID:          op.ID,
			Date:        op.Date.Format("2006-01-02"),
			Amount:      op.Amount,
			Category:    op.Category,
			Description: op.Description,
			IsIncome:    op.Amount >= 0,
			Manual:      true,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date < out[j].Date
	})
	return out, nil
}

func (s *LedgerService) recentOperations(userID uint64, limit int) []dto.OperationDTO {
	var ops []model.Operation
	if err := s.db.Where("user_id = ?", userID).Order("date DESC, id DESC").Limit(limit).Find(&ops).Error; err != nil {
		return []dto.OperationDTO{}
	}
	manuals, _ := s.manualOperations(userID)
	out := make([]dto.OperationDTO, 0, len(ops)+len(manuals))
	for _, op := range ops {
		out = append(out, dto.OperationDTO{
			ID:          op.ID,
			Date:        op.Date.Format("2006-01-02"),
			Amount:      op.Amount,
			Category:    op.Category,
			Description: op.Description,
			IsIncome:    op.Amount >= 0,
		})
	}
	for _, op := range manuals {
		out = append(out, dto.OperationDTO{
			ID:          op.ID,
			Date:        op.Date.Format("2006-01-02"),
			Amount:      op.Amount,
			Category:    op.Category,
			Description: op.Description,
			IsIncome:    op.Amount >= 0,
			Manual:      true,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date == out[j].Date {
			return out[i].Manual && !out[j].Manual
		}
		return out[i].Date > out[j].Date
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
