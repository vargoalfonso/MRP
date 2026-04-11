package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/employee/models"
	employeeRepo "github.com/ganasa18/go-template/internal/employee/repository"
)

type IEmployeeService interface {
	GetAll(ctx context.Context) ([]models.Employee, error)
	GetByID(ctx context.Context, id int64) (*models.Employee, error)
	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error)
	Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo employeeRepo.IEmployeeRepository
}

func New(repo employeeRepo.IEmployeeRepository) IEmployeeService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.Employee, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
