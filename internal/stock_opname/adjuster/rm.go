package adjuster

import (
	"context"
	"fmt"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RMAdjuster struct{}

func NewRMAdjuster() InventoryAdjuster { return &RMAdjuster{} }

func (a *RMAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row invModels.RawMaterial
	err := tx.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("raw material uniq_code %s not found", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve RM uniq: " + err.Error())
	}
	return &UniqSnapshot{EntityID: &row.ID, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQty: row.StockQty, WeightKg: row.StockWeightKg}, nil
}

func (a *RMAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []invModels.RawMaterial
	query := tx.WithContext(ctx).Where("deleted_at IS NULL")
	if q != "" {
		query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search RM uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].StockQty, WeightKg: rows[i].StockWeightKg}})
	}
	return items, nil
}

func (a *RMAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row invModels.RawMaterial
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", entry.EntityID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("raw material record not found during stock opname")
		}
		return nil, apperror.Internal("lock RM row: " + err.Error())
	}

	qtyChange := entry.CountedQty - row.StockQty
	now := time.Now()
	updates := map[string]interface{}{
		"stock_qty":   entry.CountedQty,
		"status":      inventoryStatus(entry.CountedQty, row.SafetyStockQty),
		"stock_days":  inventoryStockDays(entry.CountedQty, row.DailyUsageQty),
		"buy_not_buy": inventoryBuyNotBuyRM(entry.CountedQty, row.SafetyStockQty, row.RawMaterialType),
		"updated_by":  actor,
		"updated_at":  now,
	}
	if entry.WeightKg != nil {
		updates["stock_weight_kg"] = *entry.WeightKg
	}
	if err := tx.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, apperror.Internal("update RM stock opname: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: qtyChange, WeightChange: entry.WeightKg}, nil
}
