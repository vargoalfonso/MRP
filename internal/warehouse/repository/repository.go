package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/warehouse/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, warehouse *models.Warehouse) error
	FindByUUID(ctx context.Context, uuid string) (*models.Warehouse, error)
	List(ctx context.Context, filters models.WarehouseListFilters) ([]models.Warehouse, int64, error)
	Update(ctx context.Context, warehouse *models.Warehouse) error
	Delete(ctx context.Context, warehouse *models.Warehouse) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, warehouse *models.Warehouse) error {
	if err := r.db.WithContext(ctx).Create(warehouse).Error; err != nil {
		return apperror.InternalWrap("create warehouse failed", err)
	}
	return nil
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.Warehouse, error) {
	var warehouse models.Warehouse
	if err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&warehouse).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("warehouse not found")
		}
		return nil, apperror.InternalWrap("find warehouse failed", err)
	}
	return &warehouse, nil
}

func (r *repository) List(ctx context.Context, filters models.WarehouseListFilters) ([]models.Warehouse, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Warehouse{})

	if filters.Search != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where("warehouse_name ILIKE ? OR type_warehouse ILIKE ? OR plant_id ILIKE ?", search, search, search)
	}
	if filters.TypeWarehouse != nil {
		query = query.Where("type_warehouse = ?", *filters.TypeWarehouse)
	}
	if filters.PlantID != nil {
		query = query.Where("plant_id = ?", *filters.PlantID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count warehouses failed", err)
	}

	var items []models.Warehouse
	if err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list warehouses failed", err)
	}

	return items, total, nil
}

func (r *repository) Update(ctx context.Context, warehouse *models.Warehouse) error {
	if err := r.db.WithContext(ctx).Save(warehouse).Error; err != nil {
		return apperror.InternalWrap("update warehouse failed", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, warehouse *models.Warehouse) error {
	if err := r.db.WithContext(ctx).Delete(warehouse).Error; err != nil {
		return apperror.InternalWrap("delete warehouse failed", err)
	}
	return nil
}
