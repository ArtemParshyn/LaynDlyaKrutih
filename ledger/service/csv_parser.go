package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"ledger/model"
	"math"
	"strconv"
	"strings"
	"time"
)

type ParsedRow struct {
	Date        time.Time
	Amount      int64 // kopecks
	Category    string
	Description string
}

// ParseCSV parses statement CSV data according to the specification
func ParseCSV(data []byte, userID uint64) ([]model.Operation, error) {
	// Strip UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return nil, errors.New("empty CSV file")
	}

	// Detect delimiter from first non-empty line
	var firstLine string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			firstLine = trimmed
			break
		}
	}

	delimiter := ','
	if strings.Contains(firstLine, ";") {
		delimiter = ';'
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1 // Allow variable fields if trailing commas
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV syntax: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV has no records")
	}

	// Locate header row
	headerIdx := -1
	dateCol, amountCol, typeCol, categoryCol, descCol := -1, -1, -1, -1, -1

	for i, row := range records {
		d, a, t, c, desc := detectColumns(row)
		if d != -1 && a != -1 {
			headerIdx = i
			dateCol, amountCol, typeCol, categoryCol, descCol = d, a, t, c, desc
			break
		}
	}

	if headerIdx == -1 {
		return nil, errors.New("CSV missing required header columns (Дата, Сумма)")
	}

	var operations []model.Operation

	for rowIdx := headerIdx + 1; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]

		// Skip empty lines
		if isRowEmpty(row) {
			continue
		}

		if dateCol >= len(row) || amountCol >= len(row) {
			return nil, fmt.Errorf("row %d has missing date or amount column", rowIdx+1)
		}

		dateStr := strings.TrimSpace(row[dateCol])
		amountStr := strings.TrimSpace(row[amountCol])

		if dateStr == "" {
			return nil, fmt.Errorf("row %d has empty date", rowIdx+1)
		}
		if amountStr == "" {
			return nil, fmt.Errorf("row %d has empty amount", rowIdx+1)
		}

		parsedDate, err := parseDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("row %d invalid date '%s': %w", rowIdx+1, dateStr, err)
		}

		kopecks, err := parseAmount(amountStr)
		if err != nil {
			return nil, fmt.Errorf("row %d invalid amount '%s': %w", rowIdx+1, amountStr, err)
		}

		// Adjust sign based on Type column if provided
		if typeCol != -1 && typeCol < len(row) {
			typeStr := strings.ToLower(strings.TrimSpace(row[typeCol]))
			if (strings.Contains(typeStr, "расход") || strings.Contains(typeStr, "списание") || strings.Contains(typeStr, "out")) && kopecks > 0 {
				kopecks = -kopecks
			} else if (strings.Contains(typeStr, "доход") || strings.Contains(typeStr, "приход") || strings.Contains(typeStr, "in")) && kopecks < 0 {
				kopecks = -kopecks
			}
		}

		category := ""
		if categoryCol != -1 && categoryCol < len(row) {
			category = strings.TrimSpace(row[categoryCol])
		}

		description := ""
		if descCol != -1 && descCol < len(row) {
			description = strings.TrimSpace(row[descCol])
		}

		// If category from CSV is empty or "Прочее", classify using smart regex patterns
		if category == "" || strings.EqualFold(category, "прочее") {
			isIncome := kopecks > 0
			category = ClassifyCategory(description, isIncome)
		}

		operations = append(operations, model.Operation{
			UserID:      userID,
			Date:        parsedDate,
			Amount:      kopecks,
			Category:    category,
			Description: description,
			Recurring:   false,
		})
	}

	if len(operations) == 0 {
		return nil, io.EOF
	}

	return operations, nil
}

func detectColumns(row []string) (dateCol, amountCol, typeCol, categoryCol, descCol int) {
	dateCol, amountCol, typeCol, categoryCol, descCol = -1, -1, -1, -1, -1

	for i, col := range row {
		clean := strings.ToLower(strings.TrimSpace(col))
		clean = strings.TrimPrefix(clean, "\ufeff")

		switch {
		case strings.Contains(clean, "дата") || clean == "date":
			dateCol = i
		case strings.Contains(clean, "сумма") || clean == "amount":
			amountCol = i
		case strings.Contains(clean, "тип") || clean == "type":
			typeCol = i
		case strings.Contains(clean, "категори") || clean == "category":
			categoryCol = i
		case strings.Contains(clean, "описани") || clean == "description" || strings.Contains(clean, "назначени"):
			descCol = i
		}
	}
	return
}

func isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseDate(s string) (time.Time, error) {
	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// Try DD.MM.YYYY
	if t, err := time.Parse("02.01.2006", s); err == nil {
		return t, nil
	}
	// Try YYYY/MM/DD
	if t, err := time.Parse("2006/01/02", s); err == nil {
		return t, nil
	}
	// Try DD/MM/YYYY
	if t, err := time.Parse("02/01/2006", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or DD.MM.YYYY")
}

func parseAmount(s string) (int64, error) {
	// Remove spaces (including non-breaking spaces \u00a0)
	clean := strings.ReplaceAll(s, " ", "")
	clean = strings.ReplaceAll(clean, "\u00a0", "")
	clean = strings.ReplaceAll(clean, ",", ".")

	val, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, err
	}

	if val >= 0 {
		return int64(math.Round(val * 100)), nil
	}
	return -int64(math.Round(math.Abs(val) * 100)), nil
}
