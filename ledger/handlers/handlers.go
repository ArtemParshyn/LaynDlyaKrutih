package handlers

import (
	"errors"
	"io"
	"ledger/dto"
	"ledger/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LedgerHandler struct {
	svc *service.LedgerService
}

func NewLedgerHandler(svc *service.LedgerService) *LedgerHandler {
	return &LedgerHandler{svc: svc}
}

func getUserID(c *gin.Context) uint64 {
	return c.MustGet("user_id").(uint64)
}

// UploadStatements handles POST /statements (file in 'file' field)
func (h *LedgerHandler) UploadStatements(c *gin.Context) {
	userID := getUserID(c)

	var filename string
	var fileBytes []byte

	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		filename = header.Filename
		fileBytes, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, dto.ErrorResponse{Error: "failed to read uploaded file"})
			return
		}
	} else {
		// Fallback check if body sent directly
		body, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil || len(body) == 0 {
			c.JSON(http.StatusUnprocessableEntity, dto.ErrorResponse{Error: "file field is required in form-data"})
			return
		}
		filename = "statement.csv"
		fileBytes = body
	}

	if err := h.svc.ProcessAndSaveStatement(userID, filename, fileBytes); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetData handles GET /data (as_of, opening_balance, history, recurring_payments)
func (h *LedgerHandler) GetData(c *gin.Context) {
	userID := getUserID(c)

	data, err := h.svc.GetData(userID)
	if err != nil {
		if errors.Is(err, service.ErrNoData) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "no statements found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// ArchiveForecast handles POST /forecast/archive
func (h *LedgerHandler) ArchiveForecast(c *gin.Context) {
	userID := getUserID(c)

	if err := h.svc.ArchiveForecast(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SaveForecast handles POST /forecast
func (h *LedgerHandler) SaveForecast(c *gin.Context) {
	userID := getUserID(c)

	var payload dto.ForecastPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.svc.SaveForecast(userID, payload); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetForecast handles GET /forecast
func (h *LedgerHandler) GetForecast(c *gin.Context) {
	userID := getUserID(c)

	forecast, err := h.svc.GetForecast(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// Health handles GET /health
func (h *LedgerHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
