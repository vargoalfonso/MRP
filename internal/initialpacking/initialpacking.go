// Package initialpacking membuat "Initial Packing" untuk raw material yang
// punya stok tetapi belum punya Packing ID (mis. opening stock yang di-inject
// langsung ke raw_materials.stock_qty tanpa melalui Delivery Note supplier).
//
// Packing disimpan sebagai delivery_note_items pada satu DN sintetis
// ("OPENING-STOCK", delivery_notes.is_opening = TRUE), karena seluruh alur scan
// produksi (lookup packing, deduct qty, repacking, stock opname) membaca
// delivery_note_items.
//
// PENTING: paket ini TIDAK PERNAH mengubah raw_materials.stock_qty. Ia hanya
// membuat baris packing. Stok master sudah dihitung saat inject, jadi menaikkan
// stock_qty lagi akan menghitung stok dua kali.
package initialpacking

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	// OpeningDNNumber adalah nomor DN sintetis penampung semua Initial Packing.
	OpeningDNNumber = "OPENING-STOCK"
	// PackingPrefix adalah awalan nomor packing: OPN-<UNIQ>-001.
	PackingPrefix = "OPN"

	maxPackingNumberLen = 64 // delivery_note_items.packing_number varchar(64)
	epsilon             = 1e-6
)

// Result ringkasan pembuatan packing untuk satu raw material.
type Result struct {
	UniqCode       string   `json:"uniq_code"`
	Created        int      `json:"created"`
	TotalQty       float64  `json:"total_qty"`
	PackingNumbers []string `json:"packing_numbers"`
	Skipped        string   `json:"skipped,omitempty"` // alasan bila tidak ada yang dibuat
}

type Generator struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Generator { return &Generator{db: db} }

type rmRow struct {
	UniqCode          string
	StockQty          float64
	UOM               *string
	KanbanStandardQty *int
}

// GenerateForQty membuat packing untuk qty tertentu (dipakai saat inject
// opening stock: qty = stok yang baru di-inject). Aman dipanggil di luar
// transaksi pemanggil; transaksi dibuka sendiri.
func (g *Generator) GenerateForQty(ctx context.Context, uniqCode string, qty float64, createdBy string) (*Result, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	if uniqCode == "" {
		return nil, errors.New("uniq_code wajib diisi")
	}
	if qty <= epsilon {
		return &Result{UniqCode: uniqCode, Skipped: "qty <= 0"}, nil
	}

	var res *Result
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lock(tx, uniqCode); err != nil {
			return err
		}
		r, err := generate(tx, uniqCode, qty, createdBy)
		res = r
		return err
	})
	return res, err
}

// EnsureForStock membuat packing dari seluruh stock_qty HANYA bila raw material
// punya stok > 0 dan belum punya packing sama sekali. Idempotent: pemanggilan
// berulang/bersamaan tidak membuat packing ganda (dikunci per uniq_code).
func (g *Generator) EnsureForStock(ctx context.Context, uniqCode string, createdBy string) (*Result, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	if uniqCode == "" {
		return nil, errors.New("uniq_code wajib diisi")
	}

	var res *Result
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lock(tx, uniqCode); err != nil {
			return err
		}

		var rm rmRow
		if err := tx.Raw(`
			SELECT uniq_code, stock_qty, uom, kanban_standard_qty
			FROM raw_materials
			WHERE uniq_code = ? AND deleted_at IS NULL
			LIMIT 1`, uniqCode).Scan(&rm).Error; err != nil {
			return fmt.Errorf("baca raw material: %w", err)
		}
		if rm.UniqCode == "" {
			res = &Result{UniqCode: uniqCode, Skipped: "raw material tidak ditemukan"}
			return nil
		}
		if rm.StockQty <= epsilon {
			res = &Result{UniqCode: uniqCode, Skipped: "stok 0"}
			return nil
		}

		has, err := hasAnyPacking(tx, uniqCode)
		if err != nil {
			return err
		}
		if has {
			res = &Result{UniqCode: uniqCode, Skipped: "sudah punya packing"}
			return nil
		}

		r, err := generate(tx, uniqCode, rm.StockQty, createdBy)
		res = r
		return err
	})
	return res, err
}

// Backfill menjalankan EnsureForStock untuk semua raw material bertock > 0 yang
// belum punya packing. Dipakai sekali untuk opening stock yang sudah terlanjur
// di-inject sebelum fitur ini ada.
func (g *Generator) Backfill(ctx context.Context, createdBy string) ([]Result, error) {
	var codes []string
	if err := g.db.WithContext(ctx).Raw(`
		SELECT rm.uniq_code
		FROM raw_materials rm
		WHERE rm.deleted_at IS NULL
		  AND rm.stock_qty > 0
		  AND NOT EXISTS (
		    SELECT 1 FROM delivery_note_items dni
		    WHERE dni.item_uniq_code = rm.uniq_code
		      AND COALESCE(dni.packing_number, '') <> '')
		  AND NOT EXISTS (
		    SELECT 1 FROM work_order_items woi
		    WHERE woi.item_uniq_code = rm.uniq_code
		      AND COALESCE(woi.kanban_number, '') <> '')
		ORDER BY rm.uniq_code`).Scan(&codes).Error; err != nil {
		return nil, fmt.Errorf("daftar RM tanpa packing: %w", err)
	}

	out := make([]Result, 0, len(codes))
	for _, c := range codes {
		r, err := g.EnsureForStock(ctx, c, createdBy)
		if err != nil {
			out = append(out, Result{UniqCode: c, Skipped: "error: " + err.Error()})
			continue
		}
		if r != nil {
			out = append(out, *r)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------

func lock(tx *gorm.DB, uniqCode string) error {
	// Kunci transaksi per uniq_code supaya dua scan bersamaan tidak membuat
	// packing ganda.
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "initial_packing:"+uniqCode).Error
}

func hasAnyPacking(tx *gorm.DB, uniqCode string) (bool, error) {
	var n int64
	if err := tx.Raw(`
		SELECT (
		  (SELECT COUNT(*) FROM delivery_note_items
		   WHERE item_uniq_code = ? AND COALESCE(packing_number, '') <> '')
		+ (SELECT COUNT(*) FROM work_order_items
		   WHERE item_uniq_code = ? AND COALESCE(kanban_number, '') <> '')
		)`, uniqCode, uniqCode).Scan(&n).Error; err != nil {
		return false, fmt.Errorf("cek packing: %w", err)
	}
	return n > 0, nil
}

func ensureOpeningDN(tx *gorm.DB, createdBy string) (int64, error) {
	if err := tx.Exec(`
		INSERT INTO delivery_notes (dn_number, po_number, type, status, incoming_date, created_by, is_opening)
		VALUES (?, ?, 'opening', 'completed', CURRENT_DATE, ?, TRUE)
		ON CONFLICT (dn_number) DO NOTHING`,
		OpeningDNNumber, OpeningDNNumber, createdBy).Error; err != nil {
		return 0, fmt.Errorf("buat DN opening: %w", err)
	}
	var id int64
	if err := tx.Raw(`SELECT id FROM delivery_notes WHERE dn_number = ? LIMIT 1`, OpeningDNNumber).
		Scan(&id).Error; err != nil {
		return 0, fmt.Errorf("baca DN opening: %w", err)
	}
	if id == 0 {
		return 0, errors.New("DN opening tidak ditemukan")
	}
	return id, nil
}

// packSize: kanban_standard_qty RM -> kanban_parameters.kanban_qty -> 0 (satu packing).
func packSize(tx *gorm.DB, uniqCode string) (float64, *string, error) {
	var rm rmRow
	if err := tx.Raw(`
		SELECT uniq_code, stock_qty, uom, kanban_standard_qty
		FROM raw_materials WHERE uniq_code = ? AND deleted_at IS NULL LIMIT 1`, uniqCode).
		Scan(&rm).Error; err != nil {
		return 0, nil, fmt.Errorf("baca raw material: %w", err)
	}
	if rm.KanbanStandardQty != nil && *rm.KanbanStandardQty > 0 {
		return float64(*rm.KanbanStandardQty), rm.UOM, nil
	}
	var kq int
	if err := tx.Raw(`
		SELECT COALESCE(kanban_qty, 0) FROM kanban_parameters
		WHERE item_uniq_code = ? ORDER BY id DESC LIMIT 1`, uniqCode).Scan(&kq).Error; err == nil && kq > 0 {
		return float64(kq), rm.UOM, nil
	}
	return 0, rm.UOM, nil
}

func generate(tx *gorm.DB, uniqCode string, qty float64, createdBy string) (*Result, error) {
	dnID, err := ensureOpeningDN(tx, createdBy)
	if err != nil {
		return nil, err
	}
	size, uom, err := packSize(tx, uniqCode)
	if err != nil {
		return nil, err
	}
	if size <= 0 || size > qty {
		size = qty // tidak ada standar kanban: satu packing berisi seluruh qty
	}

	// Lanjutkan penomoran dari packing opening yang sudah ada untuk uniq ini.
	var existing int64
	if err := tx.Raw(`SELECT COUNT(*) FROM delivery_note_items
		WHERE dn_id = ? AND item_uniq_code = ?`, dnID, uniqCode).Scan(&existing).Error; err != nil {
		return nil, fmt.Errorf("hitung packing opening: %w", err)
	}

	res := &Result{UniqCode: uniqCode, PackingNumbers: []string{}}
	now := time.Now()
	remaining := qty
	seq := int(existing)
	for remaining > epsilon {
		seq++
		q := math.Min(size, remaining)
		q = math.Round(q*10000) / 10000
		remaining = math.Round((remaining-q)*10000) / 10000

		packing := fmt.Sprintf("%s-%s-%03d", PackingPrefix, uniqCode, seq)
		if len(packing) > maxPackingNumberLen {
			return nil, fmt.Errorf("nomor packing %q melebihi %d karakter", packing, maxPackingNumberLen)
		}

		qInt := int(math.Ceil(q - epsilon))
		var opname interface{} // NULL bila qty bulat (belum pernah diopname)
		if math.Abs(q-float64(qInt)) > epsilon {
			opname = q
		}
		pcsPerKanban := int(math.Ceil(size - epsilon))

		if err := tx.Exec(`
			INSERT INTO delivery_note_items
			  (dn_id, item_uniq_code, quantity, uom, qr, order_qty, date_incoming,
			   qty_stated, qty_received, quality_status, pcs_per_kanban,
			   received_at, packing_number, qty_opname)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_DATE, ?, ?, 'Approved', ?, ?, ?, ?)`,
			dnID, uniqCode, qInt, uom, packing, qInt,
			qInt, qInt, pcsPerKanban, now, packing, opname).Error; err != nil {
			return nil, fmt.Errorf("buat packing %s: %w", packing, err)
		}
		res.Created++
		res.TotalQty += q
		res.PackingNumbers = append(res.PackingNumbers, packing)
	}
	res.TotalQty = math.Round(res.TotalQty*10000) / 10000
	return res, nil
}
