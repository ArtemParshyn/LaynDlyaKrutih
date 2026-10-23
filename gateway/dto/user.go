package dto

import "time"

type CreateUserRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=6"`
	CompanyName string `json:"company_name" binding:"required,min=2,max=64"`
}

type UpdateUserRequest struct {
	CompanyName *string `json:"company_name,omitempty" binding:"omitempty,min=2,max=64"`
	Password    *string `json:"password,omitempty" binding:"omitempty,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"omitempty"`
}

type UserInfo struct {
	ID          uint      `json:"id"`
	Email       string    `json:"email"`
	CompanyName string    `json:"company_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
