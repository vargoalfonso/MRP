// Package repository provides database access for the Finished Goods module.
package repository

import (
	"context"
	"math"
	"strings"

	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Filter
// ---------------------------------------------------------------------------

// FinishedGoodsFilter holds query params for the list endpoint.
type FinishedGoodsFilter struct {
	Search            string // matches uniq_code OR part_name
	Model             string
	Status            string
	WarehouseLocation string
	Page              int
	Limit             int
	Offset            int
}

// StatusMonitoringFilter holds query params for the status-monitoring endpoint.
type StatusMonitoringFilter struct {
	AlertType string // low_on_stock | overstock (empty = all non-normal)
	Page      int
	Limit     int
	Offset    int
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IRepository interface {
	GetSummary(ctx context.Context) (*fgModels.FGSummary, error)
	GetStatusMonitoring(ctx context.Context, f StatusMonitoringFilter) (*fgModels.FGStatusMonitoringResponse, error)

	ListFinishedGoods(ctx context.Context, f FinishedGoodsFilter) ([]fgModels.FinishedGoods, int64, error)
	GetFinishedGoodsByID(ctx context.Context, id int64) (*fgModels.FinishedGoods, error)
	CreateFinishedGoods(ctx context.Context, fg *fgModels.FinishedGoods) error
	UpdateFinishedGoods(ctx context.Context, id int64, updates map[string]interface{}) error

	AppendMovementLog(ctx context.Context, log *fgModels.FGMovementLog) error
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository { return &repository{db: db} }

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

func (r *repository) GetSummary(ctx context.Context) (*fgModels.FGSummary, error) {
	type row struct {
		TotalFGItems  int64   `gorm:"column:total_fg_items"`
		LowStockItems int64   `gorm:"column:low_stock_items"`
		TotalStock    float64 `gorm:"column:total_stock"`
		ActiveAlerts  int64   `gorm:"column:active_alerts"`
	}
	var res row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*)                                                    AS total_fg_items,
			COUNT(*) FILTER (WHERE status = 'low_on_stock')            AS low_stock_items,
			COALESCE(SUM(stock_qty), 0)                                AS total_stock,
			COUNT(*) FILTER (WHERE status <> 'normal')                 AS active_alerts
		FROM finished_goods
		WHERE deleted_at IS NULL
	`).Scan(&res).Error
	if err != nil {
		return nil, apperror.Internal("fg get summary: " + err.Error())
	}
	return &fgModels.FGSummary{
		TotalFGItems:  res.TotalFGItems,
		LowStockItems: res.LowStockItems,
		TotalStock:    res.TotalStock,
		ActiveAlerts:  res.ActiveAlerts,
	}, nil
}

// ---------------------------------------------------------------------------
// Status Monitoring
// ---------------------------------------------------------------------------

func (r *repository) GetStatusMonitoring(ctx context.Context, f StatusMonitoringFilter) (*fgModels.FGStatusMonitoringResponse, error) {
	// Summary counts (always full, ignoring pagination)
	type summaryRow struct {
		LowStockCount  int64 `gorm:"column:low_stock_count"`
		OverstockCount int64 `gorm:"column:overstock_count"`
		NormalCount    int64 `gorm:"column:normal_count"`
	}
	var sr summaryRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status = 'low_on_stock') AS low_stock_count,
			COUNT(*) FILTER (WHERE status = 'overstock')    AS overstock_count,
			COUNT(*) FILTER (WHERE status = 'normal')       AS normal_count
		FROM finished_goods
		WHERE deleted_at IS NULL
	`).Scan(&sr).Error; err != nil {
		return nil, apperror.Internal("fg status monitoring summary: " + err.Error())
	}

	// Items query
	q := r.db.WithContext(ctx).Model(&fgModels.FinishedGoods{}).
		Where("deleted_at IS NULL").
		Where("status <> 'normal'")
	if f.AlertType != "" {
		q = q.Where("status = ?", f.AlertType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, apperror.Internal("fg status monitoring count: " + err.Error())
	}

	var rows []fgModels.FinishedGoods
	if err := q.Order("updated_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("fg status monitoring list: " + err.Error())
	}

	items := make([]fgModels.FGAlertItem, 0, len(rows))
	for _, row := range rows {
		item := fgModels.FGAlertItem{
			ID:           row.ID,
			UniqCode:     row.UniqCode,
			PartName:     row.PartName,
			Model:        row.Model,
			AlertType:    row.Status,
			CurrentStock: row.StockQty,
			UpdatedAt:    row.UpdatedAt,
		}
		switch row.Status {
		case fgModels.FGStatusLowStock:
			item.Priority = "High"
			item.Recommendation = "Schedule production immediately"
			if row.MinThreshold != nil {
				item.Threshold = *row.MinThreshold
			}
		case fgModels.FGStatusOverstock:
			item.Priority = "Medium"
			item.Recommendation = "Consider delivery acceleration"
			if row.MaxThreshold != nil {
				item.Threshold = *row.MaxThreshold
			}
		}
		items = append(items, item)
	}

	totalPages := 1
	if f.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(f.Limit)))
	}

	return &fgModels.FGStatusMonitoringResponse{
		Summary: fgModels.FGStatusMonitoringSummary{
			LowStockCount:  sr.LowStockCount,
			OverstockCount: sr.OverstockCount,
			NormalCount:    sr.NormalCount,
			TotalAlerts:    sr.LowStockCount + sr.OverstockCount,
		},
		Items: items,
		Pagination: fgModels.FGPagination{
			Total:      total,
			Page:       f.Page,
			Limit:      f.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func (r *repository) ListFinishedGoods(ctx context.Context, f FinishedGoodsFilter) ([]fgModels.FinishedGoods, int64, error) {
	q := r.db.WithContext(ctx).Model(&fgModels.FinishedGoods{}).Where("deleted_at IS NULL")

	if f.Search != "" {
		like := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where("LOWER(uniq_code) LIKE ? OR LOWER(part_name) LIKE ?", like, like)
	}
	if f.Model != "" {
		q = q.Where("LOWER(model) = ?", strings.ToLower(f.Model))
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.WarehouseLocation != "" {
		q = q.Where("LOWER(warehouse_location) = ?", strings.ToLower(f.WarehouseLocation))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("fg list count: " + err.Error())
	}

	var rows []fgModels.FinishedGoods
	if err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("fg list: " + err.Error())
	}
	return rows, total, nil
}

// ---------------------------------------------------------------------------
// Get by ID
// ---------------------------------------------------------------------------

func (r *repository) GetFinishedGoodsByID(ctx context.Context, id int64) (*fgModels.FinishedGoods, error) {
	var fg fgModels.FinishedGoods
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&fg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("finished goods not found")
		}
		return nil, apperror.Internal("fg get by id: " + err.Error())
	}
	return &fg, nil
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func (r *repository) CreateFinishedGoods(ctx context.Context, fg *fgModels.FinishedGoods) error {
	fg.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(fg).Error; err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return apperror.Conflict("uniq_code '" + fg.UniqCode + "' already exists in finished goods")
		}
		return apperror.Internal("fg create: " + err.Error())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (r *repository) UpdateFinishedGoods(ctx context.Context, id int64, updates map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&fgModels.FinishedGoods{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates).Error; err != nil {
		return apperror.Internal("fg update: " + err.Error())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Movement log
// ---------------------------------------------------------------------------

func (r *repository) AppendMovementLog(ctx context.Context, log *fgModels.FGMovementLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return apperror.Internal("fg movement log: " + err.Error())
	}
	return nil
}
