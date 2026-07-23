package service

import (
	"time"

	"gorm.io/gorm"
)

// appendFGMovementLog mencatat satu baris pergerakan stok Finished Goods ke
// tabel fg_movement_logs. Tabel inilah yang dibaca endpoint
// GET /api/v1/finished-goods/history untuk menampilkan History Log pada halaman
// detail Finished Goods (ERP).
//
// Sebelumnya action-ui hanya menulis ke inventory_movement_logs, sehingga
// aktivitas dari action-ui (mis. QC Finish) tidak pernah muncul di History Log
// Finished Goods. Fungsi ini melengkapi kekurangan tersebut.
func (s *service) appendFGMovementLog(
	tx *gorm.DB,
	fgID int64,
	uniq string,
	movementType string,
	qtyChange float64,
	qtyBefore float64,
	qtyAfter float64,
	woNumber *string,
	dnNumber *string,
	notes *string,
	loggedBy *string,
) error {
	source := "action_ui"
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
	if woNumber != nil && *woNumber != "" {
		row["wo_number"] = *woNumber
	}
	if dnNumber != nil && *dnNumber != "" {
		row["dn_number"] = *dnNumber
	}
	if notes != nil && *notes != "" {
		row["notes"] = *notes
	}
	if loggedBy != nil && *loggedBy != "" {
		row["logged_by"] = *loggedBy
	}
	return tx.Table("fg_movement_logs").Create(row).Error
}
