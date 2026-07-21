package service

import (
	"time"

	"gorm.io/gorm"
)

// appendFGMovementLog mencatat pergerakan stok Finished Goods (mis. pengurangan
// saat DN customer) ke tabel fg_movement_logs. Tabel ini yang ditampilkan pada
// History Log di halaman detail Finished Goods (ERP).
func (s *deliveryNoteService) appendFGMovementLog(
	tx *gorm.DB,
	fgID int64,
	uniq string,
	movementType string,
	qtyChange float64,
	qtyBefore float64,
	qtyAfter float64,
	dnNumber string,
	notes string,
	loggedBy string,
) error {
	source := "delivery"
	row := map[string]interface{}{
		"fg_id":         fgID,
		"uniq_code":     uniq,
		"movement_type": movementType,
		"qty_change":    qtyChange,
		"qty_before":    qtyBefore,
		"qty_after":     qtyAfter,
		"source_flag":   source,
		"logged_at":     time.Now(),
	}
	if dnNumber != "" {
		row["dn_number"] = dnNumber
	}
	if notes != "" {
		row["notes"] = notes
	}
	if loggedBy != "" {
		row["logged_by"] = loggedBy
	}
	return tx.Table("fg_movement_logs").Create(row).Error
}
