package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/ganasa18/go-template/internal/warehouse/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func wrapWarehouseDBError(msg string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		lowerMsg := strings.ToLower(pgErr.Message)

		// Undefined table (migration not applied).
		if pgErr.Code == "42P01" && strings.Contains(lowerMsg, "warehouse") {
			return apperror.BadRequest(
				"warehouse table is missing in DB; run migration scripts/migrations/0035_create_warehouse_up.sql",
			)
		}

		// Undefined column (schema mismatch).
		if pgErr.Code == "42703" {
			return apperror.BadRequest(
				"warehouse table schema is out of date (missing column); run migration scripts/migrations/0046_warehouse_sync_schema_up.sql",
			)
		}

		// Unique constraint violation.
		if pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "warehouse_name_plant_key") {
				return apperror.Conflict("warehouse_name already exists for this plant_id")
			}
			return apperror.Conflict("duplicate value violates unique constraint")
		}

		// Check constraint violation (e.g. invalid enum value)
		if pgErr.Code == "23514" {
			if strings.Contains(pgErr.ConstraintName, "warehouse_type_check") {
				return apperror.BadRequest("type_warehouse is not valid; allowed: raw_material, indirect_raw_material, finished_goods, subcon, general")
			}
			return apperror.BadRequest("check constraint violated: " + pgErr.Message)
		}

		// Not-null violation
		if pgErr.Code == "23502" {
			col := pgErr.ColumnName
			if col == "" {
				return apperror.BadRequest("missing required value")
			}
			return apperror.BadRequest("missing required value: " + col)
		}
	}

	return apperror.InternalWrap(msg, err)
}

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
		return wrapWarehouseDBError("create warehouse failed", err)
	}
	return nil
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.Warehouse, error) {
	var warehouse models.Warehouse
	if err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&warehouse).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("warehouse not found")
		}
		return nil, wrapWarehouseDBError("find warehouse failed", err)
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
		return nil, 0, wrapWarehouseDBError("count warehouses failed", err)
	}

	var items []models.Warehouse
	if err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&items).Error; err != nil {
		return nil, 0, wrapWarehouseDBError("list warehouses failed", err)
	}

	return items, total, nil
}

func (r *repository) Update(ctx context.Context, warehouse *models.Warehouse) error {
	if err := r.db.WithContext(ctx).Save(warehouse).Error; err != nil {
		return wrapWarehouseDBError("update warehouse failed", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, warehouse *models.Warehouse) error {
	if err := r.db.WithContext(ctx).Delete(warehouse).Error; err != nil {
		return wrapWarehouseDBError("delete warehouse failed", err)
	}
	return nil
}
