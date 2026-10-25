package dto

import "time"

// PulseCard is one graph in the cabinet. History is only enough for the
// small line on the card, not the full statement.
type PulseCard struct {
	ID        uint           `json:"id"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	AsOf      string         `json:"as_of,omitempty"`
	Balance   int64          `json:"balance"`
	History   []BalancePoint `json:"history,omitempty"`
}
