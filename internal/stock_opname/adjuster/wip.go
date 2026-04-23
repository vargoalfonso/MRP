package adjuster

import (
	"context"
	"fmt"
	"math"
	"time"

	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	wipModels "github.com/ganasa18/go-template/internal/wip/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WIPAdjuster struct{}

func NewWIPAdjuster() InventoryAdjuster { return &WIPAdjuster{} }

func (a *WIPAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row wipModels.WIPItem
	err := tx.WithContext(ctx).Where("uniq = ?", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("wip uniq %s not found", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve WIP uniq: " + err.Error())
	}
	qty := float64(row.Stock)
	uom := row.UOM
	return &UniqSnapshot{EntityID: &row.ID, PartName: &row.ProcessName, UOM: &uom, SystemQty: qty}, nil
}

func (a *WIPAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []wipModels.WIPItem
	query := tx.WithContext(ctx).Model(&wipModels.WIPItem{})
	if q != "" {
		query = query.Where("uniq ILIKE ? OR packing_number ILIKE ? OR process_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search WIP uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		uom := rows[i].UOM
		processName := rows[i].ProcessName
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].Uniq, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartName: &processName, UOM: &uom, SystemQty: float64(rows[i].Stock)}})
	}
	return items, nil
}

func (a *WIPAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row wipModels.WIPItem
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", entry.EntityID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("wip item not found during stock opname")
		}
		return nil, apperror.Internal("lock WIP row: " + err.Error())
	}

	after := int(math.Round(entry.CountedQty))
	delta := after - row.Stock
	now := time.Now()
	// WIP stock opname is audit-only in v1: keep current WIP stock untouched and only record the variance.
	log := &wipModels.WIPLog{WipItemID: row.ID, Action: "stock_opname", Qty: delta, CreatedAt: now}
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, apperror.Internal("append WIP log: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: float64(delta)}, nil
}
