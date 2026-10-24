package service

import (
	"strings"
	"testing"
	"time"
)

func TestParseCSV_ValidComma(t *testing.T) {
	csvData := `Дата,Сумма,Тип,Категория,Описание
2026-09-01,1000.50,Доход,Зарплата,Основная зарплата
2026-09-05,-200.00,Расход,Продукты,Магазин у дома
2026-09-10,300,Расход,Транспорт,Метро
`
	ops, err := ParseCSV([]byte(csvData), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ops) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(ops))
	}

	// 1000.50 -> 100050 kopecks
	if ops[0].Amount != 100050 {
		t.Errorf("expected 100050, got %d", ops[0].Amount)
	}

	// -200.00 with type Расход -> -20000 kopecks
	if ops[1].Amount != -20000 {
		t.Errorf("expected -20000, got %d", ops[1].Amount)
	}

	// 300 with type Расход -> -30000 kopecks (flipped sign!)
	if ops[2].Amount != -30000 {
		t.Errorf("expected -30000, got %d", ops[2].Amount)
	}
}

func TestParseCSV_ValidSemicolonAndRussianDate(t *testing.T) {
	csvData := `Дата;Сумма;Тип;Категория;Описание
01.09.2026;1234,56;Доход;;Выплата клиенту
15.09.2026;500,00;Расход;Аренда;Офис
`
	ops, err := ParseCSV([]byte(csvData), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}

	// 1234,56 -> 123456
	if ops[0].Amount != 123456 {
		t.Errorf("expected 123456, got %d", ops[0].Amount)
	}

	// Empty category -> "Прочее"
	if ops[0].Category != "Прочее" {
		t.Errorf("expected category 'Прочее', got '%s'", ops[0].Category)
	}

	expectedDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !ops[0].Date.Equal(expectedDate) {
		t.Errorf("expected date %v, got %v", expectedDate, ops[0].Date)
	}
}

func TestParseCSV_BadFileRejection(t *testing.T) {
	// Missing date in row
	badCSV1 := `Дата,Сумма,Тип,Категория,Описание
,1000,Доход,Зарплата,Зп
`
	_, err := ParseCSV([]byte(badCSV1), 1)
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}

	// Non-numeric amount
	badCSV2 := `Дата,Сумма,Тип,Категория,Описание
2026-09-01,не_число,Доход,Зарплата,Зп
`
	_, err = ParseCSV([]byte(badCSV2), 1)
	if err == nil {
		t.Fatal("expected error for non-numeric amount, got nil")
	}

	// Missing header
	badCSV3 := `Непонятно,Колонка
1,2
`
	_, err = ParseCSV([]byte(badCSV3), 1)
	if err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("expected missing header error, got %v", err)
	}
}
