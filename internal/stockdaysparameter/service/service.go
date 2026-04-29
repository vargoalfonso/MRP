package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/stockdaysparameter/models"
	"github.com/ganasa18/go-template/internal/stockdaysparameter/repository"
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
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	data := models.StockdaysParameter{
		IventoryType:    req.IventoryType,
		ItemCode:        req.ItemCode,
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
	updateData := map[string]interface{}{}

	if req.IventoryType != "" {
		updateData["ivertory_type"] = req.IventoryType
	}

	if req.ItemCode != "" {
		updateData["item_code"] = req.ItemCode
	}

	if req.CalculationType != "" {
		updateData["calculation_type"] = req.CalculationType
	}

	if req.Constanta != 0 {
		updateData["constanta"] = req.Constanta
	}

	if req.Status != nil {
		updateData["status"] = *req.Status
	}

	if err := s.repo.Update(ctx, id, updateData); err != nil {
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
			IventoryType:    item.IventoryType,
			ItemCode:        item.ItemCode,
			CalculationType: item.CalculationType,
			Constanta:       item.Constanta,
			Status:          "active",
		})
	}

	return s.repo.BulkCreate(ctx, data)
}
