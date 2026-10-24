package handlers

import (
	"net/http"
	"planning/dto"
	"planning/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.PlanningService
}

func New(svc *service.PlanningService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListEvents(c *gin.Context) {
	userID := c.MustGet("user_id").(uint64)
	events, err := h.svc.ListFuture(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) CreateEvent(c *gin.Context) {
	userID := c.MustGet("user_id").(uint64)

	var event dto.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	if event.Amount <= 0 || event.Category == "" || event.Date == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "нужны дата, сумма больше нуля и категория"})
		return
	}

	if err := h.svc.Create(userID, event); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}
