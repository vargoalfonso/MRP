package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/stockdays_parameter/models"
	"github.com/ganasa18/go-template/internal/stockdays_parameter/repository"
)

type IStockdayService interface {
	GetAll(ctx context.Context) ([]models.StockdaysParameter, error)
	GetByID(ctx context.Context, id int64) (*models.StockdaysParameter, error)
	Create(ctx context.Context, req models.CreateStockdaysRequest) (*models.StockdaysParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateStockdaysRequest) (*models.StockdaysParameter, error)
	Delete(ctx context.Context, id int64) error
	BulkCreate(ctx context.Context, req models.BulkCreateStockdaysRequest) error
}

type service struct {
	repo repository.IStockdaysRepository
}

func New(repo repository.IStockdaysRepository) IStockdayService {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]models.StockdaysParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.StockdaysParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateStockdaysRequest) (*models.StockdaysParameter, error) {

	status := "active"
	if req.Status != nil {
		status = *req.Status
	}

	data := models.StockdaysParameter{
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

func (s *service) Update(ctx context.Context, id int64, req models.UpdateStockdaysRequest) (*models.StockdaysParameter, error) {
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

func (s *service) BulkCreate(ctx context.Context, req models.BulkCreateStockdaysRequest) error {
	var data []models.StockdaysParameter

	for _, item := range req.Items {
		data = append(data, models.StockdaysParameter{
			InventoryType:   item.InventoryType,
			ItemUniqCode:    item.ItemUniqCode,
			CalculationType: item.CalculationType,
			Constanta:       item.Constanta,
			Status:          "active",
		})
	}

	return s.repo.BulkCreate(ctx, data)
}
