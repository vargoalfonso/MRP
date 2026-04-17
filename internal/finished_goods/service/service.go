// Package service provides business logic for the Finished Goods module.
package service

import (
	"context"
	"math"
	"strings"
	"time"

	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	"github.com/ganasa18/go-template/internal/finished_goods/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IService interface {
	// Dashboard summary (4 cards)
	GetSummary(ctx context.Context) (*fgModels.FGSummary, error)

	// Status Monitoring tab
	GetStatusMonitoring(ctx context.Context, f repository.StatusMonitoringFilter) (*fgModels.FGStatusMonitoringResponse, error)

	// FG Inventory tab
	ListFinishedGoods(ctx context.Context, f repository.FinishedGoodsFilter) (*fgModels.FinishedGoodsListResponse, error)
	ListCreateUniqOptions(ctx context.Context, q string, limit int) (*fgModels.FGCreateUniqOptionsResponse, error)
	GetFinishedGoodsByID(ctx context.Context, id int64) (*fgModels.FinishedGoodsItem, error)
	CreateFinishedGoods(ctx context.Context, req fgModels.CreateFinishedGoodsRequest, createdBy string) (*fgModels.FinishedGoodsItem, error)
	UpdateFinishedGoods(ctx context.Context, id int64, req fgModels.UpdateFinishedGoodsRequest, updatedBy string) (*fgModels.FinishedGoodsItem, error)
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type service struct {
	repo repository.IRepository
	db   *gorm.DB
}

func New(repo repository.IRepository, db *gorm.DB) IService {
	return &service{repo: repo, db: db}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// computeStatus derives the FG status from stock vs thresholds.
func computeStatus(stockQty float64, minThreshold, maxThreshold *float64) string {
	if minThreshold != nil && stockQty < *minThreshold {
		return fgModels.FGStatusLowStock
	}
	if maxThreshold != nil && stockQty > *maxThreshold {
		return fgModels.FGStatusOverstock
	}
	return fgModels.FGStatusNormal
}

// computeKanbanCount returns floor(stock / kanbanStd), 0 if kanbanStd is nil or 0.
func computeKanbanCount(stockQty float64, kanbanStd *int) *int {
	if kanbanStd == nil || *kanbanStd == 0 {
		return nil
	}
	v := int(math.Floor(stockQty / float64(*kanbanStd)))
	return &v
}

// computeStockToComplete returns max(0, safety - stock).
func computeStockToComplete(stockQty float64, safetyStockQty *float64) *float64 {
	if safetyStockQty == nil {
		return nil
	}
	v := math.Max(0, *safetyStockQty-stockQty)
	return &v
}

// kanbanProgressPct returns floor(stock / safety * 100), capped at 100.
func kanbanProgressPct(stockQty float64, safetyStockQty *float64) int {
	if safetyStockQty == nil || *safetyStockQty == 0 {
		return 0
	}
	pct := int(math.Floor(stockQty / *safetyStockQty * 100))
	if pct > 100 {
		return 100
	}
	return pct
}

// toItem converts a DB model to the API response shape.
func toItem(fg *fgModels.FinishedGoods) *fgModels.FinishedGoodsItem {
	item := &fgModels.FinishedGoodsItem{
		ID:                    fg.ID,
		UUID:                  fg.UUID.String(),
		UniqCode:              fg.UniqCode,
		PartNumber:            fg.PartNumber,
		PartName:              fg.PartName,
		Model:                 fg.Model,
		WONumber:              fg.WONumber,
		WarehouseLocation:     fg.WarehouseLocation,
		StockQty:              fg.StockQty,
		UOM:                   fg.UOM,
		KanbanCount:           fg.KanbanCount,
		KanbanStandardQty:     fg.KanbanStandardQty,
		SafetyStockQty:        fg.SafetyStockQty,
		MinThreshold:          fg.MinThreshold,
		MaxThreshold:          fg.MaxThreshold,
		StockToCompleteKanban: fg.StockToCompleteKanban,
		KanbanProgress:        kanbanProgressPct(fg.StockQty, fg.SafetyStockQty),
		Status:                fg.Status,
		CreatedBy:             fg.CreatedBy,
		CreatedAt:             fg.CreatedAt,
		UpdatedAt:             fg.UpdatedAt,
	}
	return item
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

func (s *service) GetSummary(ctx context.Context) (*fgModels.FGSummary, error) {
	return s.repo.GetSummary(ctx)
}

// ---------------------------------------------------------------------------
// Status Monitoring
// ---------------------------------------------------------------------------

func (s *service) GetStatusMonitoring(ctx context.Context, f repository.StatusMonitoringFilter) (*fgModels.FGStatusMonitoringResponse, error) {
	return s.repo.GetStatusMonitoring(ctx, f)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func (s *service) ListFinishedGoods(ctx context.Context, f repository.FinishedGoodsFilter) (*fgModels.FinishedGoodsListResponse, error) {
	rows, total, err := s.repo.ListFinishedGoods(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]fgModels.FinishedGoodsItem, 0, len(rows))
	for i := range rows {
		items = append(items, *toItem(&rows[i]))
	}

	totalPages := 1
	if f.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(f.Limit)))
	}

	return &fgModels.FinishedGoodsListResponse{
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
// Create Form Options
// ---------------------------------------------------------------------------

func (s *service) ListCreateUniqOptions(ctx context.Context, q string, limit int) (*fgModels.FGCreateUniqOptionsResponse, error) {
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	type row struct {
		UniqCode     string   `gorm:"column:uniq_code"`
		PartNumber   *string  `gorm:"column:part_number"`
		PartName     *string  `gorm:"column:part_name"`
		Model        *string  `gorm:"column:model"`
		LastWONumber *string  `gorm:"column:last_wo_number"`
		KanbanQty    *int     `gorm:"column:kanban_qty"`
		MinThreshold *float64 `gorm:"column:min_threshold"`
		MaxThreshold *float64 `gorm:"column:max_threshold"`
	}

	rows := make([]row, 0, limit)
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			bi.uniq_code,
			MAX(bi.part_number) AS part_number,
			MAX(bi.part_name) AS part_name,
			NULL::text AS model,
			wo.wo_number AS last_wo_number,
			MAX(kp.kanban_qty) AS kanban_qty,
			MAX(kp.min_stock) AS min_threshold,
			MAX(kp.max_stock) AS max_threshold
		FROM bom_items bi
		LEFT JOIN LATERAL (
			SELECT w.wo_number
			FROM work_orders w
			WHERE w.uniq_code = bi.uniq_code AND w.deleted_at IS NULL
			ORDER BY w.created_at DESC
			LIMIT 1
		) wo ON TRUE
		LEFT JOIN kanban_parameters kp
			ON kp.item_uniq_code = bi.uniq_code
			AND kp.deleted_at IS NULL
			AND kp.status ILIKE 'active'
		WHERE bi.deleted_at IS NULL
			AND ($1 = ''
				OR bi.uniq_code ILIKE '%' || $1 || '%'
				OR COALESCE(bi.part_name, '') ILIKE '%' || $1 || '%'
				OR COALESCE(bi.part_number, '') ILIKE '%' || $1 || '%')
		GROUP BY bi.uniq_code, wo.wo_number
		ORDER BY bi.uniq_code
		LIMIT $2
	`, q, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to list finished-goods create uniq options", err)
	}

	out := make([]fgModels.FGCreateUniqOptionItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, fgModels.FGCreateUniqOptionItem{
			UniqCode:     r.UniqCode,
			PartNumber:   r.PartNumber,
			PartName:     r.PartName,
			Model:        r.Model,
			LastWONumber: r.LastWONumber,
			KanbanQty:    r.KanbanQty,
			MinThreshold: r.MinThreshold,
			MaxThreshold: r.MaxThreshold,
		})
	}

	return &fgModels.FGCreateUniqOptionsResponse{Items: out}, nil
}

// ---------------------------------------------------------------------------
// Get by ID
// ---------------------------------------------------------------------------

func (s *service) GetFinishedGoodsByID(ctx context.Context, id int64) (*fgModels.FinishedGoodsItem, error) {
	fg, err := s.repo.GetFinishedGoodsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toItem(fg), nil
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func (s *service) CreateFinishedGoods(ctx context.Context, req fgModels.CreateFinishedGoodsRequest, createdBy string) (*fgModels.FinishedGoodsItem, error) {
	// Look up BOM item for part details and kanban parameters
	type bomRow struct {
		PartNumber *string `gorm:"column:part_number"`
		PartName   string  `gorm:"column:part_name"`
		Model      *string `gorm:"column:model"`
	}
	var bom bomRow
	_ = s.db.WithContext(ctx).Raw(`
		SELECT part_number, part_name, NULL AS model
		FROM bom_items
		WHERE uniq_code = ? AND deleted_at IS NULL
		LIMIT 1
	`, req.UniqCode).Scan(&bom)

	// Look up last WO for this uniq
	var woNumber *string
	var woResult struct {
		WONumber string `gorm:"column:wo_number"`
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT wo_number FROM work_orders
		WHERE uniq_code = ? AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, req.UniqCode).Scan(&woResult).Error; err == nil && woResult.WONumber != "" {
		woNumber = &woResult.WONumber
	}

	// Look up kanban parameters
	type kanbanRow struct {
		KanbanQty int     `gorm:"column:kanban_qty"`
		MinStock  float64 `gorm:"column:min_stock"`
		MaxStock  float64 `gorm:"column:max_stock"`
	}
	var kanban kanbanRow
	_ = s.db.WithContext(ctx).Raw(`
		SELECT kanban_qty, min_stock, max_stock
		FROM kanban_parameters
		WHERE item_uniq_code = ?
		LIMIT 1
	`, req.UniqCode).Scan(&kanban)

	// Compute derived fields
	var minThreshold, maxThreshold, safetyStockQty *float64
	var kanbanStandardQty *int
	if kanban.KanbanQty > 0 {
		kanbanStandardQty = &kanban.KanbanQty
	}
	if kanban.MinStock > 0 {
		minThreshold = &kanban.MinStock
	}
	if kanban.MaxStock > 0 {
		maxThreshold = &kanban.MaxStock
		safetyStockQty = &kanban.MaxStock // Target = max_stock (shown as "Target" in UI)
	}

	stockToComplete := computeStockToComplete(0, safetyStockQty)
	kanbanCount := computeKanbanCount(0, kanbanStandardQty)
	status := computeStatus(0, minThreshold, maxThreshold)

	now := time.Now()
	fg := &fgModels.FinishedGoods{
		UniqCode:              req.UniqCode,
		PartNumber:            bom.PartNumber,
		PartName:              &bom.PartName,
		Model:                 bom.Model,
		WONumber:              woNumber,
		WarehouseLocation:     &req.WarehouseLocation,
		StockQty:              0,
		KanbanCount:           kanbanCount,
		KanbanStandardQty:     kanbanStandardQty,
		MinThreshold:          minThreshold,
		MaxThreshold:          maxThreshold,
		SafetyStockQty:        safetyStockQty,
		StockToCompleteKanban: stockToComplete,
		Status:                status,
		CreatedBy:             &createdBy,
		UpdatedBy:             &createdBy,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.CreateFinishedGoods(ctx, fg); err != nil {
		return nil, err
	}

	// Append movement log (initial entry, qty_change = 0)
	src := "manual"
	_ = s.repo.AppendMovementLog(ctx, &fgModels.FGMovementLog{
		FgID:         fg.ID,
		UniqCode:     fg.UniqCode,
		MovementType: "manual_add",
		QtyChange:    0,
		QtyBefore:    0,
		QtyAfter:     0,
		SourceFlag:   &src,
		WONumber:     fg.WONumber,
		LoggedBy:     &createdBy,
		LoggedAt:     now,
	})

	return toItem(fg), nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (s *service) UpdateFinishedGoods(ctx context.Context, id int64, req fgModels.UpdateFinishedGoodsRequest, updatedBy string) (*fgModels.FinishedGoodsItem, error) {
	existing, err := s.repo.GetFinishedGoodsByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.StockQty != nil && *req.StockQty < 0 {
		return nil, apperror.BadRequest("stock_qty cannot be negative")
	}

	updates := map[string]interface{}{
		"updated_by": updatedBy,
		"updated_at": time.Now(),
	}

	newStock := existing.StockQty
	if req.WONumber != nil {
		updates["wo_number"] = req.WONumber
	}
	if req.WarehouseLocation != nil {
		updates["warehouse_location"] = req.WarehouseLocation
	}
	if req.StockQty != nil {
		newStock = *req.StockQty
		updates["stock_qty"] = newStock
		// Recompute derived fields
		updates["kanban_count"] = computeKanbanCount(newStock, existing.KanbanStandardQty)
		updates["stock_to_complete_kanban"] = computeStockToComplete(newStock, existing.SafetyStockQty)
		updates["status"] = computeStatus(newStock, existing.MinThreshold, existing.MaxThreshold)
	}

	if err := s.repo.UpdateFinishedGoods(ctx, id, updates); err != nil {
		return nil, err
	}

	// Append movement log if stock changed
	if req.StockQty != nil {
		src := "manual"
		_ = s.repo.AppendMovementLog(ctx, &fgModels.FGMovementLog{
			FgID:         id,
			UniqCode:     existing.UniqCode,
			MovementType: "manual_deduct",
			QtyChange:    newStock - existing.StockQty,
			QtyBefore:    existing.StockQty,
			QtyAfter:     newStock,
			SourceFlag:   &src,
			Notes:        req.Remarks,
			LoggedBy:     &updatedBy,
			LoggedAt:     time.Now(),
		})
	}

	return s.GetFinishedGoodsByID(ctx, id)
}
