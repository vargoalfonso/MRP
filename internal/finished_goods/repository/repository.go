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
	ListMovementLogs(ctx context.Context, uniqCode string, limit, offset int) ([]fgModels.FGMovementLog, int64, error)

	// FindWOQRByUniqCode returns the QR image (base64 data URL) that the work
	// order already generated and stored per item (work_order_items.qr_image_base64)
	// for the given finished-good uniq code. One uniq resolves to exactly one QR.
	FindWOQRByUniqCode(ctx context.Context, uniqCode string) (qr string, err error)

	// ListPackingList returns every packing/kanban number that belongs to one
	// finished-good uniq code. Rows come from the work order barcode
	// (work_order_items.kanban_number) and are enriched with the delivery note
	// that consumes the same packing number.
	ListPackingList(ctx context.Context, uniqCode string) ([]FGPackingRow, error)
}

// FGPackingRow is the raw scan target for the packing-list query.
type FGPackingRow struct {
	DNNumber      *string `gorm:"column:dn_number"`
	PackingNumber string  `gorm:"column:packing_number"`
	Quantity      float64 `gorm:"column:quantity"`
	QtyCurrent    float64 `gorm:"column:qty_current"`
	QtyMax        float64 `gorm:"column:qty_max"`
	Status        *string `gorm:"column:status"`
	WONumber      *string `gorm:"column:wo_number"`
	Source        string  `gorm:"column:source"`
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
			return nil, apperror.NotFound("finished goods tidak ditemukan")
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

// ListMovementLogs returns the In-Out Activity Log rows for one uniq_code,
// newest first, with total count for pagination.
func (r *repository) ListMovementLogs(ctx context.Context, uniqCode string, limit, offset int) ([]fgModels.FGMovementLog, int64, error) {
	var (
		rows  []fgModels.FGMovementLog
		total int64
	)
	base := r.db.WithContext(ctx).Model(&fgModels.FGMovementLog{}).
		Where("uniq_code = ?", uniqCode)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("fg list movement logs count: " + err.Error())
	}
	if err := base.
		Order("logged_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("fg list movement logs: " + err.Error())
	}
	return rows, total, nil
}

// FindWOQRByUniqCode returns the QR image (base64 data URL) that the work order
// already generated and stored for the item (work_order_items.qr_image_base64).
// The QR is specific per uniq. Among the item's rows we prefer one that has a
// stored QR, then the most recent, so one uniq always maps to exactly one QR.
func (r *repository) FindWOQRByUniqCode(ctx context.Context, uniqCode string) (string, error) {
	var qr string
	if err := r.db.WithContext(ctx).
		Table("work_order_items").
		Select("qr_image_base64").
		Where("item_uniq_code = ?", uniqCode).
		Order("CASE WHEN COALESCE(qr_image_base64, '') <> '' THEN 0 ELSE 1 END, id DESC").
		Limit(1).
		Scan(&qr).Error; err != nil {
		return "", apperror.Internal("fg find wo qr: " + err.Error())
	}

	return qr, nil
}

// ---------------------------------------------------------------------------
// Packing List
// ---------------------------------------------------------------------------

// ListPackingList resolves the packing/kanban list for one finished-good uniq.
//
// Source of truth:
//   - work_order_items: 1 row = 1 kanban/packing generated + barcoded by the WO
//   - delivery_note_items: matched on packing_number to expose the DN number
//     and the latest opname qty for that packing
//   - kanban_parameters.kanban_qty: qty maksimal per packing
//
// DN items whose packing number has no work order row are appended so packings
// created outside the WO flow still show up.
func (r *repository) ListPackingList(ctx context.Context, uniqCode string) ([]FGPackingRow, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	if uniqCode == "" {
		return []FGPackingRow{}, nil
	}

	var rows []FGPackingRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT * FROM (
			SELECT
				dn.dn_number      AS dn_number,
				woi.kanban_number AS packing_number,
				COALESCE(woi.quantity, 0) AS quantity,
				COALESCE(
					dni.qty_opname,
					NULLIF(woi.total_good_qty, 0),
					dni.quantity,
					0
				) AS qty_current,
				COALESCE(
					NULLIF(kp.kanban_qty, 0),
					NULLIF(dni.pcs_per_kanban, 0),
					NULLIF(woi.quantity, 0),
					0
				) AS qty_max,
				woi.status   AS status,
				w.wo_number  AS wo_number,
				'work_order' AS source,
				woi.created_at AS sort_at
			FROM work_order_items woi
			JOIN work_orders w ON w.id = woi.wo_id
			LEFT JOIN delivery_note_items dni ON dni.packing_number = woi.kanban_number
			LEFT JOIN delivery_notes dn ON dn.id = dni.dn_id
			LEFT JOIN LATERAL (
				SELECT kanban_qty
				FROM kanban_parameters
				WHERE item_uniq_code = woi.item_uniq_code
				ORDER BY id DESC
				LIMIT 1
			) kp ON TRUE
			WHERE woi.item_uniq_code = ?

			UNION ALL

			SELECT
				dn.dn_number       AS dn_number,
				dni.packing_number AS packing_number,
				COALESCE(dni.quantity, 0) AS quantity,
				COALESCE(dni.qty_opname, dni.quantity, 0) AS qty_current,
				COALESCE(
					NULLIF(dni.pcs_per_kanban, 0),
					NULLIF(kp.kanban_qty, 0),
					NULLIF(dni.quantity, 0),
					0
				) AS qty_max,
				NULLIF(dni.check, '') AS status,
				NULL           AS wo_number,
				'delivery_note' AS source,
				dni.created_at AS sort_at
			FROM delivery_note_items dni
			JOIN delivery_notes dn ON dn.id = dni.dn_id
			LEFT JOIN LATERAL (
				SELECT kanban_qty
				FROM kanban_parameters
				WHERE item_uniq_code = dni.item_uniq_code
				ORDER BY id DESC
				LIMIT 1
			) kp ON TRUE
			WHERE dni.item_uniq_code = ?
			  AND COALESCE(dni.packing_number, '') <> ''
			  AND NOT EXISTS (
				SELECT 1 FROM work_order_items woi2
				WHERE woi2.kanban_number = dni.packing_number
			  )
		) packing
		ORDER BY sort_at DESC, packing_number ASC
	`, uniqCode, uniqCode).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("fg list packing list: " + err.Error())
	}
	if rows == nil {
		rows = []FGPackingRow{}
	}
	return rows, nil
}
