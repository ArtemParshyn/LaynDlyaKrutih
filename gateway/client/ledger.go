package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gateway/dto"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LedgerClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewLedgerClient(baseURL string) *LedgerClient {
	return &LedgerClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadStatement forwards the uploaded statement CSV to the Ledger service
func (c *LedgerClient) UploadStatement(ctx context.Context, userID uint, filename string, fileData []byte) error {
	return c.upload(ctx, userID, filename, fileData, false)
}

// AppendStatement adds a CSV to the statement already stored for this user.
func (c *LedgerClient) AppendStatement(ctx context.Context, userID uint, filename string, fileData []byte) error {
	return c.upload(ctx, userID, filename, fileData, true)
}

func (c *LedgerClient) upload(ctx context.Context, userID uint, filename string, fileData []byte, appendFile bool) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("failed to create multipart form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("failed to write multipart form file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/statements", c.baseURL)
	if appendFile {
		url += "?mode=append"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger service upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger service returned error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Wipe deletes the statement stored under this key.
func (c *LedgerClient) Wipe(ctx context.Context, userID uint) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/statements", nil)
	if err != nil {
		return fmt.Errorf("failed to create wipe request: %w", err)
	}
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger wipe request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger wipe error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ArchiveForecast asks Ledger to archive previous forecast line before calculating a new one
func (c *LedgerClient) ArchiveForecast(ctx context.Context, userID uint) error {
	url := fmt.Sprintf("%s/forecast/archive", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create archive request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger archive request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger archive error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// AddManualOperation stores a past payment that was missing from the statement.
// It does not replace bank rows and is not part of the daily flow.
func (c *LedgerClient) AddManualOperation(ctx context.Context, userID uint, op dto.ManualOperation) error {
	jsonBytes, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("failed to marshal manual operation: %w", err)
	}

	url := fmt.Sprintf("%s/operations", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create manual operation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger manual operation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger manual operation error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Operations returns every stored movement, oldest first.
func (c *LedgerClient) Operations(ctx context.Context, userID uint) ([]dto.Operation, error) {
	url := fmt.Sprintf("%s/operations", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create operations request: %w", err)
	}
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ledger operations request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ledger operations error %d: %s", resp.StatusCode, string(respBody))
	}

	var ops []dto.Operation
	if err := json.NewDecoder(resp.Body).Decode(&ops); err != nil {
		return nil, fmt.Errorf("failed to decode operations: %w", err)
	}
	return ops, nil
}

// GetData requests balance, history, and recurring payments from Ledger
func (c *LedgerClient) GetData(ctx context.Context, userID uint) (*dto.LedgerData, error) {
	url := fmt.Sprintf("%s/data", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get data request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ledger get data request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ledger get data error %d: %s", resp.StatusCode, string(respBody))
	}

	var data dto.LedgerData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode ledger data response: %w", err)
	}

	return &data, nil
}

// SaveForecast sends the calculated 30-day forecast and alerts to Ledger for persistent storage
func (c *LedgerClient) SaveForecast(ctx context.Context, userID uint, forecast *dto.ForecastResponse) error {
	jsonBytes, err := json.Marshal(forecast)
	if err != nil {
		return fmt.Errorf("failed to marshal forecast: %w", err)
	}

	url := fmt.Sprintf("%s/forecast", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create save forecast request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger save forecast request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ledger save forecast error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetForecast retrieves the stored forecast and alerts from Ledger without recalculating
func (c *LedgerClient) GetForecast(ctx context.Context, userID uint) (*dto.ForecastResponse, error) {
	url := fmt.Sprintf("%s/forecast", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get forecast request: %w", err)
	}

	req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ledger get forecast request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ledger get forecast error %d: %s", resp.StatusCode, string(respBody))
	}

	var forecast dto.ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecast); err != nil {
		return nil, fmt.Errorf("failed to decode forecast response: %w", err)
	}

	return &forecast, nil
}
