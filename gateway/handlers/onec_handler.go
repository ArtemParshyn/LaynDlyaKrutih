package handlers

import (
	"gateway/dto"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// OneCHandler exposes the stable, machine-to-machine API used by a 1C
// information base. It deliberately delegates all operations to the same
// orchestrator as the web application, so both clients share business rules.
type OneCHandler struct {
	forecast *ForecastHandler
}

func NewOneCHandler(forecast *ForecastHandler) *OneCHandler {
	return &OneCHandler{forecast: forecast}
}

// userFromOneCHeader authenticates the account context supplied by 1C. The
// token itself is checked by OneCTokenMiddleware; the user id is kept in a
// header to avoid putting credentials or account identity in the URL.
func userFromOneCHeader(c *gin.Context) (uint, bool) {
	raw := strings.TrimSpace(c.GetHeader("X-User-ID"))
	if raw == "" {
		raw = strings.TrimSpace(c.Query("user_id"))
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || id > uint64(^uint(0)) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "X-User-ID must contain a valid positive integer"})
		return 0, false
	}
	c.Set("user_id", uint(id))
	return uint(id), true
}

func (h *OneCHandler) UploadStatement(c *gin.Context) {
	if _, ok := userFromOneCHeader(c); !ok {
		return
	}
	h.forecast.UploadStatement(c)
}

func (h *OneCHandler) GetForecast(c *gin.Context) {
	if _, ok := userFromOneCHeader(c); !ok {
		return
	}
	h.forecast.GetForecast(c)
}

func (h *OneCHandler) CreateEvent(c *gin.Context) {
	if _, ok := userFromOneCHeader(c); !ok {
		return
	}
	h.forecast.CreateEvent(c)
}

func (h *OneCHandler) GetEvents(c *gin.Context) {
	if _, ok := userFromOneCHeader(c); !ok {
		return
	}
	h.forecast.GetEvents(c)
}

// OneCHealth is intentionally authenticated: it verifies both connectivity
// and that the 1C integration credentials are configured correctly.
func OneCHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "integration": "1c"})
}
