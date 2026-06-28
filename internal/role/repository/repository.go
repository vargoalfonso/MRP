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
	FindByName(ctx context.Context, name string) (*models.Role, error)
	FindUsersByRoleID(ctx context.Context, roleID int64) ([]models.RoleUser, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRoleRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	if err := r.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}

	type roleCount struct {
		RoleID int64 `gorm:"column:role_id"`
		Count  int64 `gorm:"column:count"`
	}
	var counts []roleCount
	if err := r.db.WithContext(ctx).
		Raw("SELECT role_id, COUNT(*) AS count FROM employees GROUP BY role_id").
		Scan(&counts).Error; err == nil {
		cm := make(map[int64]int64, len(counts))
		for _, c := range counts {
			cm[c.RoleID] = c.Count
		}
		for i := range roles {
			roles[i].UserCount = cm[roles[i].ID]
		}
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

func (r *repository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (r *repository) FindUsersByRoleID(ctx context.Context, roleID int64) ([]models.RoleUser, error) {
	var users []models.RoleUser
	err := r.db.WithContext(ctx).
		Table("employees").
		Select("id, full_name, email, job_title, status, department_id, join_date").
		Where("role_id = ?", roleID).
		Order("id ASC").
		Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
