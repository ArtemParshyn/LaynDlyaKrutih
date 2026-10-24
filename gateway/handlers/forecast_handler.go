package handlers

import (
	"encoding/json"
	"gateway/dto"
	"gateway/service"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ForecastHandler struct {
	orchestrator *service.OrchestratorService
}

func NewForecastHandler(orchestrator *service.OrchestratorService) *ForecastHandler {
	return &ForecastHandler{orchestrator: orchestrator}
}

func getUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized"})
		return 0, false
	}
	userID, ok := val.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid user identity in token"})
		return 0, false
	}
	return userID, true
}

// UploadStatement godoc
// @Summary      Загрузка банковской выписки CSV
// @Description  Принимает CSV, отдаёт его выписке, собирает данные из выписки и планов, вызывает прогноз и сохраняет результат в выписку
// @Tags         Forecast
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "CSV файл банковской выписки"
// @Success      200 {object} dto.ForecastResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/statements [post]
func (h *ForecastHandler) UploadStatement(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var filename string
	var fileBytes []byte

	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		filename = header.Filename
		fileBytes, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "failed to read uploaded file"})
			return
		}
	} else {
		// Fallback: check if raw CSV body was sent
		body, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil || len(body) == 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "CSV file is required (form-data 'file' or request body)"})
			return
		}
		filename = "statement.csv"
		fileBytes = body
	}

	upload := h.orchestrator.UploadStatementAndRecalculate
	if c.Query("mode") == "append" {
		upload = h.orchestrator.AppendStatementAndRecalculate
	}
	forecast, err := upload(c.Request.Context(), userID, filename, fileBytes)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// Process godoc
// @Summary      Прогноз Prophet
// @Description  Забирает 30 дней у модели, которую обучение поставило в очередь при загрузке CSV. Если модель ещё учится, дни не возвращаются.
// @Tags         Forecast
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.ForecastResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/forecast/process [post]
func (h *ForecastHandler) Process(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	forecast, err := h.orchestrator.Process(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, forecast)
}

// GetForecast godoc
// @Summary      Получение готового прогноза
// @Description  Читает уже посчитанные дни и предупреждения из выписки и отдаёт сайту без пересчёта
// @Tags         Forecast
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.ForecastResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/forecast [get]
func (h *ForecastHandler) GetForecast(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	forecast, err := h.orchestrator.GetForecast(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// CreateEvent godoc
// @Summary      Добавление запланированного события
// @Description  Сохраняет событие в планах и запускает полный пересчёт прогноза
// @Tags         Forecast
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.PlanEvent true "Данные события"
// @Success      200 {object} dto.ForecastResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/events [post]
func (h *ForecastHandler) CreateEvent(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.PlanEvent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	forecast, err := h.orchestrator.CreateEventAndRecalculate(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// AddOperation godoc
// @Summary      Ручная операция
// @Description  Прошлая дата исправляет фактический остаток и не входит в обычный дневной поток. Будущая дата сохраняется как план. Прогноз пересчитывается целиком.
// @Tags         Forecast
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.ManualOperation true "Операция"
// @Success      200 {object} dto.ForecastResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/operations [post]
func (h *ForecastHandler) AddOperation(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req dto.ManualOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	forecast, err := h.orchestrator.AddOperation(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, forecast)
}

// GetEvents godoc
// @Summary      Список запланированных событий пользователя
// @Description  Возвращает созданные пользователем плановые расходы и доходы (разовые и регулярные)
// @Tags         Forecast
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.PlanEvent
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/events [get]
func (h *ForecastHandler) GetEvents(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	events, err := h.orchestrator.GetEventsList(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// DeleteEvent godoc
// @Summary      Удаление запланированного события
// @Description  Удаляет плановое событие по ID и пересчитывает прогноз
// @Tags         Forecast
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID события"
// @Success      200 {object} dto.ForecastResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/events/{id} [delete]
func (h *ForecastHandler) DeleteEvent(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	eventID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid event id"})
		return
	}

	forecast, err := h.orchestrator.DeleteEventAndRecalculate(c.Request.Context(), userID, eventID)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

// CalculateScenario godoc
// @Summary      Расчёт альтернативного сценария «что если»
// @Description  Берёт финансовый пакет, добавляет гипотетические события и возвращает вторую линию прогноза без сохранения в базу
// @Tags         Forecast
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.ScenarioRequest true "Список гипотетических событий"
// @Success      200 {object} dto.ForecastResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/scenarios [post]
func (h *ForecastHandler) CalculateScenario(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	rawBytes, err := io.ReadAll(c.Request.Body)
	if err != nil || len(rawBytes) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "request body is required"})
		return
	}

	// Support both {"events": [...]} and [...]
	var events []dto.ScheduledPayment
	var req dto.ScenarioRequest
	if err := json.Unmarshal(rawBytes, &req); err == nil && len(req.Events) > 0 {
		events = req.Events
	} else if err := json.Unmarshal(rawBytes, &events); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid scenario events format: expected list or {events: [...]}"})
		return
	}

	forecast, err := h.orchestrator.CalculateScenario(c.Request.Context(), userID, events)
	if err != nil {
		c.JSON(http.StatusBadGateway, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, forecast)
}
