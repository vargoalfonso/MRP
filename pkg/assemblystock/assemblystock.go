// Package assemblystock menyediakan logika stok komponen untuk WO Assembly:
//
//   - Components      : komponen langsung (1 level) sebuah parent dari BOM aktif.
//   - Plan            : cek ketersediaan (WIP -> raw material -> indirect RM) dan
//     hitung alokasi per sumber, dikurangi stok yang sudah di-lock WO lain.
//   - Lock / Consume / ReleaseByWO : siklus lock stok. Stok baru benar-benar
//     berkurang di Consume (Scan Out proses assembly).
//   - AdjustWIP       : tambah/kurangi saldo wip_inventories (hasil WO New child part).
//   - IsChildItem     : item muncul sebagai child di BOM aktif -> hasil produksi masuk WIP.
package assemblystock

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const (
	SourceWIP = "wip"
	SourceRM  = "raw_material"
	SourceIRM = "indirect_raw_material"

	eps = 1e-6
)

// sourceOrder: WIP dipakai lebih dulu, baru raw material, lalu indirect RM.
var sourceOrder = []string{SourceWIP, SourceRM, SourceIRM}

// Component adalah satu komponen langsung dari parent. QtyPer sudah termasuk scrap factor.
type Component struct {
	UniqCode string
	QtyPer   float64
}

// Request adalah kebutuhan produksi parent sebanyak Qty.
type Request struct {
	ParentUniq string
	Qty        float64
}

// Allocation adalah bagian kebutuhan yang dipenuhi dari satu sumber stok.
type Allocation struct {
	ParentUniq    string
	ComponentUniq string
	Source        string
	QtyPer        float64
	Qty           float64
}

// Shortage adalah komponen yang stok tersedianya tidak cukup.
type Shortage struct {
	ParentUniq    string
	ComponentUniq string
	Required      float64
	Available     float64
}

// NoBOMError dikembalikan bila parent tidak punya komponen di BOM aktif.
type NoBOMError struct{ ParentUniq string }

func (e *NoBOMError) Error() string {
	return fmt.Sprintf("item %s tidak memiliki BOM aktif dengan komponen, WO Assembly tidak dapat dibuat", e.ParentUniq)
}

// IsNoBOM melaporkan apakah err adalah *NoBOMError.
func IsNoBOM(err error) bool {
	var e *NoBOMError
	return errors.As(err, &e)
}

func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// Components mengembalikan komponen langsung (1 level) dari BOM aktif parent.
// Baris phantom dilewati (tidak punya stok). Komponen yang sama di beberapa baris dijumlahkan.
func Components(db *gorm.DB, parentUniq string) ([]Component, error) {
	type row struct {
		UniqCode string  `gorm:"column:uniq_code"`
		QtyPer   float64 `gorm:"column:qty_per"`
	}
	var rows []row
	err := db.Raw(`
		SELECT c.uniq_code AS uniq_code,
		       bl.qty_per_uniq * (1 + COALESCE(bl.scrap_factor, 0)) AS qty_per
		FROM items p
		JOIN bom_item bi  ON bi.item_id = p.id AND bi.is_current = TRUE AND bi.status IN ('Released','Active')
		JOIN bom_lines bl ON bl.bom_item_id = bi.id AND bl.parent_item_id = p.id
		JOIN items c      ON c.id = bl.child_item_id AND c.deleted_at IS NULL
		WHERE p.uniq_code = ? AND p.deleted_at IS NULL
		  AND COALESCE(bl.is_phantom, FALSE) = FALSE
		ORDER BY bl.line_no NULLS LAST, bl.id`, strings.TrimSpace(parentUniq)).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("assemblystock components %s: %w", parentUniq, err)
	}
	merged := map[string]float64{}
	order := make([]string, 0, len(rows))
	for _, r := range rows {
		if _, ok := merged[r.UniqCode]; !ok {
			order = append(order, r.UniqCode)
		}
		merged[r.UniqCode] += r.QtyPer
	}
	out := make([]Component, 0, len(order))
	for _, u := range order {
		out = append(out, Component{UniqCode: u, QtyPer: merged[u]})
	}
	return out, nil
}

// IsChildItem true bila uniq punya parent, yaitu menjadi child di BOM aktif manapun
// (parent masih aktif). Hasil WO (non-Assembly) dari item seperti ini masuk Inventory
// WIP walaupun seluruh prosesnya sudah selesai. Child part tanpa parent masuk FG.
func IsChildItem(db *gorm.DB, uniq string) (bool, error) {
	var n int64
	err := db.Raw(`
		SELECT COUNT(1)
		FROM bom_lines bl
		JOIN bom_item bi ON bi.id = bl.bom_item_id AND bi.is_current = TRUE AND bi.status IN ('Released','Active')
		JOIN items c     ON c.id = bl.child_item_id
		JOIN items p     ON p.id = bl.parent_item_id AND p.deleted_at IS NULL
		WHERE c.uniq_code = ? AND c.deleted_at IS NULL`, strings.TrimSpace(uniq)).Scan(&n).Error
	if err != nil {
		return false, fmt.Errorf("assemblystock is child %s: %w", uniq, err)
	}
	return n > 0, nil
}

func stockOf(db *gorm.DB, source, uniq string) (float64, error) {
	var q string
	switch source {
	case SourceWIP:
		q = `SELECT COALESCE(SUM(stock_qty), 0) FROM wip_inventories WHERE uniq_code = ?`
	case SourceRM:
		q = `SELECT COALESCE(SUM(stock_qty), 0) FROM raw_materials WHERE uniq_code = ? AND deleted_at IS NULL`
	case SourceIRM:
		q = `SELECT COALESCE(SUM(stock_qty), 0) FROM indirect_raw_materials WHERE uniq_code = ? AND deleted_at IS NULL`
	default:
		return 0, fmt.Errorf("assemblystock: sumber stok tidak dikenal %q", source)
	}
	var v float64
	if err := db.Raw(q, uniq).Scan(&v).Error; err != nil {
		return 0, fmt.Errorf("assemblystock stock %s/%s: %w", source, uniq, err)
	}
	return v, nil
}

func lockedOf(db *gorm.DB, source, uniq string) (float64, error) {
	var v float64
	err := db.Raw(`
		SELECT COALESCE(SUM(qty_locked - qty_consumed), 0)
		FROM assembly_stock_locks
		WHERE component_uniq_code = ? AND source = ? AND status = 'active'`, uniq, source).Scan(&v).Error
	if err != nil {
		return 0, fmt.Errorf("assemblystock locked %s/%s: %w", source, uniq, err)
	}
	return v, nil
}

// Available = stok - qty yang masih terkunci oleh WO Assembly aktif.
func Available(db *gorm.DB, source, uniq string) (float64, error) {
	stock, err := stockOf(db, source, uniq)
	if err != nil {
		return 0, err
	}
	locked, err := lockedOf(db, source, uniq)
	if err != nil {
		return 0, err
	}
	if v := stock - locked; v > 0 {
		return v, nil
	}
	return 0, nil
}

// Plan menghitung kebutuhan komponen untuk semua request dan mengalokasikannya
// ke sumber stok (WIP -> RM -> indirect RM). Request dengan parent sama digabung.
// Bila lock=true, db HARUS berupa transaksi: advisory lock per komponen dipasang
// sehingga dua WO Assembly yang dibuat bersamaan tidak memakai stok yang sama.
func Plan(db *gorm.DB, reqs []Request, lock bool) ([]Allocation, []Shortage, error) {
	// gabungkan request per parent (urutan kemunculan dipertahankan)
	qtyByParent := map[string]float64{}
	parents := make([]string, 0, len(reqs))
	for _, r := range reqs {
		p := strings.TrimSpace(r.ParentUniq)
		if p == "" || r.Qty <= 0 {
			continue
		}
		if _, ok := qtyByParent[p]; !ok {
			parents = append(parents, p)
		}
		qtyByParent[p] += r.Qty
	}

	used := map[string]float64{} // "source|component" -> qty yang sudah dialokasikan di plan ini
	locked := map[string]bool{}
	var allocs []Allocation
	var shortages []Shortage

	for _, parent := range parents {
		comps, err := Components(db, parent)
		if err != nil {
			return nil, nil, err
		}
		if len(comps) == 0 {
			return nil, nil, &NoBOMError{ParentUniq: parent}
		}
		for _, comp := range comps {
			if lock && !locked[comp.UniqCode] {
				if err := db.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "assembly_stock:"+comp.UniqCode).Error; err != nil {
					return nil, nil, fmt.Errorf("assemblystock advisory lock %s: %w", comp.UniqCode, err)
				}
				locked[comp.UniqCode] = true
			}
			need := round4(qtyByParent[parent] * comp.QtyPer)
			remaining := need
			for _, src := range sourceOrder {
				if remaining <= eps {
					break
				}
				avail, err := Available(db, src, comp.UniqCode)
				if err != nil {
					return nil, nil, err
				}
				key := src + "|" + comp.UniqCode
				avail -= used[key]
				if avail <= eps {
					continue
				}
				take := math.Min(remaining, avail)
				take = round4(take)
				if take <= 0 {
					continue
				}
				used[key] += take
				remaining = round4(remaining - take)
				allocs = append(allocs, Allocation{
					ParentUniq: parent, ComponentUniq: comp.UniqCode,
					Source: src, QtyPer: comp.QtyPer, Qty: take,
				})
			}
			if remaining > eps {
				shortages = append(shortages, Shortage{
					ParentUniq: parent, ComponentUniq: comp.UniqCode,
					Required: need, Available: round4(need - remaining),
				})
			}
		}
	}
	return allocs, shortages, nil
}

func fmtQty(v float64) string {
	if math.Abs(v-math.Round(v)) < eps {
		return fmt.Sprintf("%d", int64(math.Round(v)))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}

// FormatShortages menyusun pesan untuk user, mis.
// "Stok komponen belum cukup untuk WO Assembly. Buat WO New terlebih dahulu. [A] B: butuh 100, tersedia 60 (kurang 40)".
func FormatShortages(shortages []Shortage) string {
	if len(shortages) == 0 {
		return ""
	}
	sorted := make([]Shortage, len(shortages))
	copy(sorted, shortages)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ParentUniq != sorted[j].ParentUniq {
			return sorted[i].ParentUniq < sorted[j].ParentUniq
		}
		return sorted[i].ComponentUniq < sorted[j].ComponentUniq
	})
	parts := make([]string, 0, len(sorted))
	for _, s := range sorted {
		parts = append(parts, fmt.Sprintf("[%s] %s: butuh %s, tersedia %s (kurang %s)",
			s.ParentUniq, s.ComponentUniq, fmtQty(s.Required), fmtQty(s.Available), fmtQty(round4(s.Required-s.Available))))
	}
	return "Stok komponen (WIP / raw material) belum cukup untuk WO Assembly. Buat WO New terlebih dahulu untuk komponen yang kurang. " +
		strings.Join(parts, "; ")
}

// Lock menyimpan alokasi sebagai lock aktif untuk WO. Panggil di transaksi yang sama dengan Plan(lock=true).
func Lock(tx *gorm.DB, woID int64, woNumber string, allocs []Allocation) error {
	for _, a := range allocs {
		if a.Qty <= 0 {
			continue
		}
		if err := tx.Table("assembly_stock_locks").Create(map[string]interface{}{
			"wo_id":               woID,
			"wo_number":           woNumber,
			"parent_uniq_code":    a.ParentUniq,
			"component_uniq_code": a.ComponentUniq,
			"source":              a.Source,
			"qty_per_uniq":        a.QtyPer,
			"qty_locked":          a.Qty,
			"qty_consumed":        0,
			"status":              "active",
		}).Error; err != nil {
			return fmt.Errorf("assemblystock lock insert: %w", err)
		}
	}
	return nil
}

// ReleaseByWO melepas sisa lock aktif WO (WO ditolak / selesai). Stok yang sudah
// di-consume tidak dikembalikan.
func ReleaseByWO(tx *gorm.DB, woID int64) error {
	return tx.Exec(`UPDATE assembly_stock_locks SET status = 'released', updated_at = NOW()
		WHERE wo_id = ? AND status = 'active'`, woID).Error
}

// Consumed adalah hasil pengurangan stok satu komponen dari satu sumber.
type Consumed struct {
	Source        string
	ComponentUniq string
	EntityID      int64
	Qty           float64 // positif = jumlah yang dikurangi
	Before        float64
	After         float64
}

// Consume mengurangi stok komponen sebesar qty*qty_per_uniq untuk parent pada WO,
// dibatasi sisa lock. Dipanggil saat Scan Out proses assembly.
// Pencatatan movement log RM/IRM dilakukan pemanggil dari hasil return;
// movement WIP dicatat di sini (wip_inventory_movements).
func Consume(tx *gorm.DB, woID int64, woNumber, parentUniq string, qty float64, by string) ([]Consumed, error) {
	if qty <= 0 {
		return nil, nil
	}
	type lockRow struct {
		ID          int64   `gorm:"column:id"`
		Component   string  `gorm:"column:component_uniq_code"`
		Source      string  `gorm:"column:source"`
		QtyPer      float64 `gorm:"column:qty_per_uniq"`
		QtyLocked   float64 `gorm:"column:qty_locked"`
		QtyConsumed float64 `gorm:"column:qty_consumed"`
	}
	var rows []lockRow
	if err := tx.Raw(`
		SELECT id, component_uniq_code, source, qty_per_uniq, qty_locked, qty_consumed
		FROM assembly_stock_locks
		WHERE wo_id = ? AND parent_uniq_code = ? AND status = 'active'
		ORDER BY id
		FOR UPDATE`, woID, parentUniq).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("assemblystock consume select: %w", err)
	}

	needLeft := map[string]float64{}
	var out []Consumed
	for _, r := range rows {
		if _, ok := needLeft[r.Component]; !ok {
			needLeft[r.Component] = round4(qty * r.QtyPer)
		}
		remainingLock := round4(r.QtyLocked - r.QtyConsumed)
		take := round4(math.Min(needLeft[r.Component], remainingLock))
		if take <= 0 {
			continue
		}
		res, err := deductStock(tx, r.Source, r.Component, take, woNumber, by)
		if err != nil {
			return nil, err
		}
		needLeft[r.Component] = round4(needLeft[r.Component] - take)

		newConsumed := round4(r.QtyConsumed + take)
		status := "active"
		if r.QtyLocked-newConsumed <= eps {
			status = "consumed"
		}
		if err := tx.Exec(`UPDATE assembly_stock_locks
			SET qty_consumed = ?, status = ?, updated_at = NOW() WHERE id = ?`,
			newConsumed, status, r.ID).Error; err != nil {
			return nil, fmt.Errorf("assemblystock consume update: %w", err)
		}
		if res != nil {
			out = append(out, *res)
		}
	}
	return out, nil
}

// deductStock mengurangi stok satu sumber (baris dikunci FOR UPDATE).
func deductStock(tx *gorm.DB, source, uniq string, qty float64, woNumber, by string) (*Consumed, error) {
	var table string
	switch source {
	case SourceWIP:
		table = "wip_inventories"
	case SourceRM:
		table = "raw_materials"
	case SourceIRM:
		table = "indirect_raw_materials"
	default:
		return nil, fmt.Errorf("assemblystock: sumber stok tidak dikenal %q", source)
	}

	type stockRow struct {
		ID       int64   `gorm:"column:id"`
		StockQty float64 `gorm:"column:stock_qty"`
	}
	var row stockRow
	where := "uniq_code = ?"
	if source != SourceWIP {
		where += " AND deleted_at IS NULL"
	}
	if err := tx.Raw(fmt.Sprintf(`SELECT id, stock_qty FROM %s WHERE %s ORDER BY id LIMIT 1 FOR UPDATE`, table, where), uniq).
		Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("assemblystock deduct select %s/%s: %w", source, uniq, err)
	}
	if row.ID == 0 {
		// Baris stok sudah tidak ada (mis. dihapus/koreksi manual setelah di-lock).
		// Jangan blokir lantai produksi: lock tetap ditandai terpakai, stok tidak berubah.
		return nil, nil
	}

	change := qty
	if row.StockQty-change < 0 {
		change = row.StockQty
	}
	after := round4(row.StockQty - change)

	if source == SourceWIP {
		if err := tx.Exec(`UPDATE wip_inventories SET stock_qty = ?, updated_at = NOW() WHERE id = ?`, after, row.ID).Error; err != nil {
			return nil, fmt.Errorf("assemblystock deduct update wip: %w", err)
		}
		if err := insertWIPMovement(tx, row.ID, uniq, "assembly_consume", -change, row.StockQty, after, woNumber, "",
			"Dipakai sebagai komponen WO Assembly", by); err != nil {
			return nil, err
		}
	} else {
		if err := tx.Exec(fmt.Sprintf(`UPDATE %s SET stock_qty = ?, updated_by = ?, updated_at = NOW() WHERE id = ?`, table),
			after, by, row.ID).Error; err != nil {
			return nil, fmt.Errorf("assemblystock deduct update %s: %w", table, err)
		}
	}
	return &Consumed{Source: source, ComponentUniq: uniq, EntityID: row.ID, Qty: change, Before: row.StockQty, After: after}, nil
}

func nullable(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullableID(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func insertWIPMovement(tx *gorm.DB, wipID int64, uniq, mvType string, change, before, after float64, woNumber, kanban, notes, by string) error {
	if err := tx.Table("wip_inventory_movements").Create(map[string]interface{}{
		"wip_inventory_id": wipID,
		"uniq_code":        uniq,
		"movement_type":    mvType,
		"qty_change":       change,
		"qty_before":       before,
		"qty_after":        after,
		"wo_number":        nullable(woNumber),
		"kanban_number":    nullable(kanban),
		"notes":            nullable(notes),
		"created_by":       nullable(by),
	}).Error; err != nil {
		return fmt.Errorf("assemblystock wip movement: %w", err)
	}
	return nil
}

// WIPPost adalah perubahan saldo wip_inventories.
type WIPPost struct {
	UniqCode     string
	ItemID       int64
	PartNumber   string
	PartName     string
	Model        string
	UOM          string
	WONumber     string
	KanbanNumber string
	MovementType string // mis. production_in | qc_adjustment
	Notes        string
	By           string
	Delta        float64 // + masuk, - keluar
}

// AdjustWIP menambah/mengurangi saldo stok WIP satu UNIQ (baris dikunci). Saldo tidak pernah < 0.
// Mengembalikan id baris, saldo sebelum, dan sesudah.
func AdjustWIP(tx *gorm.DB, p WIPPost) (int64, float64, float64, error) {
	type stockRow struct {
		ID       int64   `gorm:"column:id"`
		StockQty float64 `gorm:"column:stock_qty"`
	}
	var row stockRow
	if err := tx.Raw(`SELECT id, stock_qty FROM wip_inventories WHERE uniq_code = ? FOR UPDATE`, p.UniqCode).Scan(&row).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("assemblystock wip select: %w", err)
	}

	if row.ID == 0 {
		if p.Delta <= 0 {
			return 0, 0, 0, nil // belum ada stok WIP: tidak ada yang dikurangi
		}
		after := round4(p.Delta)
		if err := tx.Raw(`
			INSERT INTO wip_inventories (uniq_code, item_id, part_number, part_name, model, uom, stock_qty, wo_number)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
			p.UniqCode, nullableID(p.ItemID), nullable(p.PartNumber), nullable(p.PartName), nullable(p.Model), nullable(p.UOM),
			after, nullable(p.WONumber)).Scan(&row.ID).Error; err != nil {
			return 0, 0, 0, fmt.Errorf("assemblystock wip insert: %w", err)
		}
		if err := insertWIPMovement(tx, row.ID, p.UniqCode, p.MovementType, after, 0, after, p.WONumber, p.KanbanNumber, p.Notes, p.By); err != nil {
			return 0, 0, 0, err
		}
		return row.ID, 0, after, nil
	}

	change := p.Delta
	if row.StockQty+change < 0 {
		change = -row.StockQty
	}
	after := round4(row.StockQty + change)
	if err := tx.Exec(`UPDATE wip_inventories SET stock_qty = ?, wo_number = COALESCE(?, wo_number), updated_at = NOW() WHERE id = ?`,
		after, nullable(p.WONumber), row.ID).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("assemblystock wip update: %w", err)
	}
	if err := insertWIPMovement(tx, row.ID, p.UniqCode, p.MovementType, change, row.StockQty, after, p.WONumber, p.KanbanNumber, p.Notes, p.By); err != nil {
		return 0, 0, 0, err
	}
	return row.ID, row.StockQty, after, nil
}
