package adjuster

import (
	"context"
	"strings"
	"time"

	bomModels "github.com/ganasa18/go-template/internal/billmaterial/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UniqSnapshot struct {
	EntityID   *int64
	PartNumber *string
	PartName   *string
	UOM        *string
	SystemQty  float64
	WeightKg   *float64
	// RawMaterialType is only populated for Raw Materials (e.g. "wire",
	// "sheet_plate", "ssp", "others"); empty for other inventory types.
	RawMaterialType string
}

type InventoryAdjuster interface {
	ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error)
	SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error)
	ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error)
}

type AdjustmentResult struct {
	QtyChange    float64
	WeightChange *float64
	// EntityID is the inventory row that was actually adjusted. It can differ
	// from entry.EntityID when the row had to be re-resolved by uniq code.
	EntityID *int64
}

// lockInventoryRow locks the inventory row that a stock opname entry must
// adjust at approval time. The uniq code parsed from the stock opname entry is
// the source of truth: entry.EntityID (snapshot taken when the entry was
// created) is used only as a hint to pick the exact row when it still belongs
// to that uniq. If the snapshot is missing, stale or points at another uniq,
// the row is re-resolved by uniq code (case/space-insensitive).
//
// extraUniqWhere is applied only to the by-uniq fallback (e.g. subcon limits
// opname to rows still "in vendor").
func lockInventoryRow(ctx context.Context, tx *gorm.DB, dest interface{}, entry *stockModels.StockOpnameEntry, extraUniqWhere string) error {
	base := func() *gorm.DB {
		return tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("deleted_at IS NULL")
	}
	if entry.EntityID != nil {
		err := base().
			Where("id = ? AND UPPER(BTRIM(uniq_code)) = UPPER(BTRIM(?))", *entry.EntityID, entry.UniqCode).
			Take(dest).Error
		if err == nil {
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	q := base().Where("UPPER(BTRIM(uniq_code)) = UPPER(BTRIM(?))", entry.UniqCode)
	if extraUniqWhere != "" {
		q = q.Where(extraUniqWhere)
	}
	return q.Order("id").Take(dest).Error
}

type UniqSnapshotResult struct {
	UniqCode string
	UniqSnapshot
}

// findBOMItem mencari item BOM (tidak soft-deleted) berdasarkan uniq code.
// Mengembalikan nil tanpa error bila tidak ada.
func findBOMItem(ctx context.Context, tx *gorm.DB, uniqCode string) (*bomModels.Item, error) {
	var item bomModels.Item
	err := tx.WithContext(ctx).
		Where("LOWER(TRIM(uniq_code)) = LOWER(TRIM(?)) AND deleted_at IS NULL", uniqCode).
		Order("id").Take(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, apperror.Internal("resolve BOM item: " + err.Error())
	}
	return &item, nil
}

// newUniqSnapshot dipakai saat uniq belum ada di inventory: tidak ada
// validasi not-found, SystemQty 0, info part diambil dari BOM bila ada.
func newUniqSnapshot(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	snap := &UniqSnapshot{SystemQty: 0}
	item, err := findBOMItem(ctx, tx, strings.TrimSpace(uniqCode))
	if err != nil {
		return nil, err
	}
	if item != nil {
		snap.PartNumber = item.PartNumber
		snap.PartName = &item.PartName
		snap.UOM = &item.Uom
	}
	return snap, nil
}

// liveOrRevive memastikan ada baris aktif untuk uniq. Urutan:
//  1. baris aktif (tidak soft-deleted) -> dipakai apa adanya
//  2. baris soft-deleted -> dipulihkan dengan stok 0 (uniq_code unik)
//
// Mengembalikan found=false bila belum pernah ada, sehingga pemanggil harus
// membuat baris baru. dest harus pointer ke model inventory.
func liveOrRevive(ctx context.Context, tx *gorm.DB, dest interface{}, uniqCode, stockCol, actor string) (bool, error) {
	uniq := strings.TrimSpace(uniqCode)
	match := "UPPER(BTRIM(uniq_code)) = UPPER(BTRIM(?))"

	err := tx.WithContext(ctx).Where(match+" AND deleted_at IS NULL", uniq).Order("id").Take(dest).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, apperror.Internal("lookup inventory row: " + err.Error())
	}

	err = tx.WithContext(ctx).Unscoped().Where(match+" AND deleted_at IS NOT NULL", uniq).Order("id").Take(dest).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, apperror.Internal("lookup soft-deleted inventory row: " + err.Error())
	}
	if err := tx.WithContext(ctx).Unscoped().Model(dest).Updates(map[string]interface{}{
		"deleted_at": nil,
		stockCol:     0,
		"updated_by": actor,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return false, apperror.Internal("restore inventory row: " + err.Error())
	}
	if err := tx.WithContext(ctx).Where(match+" AND deleted_at IS NULL", uniq).Order("id").Take(dest).Error; err != nil {
		return false, apperror.Internal("reload restored inventory row: " + err.Error())
	}
	return true, nil
}
