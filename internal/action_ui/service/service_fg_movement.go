package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	"github.com/ganasa18/go-template/internal/action_ui/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Jenis pergerakan Finished Goods dari action-ui. Label tampilnya dipetakan di
// finished_goods.movementReasonLabel (History Log detail FG di ERP).
const (
	fgMoveIncomingProduction = "incoming_production" // Scan Out proses terakhir
	fgMoveProductionUsage    = "production_usage"    // dipakai parent WO Assembly
	fgMoveQCFinal            = "qc_final"            // QC Round 3 selesai, qty good sesuai
	fgMoveQCNGRework         = "qc_ng_rework"        // QC Round 3: NG -> WO Rework
	fgMoveQCScrap            = "qc_scrap"            // QC Round 3: scrap
	fgMoveQCAdjustment       = "qc_adjustment"       // QC Round 3: selisih lain
)

// appendFGMovementLog mencatat satu baris pergerakan stok Finished Goods ke
// tabel fg_movement_logs. Tabel inilah yang dibaca endpoint
// GET /api/v1/finished-goods/history untuk menampilkan History Log pada halaman
// detail Finished Goods (ERP).
//
// Sebelumnya action-ui hanya menulis ke inventory_movement_logs, sehingga
// aktivitas dari action-ui (mis. QC Finish) tidak pernah muncul di History Log
// Finished Goods. Fungsi ini melengkapi kekurangan tersebut.
// [fg-history] referenceID = kanban / packing yang terlibat (kolom
// "Kanban / Packing List" di History Log).
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
	referenceID *string,
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
	if referenceID != nil && *referenceID != "" {
		row["reference_id"] = *referenceID
	}
	if notes != nil && *notes != "" {
		row["notes"] = *notes
	}
	if loggedBy != nil && *loggedBy != "" {
		row["logged_by"] = *loggedBy
	}
	return tx.Table("fg_movement_logs").Create(row).Error
}

// fgMove = satu perubahan stok FG yang akan dicatat.
type fgMove struct {
	movementType string
	qty          float64 // bertanda: + masuk, - keluar, 0 = catatan saja
	reference    string
	notes        string
}

// applyFGMoves mengubah stok finished_goods secara berurutan dan mencatat tiap
// perubahan ke fg_movement_logs + inventory_movement_logs. Stok tidak pernah
// dibuat negatif; qty yang benar-benar terpotong yang dicatat.
func (s *service) applyFGMoves(tx *gorm.DB, uniq, woNumber, performedBy string, now time.Time, moves []fgMove) error {
	if len(moves) == 0 {
		return nil
	}
	var fg models.FinishedGoods
	// [concurrency] kunci baris FG: dua transaksi tidak saling menimpa stok.
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uniq_code = ?", uniq).First(&fg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // belum ada FG untuk UNIQ ini: tidak ada yang dicatat
		}
		return err
	}

	stock := fg.StockQty
	for _, m := range moves {
		change := m.qty
		if stock+change < 0 {
			change = -stock
		}
		before := stock
		stock += change

		ref := strings.TrimSpace(m.reference)
		if err := s.appendFGMovementLog(tx, fg.ID, uniq, m.movementType,
			change, before, stock,
			stringPtr(woNumber), nil, stringPtr(ref), stringPtr(m.notes), stringPtr(performedBy)); err != nil {
			return err
		}
		if change == 0 {
			continue
		}
		mvType := "incoming"
		if change < 0 {
			mvType = "outgoing"
		}
		if err := s.createInventoryMovementLog(tx, "finished_goods", mvType,
			uniq, &fg.ID, before, change, stock,
			stringPtr(woNumber), stringPtr(strings.ToUpper(m.movementType)),
			stringPtr(m.notes), stringPtr(performedBy)); err != nil {
			return err
		}
	}

	return tx.Model(&models.FinishedGoods{}).Where("id = ?", fg.ID).
		Updates(map[string]interface{}{"stock_qty": stock, "updated_at": now}).Error
}

// [fg-history] consumeFGForProduction memotong stok FG child UNIQ yang dipakai
// parent WO Assembly saat Scan Out. Dicatat per kanban FG yang discan di Step 1
// supaya History Log menunjukkan kanban mana yang terpakai.
func (s *service) consumeFGForProduction(tx *gorm.DB, item models.WorkOrderItem, rm dto.ScanOutRawMaterial, code, woNumber, scannedBy string, now time.Time) error {
	if rm.QtyUsed <= 0 {
		return nil
	}
	notes := fmt.Sprintf("Dipakai produksi %s (kanban %s, proses %s)",
		item.ItemUniqCode, item.KanbanNumber, currentProcessNameOf(item))

	moves := make([]fgMove, 0, len(rm.Packings))
	var allocated float64
	refs := make([]string, 0, len(rm.Packings))
	for _, p := range rm.Packings {
		pn := strings.TrimSpace(p.PackingNumber)
		if pn == "" {
			continue
		}
		refs = append(refs, pn)
		if !p.DeductOnly || p.DeductQty <= 0 {
			continue
		}
		moves = append(moves, fgMove{movementType: fgMoveProductionUsage, qty: -p.DeductQty, reference: pn, notes: notes})
		allocated += p.DeductQty
	}
	// Alokasi per kanban tidak lengkap / tidak cocok dengan qty dipakai:
	// catat satu baris dengan semua kanban sebagai referensi.
	if len(moves) == 0 || allocated < rm.QtyUsed-1e-6 || allocated > rm.QtyUsed+1e-6 {
		moves = []fgMove{{movementType: fgMoveProductionUsage, qty: -rm.QtyUsed, reference: strings.Join(refs, ", "), notes: notes}}
	}
	return s.applyFGMoves(tx, code, woNumber, scannedBy, now, moves)
}

// [fg-history] reconcileFGAfterQCFinish dijalankan saat QC Round 3 (QC Finish)
// untuk task proses TERAKHIR. Qty sudah masuk FG saat Scan Out (Total
// Produksi), jadi di sini hasil QC dicatat ke History Log FG:
//   - NG     -> stok FG dikurangi, barang dibuatkan WO Rework
//   - Scrap  -> stok FG dikurangi, barang masuk Scrap
//   - selisih lain antara Total Produksi dan qty good -> QC Adjustment
//   - tidak ada selisih -> baris "QC Final" (qty 0) sebagai jejak.
func (s *service) reconcileFGAfterQCFinish(ctx context.Context, tx *gorm.DB, item models.WorkOrderItem, wo models.WorkOrder, task models.QCTask, req dto.QCFinishRequest, performedBy string, now time.Time) error {
	flow := resolveProcessFlow(item)
	if len(flow) == 0 {
		return nil
	}
	lastProcess := strings.TrimSpace(flow[len(flow)-1].ProcessName)
	taskProcess := strings.TrimSpace(task.ProcessName)
	if taskProcess != "" && !strings.EqualFold(taskProcess, lastProcess) {
		return nil // QC proses tengah: barang masih di WIP, bukan FG
	}
	if !isUniqFinished(item.Status) {
		return nil // belum Scan Out proses terakhir: belum ada qty di FG
	}
	if strings.EqualFold(strings.TrimSpace(wo.WOKind), "rm_processing") {
		return nil // output WO rm_processing berupa raw material, bukan FG
	}

	var scanOut models.ProductionScanLog
	if err := tx.WithContext(ctx).
		Where("wo_item_id = ? AND scan_type = ? AND process_name = ?", item.ID, "SCAN_OUT", lastProcess).
		Order("id DESC").First(&scanOut).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	produced := scanOut.QtyOutput
	good := req.TotalProductionQty
	kanban := item.KanbanNumber
	var moves []fgMove

	if diff := produced - good; diff > 1e-6 {
		remaining := diff
		if take := minFloat(req.NGDefectQty, remaining); take > 0 {
			moves = append(moves, fgMove{fgMoveQCNGRework, -take, kanban,
				fmt.Sprintf("QC Round 3: NG %s dipindah ke WO Rework", fmtQty(take))})
			remaining -= take
		}
		if take := minFloat(req.TotalScrapInBox, remaining); take > 0 {
			moves = append(moves, fgMove{fgMoveQCScrap, -take, kanban,
				fmt.Sprintf("QC Round 3: scrap %s dipindah ke Scrap", fmtQty(take))})
			remaining -= take
		}
		if remaining > 1e-6 {
			moves = append(moves, fgMove{fgMoveQCAdjustment, -remaining, kanban,
				fmt.Sprintf("QC Round 3: good %s dari total produksi %s", fmtQty(good), fmtQty(produced))})
		}
	} else if diff < -1e-6 {
		moves = append(moves, fgMove{fgMoveQCAdjustment, -diff, kanban,
			fmt.Sprintf("QC Round 3: good %s melebihi total produksi %s", fmtQty(good), fmtQty(produced))})
	}

	if len(moves) == 0 {
		moves = append(moves, fgMove{fgMoveQCFinal, 0, kanban,
			fmt.Sprintf("QC Round 3 selesai: good %s, tanpa NG/scrap", fmtQty(good))})
	}
	// [assembly-stock] hasil WO ini masuk Inventory WIP: koreksi QC diterapkan ke WIP.
	if toWIP, err := s.finalOutputGoesToWIP(tx, item.ItemUniqCode, wo); err != nil {
		return err
	} else if toWIP {
		return s.applyWIPMoves(tx, item, wo, performedBy, moves)
	}
	return s.applyFGMoves(tx, item.ItemUniqCode, wo.WONumber, performedBy, now, moves)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func fmtQty(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}
