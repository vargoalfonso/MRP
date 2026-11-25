package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/process_parameter/models"
	processParameterRepo "github.com/ganasa18/go-template/internal/process_parameter/repository"
)

type IProcessParameterService interface {
	GetAll(ctx context.Context) ([]models.ProcessParameter, error)
	Create(ctx context.Context, req models.CreateProcessRequest) (*models.ProcessParameter, error)
	GetByID(ctx context.Context, id int64) (*models.ProcessParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateProcessRequest) (*models.ProcessParameter, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo processParameterRepo.IProcessParameterRepository
}

func New(repo processParameterRepo.IProcessParameterRepository) IProcessParameterService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.ProcessParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.ProcessParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateProcessRequest) (*models.ProcessParameter, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateProcessRequest) (*models.ProcessParameter, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
