package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/customer_order/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, doc *models.CustomerOrderDocument) error
	FindByUUID(ctx context.Context, uuid string) (*models.CustomerOrderDocument, error)
	List(ctx context.Context, f models.ListFilters) ([]models.CustomerOrderDocument, int64, error)
	UpdateStatus(ctx context.Context, doc *models.CustomerOrderDocument) error
	SoftDelete(ctx context.Context, doc *models.CustomerOrderDocument) error
	GetActivePeriode(ctx context.Context) (string, error)
	GetCustomerNameByID(ctx context.Context, customerID int64) (string, error)
	GetItemSnapshot(ctx context.Context, uniqCode string) (*models.ItemSnapshot, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, doc *models.CustomerOrderDocument) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE public.customer_order_documents IN EXCLUSIVE MODE").Error; err != nil {
			return apperror.InternalWrap("lock table failed", err)
		}

		num, err := nextDocumentNumber(tx, doc.DocumentType)
		if err != nil {
			return err
		}
		doc.DocumentNumber = num

		if err := tx.Create(doc).Error; err != nil {
			return apperror.InternalWrap("create customer order failed", err)
		}
		return nil
	})
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.CustomerOrderDocument, error) {
	var doc models.CustomerOrderDocument
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("line_no ASC")
		}).
		Where("uuid = ? AND deleted_at IS NULL", uuid).
		First(&doc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("customer order not found")
		}
		return nil, apperror.InternalWrap("find customer order failed", err)
	}
	return &doc, nil
}

func (r *repository) List(ctx context.Context, f models.ListFilters) ([]models.CustomerOrderDocument, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.CustomerOrderDocument{}).Where("deleted_at IS NULL")

	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		q = q.Where("document_number ILIKE ? OR customer_name_snapshot ILIKE ?", like, like)
	}
	if f.DocumentType != "" {
		q = q.Where("document_type = ?", f.DocumentType)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.CustomerID > 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Period != "" {
		q = q.Where("period_schedule = ?", f.Period)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count customer orders failed", err)
	}

	var docs []models.CustomerOrderDocument
	err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&docs).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list customer orders failed", err)
	}
	return docs, total, nil
}

func (r *repository) UpdateStatus(ctx context.Context, doc *models.CustomerOrderDocument) error {
	err := r.db.WithContext(ctx).
		Model(doc).
		Update("status", doc.Status).Error
	if err != nil {
		return apperror.InternalWrap("update status failed", err)
	}
	return nil
}

func (r *repository) SoftDelete(ctx context.Context, doc *models.CustomerOrderDocument) error {
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(doc).
		Update("deleted_at", now).Error
	if err != nil {
		return apperror.InternalWrap("delete customer order failed", err)
	}
	return nil
}

func (r *repository) GetActivePeriode(ctx context.Context) (string, error) {
	var result struct {
		Period string `gorm:"column:period"`
	}
	err := r.db.WithContext(ctx).
		Table("global_parameters").
		Select("period").
		Where("parameter_group = 'working_days' AND status = 'active'").
		Order("id DESC").
		Limit(1).
		Scan(&result).Error
	return result.Period, err
}

func (r *repository) GetCustomerNameByID(ctx context.Context, customerID int64) (string, error) {
	var result struct {
		CustomerName string `gorm:"column:customer_name"`
	}
	err := r.db.WithContext(ctx).
		Table("customers").
		Select("customer_name").
		Where("id = ? AND deleted_at IS NULL", customerID).
		Scan(&result).Error
	if err != nil {
		return "", apperror.InternalWrap("fetch customer failed", err)
	}
	if result.CustomerName == "" {
		return "", apperror.NotFound("customer not found")
	}
	return result.CustomerName, nil
}

func (r *repository) GetItemSnapshot(ctx context.Context, uniqCode string) (*models.ItemSnapshot, error) {
	var snap models.ItemSnapshot
	err := r.db.WithContext(ctx).
		Table("items").
		Select("part_name, part_number, model").
		Where("uniq_code = ?", uniqCode).
		Scan(&snap).Error
	if err != nil {
		return nil, apperror.InternalWrap("fetch item snapshot failed", err)
	}
	if snap.PartName == "" {
		return nil, apperror.NotFound(fmt.Sprintf("item uniq_code '%s' not found", uniqCode))
	}
	return &snap, nil
}

func nextDocumentNumber(tx *gorm.DB, docType string) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("%s-%d-", docType, year)

	var latest struct {
		DocumentNumber string `gorm:"column:document_number"`
	}
	err := tx.Unscoped().
		Table("customer_order_documents").
		Select("document_number").
		Where("document_type = ? AND document_number LIKE ?", docType, prefix+"%").
		Order("document_number DESC").
		Limit(1).
		Scan(&latest).Error
	if err != nil {
		return "", apperror.InternalWrap("get latest document number failed", err)
	}

	seq := 1
	if latest.DocumentNumber != "" {
		parts := strings.Split(latest.DocumentNumber, "-")
		if len(parts) >= 3 {
			last, convErr := strconv.Atoi(parts[len(parts)-1])
			if convErr == nil {
				seq = last + 1
			}
		}
	}

	return fmt.Sprintf("%s%04d", prefix, seq), nil
}
