package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gateway/dto"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PlanningClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPlanningClient(baseURL string) *PlanningClient {
	return &PlanningClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetEvents fetches future planned events from the Planning service
func (c *PlanningClient) GetEvents(ctx context.Context, userID uint) ([]dto.ScheduledPayment, error) {
	url := fmt.Sprintf("%s/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get events request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("planning service get events failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("planning service error %d: %s", resp.StatusCode, string(respBody))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read planning events response: %w", err)
	}

	// Support both JSON array `[...]` and wrapped object `{"events": [...]}`
	var events []dto.ScheduledPayment
	if err := json.Unmarshal(respBytes, &events); err == nil {
		return events, nil
	}

	var wrapped struct {
		Events []dto.ScheduledPayment `json:"events"`
	}
	if err := json.Unmarshal(respBytes, &wrapped); err == nil {
		return wrapped.Events, nil
	}

	return nil, fmt.Errorf("failed to parse planning events response")
}

// CreateEvent sends a new planned event to the Planning service
func (c *PlanningClient) CreateEvent(ctx context.Context, userID uint, event dto.PlanEvent) error {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal plan event: %w", err)
	}

	url := fmt.Sprintf("%s/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create plan event request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("planning create event request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planning create event error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetEventsList retrieves raw user-created events from Planning service
func (c *PlanningClient) GetEventsList(ctx context.Context, userID uint) ([]dto.PlanEvent, error) {
	url := fmt.Sprintf("%s/events?raw=true", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get events list request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("planning service get events list failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("planning service error %d: %s", resp.StatusCode, string(respBody))
	}

	var list []dto.PlanEvent
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode planning events list: %w", err)
	}

	return list, nil
}

// ClearEvents removes every plan of the user. Called when a new statement arrives.
func (c *PlanningClient) ClearEvents(ctx context.Context, userID uint) error {
	url := fmt.Sprintf("%s/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create clear events request: %w", err)
	}
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("planning clear events failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planning clear events error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DeleteEvent deletes a planned event in Planning service
func (c *PlanningClient) DeleteEvent(ctx context.Context, userID uint, eventID uint64) error {
	url := fmt.Sprintf("%s/events/%d", c.baseURL, eventID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete event request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("planning service delete event failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("planning service delete event error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
