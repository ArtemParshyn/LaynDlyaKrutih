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

	save := h.svc.ProcessAndSaveStatement
	if c.Query("mode") == "append" {
		save = h.svc.AppendStatement
	}
	if err := save(userID, filename, fileBytes); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Wipe removes the statement, forecast and hand-entered payments of one pulse.
func (h *LedgerHandler) Wipe(c *gin.Context) {
	if err := h.svc.Wipe(getUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
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

// AddManualOperation handles POST /operations. The date must be on or before
// the statement end; later dates are plans and do not belong in the ledger.
func (h *LedgerHandler) AddManualOperation(c *gin.Context) {
	userID := getUserID(c)

	var req dto.ManualOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.AddManualOperation(userID, req); err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, service.ErrNoData) {
			status = http.StatusNotFound
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Operations handles GET /operations. Prophet trains on this full list.
func (h *LedgerHandler) Operations(c *gin.Context) {
	userID := getUserID(c)
	ops, err := h.svc.Operations(userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNoData) {
			status = http.StatusNotFound
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ops)
}

// Health handles GET /health
func (h *LedgerHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
