package repository

import (
	"context"
	"errors"
	"strings"

	supplierModels "github.com/ganasa18/go-template/internal/supplier/models"
	"github.com/ganasa18/go-template/internal/supplier_item/models"
	warehouseModels "github.com/ganasa18/go-template/internal/warehouse/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// wrapSupplierItemDBError maps well-known Postgres errors to client-facing
// errors instead of a generic 500. The raw cause stays attached to the
// AppError so it can be logged server-side.
func wrapSupplierItemDBError(msg string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperror.Conflict("uniq code sudah terdaftar untuk supplier ini pada kategori tersebut")
		case "23502": // not_null_violation
			return apperror.BadRequest("kolom wajib belum diisi: " + pgErr.ColumnName)
		case "22001": // string_data_right_truncation
			return apperror.BadRequest("salah satu nilai terlalu panjang untuk kolomnya")
		}
	}
	return apperror.InternalWrap(msg, err)
}

type IRepository interface {
	Create(ctx context.Context, item *models.SupplierItem) error
	FindByUUID(ctx context.Context, uuid string) (*models.SupplierItem, error)
	List(ctx context.Context, filters models.SupplierItemListFilters) ([]models.SupplierItem, int64, error)
	Update(ctx context.Context, item *models.SupplierItem) error
	Delete(ctx context.Context, item *models.SupplierItem) error
	FindSupplierByUUID(ctx context.Context, uuid string) (*supplierModels.Supplier, error)
	FindWarehouseByUUID(ctx context.Context, uuid string) (*warehouseModels.Warehouse, error)
	ExistsBySupplierAndUniqType(ctx context.Context, supplierUUID, uniqCode, itemType, excludeUUID string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, item *models.SupplierItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return wrapSupplierItemDBError("create supplier item failed", err)
	}
	return nil
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.SupplierItem, error) {
	var item models.SupplierItem
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("supplier item tidak ditemukan")
		}
		return nil, apperror.InternalWrap("find supplier item failed", err)
	}
	return &item, nil
}

func (r *repository) List(ctx context.Context, filters models.SupplierItemListFilters) ([]models.SupplierItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.SupplierItem{})

	if filters.Search != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where(
			"supplier_name ILIKE ? OR sebango_code ILIKE ? OR uniq_code ILIKE ? OR type ILIKE ? OR COALESCE(description, '') ILIKE ?",
			search, search, search, search, search,
		)
	}
	if filters.SupplierUUID != nil {
		query = query.Where("supplier_uuid = ?", *filters.SupplierUUID)
	}
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count supplier items failed", err)
	}

	var items []models.SupplierItem
	err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&items).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list supplier items failed", err)
	}
	return items, total, nil
}

func (r *repository) Update(ctx context.Context, item *models.SupplierItem) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return wrapSupplierItemDBError("update supplier item failed", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, item *models.SupplierItem) error {
	if err := r.db.WithContext(ctx).Delete(item).Error; err != nil {
		return apperror.InternalWrap("delete supplier item failed", err)
	}
	return nil
}

func (r *repository) FindSupplierByUUID(ctx context.Context, uuid string) (*supplierModels.Supplier, error) {
	var supplier supplierModels.Supplier
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&supplier).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("supplier tidak ditemukan")
		}
		return nil, apperror.InternalWrap("find supplier failed", err)
	}
	return &supplier, nil
}

func (r *repository) FindWarehouseByUUID(ctx context.Context, uuid string) (*warehouseModels.Warehouse, error) {
	var wh warehouseModels.Warehouse
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&wh).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("warehouse tidak ditemukan")
		}
		return nil, apperror.InternalWrap("find warehouse failed", err)
	}
	return &wh, nil
}

func (r *repository) ExistsBySupplierAndUniqType(ctx context.Context, supplierUUID, uniqCode, itemType, excludeUUID string) (bool, error) {
	query := r.db.WithContext(ctx).Model(&models.SupplierItem{}).
		Where(
			"supplier_uuid = ? AND uniq_code = ? AND type = ? AND deleted_at IS NULL",
			supplierUUID,
			strings.ToUpper(strings.TrimSpace(uniqCode)),
			strings.ToLower(strings.TrimSpace(itemType)),
		)
	if strings.TrimSpace(excludeUUID) != "" {
		query = query.Where("uuid <> ?", strings.TrimSpace(excludeUUID))
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, apperror.InternalWrap("check supplier item duplicate failed", err)
	}
	return count > 0, nil
}
