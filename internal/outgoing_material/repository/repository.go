package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	outModels "github.com/ganasa18/go-template/internal/outgoing_material/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Filter & Row types
// ---------------------------------------------------------------------------

type ListFilter struct {
	Search         string
	DateFrom       string
	DateTo         string
	Reason         string
	Uniq           string
	TransactionID  string
	WorkOrderNo    string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type OutgoingRow struct {
	ID              int64     `gorm:"column:id"`
	TransactionID   string    `gorm:"column:transaction_id"`
	TransactionDate time.Time `gorm:"column:transaction_date"`
	Uniq            string    `gorm:"column:uniq"`
	RMName          *string   `gorm:"column:rm_name"`
	PackingListRM   *string   `gorm:"column:packing_list_rm"`
	Unit            *string   `gorm:"column:unit"`
	QuantityOut     float64   `gorm:"column:quantity_out"`
	StockBefore     float64   `gorm:"column:stock_before"`
	StockAfter      float64   `gorm:"column:stock_after"`
	Reason          string    `gorm:"column:reason"`
	Purpose         *string   `gorm:"column:purpose"`
	WorkOrderNo         *string   `gorm:"column:work_order_no"`
	DestinationLocation *string   `gorm:"column:destination_location"`
	RequestedBy     *string   `gorm:"column:requested_by"`
	Remarks         *string   `gorm:"column:remarks"`
	CreatedBy       *string   `gorm:"column:created_by"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IRepository interface {
	List(ctx context.Context, f ListFilter) ([]OutgoingRow, int64, error)
	GetByID(ctx context.Context, id int64) (*outModels.OutgoingRawMaterial, error)
	// ProcessTx runs the outgoing transaction atomically:
	// 1. Locks and reads raw_materials stock for the given uniq code.
	// 2. Validates stock >= quantity_out.
	// 3. Generates transaction_id, sets stock_before/stock_after, auto-fills rm_name and unit.
	// 4. Inserts outgoing_raw_material row.
	// 5. Deducts stock from raw_materials.
	// The caller receives the populated orm after commit.
	ProcessTx(ctx context.Context, orm *outModels.OutgoingRawMaterial) error
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) List(ctx context.Context, f ListFilter) ([]OutgoingRow, int64, error) {
	q := r.db.WithContext(ctx).Table("outgoing_raw_material").Where("deleted_at IS NULL")

	if f.Search != "" {
		s := "%" + f.Search + "%"
		q = q.Where("transaction_id ILIKE ? OR uniq ILIKE ? OR rm_name ILIKE ?", s, s, s)
	}
	if f.Uniq != "" {
		q = q.Where("uniq = ?", f.Uniq)
	}
	if f.TransactionID != "" {
		q = q.Where("transaction_id = ?", f.TransactionID)
	}
	if f.Reason != "" {
		q = q.Where("reason = ?", f.Reason)
	}
	if f.WorkOrderNo != "" {
		q = q.Where("work_order_no = ?", f.WorkOrderNo)
	}
	if f.DateFrom != "" {
		q = q.Where("transaction_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("transaction_date <= ?", f.DateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "created_at"
	if f.OrderBy != "" {
		orderBy = f.OrderBy
	}
	dir := "DESC"
	if strings.ToLower(f.OrderDirection) == "asc" {
		dir = "ASC"
	}
	q = q.Order(fmt.Sprintf("%s %s", orderBy, dir))

	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}

	var rows []OutgoingRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repo) GetByID(ctx context.Context, id int64) (*outModels.OutgoingRawMaterial, error) {
	var orm outModels.OutgoingRawMaterial
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&orm).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.New(http.StatusNotFound, apperror.CodeNotFound, "outgoing transaction not found")
	}
	return &orm, err
}

func (r *repo) ProcessTx(ctx context.Context, orm *outModels.OutgoingRawMaterial) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	type rmRow struct {
		ID       int64   `gorm:"column:id"`
		StockQty float64 `gorm:"column:stock_qty"`
		UOM      *string `gorm:"column:uom"`
		PartName *string `gorm:"column:part_name"`
	}
	var rm rmRow
	if err := tx.Raw(
		`SELECT id, stock_qty, uom, part_name FROM raw_materials
		 WHERE uniq_code = ? AND deleted_at IS NULL FOR UPDATE`,
		orm.Uniq,
	).Scan(&rm).Error; err != nil {
		tx.Rollback()
		return err
	}
	if rm.ID == 0 {
		tx.Rollback()
		return apperror.New(http.StatusNotFound, apperror.CodeNotFound, "raw material not found: "+orm.Uniq)
	}
	if rm.StockQty < orm.QuantityOut {
		tx.Rollback()
		return apperror.New(http.StatusUnprocessableEntity, apperror.CodeUnprocessable,
			fmt.Sprintf("insufficient stock: available %.4f, requested %.4f", rm.StockQty, orm.QuantityOut))
	}

	var count int64
	tx.Raw("SELECT COUNT(*) FROM outgoing_raw_material").Scan(&count)

	orm.UUID = uuid.New()
	orm.TransactionID = fmt.Sprintf("OUT-RM-%05d", count+1)
	orm.StockBefore = rm.StockQty
	orm.StockAfter = rm.StockQty - orm.QuantityOut
	orm.RawMaterialID = &rm.ID
	if orm.RMName == nil && rm.PartName != nil {
		orm.RMName = rm.PartName
	}
	if orm.Unit == nil && rm.UOM != nil {
		orm.Unit = rm.UOM
	}

	if err := tx.Create(orm).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec(
		`UPDATE raw_materials SET stock_qty = stock_qty - ?, updated_at = NOW(), updated_by = ?
		 WHERE id = ?`,
		orm.QuantityOut, orm.CreatedBy, rm.ID,
	).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
