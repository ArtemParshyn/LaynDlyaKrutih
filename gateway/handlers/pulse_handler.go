package handlers

import (
	"errors"
	"gateway/dto"
	"gateway/entity"
	"gateway/repository"
	"gateway/service"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type PulseHandler struct {
	repo         *repository.PulseRepository
	orchestrator *service.OrchestratorService
}

func NewPulseHandler(repo *repository.PulseRepository, orchestrator *service.OrchestratorService) *PulseHandler {
	return &PulseHandler{repo: repo, orchestrator: orchestrator}
}

func (h *PulseHandler) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	pulses, err := h.pulses(c, userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}
	cards := make([]dto.PulseCard, 0, len(pulses))
	for _, pulse := range pulses {
		view, err := h.orchestrator.GetForecast(c.Request.Context(), uint(pulse.LedgerKey))
		if err != nil {
			cards = append(cards, pulseCard(pulse, nil))
			continue
		}
		cards = append(cards, pulseCard(pulse, view))
	}
	c.JSON(http.StatusOK, cards)
}

func (h *PulseHandler) Get(c *gin.Context) {
	pulse, ok := h.owned(c)
	if !ok {
		return
	}
	view, err := h.orchestrator.GetForecast(c.Request.Context(), uint(pulse.LedgerKey))
	if err != nil {
		c.JSON(http.StatusOK, pulseCard(*pulse, nil))
		return
	}
	c.JSON(http.StatusOK, pulseCard(*pulse, view))
}

func (h *PulseHandler) Create(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	name := "Новый пульс"
	file, header, fileErr := c.Request.FormFile("file")
	if fileErr == nil {
		defer file.Close()
		if posted := strings.TrimSpace(c.PostForm("name")); posted != "" {
			name = posted
		}
	} else if c.ContentType() == "application/json" {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err == nil && strings.TrimSpace(body.Name) != "" {
			name = strings.TrimSpace(body.Name)
		}
	}
	if name == "" || utf8.RuneCountInString(name) > 80 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Укажите имя пульса, до 80 символов"})
		return
	}

	pulse, err := h.repo.Create(userID, name)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: "Не удалось создать пульс"})
		return
	}
	if fileErr != nil {
		c.JSON(http.StatusCreated, pulseCard(*pulse, nil))
		return
	}
	payload, err := io.ReadAll(file)
	if err != nil || len(payload) == 0 {
		_ = h.repo.Delete(userID, pulse.ID)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Не удалось прочитать CSV"})
		return
	}
	view, err := h.orchestrator.UploadStatementAndRecalculate(c.Request.Context(), uint(pulse.LedgerKey), header.Filename, payload)
	if err != nil {
		_ = h.orchestrator.Forget(c.Request.Context(), uint(pulse.LedgerKey))
		_ = h.repo.Delete(userID, pulse.ID)
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pulseCard(*pulse, view))
}

func (h *PulseHandler) Rename(c *gin.Context) {
	pulse, ok := h.owned(c)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Укажите имя пульса"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Укажите имя пульса, до 80 символов"})
		return
	}
	if err := h.repo.Rename(pulse.UserID, pulse.ID, name); err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: "Не удалось переименовать пульс"})
		return
	}
	pulse.Name = name
	c.JSON(http.StatusOK, pulseCard(*pulse, nil))
}

func (h *PulseHandler) Delete(c *gin.Context) {
	pulse, ok := h.owned(c)
	if !ok {
		return
	}
	if err := h.orchestrator.Forget(c.Request.Context(), uint(pulse.LedgerKey)); err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.repo.Delete(pulse.UserID, pulse.ID); err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: "выписка удалена, но карточка осталась"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PulseHandler) pulses(c *gin.Context, userID uint) ([]entity.Pulse, error) {
	pulses, err := h.repo.List(userID)
	if err != nil || len(pulses) > 0 {
		return pulses, err
	}
	view, err := h.orchestrator.GetForecast(c.Request.Context(), userID)
	if err != nil || view == nil || len(view.History) == 0 {
		return pulses, nil
	}
	if _, err := h.repo.CreateLegacy(userID); err != nil && !errors.Is(err, repository.ErrPulseNotFound) {
		return nil, err
	}
	return h.repo.List(userID)
}

func (h *PulseHandler) owned(c *gin.Context) (*entity.Pulse, bool) {
	userID, ok := getUserID(c)
	if !ok {
		return nil, false
	}
	id, err := strconv.ParseUint(c.Param("pulseID"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "некорректный пульс"})
		return nil, false
	}
	pulse, err := h.repo.Owned(userID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "пульс не найден"})
		return nil, false
	}
	return pulse, true
}

func pulseCard(pulse entity.Pulse, view *dto.ForecastResponse) dto.PulseCard {
	card := dto.PulseCard{
		ID:        pulse.ID,
		Name:      pulse.Name,
		CreatedAt: pulse.CreatedAt,
		History:   []dto.BalancePoint{},
	}
	if view == nil || len(view.History) == 0 {
		return card
	}
	history := view.History
	last := history[len(history)-1]
	card.AsOf = last.Date
	card.Balance = last.Balance
	if len(history) > 40 {
		history = history[len(history)-40:]
	}
	card.History = history
	return card
}
