package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/role/models"
	roleRepo "github.com/ganasa18/go-template/internal/role/repository"
)

// IRoleService defines all role operations
type IRoleService interface {
	GetAll(ctx context.Context) ([]models.Role, error)
	GetByID(ctx context.Context, id int64) (*models.Role, error)
	Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error)
	Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error)
	Delete(ctx context.Context, id int64) error

	// 🔥 RBAC
	GetPermissions(ctx context.Context, role string) (map[string]interface{}, error)
}

// implementation
type service struct {
	repo roleRepo.IRoleRepository
}

// constructor (mirip auth.New)
func New(repo roleRepo.IRoleRepository) IRoleService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================

func (s *service) GetAll(ctx context.Context) ([]models.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Role, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// =========================
// RBAC (IMPORTANT)
// =========================

func (s *service) GetPermissions(ctx context.Context, role string) (map[string]interface{}, error) {
	return s.repo.GetPermissionsByRole(ctx, role)
}
