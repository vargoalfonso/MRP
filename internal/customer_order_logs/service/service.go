package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/customer_order_logs/models"
	"github.com/ganasa18/go-template/internal/customer_order_logs/repository"
	"github.com/google/uuid"
)

type IService interface {
	Create(ctx context.Context, req models.CreateLogRequest) (*models.LogResponse, error)
	List(ctx context.Context, q models.ListLogQuery) (*models.ListLogResponse, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateLogRequest) (*models.LogResponse, error) {
	source := req.Source
	if source == "" {
		source = "automation"
	}

	log := &models.CustomerOrderAutomationLog{
		UUID:                uuid.NewString(),
		DocumentNumber:      req.DocumentNumber,
		RowNo:               req.RowNo,
		ItemUniqCode:        req.ItemUniqCode,
		PartName:            req.PartName,
		Description:         req.Description,
		QtyActive:           req.QtyActive,
		FailureReason:       req.FailureReason,
		SpecialInstructions: req.SpecialInstructions,
		Source:              source,
		Status:              "failed",
	}

	if err := s.repo.Create(ctx, log); err != nil {
		return nil, err
	}

	resp := models.ToLogResponse(log)
	return &resp, nil
}

func (s *service) List(ctx context.Context, q models.ListLogQuery) (*models.ListLogResponse, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}

	filters := models.ListLogFilters{
		Search:         q.Search,
		DocumentNumber: q.DocumentNumber,
		Limit:          limit,
		Offset:         (page - 1) * limit,
	}

	rows, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	items := make([]models.LogResponse, 0, len(rows))
	for i := range rows {
		items = append(items, models.ToLogResponse(&rows[i]))
	}

	return &models.ListLogResponse{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}
