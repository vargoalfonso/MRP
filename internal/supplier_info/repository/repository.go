package repository

import (
	"context"
	"strings"

	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	supplierItemModels "github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, info *siModels.SupplierInfo) error
	FindByUUID(ctx context.Context, uuid string) (*siModels.SupplierInfo, error)
	List(ctx context.Context, query siModels.ListSupplierInfoQuery) ([]siModels.SupplierInfo, int64, error)
	Update(ctx context.Context, info *siModels.SupplierInfo) error
	Delete(ctx context.Context, info *siModels.SupplierInfo) error
	FindSupplierItem(ctx context.Context, uniq, supplierName, itemType string) (*supplierItemModels.SupplierItem, error)
	ExistsByUniqSupplierType(ctx context.Context, uniq, supplierName, typeLabel string) (bool, error)
	// ListActiveSupplierItems dan ListAll dipakai bulk import agar validasi
	// dilakukan in-memory (tanpa query per baris).
	ListActiveSupplierItems(ctx context.Context) ([]supplierItemModels.SupplierItem, error)
	ListAll(ctx context.Context) ([]siModels.SupplierInfo, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, info *siModels.SupplierInfo) error {
	if err := r.db.WithContext(ctx).Create(info).Error; err != nil {
		return apperror.InternalWrap("create supplier info failed", err)
	}
	return nil
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*siModels.SupplierInfo, error) {
	var info siModels.SupplierInfo
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&info).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("supplier info tidak ditemukan")
		}
		return nil, apperror.InternalWrap("find supplier info failed", err)
	}
	return &info, nil
}

func (r *repository) List(ctx context.Context, query siModels.ListSupplierInfoQuery) ([]siModels.SupplierInfo, int64, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	db := r.db.WithContext(ctx).Model(&siModels.SupplierInfo{})
	if strings.TrimSpace(query.Search) != "" {
		s := "%" + strings.TrimSpace(query.Search) + "%"
		db = db.Where("uniq ILIKE ? OR uniq_zahir ILIKE ? OR supplier_name ILIKE ? OR type ILIKE ?", s, s, s, s)
	}
	if strings.TrimSpace(query.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(query.Status))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count supplier info failed", err)
	}

	var items []siModels.SupplierInfo
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list supplier info failed", err)
	}
	return items, total, nil
}

func (r *repository) Update(ctx context.Context, info *siModels.SupplierInfo) error {
	if err := r.db.WithContext(ctx).Save(info).Error; err != nil {
		return apperror.InternalWrap("update supplier info failed", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, info *siModels.SupplierInfo) error {
	if err := r.db.WithContext(ctx).Delete(info).Error; err != nil {
		return apperror.InternalWrap("delete supplier info failed", err)
	}
	return nil
}

// FindSupplierItem mencari supplier_item berdasarkan kombinasi (uniq, supplier, type).
// Satu UNIQ boleh dimiliki beberapa supplier/type (lihat migrasi 0103), sehingga
// pencarian hanya berdasarkan uniq_code akan non-deterministik.
func (r *repository) FindSupplierItem(ctx context.Context, uniq, supplierName, itemType string) (*supplierItemModels.SupplierItem, error) {
	var item supplierItemModels.SupplierItem
	err := r.db.WithContext(ctx).
		Where("uniq_code = ? AND supplier_name = ? AND type = ? AND status = 'active'",
			strings.TrimSpace(uniq), strings.TrimSpace(supplierName), itemType).
		Order("id ASC").
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("kombinasi uniq, supplier, dan type tidak ditemukan di supplier item")
		}
		return nil, apperror.InternalWrap("find supplier item failed", err)
	}
	return &item, nil
}

// ExistsByUniqSupplierType memeriksa duplikat per kombinasi (uniq, supplier_name, type).
func (r *repository) ExistsByUniqSupplierType(ctx context.Context, uniq, supplierName, typeLabel string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&siModels.SupplierInfo{}).
		Where("uniq = ? AND supplier_name = ? AND type = ?",
			strings.TrimSpace(uniq), strings.TrimSpace(supplierName), typeLabel).
		Count(&count).Error
	if err != nil {
		return false, apperror.InternalWrap("check supplier info duplicate failed", err)
	}
	return count > 0, nil
}

// ListActiveSupplierItems mengembalikan seluruh supplier_item aktif yang punya uniq_code.
func (r *repository) ListActiveSupplierItems(ctx context.Context) ([]supplierItemModels.SupplierItem, error) {
	var items []supplierItemModels.SupplierItem
	err := r.db.WithContext(ctx).
		Where("status = 'active' AND uniq_code IS NOT NULL AND TRIM(uniq_code) <> ''").
		Order("uniq_code ASC, supplier_name ASC, type ASC").
		Find(&items).Error
	if err != nil {
		return nil, apperror.InternalWrap("list active supplier items failed", err)
	}
	return items, nil
}

// ListAll mengembalikan seluruh supplier_info (tanpa pagination).
func (r *repository) ListAll(ctx context.Context) ([]siModels.SupplierInfo, error) {
	var items []siModels.SupplierInfo
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, apperror.InternalWrap("list all supplier info failed", err)
	}
	return items, nil
}
