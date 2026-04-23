// Package repository provides database access for the Scrap Stock module.
package repository

import (
	"context"
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
	AddScrapQty(ctx context.Context, id int64, delta float64, updatedBy string) error

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
		return nil, apperror.NotFound(fmt.Sprintf("scrap stock id %d not found", id))
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
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return apperror.Internal("create scrap stock: " + err.Error())
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
		return apperror.NotFound(fmt.Sprintf("scrap stock id %d not found", id))
	}
	return nil
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
		return nil, apperror.NotFound(fmt.Sprintf("scrap release id %d not found", id))
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
				return apperror.NotFound(fmt.Sprintf("scrap release id %d not found", id))
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

		// Deduct stock only on Completed (approved)
		if action == scrapModels.ApprovalStatusCompleted {
			res := tx.Model(&scrapModels.ScrapStock{}).
				Where("id = ? AND deleted_at IS NULL", rel.ScrapStockID).
				Updates(map[string]interface{}{
					"quantity":   gorm.Expr("quantity - ?", rel.ReleaseQty),
					"updated_by": approvedBy,
					"updated_at": now,
				})
			if res.Error != nil {
				return apperror.Internal("deduct scrap qty: " + res.Error.Error())
			}
			if res.RowsAffected == 0 {
				return apperror.NotFound("scrap stock record not found during deduction")
			}
		}
		return nil
	})
}
