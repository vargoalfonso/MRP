package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/global_parameter/models"
	globalParameterRepo "github.com/ganasa18/go-template/internal/global_parameter/repository"
)

type IGlobalParameterService interface {
	GetAll(ctx context.Context) ([]models.GlobalParameter, error)
	Create(ctx context.Context, req models.CreateGlobalParameterRequest) (*models.GlobalParameter, error)
	GetByID(ctx context.Context, id int64) (*models.GlobalParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateGlobalParameterRequest) (*models.GlobalParameter, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo globalParameterRepo.IGlobalParameterRepository
}

func New(repo globalParameterRepo.IGlobalParameterRepository) IGlobalParameterService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.GlobalParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.GlobalParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateGlobalParameterRequest) (*models.GlobalParameter, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateGlobalParameterRequest) (*models.GlobalParameter, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
