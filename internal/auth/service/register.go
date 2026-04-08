package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

// registerUser is shared by both stateless and stateful implementations.
func registerUser(ctx context.Context, repo authRepo.IRepository, req models.RegisterRequest) error {
	// Check email not already taken.
	existing, _ := repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return apperror.Conflict("email already registered")
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Provider: "local",
	}
	if err := user.HashPassword(req.Password); err != nil {
		return apperror.InternalWrap("failed to hash password", err)
	}

	return repo.Create(ctx, user)
}
