package service

import (
	"context"
	"errors"

	"github.com/ganasa18/go-template/internal/safety_stock_parameter/constant"
	"github.com/ganasa18/go-template/internal/safety_stock_parameter/models"
	safetyStockRepo "github.com/ganasa18/go-template/internal/safety_stock_parameter/repository"
)

type ISafetyStockService interface {
	GetAll(ctx context.Context) ([]models.SafetyStockParameter, error)
	GetByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error)
	Create(ctx context.Context, req models.CreateSafetyStockRequest) (*models.SafetyStockParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateSafetyStockRequest) (*models.SafetyStockParameter, error)
	Delete(ctx context.Context, id int64) error
	BulkCreate(ctx context.Context, req models.BulkCreateSafetyStockRequest) error
	Calculate(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, error)
}

// implementation
type service struct {
	repo safetyStockRepo.ISafetyStockRepository
}

func New(repo safetyStockRepo.ISafetyStockRepository) ISafetyStockService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================

func (s *service) GetAll(ctx context.Context) ([]models.SafetyStockParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateSafetyStockRequest) (*models.SafetyStockParameter, error) {

	_, err := s.repo.FindByItemCode(ctx, req.ItemUniqCode)
	if err == nil {
		return nil, errors.New("item sudah ada")
	}

	// 🔥 VALIDATION DI SINI
	if err := validateCalculationType(req.CalculationType); err != nil {
		return nil, err
	}

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	data := models.SafetyStockParameter{
		InventoryType:   req.InventoryType,
		ItemUniqCode:    req.ItemUniqCode,
		CalculationType: req.CalculationType,
		Constanta:       req.Constanta,
		Status:          status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateSafetyStockRequest) (*models.SafetyStockParameter, error) {

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	err := s.repo.Update(ctx, id, map[string]interface{}{
		"calculation_type": req.CalculationType,
		"constanta":        req.Constanta,
		"status":           status,
	})
	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) BulkCreate(ctx context.Context, req models.BulkCreateSafetyStockRequest) error {
	var data []models.SafetyStockParameter

	for _, item := range req.Items {
		data = append(data, models.SafetyStockParameter{
			InventoryType:   item.InventoryType,
			ItemUniqCode:    item.ItemUniqCode,
			CalculationType: item.CalculationType,
			Constanta:       item.Constanta,
		})
	}

	return s.repo.BulkCreate(ctx, data)
}

func (s *service) Calculate(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, error) {

	param, err := s.repo.FindByItemCode(ctx, itemCode)
	if err != nil {
		return 0, errors.New("parameter not found")
	}

	var forecast float64

	if param.CalculationType == "forecast" {
		f, err := s.repo.GetForecastByItem(ctx, itemCode)
		if err != nil {
			return 0, err
		}
		forecast = f
	}

	result := calculateSafetyStock(
		param.CalculationType,
		prl,
		po,
		workingDays,
		param.Constanta,
		forecast,
	)

	return result, nil
}

func calculateSafetyStock(calcType string, prl, po, workingDays, constanta, forecast float64) float64 {
	if workingDays == 0 {
		return 0
	}

	daily := (prl + po) / workingDays

	switch calcType {
	case string(constant.CalcDays):
		return daily * constanta

	case string(constant.CalcPercentage):
		return daily * (constanta / 100)

	case string(constant.CalcForecast):
		return forecast
	}

	return 0
}
