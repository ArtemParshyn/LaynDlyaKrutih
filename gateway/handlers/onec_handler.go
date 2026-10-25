package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"gateway/dto"
	"gateway/entity"
	"gateway/database"
	"gateway/repository"
	"net/http"
	"os"
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

func (h *OneCHandler) Settings(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok { return }
	pulses, err := h.pulses.List(userID)
	if err != nil { c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()}); return }
	var user entity.User
	if err := database.GetDB().First(&user, userID).Error; err != nil { c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"}); return }
	if user.OneCToken == "" {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil { c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to create token"}); return }
		user.OneCToken = "onec_" + hex.EncodeToString(buf)
		if err := database.GetDB().Save(&user).Error; err != nil { c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to save token"}); return }
	}
	base := strings.TrimRight(os.Getenv("ONEC_PUBLIC_URL"), "/")
	if base == "" { base = "http://217.144.184.23:8080" }
	items := make([]gin.H, 0, len(pulses))
	for _, pulse := range pulses {
		items = append(items, gin.H{"id": pulse.ID, "name": pulse.Name, "upload_url": base + "/integration/1c/statements?user_id=" + strconv.Itoa(int(userID)) + "&pulse_id=" + strconv.Itoa(int(pulse.ID)) + "&token=" + user.OneCToken})
	}
	c.JSON(http.StatusOK, gin.H{"token": user.OneCToken, "pulses": items})
}

func (h *OneCHandler) Regenerate(c *gin.Context) {
	userID, ok := getUserID(c); if !ok { return }
	var user entity.User
	if err := database.GetDB().First(&user, userID).Error; err != nil { c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"}); return }
	buf := make([]byte, 24); if _, err := rand.Read(buf); err != nil { c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to create token"}); return }
	user.OneCToken = "onec_" + hex.EncodeToString(buf)
	if err := database.GetDB().Save(&user).Error; err != nil { c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to save token"}); return }
	h.Settings(c)
}

func OneCHealth(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "integration": "1c"}) }
