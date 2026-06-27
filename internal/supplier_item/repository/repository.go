package repository

import (
	"context"
	"strings"

	supplierModels "github.com/ganasa18/go-template/internal/supplier/models"
	"github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, item *models.SupplierItem) error
	FindByUUID(ctx context.Context, uuid string) (*models.SupplierItem, error)
	List(ctx context.Context, filters models.SupplierItemListFilters) ([]models.SupplierItem, int64, error)
	Update(ctx context.Context, item *models.SupplierItem) error
	Delete(ctx context.Context, item *models.SupplierItem) error
	FindSupplierByUUID(ctx context.Context, uuid string) (*supplierModels.Supplier, error)
	ExistsBySupplierAndUniq(ctx context.Context, supplierUUID, uniqCode string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, item *models.SupplierItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return apperror.InternalWrap("create supplier item failed", err)
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
		return apperror.InternalWrap("update supplier item failed", err)
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

func (r *repository) ExistsBySupplierAndUniq(ctx context.Context, supplierUUID, uniqCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.SupplierItem{}).
		Where("supplier_uuid = ? AND uniq_code = ? AND deleted_at IS NULL", supplierUUID, strings.ToUpper(strings.TrimSpace(uniqCode))).
		Count(&count).Error
	if err != nil {
		return false, apperror.InternalWrap("check supplier item duplicate failed", err)
	}
	return count > 0, nil
}
