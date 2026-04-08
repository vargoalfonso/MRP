package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// IRepository is the auth repository contract.
type IRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUUID(ctx context.Context, uuid string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
}

type repository struct {
	db *gorm.DB
}

// New returns a repository backed by the provided *gorm.DB.
func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.InternalWrap("FindByEmail failed", err)
	}
	return &user, nil
}

// FindByUUID looks up by the public uuid column (used for JWT sub claim).
func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("uuid = ? AND deleted_at IS NULL", uuid).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.InternalWrap("FindByUUID failed", err)
	}
	return &user, nil
}

func (r *repository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return apperror.InternalWrap("Create user failed", err)
	}
	return nil
}
