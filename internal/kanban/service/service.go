package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ganasa18/go-template/internal/kanban/models"
	kanbanRepo "github.com/ganasa18/go-template/internal/kanban/repository"
)

type IKanbanParameterService interface {
	GetAll(ctx context.Context) ([]models.KanbanParameter, error)
	GetByID(ctx context.Context, id int64) (*models.KanbanParameter, error)
	GetByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error)
	Delete(ctx context.Context, id int64) error
	Create(ctx context.Context, req models.CreateKanbanParameterRequest) (*models.KanbanParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateKanbanParameterRequest) (*models.KanbanParameter, error)
}

// implementation
type service struct {
	repo kanbanRepo.IKanbanParameterRepository
}

func New(repo kanbanRepo.IKanbanParameterRepository) IKanbanParameterService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) Create(ctx context.Context, req models.CreateKanbanParameterRequest) (*models.KanbanParameter, error) {
	totalKanban, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	totalPO := totalKanban + 1

	formatted := fmt.Sprintf("%04d", totalPO)

	kanbanNumber := fmt.Sprintf("KBN-%d-%s", time.Now().Year(), formatted)

	data := models.KanbanParameter{
		KanbanNumber: kanbanNumber,
		ItemUniqCode: req.ItemUniqCode,
		KanbanQty:    req.KanbanQty,
		MinStock:     req.MinStock,
		MaxStock:     req.MaxStock,
		Status:       req.Status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) GetAll(ctx context.Context) ([]models.KanbanParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.KanbanParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) GetByItemCode(ctx context.Context, code string) (*models.KanbanParameter, error) {
	return s.repo.FindByItemCode(ctx, code)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateKanbanParameterRequest) (*models.KanbanParameter, error) {
	data, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.KanbanQty != 0 {
		data.KanbanQty = req.KanbanQty
	}
	if req.MinStock != 0 {
		data.MinStock = req.MinStock
	}
	if req.MaxStock != 0 {
		data.MaxStock = req.MaxStock
	}
	if req.Status != nil {
		data.Status = *req.Status
	}

	if err := s.repo.Update(ctx, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
