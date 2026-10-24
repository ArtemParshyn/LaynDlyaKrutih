package service

import (
	"context"
	"fmt"
	"gateway/client"
	"gateway/dto"
	"log"
)

type OrchestratorService struct {
	ledger   *client.LedgerClient
	planning *client.PlanningClient
	forecast *client.ForecastClient
}

func NewOrchestratorService(
	ledger *client.LedgerClient,
	planning *client.PlanningClient,
	forecast *client.ForecastClient,
) *OrchestratorService {
	return &OrchestratorService{
		ledger:   ledger,
		planning: planning,
		forecast: forecast,
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

	return forecastResp, nil
}

// UploadStatementAndRecalculate sends CSV to Ledger and triggers forecast recalculation
func (s *OrchestratorService) UploadStatementAndRecalculate(ctx context.Context, userID uint, filename string, fileData []byte) (*dto.ForecastResponse, error) {
	if err := s.ledger.UploadStatement(ctx, userID, filename, fileData); err != nil {
		return nil, fmt.Errorf("failed to upload statement to ledger: %w", err)
	}

	return s.Recalculate(ctx, userID)
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

	return scenarioResp, nil
}

// GetForecast reads the already calculated days and alerts from Ledger without recalculating
func (s *OrchestratorService) GetForecast(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	return s.ledger.GetForecast(ctx, userID)
}
