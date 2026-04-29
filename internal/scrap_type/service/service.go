package service

import (
	"context"
	"math"
	"strings"

	"github.com/ganasa18/go-template/internal/scrap_type/models"
	"github.com/ganasa18/go-template/internal/scrap_type/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

type IService interface {
	List(ctx context.Context, q ListQuery) (*models.ListScrapTypeResponse, error)
	GetByID(ctx context.Context, id int64) (*models.ScrapTypeResponse, error)
	Create(ctx context.Context, req models.CreateScrapTypeRequest, createdBy string) (*models.ScrapTypeResponse, error)
	Update(ctx context.Context, id int64, req models.UpdateScrapTypeRequest) (*models.ScrapTypeResponse, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

type ListQuery struct {
	Page   int
	Limit  int
	Search string
	Status string
}

func (s *service) List(ctx context.Context, q ListQuery) (*models.ListScrapTypeResponse, error) {
	// Normalize pagination
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	repoQuery := repository.ListQuery{
		Page:   page,
		Limit:  limit,
		Offset: offset,
		Search: q.Search,
		Status: q.Status,
	}

	items, total, err := s.repo.List(ctx, repoQuery)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ScrapTypeResponse, len(items))
	for i, item := range items {
		responses[i] = toResponse(item)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ListScrapTypeResponse{
		Items: responses,
		Pagination: models.Pagination{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.ScrapTypeResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toResponse(*item)
	return &resp, nil
}

func (s *service) Create(ctx context.Context, req models.CreateScrapTypeRequest, createdBy string) (*models.ScrapTypeResponse, error) {
	// Validate name uniqueness
	if existing, _ := s.repo.GetByName(ctx, req.Name); existing != nil {
		return nil, apperror.BadRequest("scrap type with this name already exists")
	}

	// Generate code
	code, err := s.repo.GetNextCode(ctx)
	if err != nil {
		return nil, err
	}

	status := "Active"
	if req.Status != "" {
		status = req.Status
	}

	item := &models.ScrapType{
		Code:        code,
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		Status:      status,
		IsSystem:    false,
	}

	if createdBy != "" {
		item.CreatedBy = &createdBy
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	resp := toResponse(*item)
	return &resp, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateScrapTypeRequest) (*models.ScrapTypeResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// System types: only allow description update
	if item.IsSystem {
		if req.Name != nil || req.Status != nil {
			return nil, apperror.BadRequest("cannot update system scrap type name or status")
		}
		if req.Description != nil {
			item.Description = req.Description
		}
		if err := s.repo.Update(ctx, item); err != nil {
			return nil, err
		}
		resp := toResponse(*item)
		return &resp, nil
	}

	// Non-system types: full update
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		// Check uniqueness
		if existing, _ := s.repo.GetByName(ctx, name); existing != nil && existing.ID != id {
			return nil, apperror.BadRequest("scrap type with this name already exists")
		}
		item.Name = name
	}

	if req.Description != nil {
		item.Description = req.Description
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		// Check if used in transactions before deactivating
		if status == "Inactive" {
			used, _ := s.repo.IsUsedInTransactions(ctx, id)
			if used {
				return nil, apperror.BadRequest("cannot deactivate scrap type that is in use")
			}
		}
		item.Status = status
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	resp := toResponse(*item)
	return &resp, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cannot delete system types
	if item.IsSystem {
		return apperror.BadRequest("cannot delete system scrap type")
	}

	// Check if used in transactions
	used, _ := s.repo.IsUsedInTransactions(ctx, id)
	if used {
		return apperror.Conflict("scrap type is in use")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func toResponse(s models.ScrapType) models.ScrapTypeResponse {
	return models.ScrapTypeResponse{
		ID:          s.ID,
		Code:        s.Code,
		Name:        s.Name,
		Description: s.Description,
		Status:      s.Status,
		IsSystem:    s.IsSystem,
		CreatedBy:   s.CreatedBy,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
