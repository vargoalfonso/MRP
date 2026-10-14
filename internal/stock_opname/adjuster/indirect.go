package adjuster

import (
	"context"
	"strings"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IndirectAdjuster struct{}

func NewIndirectAdjuster() InventoryAdjuster { return &IndirectAdjuster{} }

func (a *IndirectAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row invModels.IndirectRawMaterial
	err := tx.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return newUniqSnapshot(ctx, tx, uniqCode)
	}
	if err != nil {
		return nil, apperror.Internal("resolve indirect uniq: " + err.Error())
	}
	return &UniqSnapshot{EntityID: &row.ID, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQty: row.StockQty, WeightKg: row.StockWeightKg}, nil
}

func (a *IndirectAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []invModels.IndirectRawMaterial
	query := tx.WithContext(ctx).Where("deleted_at IS NULL")
	if q != "" {
		query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search indirect uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].StockQty, WeightKg: rows[i].StockWeightKg}})
	}
	return items, nil
}

func (a *IndirectAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row invModels.IndirectRawMaterial
	if err := lockInventoryRow(ctx, tx, &row, entry, ""); err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, apperror.Internal("lock indirect row: " + err.Error())
		}
		found, ferr := liveOrRevive(ctx, tx, &row, entry.UniqCode, "stock_qty", actor)
		if ferr != nil {
			return nil, ferr
		}
		if !found {
			now := time.Now()
			itemID := (*int64)(nil)
			if item, ierr := findBOMItem(ctx, tx, entry.UniqCode); ierr == nil && item != nil {
				itemID = &item.ID
			}
			normal := "normal"
			row = invModels.IndirectRawMaterial{
				UUID: uuid.New(), UniqCode: strings.TrimSpace(entry.UniqCode),
				PartNumber: entry.PartNumber, PartName: entry.PartName, UOM: entry.UOM,
				ItemID: itemID, Status: &normal,
				CreatedBy: &actor, UpdatedBy: &actor, CreatedAt: now, UpdatedAt: now,
			}
			if cerr := tx.WithContext(ctx).Create(&row).Error; cerr != nil {
				return nil, apperror.Internal("create indirect stock opname: " + cerr.Error())
			}
		}
	}

	qtyChange := entry.CountedQty - row.StockQty
	now := time.Now()
	updates := map[string]interface{}{
		"stock_qty":   entry.CountedQty,
		"status":      inventoryStatus(entry.CountedQty, row.SafetyStockQty),
		"stock_days":  inventoryStockDays(entry.CountedQty, row.DailyUsageQty),
		"buy_not_buy": inventoryBuyNotBuy(entry.CountedQty, row.SafetyStockQty),
		"updated_by":  actor,
		"updated_at":  now,
	}
	if entry.WeightKg != nil {
		updates["stock_weight_kg"] = *entry.WeightKg
	}
	if err := tx.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, apperror.Internal("update indirect stock opname: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: qtyChange, WeightChange: entry.WeightKg, EntityID: &row.ID}, nil
}

func inventoryStatus(stock float64, safety *float64) string {
	threshold := 10.0
	if safety != nil && *safety > 0 {
		threshold = *safety
	}
	if stock < threshold {
		return "low_on_stock"
	}
	if stock > threshold*2 {
		return "overstock"
	}
	return "normal"
}

func inventoryStockDays(stock float64, dailyUsage *float64) int {
	divisor := 1.0
	if dailyUsage != nil && *dailyUsage > 0 {
		divisor = *dailyUsage
	}
	return int(stock / divisor)
}

func inventoryBuyNotBuy(stock float64, safety *float64) string {
	threshold := 10.0
	if safety != nil && *safety > 0 {
		threshold = *safety
	}
	if stock < threshold {
		return "buy"
	}
	return "not_buy"
}

func inventoryBuyNotBuyRM(stock float64, safety *float64, rawMaterialType string) string {
	if rawMaterialType == "ssp" || rawMaterialType == "SSP" {
		return "n/a"
	}
	return inventoryBuyNotBuy(stock, safety)
}
