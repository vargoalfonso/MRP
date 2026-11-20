// Package repository provides database access for the Scrap Stock module.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	scrapModels "github.com/ganasa18/go-template/internal/scrap_stock/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Filter types
// ---------------------------------------------------------------------------

// ScrapStockFilter holds query parameters for listing scrap stocks.
type ScrapStockFilter struct {
	ScrapType     string
	UniqCode      string
	PackingNumber string
	WONumber      string
	Status        string
	DateFrom      string // YYYY-MM-DD
	DateTo        string // YYYY-MM-DD
	Page          int
	Limit         int
	Offset        int
}

// ScrapReleaseFilter holds query parameters for listing scrap releases.
type ScrapReleaseFilter struct {
	ReleaseType    string
	ApprovalStatus string
	ScrapStockID   int64
	Page           int
	Limit          int
	Offset         int
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IRepository interface {
	// Scrap Stock
	GetStats(ctx context.Context) (*scrapModels.ScrapStockStats, error)
	ListScrapStocks(ctx context.Context, f ScrapStockFilter) ([]scrapModels.ScrapStock, int64, error)
	GetScrapStockByID(ctx context.Context, id int64) (*scrapModels.ScrapStock, error)
	CreateScrapStock(ctx context.Context, s *scrapModels.ScrapStock) error
	UpdateScrapStock(ctx context.Context, id int64, fields map[string]interface{}) error
	DeleteScrapStock(ctx context.Context, id int64, deletedBy string) error
	AddScrapQty(ctx context.Context, id int64, delta float64, updatedBy string) error
	// Packing options untuk create form (sumber: scan produksi / wip_items + delivery_note_items)
	ListPackingNumbersByUniq(ctx context.Context, uniq string) ([]string, error)
	// UNIQ item options untuk create form (sumber: items — FG/RM/Indirect/subcon)
	ListItemOptions(ctx context.Context, q string, limit int) ([]ItemOption, error)

	// Scrap Release
	ListScrapReleases(ctx context.Context, f ScrapReleaseFilter) ([]scrapModels.ScrapRelease, int64, error)
	GetScrapReleaseByID(ctx context.Context, id int64) (*scrapModels.ScrapRelease, error)
	CreateScrapRelease(ctx context.Context, r *scrapModels.ScrapRelease) error
	ApproveRelease(ctx context.Context, id int64, action, approvedBy string, remarks *string) error

	// Movement History
	ListScrapMovements(ctx context.Context, scrapStockID int64, limit, offset int) ([]scrapModels.ScrapMovementRow, int64, error)
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository { return &repository{db: db} }

// ---------------------------------------------------------------------------
// Scrap Stock
// ---------------------------------------------------------------------------

// GetStats returns the 4 dashboard summary cards.
func (r *repository) GetStats(ctx context.Context) (*scrapModels.ScrapStockStats, error) {
	type statsRow struct {
		TotalItems    int64   `gorm:"column:total_items"`
		TotalQty      float64 `gorm:"column:total_qty"`
		TotalWeightKg float64 `gorm:"column:total_weight_kg"`
		ScrapTypes    int64   `gorm:"column:scrap_types"`
	}
	var row statsRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*)                        AS total_items,
			COALESCE(SUM(quantity), 0)      AS total_qty,
			COALESCE(SUM(weight_kg), 0)     AS total_weight_kg,
			COUNT(DISTINCT scrap_type)      AS scrap_types
		FROM scrap_stocks
		WHERE deleted_at IS NULL
		  AND status = 'Active'
	`).Scan(&row).Error
	if err != nil {
		return nil, apperror.Internal("get scrap stats: " + err.Error())
	}
	return &scrapModels.ScrapStockStats{
		TotalItems:    row.TotalItems,
		TotalQty:      row.TotalQty,
		TotalWeightKg: row.TotalWeightKg,
		ScrapTypes:    row.ScrapTypes,
	}, nil
}

func (r *repository) ListScrapStocks(ctx context.Context, f ScrapStockFilter) ([]scrapModels.ScrapStock, int64, error) {
	q := r.db.WithContext(ctx).Model(&scrapModels.ScrapStock{}).Where("deleted_at IS NULL")

	if f.ScrapType != "" {
		q = q.Where("scrap_type = ?", f.ScrapType)
	}
	if f.UniqCode != "" {
		q = q.Where("uniq_code ILIKE ?", "%"+f.UniqCode+"%")
	}
	if f.PackingNumber != "" {
		q = q.Where("packing_number ILIKE ?", "%"+f.PackingNumber+"%")
	}
	if f.WONumber != "" {
		q = q.Where("wo_number ILIKE ?", "%"+f.WONumber+"%")
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.DateFrom != "" {
		q = q.Where("date_received >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("date_received <= ?", f.DateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count scrap stocks: " + err.Error())
	}

	var rows []scrapModels.ScrapStock
	if err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("list scrap stocks: " + err.Error())
	}
	return rows, total, nil
}

func (r *repository) GetScrapStockByID(ctx context.Context, id int64) (*scrapModels.ScrapStock, error) {
	var s scrapModels.ScrapStock
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("scrap stock id %d tidak ditemukan", id))
	}
	if err != nil {
		return nil, apperror.Internal("get scrap stock: " + err.Error())
	}
	return &s, nil
}

func (r *repository) CreateScrapStock(ctx context.Context, s *scrapModels.ScrapStock) error {
	s.UUID = uuid.New()
	if s.Status == "" {
		s.Status = scrapModels.StockStatusActive
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Potong stok dari inventory asli
		if err := deductInventoryForScrap(tx, s); err != nil {
			return err
		}

		// 2. Simpan record scrap stock
		if err := tx.Create(s).Error; err != nil {
			return apperror.Internal("create scrap stock: " + err.Error())
		}
		return nil
	})
}

func deductInventoryForScrap(tx *gorm.DB, s *scrapModels.ScrapStock) error {
	var packing string
	if s.PackingNumber != nil {
		packing = *s.PackingNumber
	}

	switch s.ScrapType {
	case "Product Return Scrap":
		return deductProductReturn(tx, s)
	case "Setting Machine Scrap":
		return deductTable(tx, "raw_materials", "quantity", "uniq_code", s.UniqCode, packing, s.Quantity)
	case "Process Scrap":
		// Auto detect
		err := deductTable(tx, "wip_items", "stock", "uniq", s.UniqCode, packing, s.Quantity)
		if err == nil {
			return nil
		}
		err = deductTable(tx, "raw_materials", "quantity", "uniq_code", s.UniqCode, packing, s.Quantity)
		if err == nil {
			return nil
		}
		err = deductTable(tx, "finished_goods", "quantity", "uniq_code", s.UniqCode, packing, s.Quantity)
		if err == nil {
			return nil
		}
		return apperror.UnprocessableEntity("Stok tidak mencukupi atau barang tidak ditemukan di WIP/RM/FG")
	default:
		// For other types like Customer Return Scrap, we don't deduct internal inventory
		return nil
	}
}

func deductProductReturn(tx *gorm.DB, s *scrapModels.ScrapStock) error {
	type PRRow struct {
		ID             int64
		QuantityScrap  float64
		QuantityRework float64
	}
	var rows []PRRow
	q := tx.Table("product_returns").Where("uniq = ? AND (quantity_scrap > 0 OR quantity_rework > 0)", s.UniqCode)
	if s.PackingNumber != nil {
		q = q.Where("dn_number = ?", *s.PackingNumber)
	}
	if err := q.Order("created_at ASC").Scan(&rows).Error; err != nil {
		return err
	}

	remaining := s.Quantity
	for _, row := range rows {
		if remaining <= 0 {
			break
		}

		// Potong quantity_scrap dulu
		if row.QuantityScrap > 0 {
			deductScrap := remaining
			if row.QuantityScrap < deductScrap {
				deductScrap = row.QuantityScrap
			}
			if err := tx.Table("product_returns").Where("id = ?", row.ID).UpdateColumn("quantity_scrap", gorm.Expr("quantity_scrap - ?", deductScrap)).Error; err != nil {
				return err
			}
			remaining -= deductScrap
		}

		// Jika masih kurang, potong dari quantity_rework
		if remaining > 0 && row.QuantityRework > 0 {
			deductRework := remaining
			if row.QuantityRework < deductRework {
				deductRework = row.QuantityRework
			}
			if err := tx.Table("product_returns").Where("id = ?", row.ID).UpdateColumn("quantity_rework", gorm.Expr("quantity_rework - ?", deductRework)).Error; err != nil {
				return err
			}
			remaining -= deductRework
		}
	}

	if remaining > 0 {
		return apperror.UnprocessableEntity(fmt.Sprintf("Stok di Product Return tidak mencukupi. Kurang %v", remaining))
	}
	return nil
}

func deductTable(tx *gorm.DB, tableName, qtyColumn, uniqColumn, uniq, packing string, qty float64) error {
	type StockRow struct {
		ID  int64
		Qty float64 `gorm:"column:qty_val"`
	}
	var rows []StockRow
	q := tx.Table(tableName).
		Select(fmt.Sprintf("id, %s AS qty_val", qtyColumn)).
		Where(fmt.Sprintf("%s AND %s = ? AND %s > 0", getDeletedCondition(tableName), uniqColumn, qtyColumn), uniq)
	if packing != "" {
		if tableName == "wip_items" {
			q = q.Where("packing_number = ?", packing)
		} else {
			q = q.Where("packing_number = ?", packing)
		}
	}

	if err := q.Order("created_at ASC").Scan(&rows).Error; err != nil {
		return err
	}

	remaining := qty
	for _, row := range rows {
		if remaining <= 0 {
			break
		}
		deduct := remaining
		if row.Qty < deduct {
			deduct = row.Qty
		}

		res := tx.Table(tableName).Where("id = ?", row.ID).UpdateColumn(qtyColumn, gorm.Expr(fmt.Sprintf("%s - ?", qtyColumn), deduct))
		if res.Error != nil {
			return res.Error
		}
		remaining -= deduct
	}

	if remaining > 0 {
		return fmt.Errorf("stok %s tidak mencukupi (kurang %v)", tableName, remaining)
	}
	return nil
}

func getDeletedCondition(tableName string) string {
	if tableName == "wip_items" || tableName == "product_returns" {
		return "1=1"
	}
	return "deleted_at IS NULL"
}

func (r *repository) UpdateScrapStock(ctx context.Context, id int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).
		Model(&scrapModels.ScrapStock{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields)
	if res.Error != nil {
		return apperror.Internal("update scrap stock: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound(fmt.Sprintf("scrap stock id %d tidak ditemukan", id))
	}
	return nil
}

func (r *repository) DeleteScrapStock(ctx context.Context, id int64, deletedBy string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&scrapModels.ScrapStock{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_by": deletedBy,
			"updated_at": now,
		})
	if res.Error != nil {
		return apperror.Internal("delete scrap stock: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound(fmt.Sprintf("scrap stock id %d tidak ditemukan", id))
	}
	return nil
}

// AddScrapQty increments (or decrements when delta < 0) the quantity and bumps updated_at.
func (r *repository) AddScrapQty(ctx context.Context, id int64, delta float64, updatedBy string) error {
	res := r.db.WithContext(ctx).
		Model(&scrapModels.ScrapStock{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"quantity":   gorm.Expr("quantity + ?", delta),
			"updated_by": updatedBy,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return apperror.Internal("update scrap qty: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound(fmt.Sprintf("scrap stock id %d tidak ditemukan", id))
	}
	return nil
}

func (r *repository) ListPackingNumbersByUniq(ctx context.Context, uniq string) ([]string, error) {
	// Sumber packing: scan produksi (wip_items) untuk finished goods,
	// digabung dengan scan Packing List / DN (delivery_note_items) untuk
	// raw material / indirect / incoming lainnya.
	var packings []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT packing_number FROM (
			SELECT packing_number
			FROM wip_items
			WHERE uniq = ?
			  AND COALESCE(packing_number, '') <> ''
			  AND wip_type = 'production'
			  AND status = 'done'
			UNION
			SELECT packing_number
			FROM delivery_note_items
			WHERE item_uniq_code = ?
			  AND COALESCE(packing_number, '') <> ''
			UNION
			SELECT dn_number AS packing_number
			FROM product_returns
			WHERE uniq = ?
			  AND COALESCE(dn_number, '') <> ''
			  AND (quantity_scrap > 0 OR quantity_rework > 0)
		) t
		ORDER BY packing_number ASC
	`, uniq, uniq, uniq).Pluck("packing_number", &packings).Error
	if err != nil {
		return nil, apperror.Internal("list packing numbers by uniq: " + err.Error())
	}
	return packings, nil
}

// ItemOption is one row for the source-agnostic UNIQ autocomplete on the
// scrap create form. Source is the shared items table.
type ItemOption struct {
	UniqCode     string  `gorm:"column:uniq_code"`
	PartNumber   *string `gorm:"column:part_number"`
	PartName     *string `gorm:"column:part_name"`
	Model        *string `gorm:"column:model"`
	UOM          *string `gorm:"column:uom"`
	MaterialType *string `gorm:"column:material_type"`
}

func (r *repository) ListItemOptions(ctx context.Context, q string, limit int) ([]ItemOption, error) {
	// Scrap bisa berasal dari semua sumber inventory, jadi UNIQ dikumpulkan dari
	// semua tabel stok (finished_goods / raw_materials / indirect / subcon), plus
	// master items. Item yang tidak punya baris di items tetap muncul di sini.
	// DISTINCT ON + prioritas sumber memilih detail part terbaik per uniq_code.
	rows := make([]ItemOption, 0, limit)
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			uniq_code,
			MAX(part_number) AS part_number,
			MAX(part_name) AS part_name,
			MAX(model) AS model,
			MAX(uom) AS uom,
			material_type
		FROM (
			SELECT uniq_code, part_number, part_name, model, uom, 'Finished Good'::text AS material_type
			  FROM finished_goods         WHERE deleted_at IS NULL
			UNION ALL
			SELECT uniq_code, part_number, part_name, NULL::text, uom, raw_material_type AS material_type
			  FROM raw_materials          WHERE deleted_at IS NULL
			UNION ALL
			SELECT uniq_code, part_number, part_name, NULL::text, uom, 'Indirect Material'::text AS material_type
			  FROM indirect_raw_materials WHERE deleted_at IS NULL
			UNION ALL
			SELECT uniq_code, part_number, part_name, NULL::text, NULL::text, 'Subcon'::text AS material_type
			  FROM subcon_inventories     WHERE deleted_at IS NULL
			UNION ALL
			SELECT uniq, NULL::text, NULL::text, NULL::text, NULL::text, 'Product Return'::text AS material_type
			  FROM product_returns
		) t
		WHERE ($1 = ''
			OR uniq_code                 ILIKE '%' || $1 || '%'
			OR COALESCE(part_name, '')   ILIKE '%' || $1 || '%'
			OR COALESCE(part_number, '') ILIKE '%' || $1 || '%')
		GROUP BY uniq_code, material_type
		ORDER BY uniq_code
		LIMIT $2
	`, q, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("list scrap item options: " + err.Error())
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// Scrap Release
// ---------------------------------------------------------------------------

func (r *repository) ListScrapReleases(ctx context.Context, f ScrapReleaseFilter) ([]scrapModels.ScrapRelease, int64, error) {
	q := r.db.WithContext(ctx).Model(&scrapModels.ScrapRelease{}).Where("deleted_at IS NULL")

	if f.ReleaseType != "" {
		q = q.Where("release_type = ?", f.ReleaseType)
	}
	if f.ApprovalStatus != "" {
		q = q.Where("approval_status = ?", f.ApprovalStatus)
	}
	if f.ScrapStockID > 0 {
		q = q.Where("scrap_stock_id = ?", f.ScrapStockID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count scrap releases: " + err.Error())
	}

	var rows []scrapModels.ScrapRelease
	if err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("list scrap releases: " + err.Error())
	}
	return rows, total, nil
}

func (r *repository) GetScrapReleaseByID(ctx context.Context, id int64) (*scrapModels.ScrapRelease, error) {
	var rel scrapModels.ScrapRelease
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&rel).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("scrap release id %d tidak ditemukan", id))
	}
	if err != nil {
		return nil, apperror.Internal("get scrap release: " + err.Error())
	}
	return &rel, nil
}

// nextReleaseNumber generates the next SR-YYYY-NNN number inside a transaction.
func nextReleaseNumber(db *gorm.DB, year int) (string, error) {
	var count int64
	if err := db.Raw(
		`SELECT COUNT(*) FROM scrap_releases WHERE release_number LIKE ?`,
		fmt.Sprintf("SR-%d-%%", year),
	).Scan(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("SR-%d-%03d", year, count+1), nil
}

func (r *repository) CreateScrapRelease(ctx context.Context, rel *scrapModels.ScrapRelease) error {
	rel.UUID = uuid.New()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		num, err := nextReleaseNumber(tx, time.Now().Year())
		if err != nil {
			return apperror.Internal("generate release number: " + err.Error())
		}
		rel.ReleaseNumber = num
		if err := tx.Create(rel).Error; err != nil {
			return apperror.Internal("create scrap release: " + err.Error())
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Movement History
// ---------------------------------------------------------------------------

func (r *repository) ListScrapMovements(ctx context.Context, scrapStockID int64, limit, offset int) ([]scrapModels.ScrapMovementRow, int64, error) {
	q := r.db.WithContext(ctx).
		Table("inventory_movement_logs iml").
		Joins("LEFT JOIN scrap_stocks ss ON ss.id = iml.entity_id").
		Where("iml.movement_category = 'scrap' AND iml.entity_id = ?", scrapStockID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count scrap movements: " + err.Error())
	}

	var rows []scrapModels.ScrapMovementRow
	err := q.Select("iml.id, iml.uniq_code, ss.packing_number AS packing_list, iml.qty_change, iml.source_flag, iml.reference_id, iml.notes, iml.logged_by, iml.logged_at").
		Order("iml.logged_at DESC").
		Limit(limit).Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, apperror.Internal("list scrap movements: " + err.Error())
	}
	return rows, total, nil
}

// ApproveRelease transitions a release to Completed/Rejected and, when Completed,
// deducts the release_qty from the parent scrap_stock — all inside a transaction.
func (r *repository) ApproveRelease(ctx context.Context, id int64, action, approvedBy string, remarks *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rel scrapModels.ScrapRelease
		if err := tx.Where("id = ? AND deleted_at IS NULL", id).First(&rel).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperror.NotFound(fmt.Sprintf("scrap release id %d tidak ditemukan", id))
			}
			return apperror.Internal("get release: " + err.Error())
		}

		if rel.ApprovalStatus != scrapModels.ApprovalStatusPending {
			return apperror.Conflict("release already " + rel.ApprovalStatus)
		}

		now := time.Now()
		updates := map[string]interface{}{
			"approval_status": action,
			"approved_by":     approvedBy,
			"approved_at":     now,
			"updated_by":      approvedBy,
			"updated_at":      now,
		}
		if remarks != nil {
			updates["remarks"] = *remarks
		}
		if err := tx.Model(&rel).Updates(updates).Error; err != nil {
			return apperror.Internal("update release status: " + err.Error())
		}

		// Deduct stock only on Completed (approved). Multi-item aware.
		if action == scrapModels.ApprovalStatusCompleted {
			type deductLine struct {
				ScrapStockID int64   `json:"scrap_stock_id"`
				ReleaseQty   float64 `json:"release_qty"`
			}
			var deducts []deductLine
			if rel.ItemsJSON != nil && *rel.ItemsJSON != "" {
				if err := json.Unmarshal([]byte(*rel.ItemsJSON), &deducts); err != nil {
					return apperror.Internal("parse release items: " + err.Error())
				}
			}
			if len(deducts) == 0 {
				deducts = []deductLine{{ScrapStockID: rel.ScrapStockID, ReleaseQty: rel.ReleaseQty}}
			}
			for _, d := range deducts {
				res := tx.Model(&scrapModels.ScrapStock{}).
					Where("id = ? AND deleted_at IS NULL", d.ScrapStockID).
					Updates(map[string]interface{}{
						"quantity":   gorm.Expr("quantity - ?", d.ReleaseQty),
						"updated_by": approvedBy,
						"updated_at": now,
					})
				if res.Error != nil {
					return apperror.Internal("deduct scrap qty: " + res.Error.Error())
				}
				if res.RowsAffected == 0 {
					return apperror.NotFound("scrap stock record tidak ditemukan during deduction")
				}
			}
		}
		return nil
	})
}
