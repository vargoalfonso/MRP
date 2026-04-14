package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// IRepository is the auth repository contract.
type IRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUUID(ctx context.Context, uuid string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, id int64, user *models.User) error
	Delete(ctx context.Context, email string) error

	FindValidActivation(ctx context.Context, token string) (*models.UserActivation, error)
	UpdateUserPassword(ctx context.Context, userID int64, password string) error
	MarkTokenUsed(ctx context.Context, id int64) error
	UpdateUserRole(ctx context.Context, userID int64, role string) error
	FindUserByEmployeeID(ctx context.Context, employeeID int64) (*models.User, error)
}

type repository struct {
	db *gorm.DB
}

// New returns a repository backed by the provided *gorm.DB.
func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) FindUserByEmployeeID(ctx context.Context, employeeID int64) (*models.User, error) {
	var user models.User

	err := r.db.WithContext(ctx).
		Where("employee_id = ?", strconv.FormatInt(employeeID, 10)).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *repository) UpdateUserRole(ctx context.Context, userID int64, role string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("roles", role).Error
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

func (r *repository) Update(ctx context.Context, id int64, user *models.User) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Save(user).Error; err != nil {
		return apperror.InternalWrap("Update user failed", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).
		Where("email = ?", email).
		Delete(&models.User{}).Error
}

func (r *repository) FindValidActivation(ctx context.Context, token string) (*models.UserActivation, error) {
	var act models.UserActivation

	err := r.db.WithContext(ctx).
		Where("token = ? AND used = false AND expired_at > ?", token, time.Now()).
		First(&act).Error

	if err != nil {
		return nil, fmt.Errorf("token tidak valid atau expired")
	}

	return &act, nil
}

func (r *repository) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password":    password,
			"is_verified": true,
			"updated_at":  time.Now(),
		}).Error
}

func (r *repository) MarkTokenUsed(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).
		Model(&models.UserActivation{}).
		Where("id = ? AND used = false", id).
		Update("used", true)

	if res.Error != nil {
		return res.Error
	}

	// 🔥 safeguard: kalau 0 row affected → sudah dipakai
	if res.RowsAffected == 0 {
		return fmt.Errorf("token sudah digunakan")
	}

	return nil
}
