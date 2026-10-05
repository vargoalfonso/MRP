package adjuster

import (
	"context"

	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
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
