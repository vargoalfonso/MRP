package adjuster

import (
	"context"

	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"gorm.io/gorm"
)

type UniqSnapshot struct {
	EntityID   *int64
	PartNumber *string
	PartName   *string
	UOM        *string
	SystemQty  float64
	WeightKg   *float64
}

type InventoryAdjuster interface {
	ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error)
	SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error)
	ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error)
}

type AdjustmentResult struct {
	QtyChange    float64
	WeightChange *float64
}

type UniqSnapshotResult struct {
	UniqCode string
	UniqSnapshot
}
