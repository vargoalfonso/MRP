package service

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// subconStockRow is a lightweight projection of subcon_inventories used when
// posting DN Subcon OUT / IN movements.
type subconStockRow struct {
	ID               int64
	StockAtVendorQty float64
}

// addSubconStockOut records goods sent OUT to an external subcontractor (DN
// Subcon OUT). It increases subcon_inventories.stock_at_vendor_qty (Inventory >
// Subcon Material > tab "Stock In Vendor") and writes a movement log. It never
// touches Finished Goods or WIP.
func (s *deliveryNoteService) addSubconStockOut(tx *gorm.DB, uniq, dnNumber string, qty float64, scannedBy string) error {
	return s.upsertSubconStock(tx, uniq, dnNumber, qty, scannedBy, "outgoing", "stock_at_vendor_qty", "subcon_out", "Barang keluar ke vendor subcon (DN Subcon OUT)")
}

// addSubconStockReceived records goods RETURNED from an external subcontractor
// (DN Subcon IN). It increases total_received_qty (tab "Stock Received from
// Vendor") and reduces stock_at_vendor_qty (clamped at 0). It never touches
// Finished Goods or WIP.
func (s *deliveryNoteService) addSubconStockReceived(tx *gorm.DB, uniq, dnNumber string, qty float64, scannedBy string) error {
	return s.upsertSubconStock(tx, uniq, dnNumber, qty, scannedBy, "received_from_vendor", "total_received_qty", "subcon_in", "Barang selesai diproses subcon (DN Subcon IN)")
}

func (s *deliveryNoteService) upsertSubconStock(tx *gorm.DB, uniq, dnNumber string, qty float64, scannedBy, movementType, qtyColumn, sourceFlag, notes string) error {
	uniq = strings.TrimSpace(uniq)
	if uniq == "" {
		return errors.New("item uniq code kosong untuk subcon inventory")
	}
	if qty <= 0 {
		return errors.New("qty subcon harus > 0")
	}
	if strings.TrimSpace(scannedBy) == "" {
		scannedBy = "system"
	}
	now := time.Now()

	var row subconStockRow
	if err := tx.Table("subcon_inventories").
		Select("id, stock_at_vendor_qty").
		Where("uniq_code = ? AND deleted_at IS NULL", uniq).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scan(&row).Error; err != nil {
		return err
	}

	if row.ID == 0 {
		// Auto-create the subcon inventory record on first movement.
		partNumber, partName := s.lookupSubconPartInfo(tx, uniq)
		newRow := map[string]interface{}{
			"uniq_code":           uniq,
			"part_number":         partNumber,
			"part_name":           partName,
			"stock_at_vendor_qty": 0,
			"total_received_qty":  0,
			"status":              "normal",
			"created_by":          scannedBy,
			"created_at":          now,
			"updated_at":          now,
		}
		if err := tx.Table("subcon_inventories").Create(newRow).Error; err != nil {
			return err
		}
		if err := tx.Table("subcon_inventories").
			Select("id, stock_at_vendor_qty").
			Where("uniq_code = ? AND deleted_at IS NULL", uniq).
			Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return errors.New("gagal membuat subcon inventory untuk item " + uniq)
		}
	}

	updates := map[string]interface{}{
		qtyColumn:    gorm.Expr(qtyColumn+" + ?", qty),
		"updated_by": scannedBy,
		"updated_at": now,
	}
	// Goods received back leave the vendor's stock.
	if movementType == "received_from_vendor" {
		updates["stock_at_vendor_qty"] = gorm.Expr("GREATEST(stock_at_vendor_qty - ?, 0)", qty)
	}
	if err := tx.Table("subcon_inventories").Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return err
	}

	// History log (read by Inventory > Subcon Material > History).
	logRow := map[string]interface{}{
		"movement_category": "subcon",
		"movement_type":     movementType,
		"uniq_code":         uniq,
		"entity_id":         row.ID,
		"qty_change":        qty,
		"source_flag":       sourceFlag,
		"dn_number":         dnNumber,
		"reference_id":      dnNumber,
		"notes":             notes,
		"logged_by":         scannedBy,
		"logged_at":         now,
	}
	if err := tx.Table("inventory_movement_logs").Create(logRow).Error; err != nil {
		return err
	}
	return nil
}

// lookupSubconPartInfo best-effort resolves part number / name from PO items.
func (s *deliveryNoteService) lookupSubconPartInfo(tx *gorm.DB, uniq string) (interface{}, interface{}) {
	var poi struct {
		PartNumber *string
		PartName   *string
	}
	_ = tx.Table("purchase_order_items").
		Select("part_number, part_name").
		Where("item_uniq_code = ?", uniq).
		Limit(1).
		Scan(&poi).Error
	return poi.PartNumber, poi.PartName
}
