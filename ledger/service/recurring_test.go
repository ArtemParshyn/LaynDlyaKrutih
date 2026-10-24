package service

import (
	"ledger/model"
	"testing"
	"time"
)

func TestDetectAndProjectRecurring(t *testing.T) {
	d1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC) // 30 days diff
	d3 := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) // 30 days diff
	asOf := d3

	ops := []model.Operation{
		{
			Date:        d1,
			Amount:      -100000,
			Category:    "Подписка",
			Description: "Облачный сервер",
		},
		{
			Date:        d2,
			Amount:      -102000, // 2% diff -> within 5%
			Category:    "Подписка",
			Description: "Облачный сервер",
		},
		{
			Date:        d3,
			Amount:      -99000, // 1% diff -> within 5%
			Category:    "Инфраструктура",
			Description: "Облачный сервер",
		},
		{
			Date:        d2,
			Amount:      -50000,
			Category:    "Обед",
			Description: "Кафе", // non-recurring
		},
	}

	payments := DetectAndProjectRecurring(ops, asOf)

	// Check recurring flag on ops
	if !ops[0].Recurring || !ops[1].Recurring || !ops[2].Recurring {
		t.Errorf("expected cloud server ops to be marked recurring")
	}
	if ops[3].Recurring {
		t.Errorf("cafe op should NOT be recurring")
	}

	// Projected payment should be at d3 + ~30 days (around 2026-10-30)
	if len(payments) == 0 {
		t.Fatalf("expected at least 1 projected payment, got 0")
	}

	p := payments[0]
	if p.Category != "Инфраструктура" {
		t.Errorf("expected last encounter category 'Инфраструктура', got '%s'", p.Category)
	}
	if p.IsIncome {
		t.Errorf("expected is_income false for expenses")
	}
	if p.Amount < 99000 || p.Amount > 102000 {
		t.Errorf("expected average amount around 100333, got %d", p.Amount)
	}
}

func TestBuildContinuousHistory(t *testing.T) {
	d1 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)

	ops := []model.Operation{
		{
			Date:      d1,
			Amount:    50000,
			Recurring: false,
		},
		{
			Date:      d1,
			Amount:    -20000,
			Recurring: false,
		},
		{
			Date:      d3,
			Amount:    30000,
			Recurring: false,
		},
	}

	history := BuildContinuousHistory(ops, d1, d3)

	if len(history) != 3 {
		t.Fatalf("expected 3 days of continuous history, got %d", len(history))
	}

	// Day 1: in 50000, out 20000
	if history[0].Date != "2026-09-01" || history[0].IrregularIn != 50000 || history[0].IrregularOut != 20000 {
		t.Errorf("unexpected day 1: %+v", history[0])
	}

	// Day 2 (zero day): in 0, out 0
	if history[1].Date != "2026-09-02" || history[1].IrregularIn != 0 || history[1].IrregularOut != 0 {
		t.Errorf("unexpected zero day 2: %+v", history[1])
	}

	// Day 3: in 30000, out 0
	if history[2].Date != "2026-09-03" || history[2].IrregularIn != 30000 || history[2].IrregularOut != 0 {
		t.Errorf("unexpected day 3: %+v", history[2])
	}
}
