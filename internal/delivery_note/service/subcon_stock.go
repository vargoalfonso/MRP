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

// subconEnrichment holds best-effort metadata resolved from Finished Goods and
// the matching SUBCON Purchase Order, so that subcon inventory rows created from
// a DN Subcon OUT/IN scan are not left with empty vendor / PO / part fields.
type subconEnrichment struct {
	PartNumber     *string
	PartName       *string
	SafetyStockQty *float64
	VendorID       *int64
	VendorName     *string
	PONumber       *string
	POPeriod       *string
	TotalPOQty     *float64
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

	// Resolve metadata once; used both for auto-create and for backfilling an
	// existing row whose vendor / PO / part fields are still empty.
	meta := s.lookupSubconEnrichment(tx, uniq)

	var row subconStockRow
	if err := tx.Table("subcon_inventories").
		Select("id, stock_at_vendor_qty").
		Where("uniq_code = ? AND deleted_at IS NULL", uniq).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Scan(&row).Error; err != nil {
		return err
	}

	if row.ID == 0 {
		// Auto-create the subcon inventory record on first movement, filling in
		// as much metadata as we can resolve.
		newRow := map[string]interface{}{
			"uniq_code":           uniq,
			"part_number":         meta.PartNumber,
			"part_name":           meta.PartName,
			"po_number":           meta.PONumber,
			"po_period":           meta.POPeriod,
			"subcon_vendor_id":    meta.VendorID,
			"subcon_vendor_name":  meta.VendorName,
			"total_po_qty":        meta.TotalPOQty,
			"safety_stock_qty":    meta.SafetyStockQty,
			"date_delivery":       now,
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
	} else {
		// DN Subcon OUT: refresh the delivery date to the latest shipment.
		updates["date_delivery"] = now
	}

	// Backfill metadata still empty on an existing row (never overwrites values
	// already set, e.g. edited by a user).
	if meta.PartNumber != nil {
		updates["part_number"] = gorm.Expr("COALESCE(part_number, ?)", *meta.PartNumber)
	}
	if meta.PartName != nil {
		updates["part_name"] = gorm.Expr("COALESCE(part_name, ?)", *meta.PartName)
	}
	if meta.PONumber != nil {
		updates["po_number"] = gorm.Expr("COALESCE(po_number, ?)", *meta.PONumber)
	}
	if meta.POPeriod != nil {
		updates["po_period"] = gorm.Expr("COALESCE(po_period, ?)", *meta.POPeriod)
	}
	if meta.VendorID != nil {
		updates["subcon_vendor_id"] = gorm.Expr("COALESCE(subcon_vendor_id, ?)", *meta.VendorID)
	}
	if meta.VendorName != nil {
		updates["subcon_vendor_name"] = gorm.Expr("COALESCE(subcon_vendor_name, ?)", *meta.VendorName)
	}
	if meta.TotalPOQty != nil {
		updates["total_po_qty"] = gorm.Expr("COALESCE(total_po_qty, ?)", *meta.TotalPOQty)
	}
	if meta.SafetyStockQty != nil {
		updates["safety_stock_qty"] = gorm.Expr("COALESCE(safety_stock_qty, ?)", *meta.SafetyStockQty)
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

// logSubconDNScan mencatat satu baris "Delivery Note Log" untuk scan
// DN Subcon OUT / IN.
//
// [subcon-dnlog] Tab "Delivery Note Logs" di ERP membaca endpoint
// GET /inventory/subcon-materials/incoming yang sumbernya tabel
// incoming_receiving_scans (JOIN delivery_note_items + delivery_notes).
// Scan DN Subcon sebelumnya hanya menulis subcon_inventories dan
// inventory_movement_logs, sehingga tab tersebut selalu kosong.
//
// Bersifat best-effort: kalau baris DN tidak ketemu, scan tetap sukses
// dan pencatatan log dilewati saja.
func (s *deliveryNoteService) logSubconDNScan(tx *gorm.DB, uniq, dnNumber, packingNumber string, qty float64, scannedBy, direction string) error {
	if qty <= 0 {
		return nil
	}
	if strings.TrimSpace(scannedBy) == "" {
		scannedBy = "system"
	}

	var dnItem struct {
		ID int64
	}

	// Utamakan baris DN yang packing number-nya persis sama dengan yang discan.
	q := tx.Table("delivery_note_items dni").
		Select("dni.id").
		Joins("JOIN delivery_notes dn ON dn.id = dni.dn_id").
		Where("dn.dn_number = ?", strings.TrimSpace(dnNumber))
	if p := strings.TrimSpace(packingNumber); p != "" {
		q = q.Where("dni.packing_number = ?", p)
	} else {
		q = q.Where("dni.item_uniq_code = ?", uniq)
	}
	if err := q.Order("dni.id DESC").Limit(1).Scan(&dnItem).Error; err != nil {
		return err
	}

	// Cadangan: cari lewat item uniq code kalau packing number tidak cocok.
	if dnItem.ID == 0 {
		if err := tx.Table("delivery_note_items dni").
			Select("dni.id").
			Joins("JOIN delivery_notes dn ON dn.id = dni.dn_id").
			Where("dni.item_uniq_code = ?", uniq).
			Order("dni.id DESC").Limit(1).
			Scan(&dnItem).Error; err != nil {
			return err
		}
	}
	if dnItem.ID == 0 {
		return nil
	}

	now := time.Now()
	// scan_ref menyimpan arah pergerakan supaya ERP bisa memberi label,
	// dan stempel waktu menjaga unique index (incoming_dn_item_id, scan_ref)
	// tetap aman saat packing list yang sama discan lebih dari sekali.
	scanRef := direction + "/" + strings.TrimSpace(packingNumber) + "/" + now.Format("20060102150405.000")

	logRow := map[string]interface{}{
		"incoming_dn_item_id": dnItem.ID,
		"scan_ref":            scanRef,
		"qty":                 qty,
		"scanned_at":          now,
		"scanned_by":          scannedBy,
	}
	return tx.Table("incoming_receiving_scans").Create(logRow).Error
}

// lookupSubconEnrichment resolves part / vendor / PO metadata for a subcon item
// on a best-effort basis. Any source that is missing simply leaves its fields
// nil so the caller can decide whether to backfill.
func (s *deliveryNoteService) lookupSubconEnrichment(tx *gorm.DB, uniq string) subconEnrichment {
	var meta subconEnrichment

	// 1. Part identity + safety stock from Finished Goods (most reliable).
	var fg struct {
		PartNumber     *string
		PartName       *string
		SafetyStockQty *float64
	}
	if err := tx.Table("finished_goods").
		Select("part_number, part_name, safety_stock_qty").
		Where("uniq_code = ? AND deleted_at IS NULL", uniq).
		Limit(1).
		Scan(&fg).Error; err == nil {
		meta.PartNumber = fg.PartNumber
		meta.PartName = fg.PartName
		meta.SafetyStockQty = fg.SafetyStockQty
	}

	// 2. Vendor + PO info from the latest SUBCON purchase order for this item.
	var po struct {
		PartNumber   *string
		PartName     *string
		TotalPoQty   *float64
		PoNumber     *string
		Period       *string
		SupplierID   *int64
		SupplierName *string
	}
	_ = tx.Table("purchase_order_items poi").
		Select("poi.part_number AS part_number, poi.part_name AS part_name, poi.ordered_qty AS total_po_qty, po.po_number AS po_number, po.period AS period, po.supplier_id AS supplier_id, sup.supplier_name AS supplier_name").
		Joins("JOIN purchase_orders po ON po.po_id = poi.po_id").
		Joins("LEFT JOIN suppliers sup ON sup.id = po.supplier_id").
		Where("poi.item_uniq_code = ? AND po.po_type = ?", uniq, "SUBCON").
		Order("po.po_date DESC NULLS LAST, po.po_id DESC").
		Limit(1).
		Scan(&po).Error

	if po.PoNumber != nil {
		meta.PONumber = po.PoNumber
	}
	if po.Period != nil {
		meta.POPeriod = po.Period
	}
	if po.SupplierID != nil {
		meta.VendorID = po.SupplierID
	}
	if po.SupplierName != nil {
		meta.VendorName = po.SupplierName
	}
	if po.TotalPoQty != nil {
		meta.TotalPOQty = po.TotalPoQty
	}
	// Fallback part identity from the PO line when Finished Goods lacked it.
	if meta.PartNumber == nil {
		meta.PartNumber = po.PartNumber
	}
	if meta.PartName == nil {
		meta.PartName = po.PartName
	}

	return meta
}
