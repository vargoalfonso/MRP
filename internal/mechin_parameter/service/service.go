package service

import (
	"context"
	"math"

	"github.com/ganasa18/go-template/internal/mechin_parameter/models"
	mechinRepo "github.com/ganasa18/go-template/internal/mechin_parameter/repository"
)

type ListQuery struct {
	Limit  int
	Page   int
	Search string
	Status string
}

type IMechinParameterService interface {
	GetAll(ctx context.Context, q ListQuery) (*models.ListMechinParameterResponse, error)
	GetByID(ctx context.Context, id int64) (*models.MechinParameter, error)
	Create(ctx context.Context, req models.CreateMechinParameterRequest) (*models.MechinParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateMechinParameterRequest) (*models.MechinParameter, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo mechinRepo.IMechinParameterRepository
}

func New(repo mechinRepo.IMechinParameterRepository) IMechinParameterService {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context, q ListQuery) (*models.ListMechinParameterResponse, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	items, total, err := s.repo.FindAll(ctx, mechinRepo.ListQuery{
		Limit:  limit,
		Offset: offset,
		Search: q.Search,
		Status: q.Status,
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ListMechinParameterResponse{
		Items: items,
		Pagination: models.Pagination{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.MechinParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateMechinParameterRequest) (*models.MechinParameter, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateMechinParameterRequest) (*models.MechinParameter, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
