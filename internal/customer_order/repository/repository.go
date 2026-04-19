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
	GetSummary(ctx context.Context, documentType string) (*models.SummaryResponse, error)
	List(ctx context.Context, f models.ListFilters) ([]models.CustomerOrderDocument, int64, error)
	Update(ctx context.Context, doc *models.CustomerOrderDocument) error
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
	err := q.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("line_no ASC")
	}).Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&docs).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list customer orders failed", err)
	}
	return docs, total, nil
}

func (r *repository) GetSummary(ctx context.Context, documentType string) (*models.SummaryResponse, error) {
	if documentType == "ALL" {
		return r.getAllSummary(ctx)
	}

	base := r.db.WithContext(ctx).
		Table("customer_order_documents").
		Where("deleted_at IS NULL AND document_type = ?", documentType)

	if documentType == "DN" {
		base = base.Where("status = ?", "active")
	} else {
		base = base.Where("status <> ?", "cancelled")
	}

	resp := &models.SummaryResponse{DocumentType: documentType}
	if err := base.Count(&resp.TotalDocuments).Error; err != nil {
		return nil, apperror.InternalWrap("count customer order summary failed", err)
	}

	var quantitySummary struct {
		TotalQuantity float64 `gorm:"column:total_quantity"`
	}
	err := r.db.WithContext(ctx).
		Table("customer_order_document_items coi").
		Select("COALESCE(SUM(coi.quantity), 0) AS total_quantity").
		Joins("JOIN customer_order_documents cod ON cod.id = coi.document_id").
		Where("cod.deleted_at IS NULL AND cod.document_type = ?", documentType)
	if documentType == "DN" {
		err = err.Where("cod.status = ?", "active")
	} else {
		err = err.Where("cod.status <> ?", "cancelled")
	}
	scanErr := err.Scan(&quantitySummary).Error
	if scanErr != nil {
		return nil, apperror.InternalWrap("sum customer order quantity failed", scanErr)
	}
	resp.TotalQuantity = quantitySummary.TotalQuantity

	return resp, nil
}

func (r *repository) getAllSummary(ctx context.Context) (*models.SummaryResponse, error) {
	resp := &models.SummaryResponse{DocumentType: "ALL"}
	resp.DN = &models.SummaryMetric{}
	resp.PO = &models.SummaryMetric{}
	resp.SO = &models.SummaryMetric{}
	resp.Total = &models.SummaryMetric{}

	if err := r.allSummaryDocumentsQuery(ctx).
		Where("document_type = ? AND status = ?", "DN", "active").
		Count(&resp.DN.TotalDocuments).Error; err != nil {
		return nil, apperror.InternalWrap("count active dns failed", err)
	}
	if err := r.allSummaryDocumentsQuery(ctx).
		Where("document_type = ? AND status <> ?", "PO", "cancelled").
		Count(&resp.PO.TotalDocuments).Error; err != nil {
		return nil, apperror.InternalWrap("count customer pos failed", err)
	}
	if err := r.allSummaryDocumentsQuery(ctx).
		Where("document_type = ? AND status <> ?", "SO", "cancelled").
		Count(&resp.SO.TotalDocuments).Error; err != nil {
		return nil, apperror.InternalWrap("count special orders failed", err)
	}
	resp.Total.TotalDocuments = resp.DN.TotalDocuments + resp.PO.TotalDocuments + resp.SO.TotalDocuments

	var totals struct {
		DNTotalQuantity float64 `gorm:"column:dn_total_quantity"`
		POTotalQuantity float64 `gorm:"column:po_total_quantity"`
		SOTotalQuantity float64 `gorm:"column:so_total_quantity"`
		TotalQuantity   float64 `gorm:"column:total_quantity"`
	}
	if err := r.db.WithContext(ctx).
		Table("customer_order_document_items coi").
		Select("COALESCE(SUM(CASE WHEN cod.document_type = 'DN' AND cod.status = 'active' THEN coi.quantity ELSE 0 END), 0) AS dn_total_quantity").
		Select("COALESCE(SUM(CASE WHEN cod.document_type = 'PO' AND cod.status <> 'cancelled' THEN coi.quantity ELSE 0 END), 0) AS po_total_quantity").
		Select("COALESCE(SUM(CASE WHEN cod.document_type = 'SO' AND cod.status <> 'cancelled' THEN coi.quantity ELSE 0 END), 0) AS so_total_quantity").
		Select("COALESCE(SUM(coi.quantity), 0) AS total_quantity").
		Joins("JOIN customer_order_documents cod ON cod.id = coi.document_id").
		Where("cod.deleted_at IS NULL").
		Where("(cod.document_type = 'DN' AND cod.status = 'active') OR (cod.document_type IN ('PO','SO') AND cod.status <> 'cancelled')").
		Scan(&totals).Error; err != nil {
		return nil, apperror.InternalWrap("sum all customer order quantity failed", err)
	}
	resp.DN.TotalQuantity = totals.DNTotalQuantity
	resp.PO.TotalQuantity = totals.POTotalQuantity
	resp.SO.TotalQuantity = totals.SOTotalQuantity
	resp.Total.TotalQuantity = totals.TotalQuantity
	resp.TotalQuantity = totals.TotalQuantity

	return resp, nil
}

func (r *repository) allSummaryDocumentsQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("customer_order_documents").
		Where("deleted_at IS NULL")
}

func (r *repository) Update(ctx context.Context, doc *models.CustomerOrderDocument) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CustomerOrderDocument{}).
			Where("id = ? AND deleted_at IS NULL", doc.ID).
			Updates(map[string]interface{}{
				"document_date":          doc.DocumentDate,
				"customer_id":            doc.CustomerID,
				"customer_name_snapshot": doc.CustomerNameSnapshot,
				"contact_person":         doc.ContactPerson,
				"delivery_address":       doc.DeliveryAddress,
				"notes":                  doc.Notes,
				"updated_at":             time.Now(),
			}).Error; err != nil {
			return apperror.InternalWrap("update customer order failed", err)
		}

		if err := tx.Where("document_id = ?", doc.ID).Delete(&models.CustomerOrderDocumentItem{}).Error; err != nil {
			return apperror.InternalWrap("delete customer order items failed", err)
		}

		for i := range doc.Items {
			doc.Items[i].DocumentID = doc.ID
		}
		if len(doc.Items) > 0 {
			if err := tx.Create(&doc.Items).Error; err != nil {
				return apperror.InternalWrap("create customer order items failed", err)
			}
		}

		return nil
	})
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
