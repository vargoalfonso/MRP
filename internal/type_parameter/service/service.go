package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/type_parameter/models"
	typeRepo "github.com/ganasa18/go-template/internal/type_parameter/repository"
)

type ITypeService interface {
	GetAll(ctx context.Context) ([]models.TypeParameter, error)
	GetByID(ctx context.Context, id int64) (*models.TypeParameter, error)
	Create(ctx context.Context, req models.CreateTypeRequest) (*models.TypeParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateTypeRequest) (*models.TypeParameter, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo typeRepo.ITypeRepository
}

func New(repo typeRepo.ITypeRepository) ITypeService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.TypeParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.TypeParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateTypeRequest) (*models.TypeParameter, error) {
	data := models.TypeParameter{
		TypeCode: req.TypeCode,
		TypeName: req.TypeName,
		Status:   req.Status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateTypeRequest) (*models.TypeParameter, error) {

	// 🔍 cek data ada atau tidak
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 🧩 prepare update data (hanya yang diisi)
	updateData := map[string]interface{}{}

	if req.TypeName != "" {
		updateData["type_name"] = req.TypeName
	}

	if req.Status != "" {
		updateData["status"] = req.Status
	}

	// 🔄 kalau tidak ada field diupdate
	if len(updateData) == 0 {
		return existing, nil
	}

	// 📝 update
	if err := s.repo.Update(ctx, id, updateData); err != nil {
		return nil, err
	}

	// 🔁 ambil data terbaru
	updated, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
