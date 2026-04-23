package adjuster

import (
	"context"
	"fmt"
	"math"
	"time"

	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/inventoryconst"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FGAdjuster struct{}

func NewFGAdjuster() InventoryAdjuster { return &FGAdjuster{} }

func (a *FGAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row fgModels.FinishedGoods
	err := tx.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("finished goods uniq_code %s not found", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve FG uniq: " + err.Error())
	}
	return &UniqSnapshot{EntityID: &row.ID, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQty: row.StockQty}, nil
}

func (a *FGAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []fgModels.FinishedGoods
	query := tx.WithContext(ctx).Where("deleted_at IS NULL")
	if q != "" {
		query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search FG uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].StockQty}})
	}
	return items, nil
}

func (a *FGAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row fgModels.FinishedGoods
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", entry.EntityID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("finished goods record not found during stock opname")
		}
		return nil, apperror.Internal("lock FG row: " + err.Error())
	}

	before := row.StockQty
	after := entry.CountedQty
	delta := after - before
	status := fgStatus(after, row.MinThreshold, row.MaxThreshold)
	kanbanCount := fgKanbanCount(after, row.KanbanStandardQty)
	stockToComplete := fgStockToComplete(after, row.SafetyStockQty)
	now := time.Now()

	if err := tx.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"stock_qty":                after,
		"status":                   status,
		"kanban_count":             kanbanCount,
		"stock_to_complete_kanban": stockToComplete,
		"updated_by":               actor,
		"updated_at":               now,
	}).Error; err != nil {
		return nil, apperror.Internal("update FG stock opname: " + err.Error())
	}

	sourceFlag := string(inventoryconst.SourceStockOpname)
	notes := fmt.Sprintf("stock opname %s", sessionNumber)
	log := &fgModels.FGMovementLog{
		FgID:         row.ID,
		UniqCode:     row.UniqCode,
		MovementType: string(inventoryconst.MovementStockOpname),
		QtyChange:    delta,
		QtyBefore:    before,
		QtyAfter:     after,
		SourceFlag:   &sourceFlag,
		ReferenceID:  &sessionNumber,
		Notes:        &notes,
		LoggedBy:     &actor,
		LoggedAt:     now,
	}
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, apperror.Internal("append FG movement log: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: delta}, nil
}

func fgStatus(stockQty float64, minThreshold, maxThreshold *float64) string {
	if minThreshold != nil && stockQty < *minThreshold {
		return fgModels.FGStatusLowStock
	}
	if maxThreshold != nil && stockQty > *maxThreshold {
		return fgModels.FGStatusOverstock
	}
	return fgModels.FGStatusNormal
}

func fgKanbanCount(stockQty float64, std *int) *int {
	if std == nil || *std <= 0 {
		return nil
	}
	v := int(math.Floor(stockQty / float64(*std)))
	return &v
}

func fgStockToComplete(stockQty float64, safety *float64) *float64 {
	if safety == nil {
		return nil
	}
	v := math.Max(0, *safety-stockQty)
	return &v
}
