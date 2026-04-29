package service

import (
	"context"
	"math"
	"strings"

	"github.com/ganasa18/go-template/internal/machine_pattern/models"
	"github.com/ganasa18/go-template/internal/machine_pattern/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

type IService interface {
	List(ctx context.Context, q ListQuery) (*models.ListMachinePatternResponse, error)
	GetByID(ctx context.Context, id int64) (*models.MachinePatternResponse, error)
	Create(ctx context.Context, req models.CreateMachinePatternRequest, createdBy string) (*models.MachinePatternResponse, error)
	Update(ctx context.Context, id int64, req models.UpdateMachinePatternRequest) (*models.MachinePatternResponse, error)
	Delete(ctx context.Context, id int64) error
	BulkCreate(ctx context.Context, req models.BulkMachinePatternRequest, createdBy string) (*models.BulkMachinePatternResponse, error)
	GetSummary(ctx context.Context) (*models.MachinePatternSummary, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

type ListQuery struct {
	Page       int
	Limit      int
	Search     string
	MachineID  int64
	MovingType string
	Status     string
	UniqCode   string
}

func (s *service) List(ctx context.Context, q ListQuery) (*models.ListMachinePatternResponse, error) {
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
		Page:       page,
		Limit:      limit,
		Offset:     offset,
		Search:     q.Search,
		MachineID:  q.MachineID,
		MovingType: q.MovingType,
		Status:     q.Status,
		UniqCode:   q.UniqCode,
	}

	items, total, err := s.repo.List(ctx, repoQuery)
	if err != nil {
		return nil, err
	}

	responses := make([]models.MachinePatternResponse, len(items))
	for i, item := range items {
		responses[i] = toResponse(item)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ListMachinePatternResponse{
		Items: responses,
		Pagination: models.Pagination{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.MachinePatternResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toResponse(*item)
	return &resp, nil
}

func (s *service) Create(ctx context.Context, req models.CreateMachinePatternRequest, createdBy string) (*models.MachinePatternResponse, error) {
	// Validate machine exists
	exists, err := s.repo.ValidateMachineExists(ctx, req.MachineID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.BadRequest("machine not found")
	}

	// Check duplicate: uniq_code + machine_id unique
	existing, err := s.repo.GetByUniqAndMachine(ctx, req.UniqCode, req.MachineID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.Conflict("machine pattern with this uniq_code and machine already exists")
	}

	status := "Active"
	if req.Status != "" {
		status = strings.TrimSpace(req.Status)
	}

	movingType := "Normal"
	if req.MovingType != "" {
		movingType = strings.TrimSpace(req.MovingType)
	}

	item := &models.MachinePattern{
		UniqCode:     strings.TrimSpace(req.UniqCode),
		MachineID:    req.MachineID,
		CycleTime:    req.CycleTime,
		PatternValue: 1.0,
		WorkingDays:  26,
		MovingType:   movingType,
		MinOutput:    req.MinOutput,
		PRLReference: req.PRLReference,
		Status:       status,
	}

	if req.PatternValue != nil {
		item.PatternValue = *req.PatternValue
	}
	if req.WorkingDays != nil {
		item.WorkingDays = *req.WorkingDays
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

func (s *service) Update(ctx context.Context, id int64, req models.UpdateMachinePatternRequest) (*models.MachinePatternResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.CycleTime != nil {
		item.CycleTime = req.CycleTime
	}
	if req.PatternValue != nil {
		item.PatternValue = *req.PatternValue
	}
	if req.WorkingDays != nil {
		item.WorkingDays = *req.WorkingDays
	}
	if req.MovingType != "" {
		item.MovingType = strings.TrimSpace(req.MovingType)
	}
	if req.MinOutput != nil {
		item.MinOutput = req.MinOutput
	}
	if req.PRLReference != nil {
		item.PRLReference = req.PRLReference
	}
	if req.Status != "" {
		item.Status = strings.TrimSpace(req.Status)
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	resp := toResponse(*item)
	return &resp, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) GetSummary(ctx context.Context) (*models.MachinePatternSummary, error) {
	return s.repo.GetSummary(ctx)
}

func (s *service) BulkCreate(ctx context.Context, req models.BulkMachinePatternRequest, createdBy string) (*models.BulkMachinePatternResponse, error) {
	created := 0
	updated := 0

	for _, item := range req.Items {
		// Validate machine exists
		exists, err := s.repo.ValidateMachineExists(ctx, item.MachineID)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}

		existing, err := s.repo.GetByUniqAndMachine(ctx, item.UniqCode, item.MachineID)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			// Update existing
			if item.CycleTime != nil {
				existing.CycleTime = item.CycleTime
			}
			if item.PatternValue != nil {
				existing.PatternValue = *item.PatternValue
			}
			if item.WorkingDays != nil {
				existing.WorkingDays = *item.WorkingDays
			}
			if item.MovingType != "" {
				existing.MovingType = strings.TrimSpace(item.MovingType)
			}
			if item.MinOutput != nil {
				existing.MinOutput = item.MinOutput
			}
			if item.PRLReference != nil {
				existing.PRLReference = item.PRLReference
			}
			if err := s.repo.Update(ctx, existing); err != nil {
				return nil, err
			}
			updated++
		} else {
			// Create new
			movingType := "Normal"
			if item.MovingType != "" {
				movingType = strings.TrimSpace(item.MovingType)
			}

			newItem := &models.MachinePattern{
				UniqCode:     strings.TrimSpace(item.UniqCode),
				MachineID:    item.MachineID,
				CycleTime:    item.CycleTime,
				PatternValue: 1.0,
				WorkingDays:  26,
				MovingType:   movingType,
				MinOutput:    item.MinOutput,
				PRLReference: item.PRLReference,
				Status:       "Active",
			}

			if item.PatternValue != nil {
				newItem.PatternValue = *item.PatternValue
			}
			if item.WorkingDays != nil {
				newItem.WorkingDays = *item.WorkingDays
			}

			if createdBy != "" {
				newItem.CreatedBy = &createdBy
			}

			if err := s.repo.Create(ctx, newItem); err != nil {
				return nil, err
			}
			created++
		}
	}

	return &models.BulkMachinePatternResponse{
		Created: created,
		Updated: updated,
	}, nil
}

func toResponse(m models.MachinePattern) models.MachinePatternResponse {
	return models.MachinePatternResponse{
		ID:           m.ID,
		UniqCode:     m.UniqCode,
		MachineID:    m.MachineID,
		CycleTime:    m.CycleTime,
		PatternValue: m.PatternValue,
		WorkingDays:  m.WorkingDays,
		MovingType:   m.MovingType,
		MinOutput:    m.MinOutput,
		PRLReference: m.PRLReference,
		Status:       m.Status,
		CreatedBy:    m.CreatedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
