package adjuster

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	bomModels "github.com/ganasa18/go-template/internal/billmaterial/models"
	fgModels "github.com/ganasa18/go-template/internal/finished_goods/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/inventoryconst"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FGAdjuster struct{}

func NewFGAdjuster() InventoryAdjuster { return &FGAdjuster{} }

// ResolveUniq tidak lagi menolak uniq yang belum ada di finished_goods.
// Urutan: (1) pakai baris finished_goods yang aktif, (2) kalau belum ada,
// ambil data part dari BOM item (kalau ada) dengan SystemQty 0. Baris
// finished_goods baru dibuat saat entry di-approve (lihat ApplyAdjustment).
func (a *FGAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	var row fgModels.FinishedGoods
	err := tx.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).Take(&row).Error
	if err == nil {
		return &UniqSnapshot{EntityID: &row.ID, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQty: row.StockQty}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, apperror.Internal("resolve FG uniq: " + err.Error())
	}

	// FG belum ada -> data baru. Lengkapi info part dari BOM bila tersedia.
	snap := &UniqSnapshot{SystemQty: 0}
	var item bomModels.Item
	ierr := tx.WithContext(ctx).
		Where("LOWER(TRIM(uniq_code)) = LOWER(TRIM(?)) AND deleted_at IS NULL", uniqCode).
		Take(&item).Error
	if ierr == nil {
		snap.PartNumber = item.PartNumber
		snap.PartName = &item.PartName
		snap.UOM = &item.Uom
	} else if ierr != gorm.ErrRecordNotFound {
		return nil, apperror.Internal("resolve FG BOM item: " + ierr.Error())
	}
	return snap, nil
}

func (a *FGAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []fgModels.FinishedGoods
	query := tx.WithContext(ctx).Where("deleted_at IS NULL")
	if q != "" {
		query = query.Where("uniq_code ILIKE ? OR part_number ILIKE ? OR part_name ILIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, apperror.Internal("search FG uniq options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{EntityID: &rows[i].ID, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].StockQty}})
	}
	return items, nil
}

func (a *FGAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row fgModels.FinishedGoods
	if err := lockInventoryRow(ctx, tx, &row, entry, ""); err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, apperror.Internal("lock FG row: " + err.Error())
		}
		// Belum ada di finished_goods -> buat data baru (atau pulihkan yang
		// pernah di-soft-delete) lalu lanjut sebagai penyesuaian dari qty 0.
		created, cerr := a.createOrRestoreFG(ctx, tx, entry, actor)
		if cerr != nil {
			return nil, cerr
		}
		row = *created
	}

	before := row.StockQty
	after := entry.CountedQty
	delta := after - before
	status := fgStatus(after, row.MinThreshold, row.MaxThreshold)
	kanbanCount := fgKanbanCount(after, row.KanbanStandardQty)
	stockToComplete := fgStockToComplete(after, row.SafetyStockQty)
	now := time.Now()

	if err := tx.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"stock_qty":                after,
		"status":                   status,
		"kanban_count":             kanbanCount,
		"stock_to_complete_kanban": stockToComplete,
		"updated_by":               actor,
		"updated_at":               now,
	}).Error; err != nil {
		return nil, apperror.Internal("update FG stock opname: " + err.Error())
	}

	sourceFlag := string(inventoryconst.SourceStockOpname)
	notes := fmt.Sprintf("stock opname %s", sessionNumber)
	log := &fgModels.FGMovementLog{
		FgID:         row.ID,
		UniqCode:     row.UniqCode,
		MovementType: string(inventoryconst.MovementStockOpname),
		QtyChange:    delta,
		QtyBefore:    before,
		QtyAfter:     after,
		SourceFlag:   &sourceFlag,
		ReferenceID:  &sessionNumber,
		Notes:        &notes,
		LoggedBy:     &actor,
		LoggedAt:     now,
	}
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return nil, apperror.Internal("append FG movement log: " + err.Error())
	}
	return &AdjustmentResult{QtyChange: delta, EntityID: &row.ID}, nil
}

// createOrRestoreFG membuat baris finished_goods baru untuk uniq yang belum
// ada. Kalau ada baris soft-deleted dengan uniq_code sama (uniq_code unik),
// baris itu dipulihkan dengan stok 0 supaya tidak bentrok unique.
func (a *FGAdjuster) createOrRestoreFG(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, actor string) (*fgModels.FinishedGoods, error) {
	uniq := strings.TrimSpace(entry.UniqCode)
	now := time.Now()

	var existing fgModels.FinishedGoods
	err := tx.WithContext(ctx).Unscoped().Where("uniq_code = ?", uniq).Take(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"deleted_at": nil,
			"stock_qty":  0,
			"updated_by": actor,
			"updated_at": now,
		}
		if existing.PartNumber == nil && entry.PartNumber != nil {
			updates["part_number"] = *entry.PartNumber
		}
		if existing.PartName == nil && entry.PartName != nil {
			updates["part_name"] = *entry.PartName
		}
		if existing.UOM == nil && entry.UOM != nil {
			updates["uom"] = *entry.UOM
		}
		if uerr := tx.WithContext(ctx).Unscoped().Model(&fgModels.FinishedGoods{}).
			Where("id = ?", existing.ID).Updates(updates).Error; uerr != nil {
			return nil, apperror.Internal("restore FG stock opname: " + uerr.Error())
		}
		existing.DeletedAt = nil
		existing.StockQty = 0
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, apperror.Internal("lookup FG (unscoped): " + err.Error())
	}

	fg := &fgModels.FinishedGoods{
		UUID:       uuid.New(),
		UniqCode:   uniq,
		PartNumber: entry.PartNumber,
		PartName:   entry.PartName,
		UOM:        entry.UOM,
		StockQty:   0,
		Status:     fgModels.FGStatusNormal,
		CreatedBy:  &actor,
		UpdatedBy:  &actor,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if cerr := tx.WithContext(ctx).Create(fg).Error; cerr != nil {
		return nil, apperror.Internal("create FG stock opname: " + cerr.Error())
	}
	return fg, nil
}

func fgStatus(stockQty float64, minThreshold, maxThreshold *float64) string {
	if minThreshold != nil && stockQty < *minThreshold {
		return fgModels.FGStatusLowStock
	}
	if maxThreshold != nil && stockQty > *maxThreshold {
		return fgModels.FGStatusOverstock
	}
	return fgModels.FGStatusNormal
}

func fgKanbanCount(stockQty float64, std *int) *int {
	if std == nil || *std <= 0 {
		return nil
	}
	v := int(math.Floor(stockQty / float64(*std)))
	return &v
}

func fgStockToComplete(stockQty float64, safety *float64) *float64 {
	if safety == nil {
		return nil
	}
	v := math.Max(0, *safety-stockQty)
	return &v
}
