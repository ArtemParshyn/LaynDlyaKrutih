package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gateway/dto"
	"io"
	"net/http"
	"strings"
	"time"
)

type ForecastClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewForecastClient(baseURL string) *ForecastClient {
	return &ForecastClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// Calculate sends the consolidated financial package to the Forecast ML service
func (c *ForecastClient) Calculate(ctx context.Context, forecastReq *dto.ForecastRequest) (*dto.ForecastResponse, error) {
	jsonBytes, err := json.Marshal(forecastReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal forecast request: %w", err)
	}

	url := fmt.Sprintf("%s/forecast", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create forecast request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forecast service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("forecast service returned error %d: %s", resp.StatusCode, string(respBody))
	}

	var result dto.ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode forecast response: %w", err)
	}

	return &result, nil
}
