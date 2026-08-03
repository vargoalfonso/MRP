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

// SubconAdjuster handles stock opname for materials held at subcon vendors
// (subcon_inventories). The counted quantity adjusts stock_at_vendor_qty.
type SubconAdjuster struct{}

func NewSubconAdjuster() InventoryAdjuster { return &SubconAdjuster{} }

func (a *SubconAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row invModels.SubconInventory
	err := tx.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("subcon inventory uniq_code %s tidak ditemukan", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve subcon uniq: " + err.Error())
	}
	return &UniqSnapshot{EntityID: &row.ID, PartNumber: row.PartNumber, PartName: row.PartName, SystemQty: row.StockAtVendorQty}, nil
}

func (a *SubconAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []invModels.SubconInventory
	// [so-subcon] Stock opname subcon hanya menghitung uniq dari tab
	// "Stock In Vendor" (total_received_qty = 0). Baris yang sudah pindah ke
	// "Stock Received from Vendor" (total_received_qty > 0) dikecualikan agar
	// tidak ikut masuk daftar uniq saat start count.
	query := tx.WithContext(ctx).Where("deleted_at IS NULL").Where("COALESCE(total_received_qty, 0) = 0")
	if q != "" {
		query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search subcon uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, SystemQty: rows[i].StockAtVendorQty}})
	}
	return items, nil
}

func (a *SubconAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row invModels.SubconInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", entry.EntityID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("subcon inventory record tidak ditemukan during stock opname")
		}
		return nil, apperror.Internal("lock subcon row: " + err.Error())
	}

	qtyChange := entry.CountedQty - row.StockAtVendorQty
	now := time.Now()
	updates := map[string]interface{}{
		"stock_at_vendor_qty": entry.CountedQty,
		"status":              subconStatus(entry.CountedQty, row.SafetyStockQty, row.TotalPOQty),
		"updated_by":          actor,
		"updated_at":          now,
	}
	if err := tx.WithContext(ctx).Model(&row).Updates(updates).Error; err != nil {
		return nil, apperror.Internal("update subcon stock opname: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: qtyChange}, nil
}

// subconStatus mirrors the subcon inventory status logic:
// above_po | low_on_stock | overstock | normal.
func subconStatus(stock float64, safety, totalPO *float64) string {
	if totalPO != nil && *totalPO > 0 && stock > *totalPO {
		return "above_po"
	}
	return inventoryStatus(stock, safety)
}
