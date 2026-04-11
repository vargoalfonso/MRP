package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	procModels "github.com/ganasa18/go-template/internal/procurement/models"
	qcModels "github.com/ganasa18/go-template/internal/qc/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ListFilter struct {
	TaskType string
	Status   string
	Page     int
	Limit    int
	Offset   int
}

type TaskListRow struct {
	ID        int64     `gorm:"column:id"`
	TaskType  string    `gorm:"column:task_type"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`

	PackingNumber *string  `gorm:"column:packing_number"`
	DnNumber      *string  `gorm:"column:dn_number"`
	PoNumber      *string  `gorm:"column:po_number"`
	SupplierName  *string  `gorm:"column:supplier_name"`
	ItemUniqCode  *string  `gorm:"column:item_uniq_code"`
	QtyReceived   *float64 `gorm:"column:qty_received"`
	UOM           *string  `gorm:"column:uom"`
}

type IRepository interface {
	ListTasks(ctx context.Context, f ListFilter) ([]TaskListRow, int64, error)
	StartTask(ctx context.Context, taskID int64, performedBy string) (*qcModels.QCTask, error)
	ApproveIncoming(ctx context.Context, taskID int64, approvedQty, ngQty, scrapQty int, notes *string, defects []interface{}, scrapDisposition *string, performedBy string) error
	RejectIncoming(ctx context.Context, taskID int64, rejectedQty int, reason string, defects []interface{}, disposition *string, performedBy string) error
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) ListTasks(ctx context.Context, f ListFilter) ([]TaskListRow, int64, error) {
	q := r.db.WithContext(ctx).Table("qc_tasks t")
	q = q.Joins("LEFT JOIN incoming_dn_items idi ON idi.id = t.incoming_dn_item_id")
	q = q.Joins("LEFT JOIN incoming_dns idn ON idn.id = idi.incoming_dn_id")
	q = q.Joins("LEFT JOIN supplier s ON s.supplier_id = idn.supplier_id")

	if f.TaskType != "" {
		q = q.Where("t.task_type = ?", f.TaskType)
	}
	if f.Status != "" {
		q = q.Where("t.status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("ListTasks count: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var rows []TaskListRow
	err := q.Select(`
		t.id,
		t.task_type,
		t.status,
		t.created_at,
		idi.packing_number,
		idn.dn_number,
		idn.po_number,
		s.supplier_name,
		idi.item_uniq_code,
		idi.qty_received::numeric AS qty_received,
		idi.uom
	`).
		Order("t.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("ListTasks scan: %w", err)
	}
	return rows, total, nil
}

func (r *repo) StartTask(ctx context.Context, taskID int64, performedBy string) (*qcModels.QCTask, error) {
	var task qcModels.QCTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", taskID).Error; err != nil {
			return apperror.NotFound("qc task not found")
		}
		if task.Status == "approved" || task.Status == "rejected" {
			return apperror.Conflict("qc task already completed")
		}
		if task.Status == "in_progress" {
			return nil
		}

		if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Update("status", "in_progress").Error; err != nil {
			return fmt.Errorf("update qc task: %w", err)
		}
		// append small audit event
		if err := appendRoundEventTx(tx, taskID, map[string]interface{}{
			"event":        "qc_start",
			"performed_by": performedBy,
			"occurred_at":  time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}

		return tx.First(&task, "id = ?", taskID).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *repo) ApproveIncoming(ctx context.Context, taskID int64, approvedQty, ngQty, scrapQty int, notes *string, defects []interface{}, scrapDisposition *string, performedBy string) error {
	if approvedQty < 0 || ngQty < 0 || scrapQty < 0 {
		return apperror.BadRequest("qty fields must be >= 0")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task qcModels.QCTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", taskID).Error; err != nil {
			return apperror.NotFound("qc task not found")
		}
		if task.TaskType != "incoming_qc" {
			return apperror.UnprocessableEntity("task_type must be incoming_qc")
		}
		if task.Status == "approved" {
			return nil
		}
		if task.Status == "rejected" {
			return apperror.Conflict("qc task already rejected")
		}
		if task.IncomingDNItemID == nil || strings.TrimSpace(*task.IncomingDNItemID) == "" {
			return apperror.UnprocessableEntity("incoming_dn_item_id is required for incoming_qc")
		}

		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", *task.IncomingDNItemID).Error; err != nil {
			return apperror.NotFound("incoming_dn_item not found")
		}

		if approvedQty+ngQty+scrapQty > dnItem.QtyReceived {
			return apperror.UnprocessableEntity("approved_qty + ng_qty + scrap_qty must be <= qty_received")
		}

		// Update task fields
		if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":         "approved",
			"good_quantity":  approvedQty,
			"ng_quantity":    ngQty,
			"scrap_quantity": scrapQty,
			"updated_at":     time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("update qc_tasks: %w", err)
		}

		// Update DN item quality
		if err := tx.Model(&procModels.IncomingDNItem{}).Where("id = ?", dnItem.ID).Update("quality_status", "Approved").Error; err != nil {
			return fmt.Errorf("update incoming_dn_items: %w", err)
		}

		event := map[string]interface{}{
			"event":             "qc_approve",
			"approved_qty":      approvedQty,
			"ng_qty":            ngQty,
			"scrap_qty":         scrapQty,
			"scrap_disposition": scrapDisposition,
			"notes":             notes,
			"defects":           defects,
			"performed_by":      performedBy,
			"occurred_at":       time.Now().UTC().Format(time.RFC3339),
		}
		if err := appendRoundEventTx(tx, taskID, event); err != nil {
			return err
		}

		// Inventory posting:
		// We keep it best-effort by checking if target tables exist.
		// If the table doesn't exist yet in environment, we return a clear error.
		dnType, poNumber, err := loadDNTypeAndPONumber(tx, dnItem.IncomingDNID)
		if err != nil {
			return err
		}
		if approvedQty > 0 {
			if err := postToInventoryByDNType(tx, dnType, dnItem.ItemUniqCode, approvedQty, dnItem.WeightReceived); err != nil {
				return err
			}
			// Rollup purchase_orders.total_incoming by PO number if PO table exists
			if poNumber != "" {
				_ = tx.Table("purchase_orders").Where("po_number = ?", poNumber).Update("total_incoming", gorm.Expr("COALESCE(total_incoming,0) + ?", approvedQty)).Error
			}
		}

		// Scrap posting if requested
		if scrapQty > 0 && scrapDisposition != nil && *scrapDisposition == "scrap" {
			_ = postToScrap(tx, dnItem.ItemUniqCode, scrapQty, dnItem.PackingNumber)
		}

		return nil
	})
}

func (r *repo) RejectIncoming(ctx context.Context, taskID int64, rejectedQty int, reason string, defects []interface{}, disposition *string, performedBy string) error {
	if rejectedQty < 0 {
		return apperror.BadRequest("rejected_qty must be >= 0")
	}
	if strings.TrimSpace(reason) == "" {
		return apperror.BadRequest("reason is required")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task qcModels.QCTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", taskID).Error; err != nil {
			return apperror.NotFound("qc task not found")
		}
		if task.TaskType != "incoming_qc" {
			return apperror.UnprocessableEntity("task_type must be incoming_qc")
		}
		if task.Status == "rejected" {
			return nil
		}
		if task.Status == "approved" {
			return apperror.Conflict("qc task already approved")
		}
		if task.IncomingDNItemID == nil || strings.TrimSpace(*task.IncomingDNItemID) == "" {
			return apperror.UnprocessableEntity("incoming_dn_item_id is required for incoming_qc")
		}

		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", *task.IncomingDNItemID).Error; err != nil {
			return apperror.NotFound("incoming_dn_item not found")
		}

		if rejectedQty > dnItem.QtyReceived {
			return apperror.UnprocessableEntity("rejected_qty must be <= qty_received")
		}

		// Update task
		if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":      "rejected",
			"ng_quantity": rejectedQty,
			"updated_at":  time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("update qc_tasks: %w", err)
		}
		if err := tx.Model(&procModels.IncomingDNItem{}).Where("id = ?", dnItem.ID).Update("quality_status", "Rejected").Error; err != nil {
			return fmt.Errorf("update incoming_dn_items: %w", err)
		}

		event := map[string]interface{}{
			"event":        "qc_reject",
			"rejected_qty": rejectedQty,
			"reason":       reason,
			"disposition":  disposition,
			"defects":      defects,
			"performed_by": performedBy,
			"occurred_at":  time.Now().UTC().Format(time.RFC3339),
		}
		if err := appendRoundEventTx(tx, taskID, event); err != nil {
			return err
		}

		// Scrap posting only if explicitly scrap.
		if rejectedQty > 0 && disposition != nil && *disposition == "scrap" {
			_ = postToScrap(tx, dnItem.ItemUniqCode, rejectedQty, dnItem.PackingNumber)
		}

		return nil
	})
}

func appendRoundEventTx(tx *gorm.DB, taskID int64, event map[string]interface{}) error {
	// Read current round_results, append, write back.
	var task qcModels.QCTask
	if err := tx.Select("id", "round_results").First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("load round_results: %w", err)
	}

	var arr []interface{}
	if len(task.RoundResults) > 0 {
		_ = json.Unmarshal(task.RoundResults, &arr)
	}
	arr = append(arr, event)
	b, _ := json.Marshal(arr)
	if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Update("round_results", b).Error; err != nil {
		return fmt.Errorf("update round_results: %w", err)
	}
	return nil
}

func tableExists(tx *gorm.DB, table string) bool {
	var exists bool
	_ = tx.Raw("SELECT to_regclass(?) IS NOT NULL", table).Scan(&exists).Error
	return exists
}

func loadDNTypeAndPONumber(tx *gorm.DB, dnID string) (dnType string, poNumber string, err error) {
	if !tableExists(tx, "incoming_dns") {
		return "", "", apperror.UnprocessableEntity("incoming_dns table not found; apply migration 0015_dn_feature_up.sql")
	}
	var row struct {
		DnType   string `gorm:"column:dn_type"`
		PoNumber string `gorm:"column:po_number"`
	}
	if err := tx.Table("incoming_dns").Select("dn_type, po_number").Where("id = ?", dnID).Limit(1).Scan(&row).Error; err != nil {
		return "", "", fmt.Errorf("load incoming_dns: %w", err)
	}
	return row.DnType, row.PoNumber, nil
}

func pickUniqColumn(tx *gorm.DB, table string) (string, error) {
	// try common names
	candidates := []string{"uniq", "item_uniq_code", "uniq_code"}
	for _, col := range candidates {
		var exists bool
		err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name = ? AND column_name = ?
		)`, table, col).Scan(&exists).Error
		if err == nil && exists {
			return col, nil
		}
	}
	return "", apperror.UnprocessableEntity(fmt.Sprintf("cannot map uniq code for table %s", table))
}

func pickQtyColumn(tx *gorm.DB, table string) (string, error) {
	candidates := []string{"stock", "quantity", "qty"}
	for _, col := range candidates {
		var exists bool
		err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name = ? AND column_name = ?
		)`, table, col).Scan(&exists).Error
		if err == nil && exists {
			return col, nil
		}
	}
	return "", apperror.UnprocessableEntity(fmt.Sprintf("cannot map quantity column for table %s", table))
}

func pickWeightColumn(tx *gorm.DB, table string) *string {
	candidates := []string{"weight", "weight_kg", "weigh"}
	for _, col := range candidates {
		var exists bool
		err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name = ? AND column_name = ?
		)`, table, col).Scan(&exists).Error
		if err == nil && exists {
			return &col
		}
	}
	return nil
}

func postToInventoryByDNType(tx *gorm.DB, dnType string, itemUniqCode string, qty int, weightReceived *float64) error {
	dnType = strings.TrimSpace(dnType)
	var table string
	switch dnType {
	case "Raw Material":
		table = "rm_inventory"
	case "Indirect Raw Material":
		table = "indirect_raw_material"
	case "SubCon":
		table = "subcon_raw_material"
	default:
		return apperror.UnprocessableEntity("unsupported dn_type: " + dnType)
	}
	if !tableExists(tx, table) {
		return apperror.UnprocessableEntity(fmt.Sprintf("%s table not found", table))
	}

	uniqCol, err := pickUniqColumn(tx, table)
	if err != nil {
		return err
	}
	qtyCol, err := pickQtyColumn(tx, table)
	if err != nil {
		return err
	}

	// If row not exists, create a minimal row.
	// Use upsert when there's a unique constraint; if not, fallback to update then insert.
	res := tx.Table(table).Where(fmt.Sprintf("%s = ?", uniqCol), itemUniqCode).Update(qtyCol, gorm.Expr(fmt.Sprintf("COALESCE(%s,0) + ?", qtyCol), qty))
	if res.Error != nil {
		return fmt.Errorf("update %s: %w", table, res.Error)
	}
	if res.RowsAffected == 0 {
		row := map[string]interface{}{uniqCol: itemUniqCode, qtyCol: qty}
		if wcol := pickWeightColumn(tx, table); wcol != nil && weightReceived != nil {
			row[*wcol] = *weightReceived
		}
		if err := tx.Table(table).Create(row).Error; err != nil {
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}

	// Weight update (incremental) if requested.
	if weightReceived != nil {
		if wcol := pickWeightColumn(tx, table); wcol != nil {
			_ = tx.Table(table).Where(fmt.Sprintf("%s = ?", uniqCol), itemUniqCode).Update(*wcol, gorm.Expr(fmt.Sprintf("COALESCE(%s,0) + ?", *wcol), *weightReceived)).Error
		}
	}

	return nil
}

func postToScrap(tx *gorm.DB, itemUniqCode string, qty int, packingNumber *string) error {
	if !tableExists(tx, "scrap_stock") {
		return nil
	}
	uniqCol, err := pickUniqColumn(tx, "scrap_stock")
	if err != nil {
		return nil
	}
	qtyCol, err := pickQtyColumn(tx, "scrap_stock")
	if err != nil {
		return nil
	}

	res := tx.Table("scrap_stock").Where(fmt.Sprintf("%s = ?", uniqCol), itemUniqCode).Update(qtyCol, gorm.Expr(fmt.Sprintf("COALESCE(%s,0) + ?", qtyCol), qty))
	if res.Error != nil {
		return nil
	}
	if res.RowsAffected == 0 {
		row := map[string]interface{}{uniqCol: itemUniqCode, qtyCol: qty}
		if packingNumber != nil {
			row["packing_number"] = *packingNumber
		}
		row["scrap_type"] = "Incoming"
		_ = tx.Table("scrap_stock").Create(row).Error
	}
	return nil
}
