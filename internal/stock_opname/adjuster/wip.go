package adjuster

import (
	"context"
	"fmt"
	"math"
	"time"

	bomModels "github.com/ganasa18/go-template/internal/billmaterial/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	wipModels "github.com/ganasa18/go-template/internal/wip/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WIPAdjuster struct{}

func NewWIPAdjuster() InventoryAdjuster { return &WIPAdjuster{} }

// ResolveUniq is used by manual stock opname and keeps the original WIP
// behavior: manual input resolves against a live wip_items row.
func (a *WIPAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row wipModels.WIPItem
	err := tx.WithContext(ctx).Where("uniq = ?", uniqCode).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("wip uniq %s tidak ditemukan", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve WIP uniq: " + err.Error())
	}
	qty := float64(row.Stock)
	uom := row.UOM
	return &UniqSnapshot{EntityID: &row.ID, PartName: &row.ProcessName, UOM: &uom, SystemQty: qty}, nil
}

// SearchUniqs is used by manual stock opname and searches live WIP rows.
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

// ResolveWIPBOMUniq is used only by bulk stock opname. Bulk Excel rows are
// allowed to reference active BOM items even when no wip_items row exists.
func ResolveWIPBOMUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row bomModels.Item
	err := tx.WithContext(ctx).
		Where("LOWER(TRIM(uniq_code)) = LOWER(TRIM(?))", uniqCode).
		Where("deleted_at IS NULL").
		Where("LOWER(TRIM(COALESCE(status, ''))) = 'active'").
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("BOM item uniq %s tidak ditemukan atau belum Active", uniqCode))
	}
	if err != nil {
		return nil, apperror.Internal("resolve bulk WIP BOM item: " + err.Error())
	}
	return &UniqSnapshot{
		EntityID:   &row.ID,
		PartNumber: row.PartNumber,
		PartName:   &row.PartName,
		UOM:        &row.Uom,
		SystemQty:  0,
	}, nil
}

// SearchWIPBOMUniqs supplies the bulk-upload lookup/autofill list from active
// BOM items rather than from wip_items.
func SearchWIPBOMUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []bomModels.Item
	query := tx.WithContext(ctx).
		Model(&bomModels.Item{}).
		Where("deleted_at IS NULL").
		Where("LOWER(TRIM(COALESCE(status, ''))) = 'active'")
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("uniq_code ILIKE ? OR COALESCE(part_number, '') ILIKE ? OR part_name ILIKE ?", like, like, like)
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search bulk WIP BOM item options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{
			UniqCode: rows[i].UniqCode,
			UniqSnapshot: UniqSnapshot{
				EntityID:   &rows[i].ID,
				PartNumber: rows[i].PartNumber,
				PartName:   &rows[i].PartName,
				UOM:        &rows[i].Uom,
				SystemQty:  0,
			},
		})
	}
	return items, nil
}

func (a *WIPAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row wipModels.WIPItem
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", entry.EntityID).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("wip item tidak ditemukan during stock opname")
		}
		return nil, apperror.Internal("lock WIP row: " + err.Error())
	}

	after := int(math.Round(entry.CountedQty))
	delta := after - row.Stock
	now := time.Now()
	// Manual WIP stock opname is audit-only: keep current WIP stock untouched and record the variance.
	log := &wipModels.WIPLog{WipItemID: row.ID, Action: "stock_opname", Qty: delta, CreatedAt: now}
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, apperror.Internal("append WIP log: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: float64(delta)}, nil
}

// ApplyWIPBOMAdjustment is used only for bulk stock opname. It validates the
// active BOM item reference and keeps the operation audit-only because the
// bulk WIP workflow has no wip_items stock row to update.
func ApplyWIPBOMAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry) (*AdjustmentResult, error) {
	var item bomModels.Item
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", entry.EntityID).
		Take(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("BOM item tidak ditemukan during bulk WIP stock opname")
		}
		return nil, apperror.Internal("lock bulk WIP BOM item: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: entry.CountedQty}, nil
}
