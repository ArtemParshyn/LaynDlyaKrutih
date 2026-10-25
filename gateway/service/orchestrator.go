package service

import (
	"context"
	"fmt"
	"gateway/client"
	"gateway/dto"
	"log"
	"math"
	"strings"
	"sort"
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

// Recalculate starts the documented Prophet training flow. The ML service
// trains from the statement CSV and does not accept the old JSON forecast
// package or future planned payments.
func (s *OrchestratorService) Recalculate(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	view, err := s.statementView(ctx, userID)
	if err != nil { return nil, err }
	view.Days = []dto.ForecastDay{}
	view.Alerts = []dto.Alert{}
	view.Training = s.startTraining(ctx, userID)
	return view, nil
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
	view.Training = s.startTrainingCSV(ctx, userID, fileData)
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
	s.enrichWithAnalytics(ctx, userID, ready)
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
func (s *OrchestratorService) CalculateScenario(ctx context.Context, userID uint, whatIfEvents []dto.ScheduledPayment, changes []dto.ScenarioOperation) (*dto.ForecastResponse, error) {
	ops, err := s.ledger.Operations(ctx, userID)
	if err != nil { return nil, fmt.Errorf("failed to fetch operations: %w", err) }
	base, err := s.GetForecast(ctx, userID)
	if err != nil { return nil, fmt.Errorf("failed to fetch base forecast: %w", err) }
	// Replace selected rows in a temporary statement only; Ledger and the
	// user's model are untouched. ID is the stable identity from Ledger.
	temporary := append([]dto.Operation(nil), ops...)
	actualEnd := ""
	if len(ops) > 0 {
		actualEnd = ops[len(ops)-1].Date
	}
	// For a future intervention, turn the base forecast up to the intervention
	// date into temporary daily movements. Prophet must see the whole path:
	// actual statement -> original forecast -> hypothetical operation.
	forecastDates := make(map[string]bool)
	for _, day := range base.Days {
		forecastDates[day.Date] = true
	}
	lastBalance := int64(0)
	if len(base.History) > 0 {
		lastBalance = base.History[len(base.History)-1].Balance
	}
	lastDate := actualEnd
	addForecastUntil := func(until string) {
		for _, day := range base.Days {
			if day.Date <= lastDate || day.Date > until || !forecastDates[day.Date] {
				continue
			}
			balance := day.ExpectedBalance
			temporary = append(temporary, dto.Operation{
				Date: day.Date, Amount: balance - lastBalance,
				Description: "Базовый прогноз (временная история)", Category: "Сценарий",
				IsIncome: balance >= lastBalance,
			})
			lastBalance = balance
			lastDate = day.Date
		}
	}
	for _, change := range changes {
		if change.Amount <= 0 { continue }
		amount := change.Amount
		if !change.IsIncome { amount = -amount }
		if change.ID != 0 {
			found := false
			for i := range temporary {
				if temporary[i].ID != change.ID { continue }
				temporary[i].Date = change.Date
				temporary[i].Amount = amount
				temporary[i].IsIncome = change.IsIncome
				if change.Description != "" { temporary[i].Description = change.Description }
				found = true
				break
			}
			if found { continue }
		}
		if change.Date > actualEnd {
			addForecastUntil(change.Date)
		}
		temporary = append(temporary, dto.Operation{Date: change.Date, Amount: amount, IsIncome: change.IsIncome, Description: change.Description, Category: "Сценарий"})
	}
	for _, event := range whatIfEvents {
		if event.Amount <= 0 { continue }
		amount := event.Amount
		if !event.IsIncome { amount = -amount }
		if event.Date > actualEnd {
			addForecastUntil(event.Date)
		}
		temporary = append(temporary, dto.Operation{Date: event.Date, Amount: amount, IsIncome: event.IsIncome, Description: "Сценарное событие", Category: event.Category})
	}
	sort.SliceStable(temporary, func(i, j int) bool { return temporary[i].Date < temporary[j].Date })
	scenarioID := fmt.Sprintf("scenario-%d-%d", userID, time.Now().UnixNano())
	train, err := s.prophet.Train(ctx, scenarioID, prophetCSV(temporary))
	if err != nil { return nil, fmt.Errorf("scenario training failed: %w", err) }
	defer s.prophet.Delete(context.Background(), scenarioID)
	deadline := time.Now().Add(55 * time.Second)
	for {
		job, jobErr := s.prophet.Job(ctx, train.JobID)
		if jobErr == nil && job.Status == "succeeded" { break }
		if jobErr == nil && job.Status == "failed" { return nil, fmt.Errorf("scenario training failed: %s", job.Error) }
		if time.Now().After(deadline) { return nil, fmt.Errorf("scenario training is still running") }
		select { case <-ctx.Done(): return nil, ctx.Err(); case <-time.After(time.Second): }
	}
	raw, err := s.prophet.Forecast(ctx, scenarioID)
	if err != nil { return nil, fmt.Errorf("scenario forecast failed: %w", err) }
	result := mapProphet(raw)
	result.History = base.History
	result.Operations = base.Operations
	result.RecurringPayments = base.RecurringPayments
	result.Summary = "Сценарий: модель переобучена на временной копии операций"
	return result, nil
}

func (s *OrchestratorService) enrichWithAnalytics(ctx context.Context, userID uint, response *dto.ForecastResponse) {
	id := companyID(userID)
	if report, err := s.prophet.Analytics(ctx, id, "categories", ""); err == nil { response.CategoryReport = report }
	if streams, err := s.prophet.Analytics(ctx, id, "recurring", "status=active"); err == nil { response.RecurringStreams = streams }
	if calendar, err := s.prophet.Analytics(ctx, id, "calendar", "days=30"); err == nil { response.PaymentCalendar = calendar }
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
	// Analytics is stored by ML outside Ledger, so refresh it on reads instead
	// of losing it when the forecast is loaded from the ledger database.
	s.enrichWithAnalytics(ctx, userID, resp)
	// The scenario editor needs the complete statement, not only the ten rows
	// shown in the dashboard's "recent operations" list.
	resp.Operations, err = s.ledger.Operations(ctx, userID)
	if err != nil {
		resp.Operations = []dto.Operation{}
	}
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
	return s.startTrainingCSV(ctx, userID, prophetCSV(ops))
}

// startTrainingCSV forwards the original statement whenever possible. The
// Prophet service also runs categorization during /train, and categorization
// needs counterparty_inn, counterparty_name and payment_purpose. Rebuilding a
// CSV from the ledger's compact operation DTO silently loses those columns.
func (s *OrchestratorService) startTrainingCSV(ctx context.Context, userID uint, csv []byte) *dto.Training {
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
		if op.BalanceEOD != nil { balance = *op.BalanceEOD }
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
			Reason:    raw.CashGap.Reason,
			RequiredPayment: rublesToKopecks(raw.CashGap.RequiredPayment),
			ShortageStressed: rublesToKopecks(raw.CashGap.ShortageStressed),
		})
	}
	return &dto.ForecastResponse{
		Days:          days,
		Alerts:        alerts,
		Summary:       raw.Summary,
		Risks:         raw.Risks,
		Explanations:  raw.Explanations,
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
