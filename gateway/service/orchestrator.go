package service

import (
	"context"
	"fmt"
	"gateway/client"
	"gateway/dto"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

type trainJob struct {
	id     string
	status string
	err    string
}

type OrchestratorService struct {
	ledger   *client.LedgerClient
	planning *client.PlanningClient
	forecast *client.ForecastClient
	prophet  *client.ProphetClient

	mu   sync.Mutex
	jobs map[uint]trainJob
}

func NewOrchestratorService(
	ledger *client.LedgerClient,
	planning *client.PlanningClient,
	forecast *client.ForecastClient,
	prophet *client.ProphetClient,
) *OrchestratorService {
	return &OrchestratorService{
		ledger:   ledger,
		planning: planning,
		forecast: forecast,
		prophet:  prophet,
		jobs:     map[uint]trainJob{},
	}
}

// Recalculate performs the full forecasting pipeline:
// 1. Asks Ledger to archive the previous forecast line to history.
// 2. Fetches opening balance, history, and recurring payments from Ledger.
// 3. Fetches future planned events from Planning.
// 4. Glues them into a single ForecastRequest and calls Forecast ML service.
// 5. Sends the 30-day forecast and alerts to Ledger to be saved.
// 6. Returns the forecast response (including rough: true as-is).
func (s *OrchestratorService) Recalculate(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	// 1. Copy old line to history in ledger (if any exists)
	if err := s.ledger.ArchiveForecast(ctx, userID); err != nil {
		log.Printf("Notice: archive forecast for user %d returned: %v (continuing)", userID, err)
	}

	// 2. Get data from ledger
	ledgerData, err := s.ledger.GetData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ledger data: %w", err)
	}

	// 3. Get planned events
	plannedEvents, err := s.planning.GetEvents(ctx, userID)
	if err != nil {
		log.Printf("Notice: could not fetch planning events for user %d: %v", userID, err)
		plannedEvents = []dto.ScheduledPayment{}
	}

	// 4. Combine scheduled payments
	scheduled := make([]dto.ScheduledPayment, 0, len(ledgerData.RecurringPayments)+len(ledgerData.Scheduled)+len(plannedEvents))
	scheduled = append(scheduled, ledgerData.RecurringPayments...)
	scheduled = append(scheduled, ledgerData.Scheduled...)
	scheduled = append(scheduled, plannedEvents...)

	forecastReq := dto.ForecastRequest{
		AsOf:           ledgerData.AsOf,
		OpeningBalance: ledgerData.OpeningBalance,
		History:        ledgerData.History,
		Scheduled:      scheduled,
	}

	// 5. Send package to forecast service
	forecastResp, err := s.forecast.Calculate(ctx, &forecastReq)
	if err != nil {
		return nil, fmt.Errorf("forecast calculation failed: %w", err)
	}

	// 6. Save calculated days and alerts to ledger
	if err := s.ledger.SaveForecast(ctx, userID, forecastResp); err != nil {
		return nil, fmt.Errorf("failed to save forecast to ledger: %w", err)
	}

	// Read back after the save. A hand-entered past payment changes the actual
	// balance, and that line is attached by the ledger on read, not by the forecast.
	if stored, err := s.ledger.GetForecast(ctx, userID); err == nil && stored != nil {
		forecastResp.History = stored.History
		forecastResp.Operations = stored.Operations
		forecastResp.RarePayments = stored.RarePayments
	}

	forecastResp.RecurringPayments = ledgerData.RecurringPayments
	if forecastResp.RecurringPayments == nil {
		forecastResp.RecurringPayments = []dto.ScheduledPayment{}
	}
	if forecastResp.Operations == nil {
		forecastResp.Operations = s.recentOperations(ctx, userID)
	}
	if forecastResp.History == nil {
		forecastResp.History = s.balanceHistory(ctx, userID)
	}

	// Append user-defined recurring planned events to recurring payments
	if rawEvents, err := s.planning.GetEventsList(ctx, userID); err == nil {
		for _, e := range rawEvents {
			if e.Recurring {
				forecastResp.RecurringPayments = append(forecastResp.RecurringPayments, dto.ScheduledPayment{
					Date:     e.Date,
					Amount:   e.Amount,
					IsIncome: e.IsIncome,
					Category: e.Category,
				})
			}
		}
	}

	return forecastResp, nil
}

// UploadStatementAndRecalculate sends CSV to Ledger and triggers forecast recalculation
func (s *OrchestratorService) UploadStatementAndRecalculate(ctx context.Context, userID uint, filename string, fileData []byte) (*dto.ForecastResponse, error) {
	return s.storeStatement(ctx, userID, filename, fileData, false)
}

// AppendStatementAndRecalculate keeps the open statement and adds another CSV.
// Plans stay, because this is the same dataset, not a new one.
func (s *OrchestratorService) AppendStatementAndRecalculate(ctx context.Context, userID uint, filename string, fileData []byte) (*dto.ForecastResponse, error) {
	return s.storeStatement(ctx, userID, filename, fileData, true)
}

func (s *OrchestratorService) storeStatement(ctx context.Context, userID uint, filename string, fileData []byte, appendFile bool) (*dto.ForecastResponse, error) {
	var err error
	if appendFile {
		err = s.ledger.AppendStatement(ctx, userID, filename, fileData)
	} else {
		err = s.ledger.UploadStatement(ctx, userID, filename, fileData)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to upload statement to ledger: %w", err)
	}
	// A replacement file starts clean. An added file keeps plans of this dataset.
	if !appendFile {
		if err := s.planning.ClearEvents(ctx, userID); err != nil {
			log.Printf("Notice: could not clear plans for user %d: %v", userID, err)
		}
	}

	view, err := s.statementView(ctx, userID)
	if err != nil {
		return nil, err
	}
	// The chart shows history immediately. Prophet fits in the background,
	// and the forecast line appears only when the user asks to process it.
	view.Days = []dto.ForecastDay{}
	view.Alerts = []dto.Alert{}
	view.Training = s.startTraining(ctx, userID)
	return view, nil
}

// Process reads the Prophet model trained for this user. If it is still
// fitting, the statement comes back without a forecast line.
func (s *OrchestratorService) Process(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	view, err := s.statementView(ctx, userID)
	if err != nil {
		return nil, err
	}
	job := s.refreshJob(ctx, userID)
	if job.status == "queued" || job.status == "running" {
		view.Days = []dto.ForecastDay{}
		view.Alerts = []dto.Alert{}
		view.Training = &dto.Training{Status: job.status}
		return view, nil
	}
	if job.status == "failed" {
		view.Days = []dto.ForecastDay{}
		view.Alerts = []dto.Alert{}
		view.Training = &dto.Training{Status: "failed", Error: job.err}
		return view, nil
	}

	raw, err := s.prophet.Forecast(ctx, companyID(userID))
	if err != nil {
		return nil, fmt.Errorf("прогноз ещё не готов: %w", err)
	}
	ready := mapProphet(raw)
	ready.History = view.History
	ready.Operations = view.Operations
	ready.RarePayments = view.RarePayments
	ready.RecurringPayments = view.RecurringPayments
	ready.Training = &dto.Training{Status: "succeeded"}
	if err := s.ledger.SaveForecast(ctx, userID, ready); err != nil {
		return nil, fmt.Errorf("failed to save forecast to ledger: %w", err)
	}
	s.setJob(userID, trainJob{id: job.id, status: "succeeded"})
	return ready, nil
}

// AddOperation routes a hand-entered payment and rebuilds the forecast.
// A date after the statement end is a plan: it moves only the expected line.
// A date on or before that end corrects the actual balance, but stays out of
// the ordinary daily flow, so one payment cannot become a daily expense.
// A new statement still replaces only bank rows.
func (s *OrchestratorService) AddOperation(ctx context.Context, userID uint, op dto.ManualOperation) (*dto.ForecastResponse, error) {
	ledgerData, err := s.ledger.GetData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ledger data: %w", err)
	}
	asOf, err := time.Parse("2006-01-02", ledgerData.AsOf)
	if err != nil {
		return nil, fmt.Errorf("ledger returned an invalid statement date: %w", err)
	}
	opDate, err := time.Parse("2006-01-02", strings.TrimSpace(op.Date))
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}

	category := strings.TrimSpace(op.Category)
	if category == "" {
		category = "Прочее"
	}
	if opDate.After(asOf) {
		event := dto.PlanEvent{
			Date:        opDate.Format("2006-01-02"),
			Amount:      op.Amount,
			IsIncome:    op.IsIncome,
			Category:    category,
			Description: strings.TrimSpace(op.Description),
			Recurring:   op.Recurring,
		}
		if err := s.planning.CreateEvent(ctx, userID, event); err != nil {
			return nil, fmt.Errorf("failed to save planned operation: %w", err)
		}
	} else {
		op.Date = opDate.Format("2006-01-02")
		op.Category = category
		op.Recurring = false
		if err := s.ledger.AddManualOperation(ctx, userID, op); err != nil {
			return nil, fmt.Errorf("failed to save manual operation: %w", err)
		}
	}

	s.startTraining(ctx, userID)
	view, err := s.statementView(ctx, userID)
	if err != nil {
		return nil, err
	}
	view.Days = []dto.ForecastDay{}
	view.Alerts = []dto.Alert{}
	view.Training = s.trainingOf(userID)
	return view, nil
}

// CreateEventAndRecalculate saves a new event in Planning and triggers forecast recalculation
func (s *OrchestratorService) CreateEventAndRecalculate(ctx context.Context, userID uint, event dto.PlanEvent) (*dto.ForecastResponse, error) {
	if err := s.planning.CreateEvent(ctx, userID, event); err != nil {
		return nil, fmt.Errorf("failed to save event in planning: %w", err)
	}

	return s.Recalculate(ctx, userID)
}

// CalculateScenario computes an alternative "what if" second line without saving or overwriting the base forecast
func (s *OrchestratorService) CalculateScenario(ctx context.Context, userID uint, whatIfEvents []dto.ScheduledPayment) (*dto.ForecastResponse, error) {
	ledgerData, err := s.ledger.GetData(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ledger data: %w", err)
	}

	plannedEvents, err := s.planning.GetEvents(ctx, userID)
	if err != nil {
		log.Printf("Notice: could not fetch planning events for user %d: %v", userID, err)
		plannedEvents = []dto.ScheduledPayment{}
	}

	scheduled := make([]dto.ScheduledPayment, 0, len(ledgerData.RecurringPayments)+len(ledgerData.Scheduled)+len(plannedEvents)+len(whatIfEvents))
	scheduled = append(scheduled, ledgerData.RecurringPayments...)
	scheduled = append(scheduled, ledgerData.Scheduled...)
	scheduled = append(scheduled, plannedEvents...)
	scheduled = append(scheduled, whatIfEvents...)

	forecastReq := dto.ForecastRequest{
		AsOf:           ledgerData.AsOf,
		OpeningBalance: ledgerData.OpeningBalance,
		History:        ledgerData.History,
		Scheduled:      scheduled,
	}

	// Calculate scenario line only - do NOT save to ledger
	scenarioResp, err := s.forecast.Calculate(ctx, &forecastReq)
	if err != nil {
		return nil, fmt.Errorf("scenario forecast calculation failed: %w", err)
	}

	scenarioResp.RecurringPayments = ledgerData.RecurringPayments
	if scenarioResp.RecurringPayments == nil {
		scenarioResp.RecurringPayments = []dto.ScheduledPayment{}
	}

	return scenarioResp, nil
}

// GetForecast reads the already calculated days and alerts from Ledger without recalculating
func (s *OrchestratorService) GetForecast(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	resp, err := s.ledger.GetForecast(ctx, userID)
	if err != nil {
		return nil, err
	}
	if resp.RecurringPayments == nil {
		resp.RecurringPayments = []dto.ScheduledPayment{}
	}
	resp.Operations = s.recentOperations(ctx, userID)
	resp.History = s.balanceHistory(ctx, userID)

	ledgerData, err := s.ledger.GetData(ctx, userID)
	if err == nil && ledgerData != nil && len(ledgerData.RecurringPayments) > 0 {
		resp.RecurringPayments = ledgerData.RecurringPayments
	}

	// Append user-defined recurring planned events
	if rawEvents, err := s.planning.GetEventsList(ctx, userID); err == nil {
		for _, e := range rawEvents {
			if e.Recurring {
				resp.RecurringPayments = append(resp.RecurringPayments, dto.ScheduledPayment{
					Date:     e.Date,
					Amount:   e.Amount,
					IsIncome: e.IsIncome,
					Category: e.Category,
				})
			}
		}
	}

	return resp, nil
}

// Forget removes the statement, plans and model stored under this key.
func (s *OrchestratorService) Forget(ctx context.Context, userID uint) error {
	if err := s.ledger.Wipe(ctx, userID); err != nil {
		return err
	}
	if err := s.planning.ClearEvents(ctx, userID); err != nil {
		return err
	}
	if err := s.prophet.Delete(ctx, companyID(userID)); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.jobs, userID)
	s.mu.Unlock()
	return nil
}

func companyID(userID uint) string {
	return fmt.Sprintf("u%d", userID)
}

func (s *OrchestratorService) statementView(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	stored, err := s.ledger.GetForecast(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stored.Days == nil {
		stored.Days = []dto.ForecastDay{}
	}
	if stored.Alerts == nil {
		stored.Alerts = []dto.Alert{}
	}
	if stored.Operations == nil {
		stored.Operations = []dto.Operation{}
	}
	if stored.History == nil {
		stored.History = []dto.BalancePoint{}
	}
	if stored.RecurringPayments == nil {
		stored.RecurringPayments = []dto.ScheduledPayment{}
	}
	return stored, nil
}

func (s *OrchestratorService) startTraining(ctx context.Context, userID uint) *dto.Training {
	ops, err := s.ledger.Operations(ctx, userID)
	if err != nil {
		training := &dto.Training{Status: "failed", Error: err.Error()}
		s.setJob(userID, trainJob{status: "failed", err: training.Error})
		return training
	}
	csv := prophetCSV(ops)
	train, err := s.prophet.Train(ctx, companyID(userID), csv)
	if err != nil {
		training := &dto.Training{Status: "failed", Error: err.Error()}
		s.setJob(userID, trainJob{status: "failed", err: training.Error})
		return training
	}
	s.setJob(userID, trainJob{id: train.JobID, status: train.Status})
	return &dto.Training{Status: train.Status}
}

func (s *OrchestratorService) refreshJob(ctx context.Context, userID uint) trainJob {
	job := s.jobOf(userID)
	if job.id == "" || job.status == "succeeded" || job.status == "failed" {
		return job
	}
	remote, err := s.prophet.Job(ctx, job.id)
	if err != nil {
		return job
	}
	job.status = remote.Status
	job.err = remote.Error
	s.setJob(userID, job)
	return job
}

func (s *OrchestratorService) trainingOf(userID uint) *dto.Training {
	job := s.jobOf(userID)
	if job.status == "" {
		return nil
	}
	return &dto.Training{Status: job.status, Error: job.err}
}

func (s *OrchestratorService) jobOf(userID uint) trainJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[userID]
}

func (s *OrchestratorService) setJob(userID uint, job trainJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[userID] = job
}

// prophetCSV is the statement shape Prophet trains on. Amounts are rubles.
// The running balance is the end-of-day fact, because the ledger stores
// movements and not the bank's own balance column.
func prophetCSV(ops []dto.Operation) []byte {
	var b strings.Builder
	b.WriteString("operation_id;operation_date;amount;direction;balance_eod\n")
	var balance int64
	for i, op := range ops {
		balance += op.Amount
		amount := op.Amount
		if amount < 0 {
			amount = -amount
		}
		direction := "Списание"
		if op.Amount > 0 {
			direction = "Поступление"
		}
		fmt.Fprintf(&b, "%d;%s;%.2f;%s;%.2f\n", i+1, op.Date, float64(amount)/100, direction, float64(balance)/100)
	}
	return []byte(b.String())
}

func mapProphet(raw *client.ProphetForecast) *dto.ForecastResponse {
	days := make([]dto.ForecastDay, 0, len(raw.Forecast))
	for _, day := range raw.Forecast {
		days = append(days, dto.ForecastDay{
			Date:            day.Date,
			ExpectedBalance: rublesToKopecks(day.Balance),
			BalanceLow:      rublesToKopecks(day.BalanceLower),
			BalanceHigh:     rublesToKopecks(day.BalanceUpper),
		})
	}
	alerts := []dto.Alert{}
	if raw.CashGap != nil {
		alerts = append(alerts, dto.Alert{
			Date:      raw.CashGap.Date,
			Shortfall: rublesToKopecks(raw.CashGap.Shortage),
			Chance:    raw.CashGap.Probability,
			Reason:    "Prophet: остаток может уйти ниже нуля",
		})
	}
	return &dto.ForecastResponse{
		Days:    days,
		Alerts:  alerts,
		Summary: raw.Summary,
	}
}

func rublesToKopecks(rubles float64) int64 {
	return int64(math.Round(rubles * 100))
}

func (s *OrchestratorService) balanceHistory(ctx context.Context, userID uint) []dto.BalancePoint {
	stored, err := s.ledger.GetForecast(ctx, userID)
	if err != nil || stored == nil || len(stored.History) == 0 {
		return []dto.BalancePoint{}
	}
	return stored.History
}

func (s *OrchestratorService) recentOperations(ctx context.Context, userID uint) []dto.Operation {
	stored, err := s.ledger.GetForecast(ctx, userID)
	if err != nil || stored == nil || len(stored.Operations) == 0 {
		return []dto.Operation{}
	}
	return stored.Operations
}

// GetEventsList retrieves raw user-created events from Planning service
func (s *OrchestratorService) GetEventsList(ctx context.Context, userID uint) ([]dto.PlanEvent, error) {
	return s.planning.GetEventsList(ctx, userID)
}

// DeleteEventAndRecalculate deletes a planned event and triggers forecast recalculation
func (s *OrchestratorService) DeleteEventAndRecalculate(ctx context.Context, userID uint, eventID uint64) (*dto.ForecastResponse, error) {
	if err := s.planning.DeleteEvent(ctx, userID, eventID); err != nil {
		return nil, fmt.Errorf("failed to delete planned event: %w", err)
	}
	return s.Recalculate(ctx, userID)
}
