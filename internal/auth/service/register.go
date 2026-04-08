package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

// registerUser is shared by both stateless and stateful implementations.
// Returns the created user so callers can build a response.
func registerUser(ctx context.Context, repo authRepo.IRepository, req models.RegisterRequest) (*models.User, error) {
	// Check email not already taken.
	existing, _ := repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, apperror.Conflict("email already registered")
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Provider: "local",
	}
	if err := user.HashPassword(req.Password); err != nil {
		return nil, apperror.InternalWrap("failed to hash password", err)
	}

	if err := repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
