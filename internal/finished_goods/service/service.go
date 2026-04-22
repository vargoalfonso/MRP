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
	GetParameterizedSummary(ctx context.Context, uniqCode string) (*fgModels.FGParameterizedSummary, error)
	ListCreateUniqOptions(ctx context.Context, q string, limit int) (*fgModels.FGCreateUniqOptionsResponse, error)
	GetFinishedGoodsByID(ctx context.Context, id int64) (*fgModels.FinishedGoodsItem, error)
	CreateFinishedGoods(ctx context.Context, req fgModels.CreateFinishedGoodsRequest, createdBy string) (*fgModels.FinishedGoodsItem, error)
	BulkCreateFinishedGoods(ctx context.Context, req fgModels.BulkCreateFinishedGoodsRequest, createdBy string) (*fgModels.FGBulkCreateResponse, error)
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

// computeKanbanNeed rounds deficit up to full kanban/box count.
func computeKanbanNeed(stockQty float64, targetStockQty *float64, kanbanStd *int) *int {
	if targetStockQty == nil || kanbanStd == nil || *kanbanStd <= 0 {
		return nil
	}
	deficit := math.Max(0, *targetStockQty-stockQty)
	v := int(math.Ceil(deficit / float64(*kanbanStd)))
	return &v
}

// computeStockToKanbanPCS converts kanban need back to pcs using full-box rounding.
func computeStockToKanbanPCS(kanbanNeed *int, kanbanStd *int) *float64 {
	if kanbanNeed == nil || kanbanStd == nil || *kanbanStd <= 0 {
		return nil
	}
	v := float64(*kanbanNeed * *kanbanStd)
	return &v
}

func computeStockAfterReplenish(stockQty float64, stockToKanbanPCS *float64) *float64 {
	if stockToKanbanPCS == nil {
		return nil
	}
	v := stockQty + *stockToKanbanPCS
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
	targetStockQty := fg.SafetyStockQty
	currentKanban := computeKanbanCount(fg.StockQty, fg.KanbanStandardQty)
	stockGapToTarget := computeStockToComplete(fg.StockQty, targetStockQty)
	kanbanNeed := computeKanbanNeed(fg.StockQty, targetStockQty, fg.KanbanStandardQty)
	stockToKanbanPCS := computeStockToKanbanPCS(kanbanNeed, fg.KanbanStandardQty)
	stockAfterReplenish := computeStockAfterReplenish(fg.StockQty, stockToKanbanPCS)

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
		TargetStockQty:        targetStockQty,
		UOM:                   fg.UOM,
		KanbanCount:           fg.KanbanCount,
		CurrentKanban:         currentKanban,
		KanbanStandardQty:     fg.KanbanStandardQty,
		SafetyStockQty:        fg.SafetyStockQty,
		MinThreshold:          fg.MinThreshold,
		MaxThreshold:          fg.MaxThreshold,
		StockToCompleteKanban: fg.StockToCompleteKanban,
		StockGapToTarget:      stockGapToTarget,
		KanbanNeed:            kanbanNeed,
		StockToKanbanPCS:      stockToKanbanPCS,
		StockAfterReplenish:   stockAfterReplenish,
		KanbanProgress:        kanbanProgressPct(fg.StockQty, fg.SafetyStockQty),
		Status:                fg.Status,
		CreatedBy:             fg.CreatedBy,
		CreatedAt:             fg.CreatedAt,
		UpdatedAt:             fg.UpdatedAt,
	}
	return item
}

func toListItem(fg *fgModels.FinishedGoods) fgModels.FinishedGoodsListItem {
	return fgModels.FinishedGoodsListItem{
		ID:                fg.ID,
		UUID:              fg.UUID.String(),
		UniqCode:          fg.UniqCode,
		PartNumber:        fg.PartNumber,
		PartName:          fg.PartName,
		Model:             fg.Model,
		WONumber:          fg.WONumber,
		WarehouseLocation: fg.WarehouseLocation,
		UOM:               fg.UOM,
		CreatedAt:         fg.CreatedAt,
		UpdatedAt:         fg.UpdatedAt,
	}
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

	items := make([]fgModels.FinishedGoodsListItem, 0, len(rows))
	for i := range rows {
		items = append(items, toListItem(&rows[i]))
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

func (s *service) GetParameterizedSummary(ctx context.Context, uniqCode string) (*fgModels.FGParameterizedSummary, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	if uniqCode == "" {
		return nil, apperror.BadRequest("uniq_code is required")
	}

	type row struct {
		UniqCode          string   `gorm:"column:uniq_code"`
		PartNumber        *string  `gorm:"column:part_number"`
		PartName          *string  `gorm:"column:part_name"`
		Model             *string  `gorm:"column:model"`
		WONumber          *string  `gorm:"column:wo_number"`
		WarehouseLocation *string  `gorm:"column:warehouse_location"`
		StockQty          float64  `gorm:"column:stock_qty"`
		UOM               *string  `gorm:"column:uom"`
		KanbanQty         *int     `gorm:"column:kanban_qty"`
		MinStock          *float64 `gorm:"column:min_stock"`
		MaxStock          *float64 `gorm:"column:max_stock"`
	}

	var r row
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			fg.uniq_code,
			fg.part_number,
			fg.part_name,
			fg.model,
			fg.wo_number,
			fg.warehouse_location,
			fg.stock_qty,
			fg.uom,
			kp.kanban_qty,
			kp.min_stock,
			kp.max_stock
		FROM finished_goods fg
		LEFT JOIN LATERAL (
			SELECT kanban_qty, min_stock, max_stock
			FROM kanban_parameters
			WHERE item_uniq_code = fg.uniq_code
				AND COALESCE(status, 'active') ILIKE 'active'
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		) kp ON TRUE
		WHERE fg.deleted_at IS NULL
			AND fg.uniq_code = ?
		LIMIT 1
	`, uniqCode).Scan(&r).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to get parameterized finished-goods summary", err)
	}
	if r.UniqCode == "" {
		return nil, apperror.NotFound("finished goods not found")
	}

	targetStockQty := r.MinStock
	currentKanban := computeKanbanCount(r.StockQty, r.KanbanQty)
	stockGapToTarget := computeStockToComplete(r.StockQty, targetStockQty)
	kanbanNeed := computeKanbanNeed(r.StockQty, targetStockQty, r.KanbanQty)
	stockToKanbanPCS := computeStockToKanbanPCS(kanbanNeed, r.KanbanQty)
	stockAfterReplenish := computeStockAfterReplenish(r.StockQty, stockToKanbanPCS)
	status := computeStatus(r.StockQty, r.MinStock, r.MaxStock)

	return &fgModels.FGParameterizedSummary{
		UniqCode:            r.UniqCode,
		PartNumber:          r.PartNumber,
		PartName:            r.PartName,
		Model:               r.Model,
		WONumber:            r.WONumber,
		WarehouseLocation:   r.WarehouseLocation,
		StockQty:            r.StockQty,
		UOM:                 r.UOM,
		KanbanStandardQty:   r.KanbanQty,
		MinThreshold:        r.MinStock,
		MaxThreshold:        r.MaxStock,
		TargetStockQty:      targetStockQty,
		CurrentKanban:       currentKanban,
		StockGapToTarget:    stockGapToTarget,
		KanbanNeed:          kanbanNeed,
		StockToKanbanPCS:    stockToKanbanPCS,
		StockAfterReplenish: stockAfterReplenish,
		Status:              status,
		ParameterSource:     "kanban_parameters",
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
			kp.item_uniq_code   AS uniq_code,
			woi.part_number     AS part_number,
			woi.part_name       AS part_name,
			i.model             AS model,
			w.wo_number         AS last_wo_number,
			kp.kanban_qty       AS kanban_qty,
			kp.min_stock        AS min_threshold,
			kp.max_stock        AS max_threshold
		FROM kanban_parameters kp
		LEFT JOIN LATERAL (
			SELECT woi2.part_number, woi2.part_name, woi2.wo_id
			FROM work_order_items woi2
			WHERE woi2.item_uniq_code = kp.item_uniq_code
			ORDER BY woi2.created_at DESC
			LIMIT 1
		) woi ON TRUE
		LEFT JOIN work_orders w ON w.id = woi.wo_id
		LEFT JOIN items i
			ON i.uniq_code = kp.item_uniq_code
			AND i.deleted_at IS NULL
		WHERE kp.status ILIKE 'active'
			AND ($1 = ''
				OR kp.item_uniq_code           ILIKE $1 || '%'
				OR COALESCE(woi.part_name, '')  ILIKE $1 || '%'
				OR COALESCE(woi.part_number, '') ILIKE $1 || '%')
		ORDER BY kp.item_uniq_code
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
	// Look up part details + last WO + initial quantity from work_order_items
	type bomRow struct {
		PartNumber *string  `gorm:"column:part_number"`
		PartName   *string  `gorm:"column:part_name"`
		Model      *string  `gorm:"column:model"`
		WONumber   *string  `gorm:"column:wo_number"`
		Quantity   float64  `gorm:"column:quantity"`
	}
	var bom bomRow
	_ = s.db.WithContext(ctx).Raw(`
		SELECT woi.part_number, woi.part_name, i.model, w.wo_number, woi.quantity
		FROM work_order_items woi
		JOIN work_orders w ON w.id = woi.wo_id
		LEFT JOIN items i ON i.uniq_code = woi.item_uniq_code AND i.deleted_at IS NULL
		WHERE woi.item_uniq_code = ?
		ORDER BY w.created_at DESC
		LIMIT 1
	`, req.UniqCode).Scan(&bom)

	var woNumber *string
	if req.WONumberOverride != nil && *req.WONumberOverride != "" {
		woNumber = req.WONumberOverride
	} else if bom.WONumber != nil && *bom.WONumber != "" {
		woNumber = bom.WONumber
	}

	// Initial stock from WO item quantity; bulk upload can override via StockQty
	initialStock := bom.Quantity

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
		safetyStockQty = &kanban.MinStock // replenishment target = min_stock
	}
	if kanban.MaxStock > 0 {
		maxThreshold = &kanban.MaxStock
	}

	now := time.Now()

	// Check if record already exists (including soft-deleted) — upsert behaviour
	var existing fgModels.FinishedGoods
	err := s.db.WithContext(ctx).
		Unscoped().
		Where("uniq_code = ?", req.UniqCode).
		First(&existing).Error

	if err == nil {
		// Record exists: restore if soft-deleted, update warehouse + kanban params
		stockToComplete := computeStockToComplete(existing.StockQty, safetyStockQty)
		kanbanCount := computeKanbanCount(existing.StockQty, kanbanStandardQty)
		status := computeStatus(existing.StockQty, minThreshold, maxThreshold)

		updates := map[string]interface{}{
			"deleted_at":              nil,
			"warehouse_location":      req.WarehouseLocation,
			"wo_number":               woNumber,
			"part_number":             bom.PartNumber,
			"part_name":               bom.PartName,
			"model":                   bom.Model,
			"kanban_standard_qty":     kanbanStandardQty,
			"min_threshold":           minThreshold,
			"max_threshold":           maxThreshold,
			"safety_stock_qty":        safetyStockQty,
			"kanban_count":            kanbanCount,
			"stock_to_complete_kanban": stockToComplete,
			"status":                  status,
			"updated_by":              createdBy,
			"updated_at":              now,
		}
		if err := s.db.WithContext(ctx).Unscoped().Model(&fgModels.FinishedGoods{}).
			Where("uniq_code = ?", req.UniqCode).
			Updates(updates).Error; err != nil {
			return nil, apperror.InternalWrap("fg upsert update", err)
		}
		existing.WarehouseLocation = &req.WarehouseLocation
		existing.WONumber = woNumber
		existing.KanbanStandardQty = kanbanStandardQty
		existing.MinThreshold = minThreshold
		existing.MaxThreshold = maxThreshold
		existing.SafetyStockQty = safetyStockQty
		existing.DeletedAt = nil
		return toItem(&existing), nil
	}

	// New record — initial stock from WO item quantity
	stockToComplete := computeStockToComplete(initialStock, safetyStockQty)
	kanbanCount := computeKanbanCount(initialStock, kanbanStandardQty)
	status := computeStatus(initialStock, minThreshold, maxThreshold)

	fg := &fgModels.FinishedGoods{
		UniqCode:              req.UniqCode,
		PartNumber:            bom.PartNumber,
		PartName:              bom.PartName,
		Model:                 bom.Model,
		WONumber:              woNumber,
		WarehouseLocation:     &req.WarehouseLocation,
		StockQty:              initialStock,
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

	src := "manual"
	_ = s.repo.AppendMovementLog(ctx, &fgModels.FGMovementLog{
		FgID:         fg.ID,
		UniqCode:     fg.UniqCode,
		MovementType: "manual_add",
		QtyChange:    initialStock,
		QtyBefore:    0,
		QtyAfter:     initialStock,
		SourceFlag:   &src,
		WONumber:     fg.WONumber,
		LoggedBy:     &createdBy,
		LoggedAt:     now,
	})

	return toItem(fg), nil
}

// ---------------------------------------------------------------------------
// Bulk Create
// ---------------------------------------------------------------------------

func (s *service) BulkCreateFinishedGoods(ctx context.Context, req fgModels.BulkCreateFinishedGoodsRequest, createdBy string) (*fgModels.FGBulkCreateResponse, error) {
	results := make([]fgModels.FGBulkCreateResult, 0, len(req.Items))
	created := 0
	failed := 0

	for i, item := range req.Items {
		singleReq := fgModels.CreateFinishedGoodsRequest{
			UniqCode:          item.UniqCode,
			WarehouseLocation: item.WarehouseLocation,
			WONumberOverride:  item.WONumber,
		}
		fg, err := s.CreateFinishedGoods(ctx, singleReq, createdBy)
		if err != nil {
			failed++
			errMsg := err.Error()
			results = append(results, fgModels.FGBulkCreateResult{
				Index:    i,
				UniqCode: item.UniqCode,
				Status:   "failed",
				Error:    &errMsg,
			})
			continue
		}

		// Apply initial stock_qty from review step if provided
		if item.StockQty != nil && *item.StockQty > 0 {
			stockReq := fgModels.UpdateFinishedGoodsRequest{StockQty: item.StockQty}
			fg, _ = s.UpdateFinishedGoods(ctx, fg.ID, stockReq, createdBy)
		}

		created++
		uuidStr := fg.UUID
		results = append(results, fgModels.FGBulkCreateResult{
			Index:    i,
			UniqCode: item.UniqCode,
			Status:   "created",
			ID:       &fg.ID,
			UUID:     &uuidStr,
		})
	}

	return &fgModels.FGBulkCreateResponse{
		Created: created,
		Failed:  failed,
		Results: results,
	}, nil
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
