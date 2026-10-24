package service

import (
	"ledger/dto"
	"ledger/model"
	"math"
	"sort"
	"strings"
	"time"
)

type recurringGroupKey struct {
	desc     string
	isIncome bool
}

// DetectAndProjectRecurring finds recurring operations, flags them in-place,
// and projects future recurring_payments up to asOf + 30 days.
func DetectAndProjectRecurring(ops []model.Operation, asOf time.Time) []dto.ScheduledPayment {
	if len(ops) < 2 {
		return []dto.ScheduledPayment{}
	}

	// 1. Group operations by normalized description and income sign
	groups := make(map[recurringGroupKey][]int)
	for i, op := range ops {
		key := recurringGroupKey{
			desc:     strings.ToLower(strings.TrimSpace(op.Description)),
			isIncome: op.Amount > 0,
		}
		groups[key] = append(groups[key], i)
	}

	var allRecurringPayments []dto.ScheduledPayment

	limitDate := asOf.AddDate(0, 0, 30)

	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}

		// Sort indices by operation date ascending
		sort.Slice(indices, func(i, j int) bool {
			return ops[indices[i]].Date.Before(ops[indices[j]].Date)
		})

		// Find chains of operations with 25-35 days distance and <= 5% amount diff
		var chains [][]int
		var currentChain []int

		for i := 0; i < len(indices); i++ {
			if len(currentChain) == 0 {
				currentChain = append(currentChain, indices[i])
				continue
			}

			prevIdx := currentChain[len(currentChain)-1]
			prevOp := ops[prevIdx]
			currOp := ops[indices[i]]

			daysDiff := int(currOp.Date.Sub(prevOp.Date).Hours() / 24)
			prevAmt := math.Abs(float64(prevOp.Amount))
			currAmt := math.Abs(float64(currOp.Amount))

			maxAmt := math.Max(prevAmt, currAmt)
			amtDiffRatio := 0.0
			if maxAmt > 0 {
				amtDiffRatio = math.Abs(currAmt-prevAmt) / maxAmt
			}

			if daysDiff >= 25 && daysDiff <= 35 && amtDiffRatio <= 0.05 {
				currentChain = append(currentChain, indices[i])
			} else {
				if len(currentChain) >= 2 {
					chains = append(chains, currentChain)
				}
				currentChain = []int{indices[i]}
			}
		}

		if len(currentChain) >= 2 {
			chains = append(chains, currentChain)
		}

		// Process each identified chain
		for _, chain := range chains {
			// Mark operations in chain as recurring
			for _, idx := range chain {
				ops[idx].Recurring = true
			}

			N := len(chain)
			lastOp := ops[chain[N-1]]
			isIncome := lastOp.Amount > 0
			category := lastOp.Category

			// Calculate average step between dates
			totalDays := 0
			var totalAmt int64 = 0

			for j := 0; j < N; j++ {
				amt := ops[chain[j]].Amount
				if amt < 0 {
					amt = -amt
				}
				totalAmt += amt

				if j > 0 {
					days := int(ops[chain[j]].Date.Sub(ops[chain[j-1]].Date).Hours() / 24)
					totalDays += days
				}
			}

			avgStep := int(math.Round(float64(totalDays) / float64(N-1)))
			if avgStep < 25 {
				avgStep = 30 // fallback to standard monthly step
			}

			avgAmount := int64(math.Round(float64(totalAmt) / float64(N)))

			// Project future dates: last encounter + avgStep up to asOf + 30 days
			currDate := lastOp.Date.AddDate(0, 0, avgStep)
			for !currDate.After(limitDate) {
				if currDate.After(asOf) {
					allRecurringPayments = append(allRecurringPayments, dto.ScheduledPayment{
						Date:     currDate.Format("2006-01-02"),
						Amount:   avgAmount,
						IsIncome: isIncome,
						Category: category,
					})
				}
				currDate = currDate.AddDate(0, 0, avgStep)
			}
		}
	}

	// Sort recurring payments by Date ascending
	sort.Slice(allRecurringPayments, func(i, j int) bool {
		return allRecurringPayments[i].Date < allRecurringPayments[j].Date
	})

	return allRecurringPayments
}

// BuildContinuousHistory builds continuous days from minDate to asOf inclusive,
// summing only non-recurring irregular operations.
func BuildContinuousHistory(ops []model.Operation, minDate, asOf time.Time) []dto.HistoryDay {
	if minDate.After(asOf) {
		return []dto.HistoryDay{}
	}

	// Group irregular amounts by date string "YYYY-MM-DD"
	type dayTotals struct {
		in  int64
		out int64
	}
	dayMap := make(map[string]*dayTotals)

	for _, op := range ops {
		if op.Recurring {
			continue // Recurring payments do NOT enter irregular history
		}
		dateKey := op.Date.Format("2006-01-02")
		totals, exists := dayMap[dateKey]
		if !exists {
			totals = &dayTotals{}
			dayMap[dateKey] = totals
		}

		if op.Amount > 0 {
			totals.in += op.Amount
		} else if op.Amount < 0 {
			totals.out += -op.Amount // module for outflow
		}
	}

	// Generate day-by-day from minDate to asOf inclusive
	var history []dto.HistoryDay
	curr := minDate
	for !curr.After(asOf) {
		dateKey := curr.Format("2006-01-02")
		var inAmt, outAmt int64
		if t, ok := dayMap[dateKey]; ok {
			inAmt = t.in
			outAmt = t.out
		}

		history = append(history, dto.HistoryDay{
			Date:         dateKey,
			IrregularIn:  inAmt,
			IrregularOut: outAmt,
		})

		curr = curr.AddDate(0, 0, 1)
	}

	return history
}
