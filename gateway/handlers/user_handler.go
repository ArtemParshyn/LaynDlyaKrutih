package handlers

import (
	"gateway/auth"
	"gateway/database"
	"gateway/dto"
	"gateway/repository"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	repo repository.UserRepositoryInterface
}

func NewUserHandler(repo repository.UserRepositoryInterface) *UserHandler {
	return &UserHandler{repo: repo}
}

// GetMe godoc
// @Summary      Профиль текущего пользователя
// @Description  Возвращает информацию об авторизованном пользователе
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.UserInfo
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := h.repo.GetByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"})
		return
	}

	c.JSON(http.StatusOK, dto.UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		CompanyName: user.CompanyName,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	})
}

// GetUser godoc
// @Summary      Получение пользователя по ID
// @Description  Возвращает данные пользователя из кэша Redis или базы PostgreSQL
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID пользователя"
// @Success      200 {object} dto.UserInfo
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid user id"})
		return
	}

	cacheKey := fmt.Sprintf("user:%d", id)

	// Try Redis cache first
	val, err := database.GetRedis().Get(c.Request.Context(), cacheKey).Result()
	if err == nil {
		var cachedUser dto.UserInfo
		if json.Unmarshal([]byte(val), &cachedUser) == nil {
			c.JSON(http.StatusOK, cachedUser)
			return
		}
	}

	// Cache miss: query PostgreSQL
	user, err := h.repo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "database error"})
		return
	}

	userInfo := dto.UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		CompanyName: user.CompanyName,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	// Save to Redis (10 minutes)
	if userBytes, err := json.Marshal(userInfo); err == nil {
		database.GetRedis().Set(c.Request.Context(), cacheKey, userBytes, 10*time.Minute)
	}

	c.JSON(http.StatusOK, userInfo)
}

// GetAllUsers godoc
// @Summary      Список всех пользователей
// @Description  Возвращает список всех пользователей из кэша Redis или базы PostgreSQL
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.UserInfo
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	cacheKey := "users:all"

	// Try Redis cache
	val, err := database.GetRedis().Get(c.Request.Context(), cacheKey).Result()
	if err == nil {
		var cachedList []dto.UserInfo
		if json.Unmarshal([]byte(val), &cachedList) == nil {
			c.JSON(http.StatusOK, cachedList)
			return
		}
	}

	// Cache miss: query DB
	users, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to retrieve users"})
		return
	}

	result := make([]dto.UserInfo, 0, len(users))
	for _, u := range users {
		result = append(result, dto.UserInfo{
			ID:          u.ID,
			Email:       u.Email,
			CompanyName: u.CompanyName,
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
		})
	}

	// Cache in Redis (5 minutes)
	if bytes, err := json.Marshal(result); err == nil {
		database.GetRedis().Set(c.Request.Context(), cacheKey, bytes, 5*time.Minute)
	}

	c.JSON(http.StatusOK, result)
}

// UpdateUser godoc
// @Summary      Обновление данных пользователя
// @Description  Обновляет пароль и/или название компании пользователя
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID пользователя"
// @Param        request body dto.UpdateUserRequest true "Поля для обновления"
// @Success      200 {object} dto.UserInfo
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /users/{id} [patch]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid user id"})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.repo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to find user"})
		return
	}

	if req.CompanyName != nil {
		user.CompanyName = *req.CompanyName
	}

	if req.Password != nil {
		hashedPass, err := auth.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to hash password"})
			return
		}
		user.Password = hashedPass
	}

	if err := h.repo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to update user"})
		return
	}

	// Invalidate Redis caches
	database.GetRedis().Del(c.Request.Context(), fmt.Sprintf("user:%d", id), "users:all")

	c.JSON(http.StatusOK, dto.UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		CompanyName: user.CompanyName,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	})
}

// DeleteUser godoc
// @Summary      Удаление пользователя
// @Description  Удаляет пользователя из базы PostgreSQL и сбрасывает кэш в Redis
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "ID пользователя"
// @Success      200 {object} dto.SuccessResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid user id"})
		return
	}

	if err := h.repo.Delete(uint(id)); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to delete user"})
		return
	}

	// Invalidate Redis caches
	database.GetRedis().Del(c.Request.Context(), fmt.Sprintf("user:%d", id), "users:all")

	c.JSON(http.StatusOK, dto.SuccessResponse{Message: "user deleted successfully"})
}
