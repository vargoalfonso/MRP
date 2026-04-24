package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/product_return/models"
	productReturnRepo "github.com/ganasa18/go-template/internal/product_return/repository"
)

type IProductReturn interface {
	GetAll(ctx context.Context, page, limit int) ([]models.ProductReturn, int64, error)
	GetByID(ctx context.Context, id int64) (*models.ProductReturn, error)
	Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error)
	Update(ctx context.Context, id int64, req models.UpdateProductReturnRequest) (*models.ProductReturn, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo productReturnRepo.IProductReturnRepository
}

func New(repo productReturnRepo.IProductReturnRepository) IProductReturn {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context, page, limit int) ([]models.ProductReturn, int64, error) {
	return s.repo.FindAll(ctx, page, limit)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.ProductReturn, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateProductReturnRequest) (*models.ProductReturn, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateProductReturnRequest) (*models.ProductReturn, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
