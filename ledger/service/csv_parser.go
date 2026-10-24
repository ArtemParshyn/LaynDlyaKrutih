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
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

type ParsedRow struct {
	Date        time.Time
	Amount      int64 // kopecks
	Category    string
	Description string
}

// normalizeEncoding converts non-UTF8 encodings (like Windows-1251 or UTF-16) to valid UTF-8
func normalizeEncoding(data []byte) []byte {
	// 1. Strip UTF-8 BOM
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		return bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	}

	// 2. Handle UTF-16 BOM
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		if decoded, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(data); err == nil {
			return bytes.TrimPrefix(decoded, []byte("\xef\xbb\xbf"))
		}
	}

	// 3. If already valid UTF-8, return as is
	if utf8.Valid(data) {
		return data
	}

	// 4. Try Windows-1251 (CP1251) which is standard for Russian banking exports
	if decoded, err := charmap.Windows1251.NewDecoder().Bytes(data); err == nil && utf8.Valid(decoded) {
		return decoded
	}

	return data
}

// ParseCSV parses statement CSV data according to the specification
func ParseCSV(rawData []byte, userID uint64) ([]model.Operation, error) {
	data := normalizeEncoding(rawData)

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return nil, errors.New("empty CSV file")
	}

	// Try delimiters in priority order: semicolon, comma, tab, pipe
	delimiters := []rune{';', ',', '\t', '|'}

	var bestRecords [][]string
	bestHeaderIdx := -1
	dateCol, amountCol, typeCol, categoryCol, descCol := -1, -1, -1, -1, -1
	creditCol, debitCol := -1, -1

	for _, delim := range delimiters {
		reader := csv.NewReader(bytes.NewReader(data))
		reader.Comma = delim
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		reader.LazyQuotes = true

		records, err := reader.ReadAll()
		if err != nil || len(records) == 0 {
			continue
		}

		for i, row := range records {
			d, a, t, c, desc, credit, debit := detectColumns(row)
			if d != -1 && (a != -1 || credit != -1 || debit != -1) {
				bestRecords = records
				bestHeaderIdx = i
				dateCol, amountCol, typeCol, categoryCol, descCol = d, a, t, c, desc
				creditCol, debitCol = credit, debit
				break
			}
		}

		if bestHeaderIdx != -1 {
			break
		}
	}

	if bestHeaderIdx == -1 {
		return nil, errors.New("CSV missing required header columns (Дата, Сумма)")
	}

	records := bestRecords
	var operations []model.Operation

	for rowIdx := bestHeaderIdx + 1; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]

		// Skip empty lines
		if isRowEmpty(row) {
			continue
		}

		if dateCol >= len(row) || (amountCol >= 0 && amountCol >= len(row)) {
			return nil, fmt.Errorf("row %d has missing date or amount column", rowIdx+1)
		}

		dateStr := strings.TrimSpace(row[dateCol])
		amountStr := ""
		if amountCol >= 0 {
			amountStr = strings.TrimSpace(row[amountCol])
		} else {
			credit, debit := cell(row, creditCol), cell(row, debitCol)
			switch {
			case credit != "" && credit != "0" && credit != "0.00":
				amountStr = credit
			case debit != "":
				amountStr = "-" + strings.TrimPrefix(debit, "-")
			}
		}

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
			if (strings.Contains(typeStr, "расход") || strings.Contains(typeStr, "списание") || strings.Contains(typeStr, "out") || strings.Contains(typeStr, "дебет")) && kopecks > 0 {
				kopecks = -kopecks
			} else if (strings.Contains(typeStr, "доход") || strings.Contains(typeStr, "приход") || strings.Contains(typeStr, "поступлен") || strings.Contains(typeStr, "in") || strings.Contains(typeStr, "кредит")) && kopecks < 0 {
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

func detectColumns(row []string) (dateCol, amountCol, typeCol, categoryCol, descCol, creditCol, debitCol int) {
	dateCol, amountCol, typeCol, categoryCol, descCol, creditCol, debitCol = -1, -1, -1, -1, -1, -1, -1

	for i, col := range row {
		clean := strings.ToLower(strings.TrimSpace(col))
		clean = strings.TrimPrefix(clean, "\ufeff")
		clean = strings.ReplaceAll(clean, "_", " ")

		switch {
		case strings.Contains(clean, "дата") || strings.Contains(clean, "date") || strings.Contains(clean, "время") || strings.Contains(clean, "день") || clean == "dt":
			if dateCol == -1 || strings.Contains(clean, "операц") || strings.Contains(clean, "operation") {
				dateCol = i
			}
		case strings.Contains(clean, "кредит") || strings.Contains(clean, "credit") || strings.Contains(clean, "приход") || strings.Contains(clean, "поступлен"):
			creditCol = i
		case strings.Contains(clean, "дебет") || strings.Contains(clean, "debit") || strings.Contains(clean, "расход") || strings.Contains(clean, "списан"):
			debitCol = i
		case strings.Contains(clean, "сумма") || clean == "amount" || clean == "sum" || strings.Contains(clean, "руб") || strings.Contains(clean, "размер") || clean == "total" || strings.Contains(clean, "amount"):
			amountCol = i
		case strings.Contains(clean, "тип") || clean == "type" || strings.Contains(clean, "направлен") || strings.Contains(clean, "вид") || strings.Contains(clean, "direction"):
			typeCol = i
		case strings.Contains(clean, "категори") || clean == "category" || strings.Contains(clean, "статья"):
			categoryCol = i
		case strings.Contains(clean, "описани") || strings.Contains(clean, "description") || strings.Contains(clean, "назначени") || strings.Contains(clean, "детал") || strings.Contains(clean, "purpose") || strings.Contains(clean, "комментар") || strings.Contains(clean, "платеж") || strings.Contains(clean, "контрагент") || strings.Contains(clean, "counterparty"):
			descCol = i
		}
	}
	return
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
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
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"2006/01/02",
		"02-01-2006",
		"2006-01-02 15:04:05",
		"02.01.2006 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format: '%s'", s)
}

func parseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	// Remove currency symbols and non-breaking spaces
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "") // non-breaking space
	s = strings.ReplaceAll(s, "\u20bd", "") // ruble sign
	s = strings.ReplaceAll(s, "руб", "")
	s = strings.ReplaceAll(s, "₽", "")

	// Convert comma decimal separator to dot
	s = strings.ReplaceAll(s, ",", ".")

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse float amount '%s': %w", s, err)
	}

	// Multiply by 100 and round to nearest kopeck
	kopecks := int64(math.Round(val * 100))
	return kopecks, nil
}
