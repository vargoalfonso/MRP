package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/unit_measurement/models"
	unitMeasurementRepo "github.com/ganasa18/go-template/internal/unit_measurement/repository"
)

type IUnitMeasurementService interface {
	GetAll(ctx context.Context) ([]models.UnitMeasurement, error)
	GetByID(ctx context.Context, id int64) (*models.UnitMeasurement, error)
	Create(ctx context.Context, req models.CreateUnitRequest) (*models.UnitMeasurement, error)
	Update(ctx context.Context, id int64, req models.UpdateUnitRequest) (*models.UnitMeasurement, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo unitMeasurementRepo.IUnitMeasurementRepository
}

func New(repo unitMeasurementRepo.IUnitMeasurementRepository) IUnitMeasurementService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.UnitMeasurement, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.UnitMeasurement, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateUnitRequest) (*models.UnitMeasurement, error) {
	data := models.UnitMeasurement{
		Name:     req.Name,
		Category: req.Category,
		Status:   req.Status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateUnitRequest) (*models.UnitMeasurement, error) {

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updateData := map[string]interface{}{}

	if req.Category != "" {
		updateData["category"] = req.Category
	}

	if req.Status != "" {
		updateData["status"] = req.Status
	}

	if len(updateData) == 0 {
		return existing, nil
	}

	if err := s.repo.Update(ctx, id, updateData); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
