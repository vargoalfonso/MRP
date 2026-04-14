package service

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/warehouse/models"
	warehouseRepo "github.com/ganasa18/go-template/internal/warehouse/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
)

type WarehouseService interface {
	Create(ctx context.Context, req models.CreateWarehouseRequest) (*models.Warehouse, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Warehouse, error)
	List(ctx context.Context, query models.ListWarehouseQuery) (*models.WarehouseListResult, error)
	Update(ctx context.Context, uuid string, req models.UpdateWarehouseRequest) (*models.Warehouse, error)
	Delete(ctx context.Context, uuid string) error
}

type service struct {
	repo warehouseRepo.IRepository
}

func New(repo warehouseRepo.IRepository) WarehouseService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateWarehouseRequest) (*models.Warehouse, error) {
	typeWarehouse, err := normalizeType(req.TypeWarehouse)
	if err != nil {
		return nil, err
	}

	warehouse := &models.Warehouse{
		UUID:          uuid.NewString(),
		WarehouseName: models.Trimmed(req.WarehouseName),
		TypeWarehouse: typeWarehouse,
		PlantID:       models.Trimmed(req.PlantID),
	}

	if err := s.repo.Create(ctx, warehouse); err != nil {
		return nil, err
	}
	return warehouse, nil
}

func (s *service) GetByUUID(ctx context.Context, uuid string) (*models.Warehouse, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, apperror.BadRequest("warehouse id is required")
	}
	return s.repo.FindByUUID(ctx, uuid)
}

func (s *service) List(ctx context.Context, query models.ListWarehouseQuery) (*models.WarehouseListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filters := models.WarehouseListFilters{
		Search: models.Trimmed(query.Search),
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
	if strings.TrimSpace(query.TypeWarehouse) != "" {
		typeWarehouse, err := normalizeType(query.TypeWarehouse)
		if err != nil {
			return nil, err
		}
		filters.TypeWarehouse = &typeWarehouse
	}
	if strings.TrimSpace(query.PlantID) != "" {
		plantID := models.Trimmed(query.PlantID)
		filters.PlantID = &plantID
	}

	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &models.WarehouseListResult{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) Update(ctx context.Context, uuid string, req models.UpdateWarehouseRequest) (*models.Warehouse, error) {
	warehouse, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	typeWarehouse, err := normalizeType(req.TypeWarehouse)
	if err != nil {
		return nil, err
	}

	warehouse.WarehouseName = models.Trimmed(req.WarehouseName)
	warehouse.TypeWarehouse = typeWarehouse
	warehouse.PlantID = models.Trimmed(req.PlantID)

	if err := s.repo.Update(ctx, warehouse); err != nil {
		return nil, err
	}
	return warehouse, nil
}

func (s *service) Delete(ctx context.Context, uuid string) error {
	warehouse, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, warehouse)
}

func normalizeType(value string) (string, error) {
	cleaned := strings.ToLower(strings.TrimSpace(value))
	if cleaned == "" {
		return "", apperror.BadRequest("type_warehouse is required")
	}

	switch cleaned {
	case models.WarehouseTypeRawMaterial,
		models.WarehouseTypeWIP,
		models.WarehouseTypeFinishedGoods,
		models.WarehouseTypeSubcon,
		models.WarehouseTypeGeneral:
		return cleaned, nil
	default:
		return "", apperror.BadRequest("type_warehouse must be one of: raw_material, wip, finished_goods, subcon, general")
	}
}
