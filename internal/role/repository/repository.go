package repository

import (
	"context"
	"encoding/json"

	"github.com/ganasa18/go-template/internal/role/models"
	"gorm.io/gorm"
)

type IRoleRepository interface {
	FindAll(ctx context.Context) ([]models.Role, error)
	FindByID(ctx context.Context, id int64) (*models.Role, error)
	Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error)
	Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error)
	Delete(ctx context.Context, id int64) error

	GetPermissionsByRole(ctx context.Context, roleName string) (map[string]interface{}, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRoleRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role

	err := r.db.WithContext(ctx).Find(&roles).Error
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.Role, error) {
	var role models.Role

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error) {
	permissionsJSON, _ := json.Marshal(req.Permissions)

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions, // langsung map
		Status:      req.Status,
	}

	err := r.db.WithContext(ctx).Create(&role).Error
	if err != nil {
		return nil, err
	}

	// re-assign JSON (optional kalau model pakai map)
	json.Unmarshal(permissionsJSON, &role.Permissions)

	return &role, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error) {
	var role models.Role

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&role).Error
	if err != nil {
		return nil, err
	}

	role.Name = req.Name
	role.Description = req.Description
	role.Permissions = req.Permissions
	role.Status = req.Status

	err = r.db.WithContext(ctx).Save(&role).Error
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Role{}).Error
}

func (r *repository) GetPermissionsByRole(ctx context.Context, roleName string) (map[string]interface{}, error) {
	var role models.Role

	err := r.db.WithContext(ctx).
		Select("permissions").
		Where("name = ?", roleName).
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return role.Permissions, nil
}
