package service

import (
	"planning/dto"
	"testing"
)

func TestParseDate(t *testing.T) {
	d, err := parseDate("2026-11-05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Year() != 2026 || d.Month() != 11 || d.Day() != 5 {
		t.Errorf("expected 2026-11-05, got %v", d)
	}

	d2, err := parseDate("05.11.2026")
	if err != nil {
		t.Fatalf("unexpected error for russian format: %v", err)
	}
	if d2.Year() != 2026 || d2.Month() != 11 || d2.Day() != 5 {
		t.Errorf("expected 2026-11-05, got %v", d2)
	}
}

func TestScheduledPaymentProjection(t *testing.T) {
	p := dto.ScheduledPayment{
		Date:     "2026-11-01",
		Amount:   5000000,
		IsIncome: false,
		Category: "Аренда",
	}

	if p.Amount != 5000000 || p.Category != "Аренда" {
		t.Errorf("invalid payment dto: %v", p)
	}
}
