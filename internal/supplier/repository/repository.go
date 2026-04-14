package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, supplier *models.Supplier) error
	FindByUUID(ctx context.Context, uuid string) (*models.Supplier, error)
	List(ctx context.Context, filters models.SupplierListFilters) ([]models.Supplier, int64, error)
	Update(ctx context.Context, supplier *models.Supplier) error
	Delete(ctx context.Context, supplier *models.Supplier) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, supplier *models.Supplier) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(supplier).Error; err != nil {
			return apperror.InternalWrap("create supplier failed", err)
		}

		supplierCode := fmt.Sprintf("SUP-%04d", supplier.ID)
		if err := tx.Model(supplier).Update("supplier_code", supplierCode).Error; err != nil {
			return apperror.InternalWrap("generate supplier code failed", err)
		}

		supplier.SupplierCode = supplierCode
		return nil
	})
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.Supplier, error) {
	var supplier models.Supplier
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&supplier).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("supplier not found")
		}
		return nil, apperror.InternalWrap("find supplier failed", err)
	}

	return &supplier, nil
}

func (r *repository) List(ctx context.Context, filters models.SupplierListFilters) ([]models.Supplier, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Supplier{})

	if filters.Search != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where(
			"supplier_code ILIKE ? OR supplier_name ILIKE ? OR contact_person ILIKE ? OR email_address ILIKE ?",
			search,
			search,
			search,
			search,
		)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.MaterialCategory != nil {
		query = query.Where("material_category = ?", *filters.MaterialCategory)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count suppliers failed", err)
	}

	var suppliers []models.Supplier
	err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&suppliers).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list suppliers failed", err)
	}

	return suppliers, total, nil
}

func (r *repository) Update(ctx context.Context, supplier *models.Supplier) error {
	if err := r.db.WithContext(ctx).Save(supplier).Error; err != nil {
		return apperror.InternalWrap("update supplier failed", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, supplier *models.Supplier) error {
	if err := r.db.WithContext(ctx).Delete(supplier).Error; err != nil {
		return apperror.InternalWrap("delete supplier failed", err)
	}

	return nil
}
