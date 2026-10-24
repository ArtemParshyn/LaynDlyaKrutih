package service

import (
	"ledger/dto"
	"ledger/model"
	"sort"
)

const rareLookbackDays = 84

// rarePayments finds large transfers that happened more than once inside the
// lookback, but not often enough to be a daily flow. Amounts stay in kopecks.
// The note has no date: the transfer is not placed on the forecast line.
func rarePayments(ops []model.Operation) []dto.RarePayment {
	if len(ops) == 0 {
		return []dto.RarePayment{}
	}
	latest := ops[0].Date
	for _, op := range ops[1:] {
		if op.Date.After(latest) {
			latest = op.Date
		}
	}
	cutoff := latest.AddDate(0, 0, -(rareLookbackDays - 1))

	var incoming, outgoing []int64
	days := map[string]struct{}{}
	for _, op := range ops {
		if op.Date.Before(cutoff) {
			continue
		}
		days[op.Date.Format("2006-01-02")] = struct{}{}
		if op.Amount > 0 {
			incoming = append(incoming, op.Amount)
		} else if op.Amount < 0 {
			outgoing = append(outgoing, -op.Amount)
		}
	}
	observed := len(days)
	if observed < 14 {
		return []dto.RarePayment{}
	}

	notes := make([]dto.RarePayment, 0, 2)
	if note, ok := rareNote(outgoing, incoming, observed, false); ok {
		notes = append(notes, note)
	}
	if note, ok := rareNote(incoming, outgoing, observed, true); ok {
		notes = append(notes, note)
	}
	return notes
}

func rareNote(amounts, other []int64, observed int, isIncome bool) (dto.RarePayment, bool) {
	if len(amounts) < 2 {
		return dto.RarePayment{}, false
	}
	sortedAmounts := append([]int64(nil), amounts...)
	sort.Slice(sortedAmounts, func(i, j int) bool { return sortedAmounts[i] < sortedAmounts[j] })

	pool := sortedAmounts
	// Two identical transfers would set their own median and then disappear.
	// Compare them with the other direction when this side is only a handful.
	if len(sortedAmounts) <= 3 && len(other) > 0 {
		pool = append([]int64(nil), other...)
		sort.Slice(pool, func(i, j int) bool { return pool[i] < pool[j] })
	}
	ordinary := medianKopecks(pool[:max(1, len(pool)/2)])
	if ordinary <= 0 {
		return dto.RarePayment{}, false
	}

	spikes := make([]int64, 0)
	for _, amount := range sortedAmounts {
		if amount >= ordinary*5 && amount >= 10_000_000 {
			spikes = append(spikes, amount)
		}
	}
	if len(spikes) < 2 || len(spikes)*2 >= observed {
		return dto.RarePayment{}, false
	}
	typical := medianKopecks(spikes)
	close := make([]int64, 0, len(spikes))
	for _, amount := range spikes {
		diff := amount - typical
		if diff < 0 {
			diff = -diff
		}
		if diff*5 <= typical {
			close = append(close, amount)
		}
	}
	if len(close) < 2 {
		return dto.RarePayment{}, false
	}
	every := observed / len(close)
	if every < 1 {
		every = 1
	}
	return dto.RarePayment{
		Amount:           medianKopecks(close),
		IsIncome:         isIncome,
		Seen:             len(close),
		RoughlyEveryDays: every,
	}, true
}

func medianKopecks(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle]
	}
	return (ordered[middle-1] + ordered[middle]) / 2
}
