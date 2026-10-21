package adjuster

import (
	"context"
	"strings"

	bomModels "github.com/ganasa18/go-template/internal/billmaterial/models"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/assemblystock"
	"gorm.io/gorm"
)

type WIPAdjuster struct{}

func NewWIPAdjuster() InventoryAdjuster { return &WIPAdjuster{} }

// wipInventoryRow adalah baris Inventory WIP (wip_inventories): stok per UNIQ
// hasil WO New child part, dipakai WO Assembly sebagai komponen.
type wipInventoryRow struct {
	ID         int64   `gorm:"column:id"`
	UniqCode   string  `gorm:"column:uniq_code"`
	PartNumber *string `gorm:"column:part_number"`
	PartName   *string `gorm:"column:part_name"`
	UOM        *string `gorm:"column:uom"`
	StockQty   float64 `gorm:"column:stock_qty"`
}

func wipStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// ResolveUniq: stok sistem diambil dari Inventory WIP. UNIQ yang belum ada di
// inventory TIDAK divalidasi: SystemQty 0 dan info part diambil dari BOM bila ada.
func (a *WIPAdjuster) ResolveUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row wipInventoryRow
	if err := tx.WithContext(ctx).Raw(`
		SELECT id, uniq_code, part_number, part_name, uom, stock_qty
		FROM wip_inventories
		WHERE UPPER(BTRIM(uniq_code)) = UPPER(BTRIM(?))
		ORDER BY id LIMIT 1`, uniqCode).Scan(&row).Error; err != nil {
		return nil, apperror.Internal("resolve WIP uniq: " + err.Error())
	}
	if row.ID == 0 {
		return newUniqSnapshot(ctx, tx, uniqCode)
	}
	id := row.ID
	return &UniqSnapshot{EntityID: &id, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQty: row.StockQty}, nil
}

// SearchWIPInventoryUniqs mencari baris yang benar-benar ada di Inventory WIP.
func SearchWIPInventoryUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	like := "%" + strings.TrimSpace(q) + "%"
	var rows []wipInventoryRow
	if err := tx.WithContext(ctx).Raw(`
		SELECT id, uniq_code, part_number, part_name, uom, stock_qty
		FROM wip_inventories
		WHERE (? = '%%' OR uniq_code ILIKE ? OR COALESCE(part_name,'') ILIKE ? OR COALESCE(part_number,'') ILIKE ?)
		ORDER BY updated_at DESC LIMIT ?`, like, like, like, like, limit).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal("search WIP inventory options: " + err.Error())
	}
	items := make([]UniqSnapshotResult, 0, len(rows))
	for i := range rows {
		id := rows[i].ID
		items = append(items, UniqSnapshotResult{UniqCode: rows[i].UniqCode, UniqSnapshot: UniqSnapshot{
			EntityID: &id, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].StockQty,
		}})
	}
	return items, nil
}

// SearchUniqs (stock opname manual): UNIQ yang sudah ada di Inventory WIP lebih dulu,
// lalu semua item BOM aktif, supaya UNIQ yang belum ada di inventory tetap bisa dipilih
// dan akan "muncul" di Inventory WIP setelah opname disetujui.
func (a *WIPAdjuster) SearchUniqs(ctx context.Context, tx *gorm.DB, q string, limit int) ([]UniqSnapshotResult, error) {
	live, err := SearchWIPInventoryUniqs(ctx, tx, q, limit)
	if err != nil {
		return nil, err
	}
	return mergeUniqWithBOM(ctx, tx, q, limit, live)
}

// ResolveWIPBOMUniq is used only by bulk stock opname. Bulk Excel rows are
// allowed to reference active BOM items even when no wip_items row exists.
func ResolveWIPBOMUniq(ctx context.Context, tx *gorm.DB, uniqCode string) (*UniqSnapshot, error) {
	var row bomModels.Item
	err := tx.WithContext(ctx).
		Where("LOWER(TRIM(uniq_code)) = LOWER(TRIM(?))", uniqCode).
		Where("deleted_at IS NULL").
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return newUniqSnapshot(ctx, tx, uniqCode)
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

// ApplyAdjustment (stock opname manual): Inventory WIP di-set ke qty hasil hitung.
// UNIQ yang belum ada di Inventory WIP dibuat baru (tanpa validasi apapun).
func (a *WIPAdjuster) ApplyAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	var row wipInventoryRow
	if err := tx.WithContext(ctx).
		Raw(`SELECT id, uniq_code, part_number, part_name, uom, stock_qty
			FROM wip_inventories WHERE UPPER(BTRIM(uniq_code)) = UPPER(BTRIM(?)) ORDER BY id LIMIT 1 FOR UPDATE`, entry.UniqCode).
		Scan(&row).Error; err != nil {
		return nil, apperror.Internal("lock WIP inventory row: " + err.Error())
	}
	return postWIPOpname(ctx, tx, entry, sessionNumber, actor, entry.CountedQty-row.StockQty)
}

// ApplyWIPBOMAdjustment (stock opname bulk): baris bulk dicatat terhadap baseline 0,
// jadi qty hitung DITAMBAHKAN ke Inventory WIP (dibuat baru bila belum ada).
func ApplyWIPBOMAdjustment(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string) (*AdjustmentResult, error) {
	return postWIPOpname(ctx, tx, entry, sessionNumber, actor, entry.CountedQty)
}

func postWIPOpname(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry, sessionNumber, actor string, delta float64) (*AdjustmentResult, error) {
	id, _, _, err := assemblystock.AdjustWIP(tx.WithContext(ctx), assemblystock.WIPPost{
		UniqCode:     strings.TrimSpace(entry.UniqCode),
		PartNumber:   wipStr(entry.PartNumber),
		PartName:     wipStr(entry.PartName),
		UOM:          wipStr(entry.UOM),
		MovementType: "stock_opname",
		Notes:        "stock opname " + sessionNumber,
		By:           actor,
		Delta:        delta,
	})
	if err != nil {
		return nil, apperror.Internal("post WIP stock opname: " + err.Error())
	}
	res := &AdjustmentResult{QtyChange: delta}
	if id > 0 {
		res.EntityID = &id
	}
	return res, nil
}
