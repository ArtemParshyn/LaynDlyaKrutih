package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type ProphetClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProphetClient(baseURL string) *ProphetClient {
	return &ProphetClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type ProphetTrain struct {
	JobID     string `json:"job_id"`
	CompanyID string `json:"company_id"`
	Status    string `json:"status"`
}

type ProphetJob struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type ProphetDay struct {
	Date         string  `json:"date"`
	Balance      float64 `json:"balance"`
	BalanceLower float64 `json:"balance_lower"`
	BalanceUpper float64 `json:"balance_upper"`
}

type ProphetGap struct {
	Date        string  `json:"date"`
	Shortage    float64 `json:"shortage"`
	Probability float64 `json:"probability"`
}

type ProphetForecast struct {
	Forecast []ProphetDay `json:"forecast"`
	CashGap  *ProphetGap  `json:"cash_gap"`
	Summary  string       `json:"summary"`
}

// Train queues a model for this company. The call returns as soon as the
// CSV is accepted; fitting itself runs in the Prophet service.
func (c *ProphetClient) Train(ctx context.Context, companyID string, csv []byte) (*ProphetTrain, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("company_id", companyID); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("file", "statement.csv")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(csv); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/train", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prophet train request failed: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", prophetDetail(payload, resp.StatusCode))
	}
	var train ProphetTrain
	if err := json.Unmarshal(payload, &train); err != nil {
		return nil, fmt.Errorf("failed to decode train response: %w", err)
	}
	return &train, nil
}

func (c *ProphetClient) Job(ctx context.Context, jobID string) (*ProphetJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/jobs/"+jobID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prophet job request failed: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", prophetDetail(payload, resp.StatusCode))
	}
	var job ProphetJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return nil, fmt.Errorf("failed to decode job: %w", err)
	}
	return &job, nil
}

func (c *ProphetClient) Forecast(ctx context.Context, companyID string) (*ProphetForecast, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/forecast/"+companyID+"?horizon_days=30", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prophet forecast request failed: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", prophetDetail(payload, resp.StatusCode))
	}
	var forecast ProphetForecast
	if err := json.Unmarshal(payload, &forecast); err != nil {
		return nil, fmt.Errorf("failed to decode prophet forecast: %w", err)
	}
	return &forecast, nil
}

func prophetDetail(payload []byte, status int) string {
	var body struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(payload, &body) == nil && body.Detail != "" {
		return body.Detail
	}
	text := strings.TrimSpace(string(payload))
	if text == "" {
		return fmt.Sprintf("prophet returned %d", status)
	}
	return text
}
