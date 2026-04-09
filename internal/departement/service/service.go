package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/departement/models"
	departementRepo "github.com/ganasa18/go-template/internal/departement/repository"
)

type IDepartement interface {
	GetAll(ctx context.Context) ([]models.Department, error)
	GetByID(ctx context.Context, id int64) (*models.Department, error)
	Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error)
	Update(ctx context.Context, id int64, req models.UpdateDepartmentRequest) (*models.Department, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo departementRepo.IDepartementRepository
}

func New(repo departementRepo.IDepartementRepository) IDepartement {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.Department, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Department, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateDepartmentRequest) (*models.Department, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
