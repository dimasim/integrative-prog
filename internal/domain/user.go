package domain

import (
	"errors"
	"time"
)

// ─── Entity ──────────────────────────────────────────────────────────────────

// User adalah representasi data user di database.
// Field Password selalu disembunyikan dari JSON response via tag json:"-"
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── Request DTOs ─────────────────────────────────────────────────────────────

type CreateUserRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Role     string `json:"role"     validate:"required,oneof=admin user"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"  validate:"omitempty,min=2,max=100"`
	Email string `json:"email" validate:"omitempty,email"`
	Role  string `json:"role"  validate:"omitempty,oneof=admin user"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ─── Response DTOs ────────────────────────────────────────────────────────────

type LoginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in_hours"`
}

// ─── Sentinel Errors ─────────────────────────────────────────────────────────
// Service & repository mengembalikan error ini.
// Handler memetakan ke HTTP status code yang tepat tanpa expose detail DB.

var (
	ErrNotFound       = errors.New("resource not found")
	ErrConflict       = errors.New("resource already exists")
	ErrInvalidCreds   = errors.New("invalid credentials")
	ErrForbidden      = errors.New("access forbidden")
	ErrInternalServer = errors.New("internal server error")
)

// ─── API Response Wrappers ────────────────────────────────────────────────────

// APIResponse adalah format response sukses yang konsisten.
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// APIError adalah format response error yang konsisten.
// Tidak pernah membocorkan detail teknis (query, stack trace, dll).
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e APIError) Error() string { return e.Message }
