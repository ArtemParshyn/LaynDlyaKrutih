package handlers

import (
	"gateway/dto"
	"gateway/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type OneCHandler struct {
	forecast *ForecastHandler
	pulses   *repository.PulseRepository
}

func NewOneCHandler(forecast *ForecastHandler, pulses *repository.PulseRepository) *OneCHandler {
	return &OneCHandler{forecast: forecast, pulses: pulses}
}

func (h *OneCHandler) user(c *gin.Context) bool {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Query("user_id")), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "user_id must be a positive integer"})
		return false
	}
	c.Set("user_id", uint(id))
	pulseID, err := strconv.ParseUint(strings.TrimSpace(c.Query("pulse_id")), 10, 64)
	if err != nil || pulseID == 0 || h.pulses == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "pulse_id must be a positive integer"})
		return false
	}
	if _, err := h.pulses.Owned(uint(id), uint(pulseID)); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "pulse not found"})
		return false
	}
	return true
}

func (h *OneCHandler) UploadStatement(c *gin.Context) {
	if h.user(c) { h.forecast.UploadStatement(c) }
}

func OneCHealth(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "integration": "1c"}) }
