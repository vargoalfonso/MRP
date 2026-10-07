package adjuster

import (
	"context"
	"strings"

	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// WarehouseItemResult is one inventory row that physically sits in a
// warehouse. SystemQty ("remain") is kept server-side only: it is never
// serialized, so counters cannot see it (anti-fraud). KanbanQty is the
// kanban/box standard qty (not the remaining stock).
type WarehouseItemResult struct {
	UniqCode          string
	PartNumber        *string
	PartName          *string
	UOM               *string
	WarehouseLocation string
	KanbanQty         *int
	SystemQty         float64
	WeightKg          *float64
	RawMaterialType   string
}

func warehouseMatch(col string) string {
	return "UPPER(BTRIM(COALESCE(" + col + ", ''))) = UPPER(BTRIM(?))"
}

// ListWarehouseItems returns every inventory row of the given type that is
// stored in the given warehouse, ordered by uniq code. FG / RM / Indirect have
// a warehouse_location column and are filtered by it. WIP and Subcon rows
// carry no warehouse, so all live rows are returned and labelled with the
// selected warehouse.
func ListWarehouseItems(ctx context.Context, db *gorm.DB, inventoryType, warehouse, q string, limit int) ([]WarehouseItemResult, error) {
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	warehouse = strings.TrimSpace(warehouse)
	like := "%" + strings.TrimSpace(q) + "%"
	hasQ := strings.TrimSpace(q) != ""

	switch inventoryType {
	case stockModels.InventoryTypeFG:
		var rows []fgModels.FinishedGoods
		query := db.WithContext(ctx).Where("deleted_at IS NULL").Where(warehouseMatch("warehouse_location"), warehouse)
		if hasQ {
			query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", like, like, like)
		}
		if err := query.Order("uniq_code ASC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, apperror.Internal("list FG warehouse items: " + err.Error())
		}
		out := make([]WarehouseItemResult, 0, len(rows))
		for i := range rows {
			out = append(out, WarehouseItemResult{UniqCode: rows[i].UniqCode, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, WarehouseLocation: warehouse, KanbanQty: rows[i].KanbanStandardQty, SystemQty: rows[i].StockQty})
		}
		return out, nil

	case stockModels.InventoryTypeRM:
		var rows []invModels.RawMaterial
		query := db.WithContext(ctx).Where("deleted_at IS NULL").Where(warehouseMatch("warehouse_location"), warehouse)
		if hasQ {
			query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", like, like, like)
		}
		if err := query.Order("uniq_code ASC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, apperror.Internal("list RM warehouse items: " + err.Error())
		}
		out := make([]WarehouseItemResult, 0, len(rows))
		for i := range rows {
			out = append(out, WarehouseItemResult{UniqCode: rows[i].UniqCode, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, WarehouseLocation: warehouse, KanbanQty: rows[i].KanbanStandardQty, SystemQty: rows[i].StockQty, WeightKg: rows[i].StockWeightKg, RawMaterialType: rows[i].RawMaterialType})
		}
		return out, nil

	case stockModels.InventoryTypeIDR:
		var rows []invModels.IndirectRawMaterial
		query := db.WithContext(ctx).Where("deleted_at IS NULL").Where(warehouseMatch("warehouse_location"), warehouse)
		if hasQ {
			query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", like, like, like)
		}
		if err := query.Order("uniq_code ASC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, apperror.Internal("list indirect warehouse items: " + err.Error())
		}
		out := make([]WarehouseItemResult, 0, len(rows))
		for i := range rows {
			out = append(out, WarehouseItemResult{UniqCode: rows[i].UniqCode, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, WarehouseLocation: warehouse, KanbanQty: rows[i].KanbanStandardQty, SystemQty: rows[i].StockQty, WeightKg: rows[i].StockWeightKg})
		}
		return out, nil

	case stockModels.InventoryTypeWIP:
		// wip_items has no warehouse column: reuse the live-row search.
		rows, err := NewWIPAdjuster().SearchUniqs(ctx, db, strings.TrimSpace(q), limit)
		return toWarehouseItems(rows, warehouse), err

	case stockModels.InventoryTypeSubcon:
		// Same "still in vendor" rule as the manual start-count list.
		rows, err := NewSubconAdjuster().SearchUniqs(ctx, db, strings.TrimSpace(q), limit)
		return toWarehouseItems(rows, warehouse), err
	}
	return nil, apperror.BadRequest("inventory type tidak valid")
}

func toWarehouseItems(rows []UniqSnapshotResult, warehouse string) []WarehouseItemResult {
	out := make([]WarehouseItemResult, 0, len(rows))
	for i := range rows {
		out = append(out, WarehouseItemResult{UniqCode: rows[i].UniqCode, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, WarehouseLocation: warehouse, SystemQty: rows[i].SystemQty, WeightKg: rows[i].WeightKg, RawMaterialType: rows[i].RawMaterialType})
	}
	return out
}
