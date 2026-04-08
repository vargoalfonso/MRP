// Package service contains the Authenticator interface and its factory.
// Switch between stateless and stateful modes by setting JWT_MODE in .env —
// no code changes required.
package service

import (
	"context"

	"github.com/ganasa18/go-template/config"
	"github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/redis/go-redis/v9"
)

// Authenticator is the single interface for all JWT operations.
// Stateless mode leaves RefreshTokens returning an error and RevokeToken a no-op.
type Authenticator interface {
	// Register creates a new local user account and returns the created user.
	Register(ctx context.Context, req models.RegisterRequest) (*models.User, error)

	// Login validates credentials and returns a token pair.
	Login(ctx context.Context, req models.LoginRequest) (*models.TokenPair, error)

	// ValidateAccessToken parses and validates an access token string.
	// Returns the embedded Claims on success.
	ValidateAccessToken(ctx context.Context, tokenStr string) (*models.Claims, error)

	// RefreshTokens issues a new token pair from a valid refresh token.
	// Returns apperror.ErrForbidden in stateless mode.
	RefreshTokens(ctx context.Context, refreshToken string) (*models.TokenPair, error)

	// RevokeToken invalidates a token by its jti (stateful only; no-op for stateless).
	RevokeToken(ctx context.Context, jti string) error

	// Mode returns "stateless" or "stateful".
	Mode() string
}

// New is the factory that selects the correct Authenticator implementation
// based on cfg.JWTMode. Callers never need to know which one they get.
func New(cfg *config.Config, repo authRepo.IRepository, rdb *redis.Client) Authenticator {
	if cfg.IsStateful() {
		return newStateful(cfg, repo, rdb)
	}
	return newStateless(cfg, repo)
}
