// Package models defines the data types shared across the auth domain.
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload used for both access and refresh tokens.
type Claims struct {
	jwt.RegisteredClaims
	// UserID mirrors RegisteredClaims.Subject but is kept here for clarity.
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
}

// TokenPair is returned on login (stateful) or login (stateless: RefreshToken is empty).
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"` // empty in stateless mode
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// LoginRequest is the expected JSON body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RefreshRequest is the expected JSON body for POST /auth/refresh (stateful only).
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RegisterRequest is the expected JSON body for POST /auth/register.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}
