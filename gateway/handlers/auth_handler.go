package handlers

import (
	"gateway/auth"
	"gateway/database"
	"gateway/dto"
	"gateway/entity"
	"gateway/repository"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type AuthHandler struct {
	repo repository.UserRepositoryInterface
}

func NewAuthHandler(repo repository.UserRepositoryInterface) *AuthHandler {
	return &AuthHandler{repo: repo}
}

type SessionData struct {
	UserID      uint   `json:"user_id"`
	Email       string `json:"email"`
	CompanyName string `json:"company_name"`
}

// Register godoc
// @Summary      Регистрация нового пользователя
// @Description  Создает нового пользователя и возвращает его данные
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateUserRequest true "Данные регистрации"
// @Success      201 {object} dto.UserInfo
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "passwords do not match"})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to hash password"})
		return
	}

	user := entity.User{
		Email:       req.Email,
		CompanyName: req.CompanyName,
		Password:    hashedPassword,
	}

	if err := h.repo.Create(&user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "user with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to create user"})
		return
	}

	// Invalidate users list cache in Redis
	database.GetRedis().Del(c.Request.Context(), "users:all")

	c.JSON(http.StatusCreated, dto.UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		CompanyName: user.CompanyName,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	})
}

// Login godoc
// @Summary      Авторизация пользователя
// @Description  Проверяет email и пароль, возвращает Access и Refresh токены, а также сохраняет сессию в Redis
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Учетные данные"
// @Success      200 {object} dto.LoginResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.repo.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid email or password"})
		return
	}

	match, err := auth.VerifyPassword(user.Password, req.Password)
	if err != nil || !match {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid email or password"})
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, user.CompanyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to generate access token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to generate refresh token"})
		return
	}

	sessionData := SessionData{
		UserID:      user.ID,
		Email:       user.Email,
		CompanyName: user.CompanyName,
	}
	sessionBytes, _ := json.Marshal(sessionData)

	// Store refresh token in Redis for 7 days
	err = database.GetRedis().Set(c.Request.Context(), "refresh_token:"+refreshToken, sessionBytes, 7*24*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to store session in redis"})
		return
	}

	// Set HTTP cookies for browser / swagger convenience
	c.SetCookie("access_token", accessToken, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/", "", false, true)

	c.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserInfo{
			ID:          user.ID,
			Email:       user.Email,
			CompanyName: user.CompanyName,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
	})
}

// Refresh godoc
// @Summary      Обновление Access токена
// @Description  Принимает Refresh токен из Cookie или body, проверяет его в Redis и выпускает новый Access токен
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshTokenRequest false "Refresh токен (опционально, если передан в Cookie)"
// @Success      200 {object} dto.RefreshTokenResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var token string

	// 1. Try reading from cookie
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
		token = cookie
	}

	// 2. Fallback to JSON body if provided
	if token == "" {
		var req dto.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
			token = req.RefreshToken
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "refresh token is required"})
		return
	}

	val, err := database.GetRedis().Get(c.Request.Context(), "refresh_token:"+token).Result()
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid or expired refresh token"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to check session"})
		return
	}

	var session SessionData
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "corrupted session data"})
		return
	}

	newAccessToken, err := auth.GenerateAccessToken(session.UserID, session.Email, session.CompanyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to generate access token"})
		return
	}

	c.SetCookie("access_token", newAccessToken, 15*60, "/", "", false, true)

	c.JSON(http.StatusOK, dto.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: token,
	})
}

// Logout godoc
// @Summary      Выход из системы
// @Description  Удаляет Refresh сессию из Redis и очищает cookies
// @Tags         Auth
// @Produce      json
// @Success      200 {object} dto.SuccessResponse
// @Router       /logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var token string
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
		token = cookie
	}

	if token == "" {
		var req dto.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
			token = req.RefreshToken
		}
	}

	if token != "" {
		database.GetRedis().Del(c.Request.Context(), "refresh_token:"+token)
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.SuccessResponse{Message: "successfully logged out"})
}
