package handlers

import (
	"errors"
	"net/http"
	"planning/dto"
	"planning/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlanningHandler struct {
	svc *service.PlanningService
}

func NewPlanningHandler(svc *service.PlanningService) *PlanningHandler {
	return &PlanningHandler{svc: svc}
}

func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "missing X-User-ID header"})
			return
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid X-User-ID header"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func getUserID(c *gin.Context) uint64 {
	return c.MustGet("user_id").(uint64)
}

func (h *PlanningHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateEvent handles POST /events
func (h *PlanningHandler) CreateEvent(c *gin.Context) {
	userID := getUserID(c)

	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	event, err := h.svc.CreateEvent(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

// GetEvents handles GET /events (if ?raw=true returns list of events, otherwise scheduled payments for forecast)
func (h *PlanningHandler) GetEvents(c *gin.Context) {
	userID := getUserID(c)

	if c.Query("raw") == "true" {
		list, err := h.svc.GetEventsList(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
		return
	}

	payments, err := h.svc.GetScheduledPayments(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, payments)
}

// ClearEvents handles DELETE /events. A new statement drops plans from the previous one.
func (h *PlanningHandler) ClearEvents(c *gin.Context) {
	userID := getUserID(c)
	if err := h.svc.ClearUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteEvent handles DELETE /events/:id
func (h *PlanningHandler) DeleteEvent(c *gin.Context) {
	userID := getUserID(c)

	idParam := c.Param("id")
	eventID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid event id"})
		return
	}

	if err := h.svc.DeleteEvent(userID, eventID); err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
