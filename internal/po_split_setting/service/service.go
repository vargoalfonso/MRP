package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/po_split_setting/models"
	poSplitSettingRepo "github.com/ganasa18/go-template/internal/po_split_setting/repository"
)

type IPOSplitSettingService interface {
	GetAll(ctx context.Context) ([]models.POSplitSetting, error)
	GetByID(ctx context.Context, id int64) (*models.POSplitSetting, error)
	Create(ctx context.Context, req models.CreatePOSplitRequest) (*models.POSplitSetting, error)
	Update(ctx context.Context, id int64, req models.UpdatePOSplitRequest) (*models.POSplitSetting, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo poSplitSettingRepo.IPOSplitSettingRepository
}

func New(repo poSplitSettingRepo.IPOSplitSettingRepository) IPOSplitSettingService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.POSplitSetting, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.POSplitSetting, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreatePOSplitRequest) (*models.POSplitSetting, error) {
	data := models.POSplitSetting{
		BudgetType:    req.BudgetType,
		MinOrderQty:   req.MinOrderQty,
		MaxSplitLines: req.MaxSplitLines,
		SplitRule:     req.SplitRule,
		Status:        req.Status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdatePOSplitRequest) (*models.POSplitSetting, error) {

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updateData := map[string]interface{}{}

	if req.MinOrderQty != 0 {
		updateData["min_order_qty"] = req.MinOrderQty
	}
	if req.MaxSplitLines != 0 {
		updateData["max_split_lines"] = req.MaxSplitLines
	}
	if req.SplitRule != "" {
		updateData["split_rule"] = req.SplitRule
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
