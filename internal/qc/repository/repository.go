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
	ApproveIncoming(ctx context.Context, taskID int64, numberOfDefects int, dateChecked string, performedBy string) error
	RejectIncoming(ctx context.Context, taskID int64, numberOfDefects int, dateChecked string, performedBy string) error
}

type repo struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repo{db: db} }

func (r *repo) ListTasks(ctx context.Context, f ListFilter) ([]TaskListRow, int64, error) {
	q := r.db.WithContext(ctx).Table("qc_tasks t")
	q = q.Joins("LEFT JOIN delivery_note_items idi ON idi.id = t.incoming_dn_item_id")
	q = q.Joins("LEFT JOIN delivery_notes idn ON idn.id = idi.dn_id")
	q = q.Joins("LEFT JOIN suppliers s ON s.id = idn.supplier_id")

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
		idn.dn_number AS packing_number,
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
	performedBy = normalizeActor(performedBy)

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

func (r *repo) ApproveIncoming(ctx context.Context, taskID int64, numberOfDefects int, dateChecked string, performedBy string) error {
	performedBy = normalizeActor(performedBy)

	if numberOfDefects < 0 {
		return apperror.BadRequest("number_of_defects must be >= 0")
	}

	// Parse date_checked (YYYY-MM-DD)
	checkedAt, err := time.Parse("2006-01-02", dateChecked)
	if err != nil {
		return apperror.BadRequest("date_checked must be in YYYY-MM-DD format")
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
		if task.IncomingDNItemID == nil || *task.IncomingDNItemID <= 0 {
			return apperror.UnprocessableEntity("incoming_dn_item_id is required for incoming_qc")
		}

		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", *task.IncomingDNItemID).Error; err != nil {
			return apperror.NotFound("incoming_dn_item not found")
		}

		if numberOfDefects > dnItem.QtyReceived {
			return apperror.UnprocessableEntity("number_of_defects must be <= qty_received")
		}

		approvedQty := dnItem.QtyReceived - numberOfDefects
		ngQty := numberOfDefects

		// Update task fields
		if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":        "approved",
			"good_quantity": approvedQty,
			"ng_quantity":   ngQty,
			"date_checked":  checkedAt,
			"updated_at":    time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("update qc_tasks: %w", err)
		}

		// Update DN item quality
		if err := tx.Model(&procModels.IncomingDNItem{}).Where("id = ?", dnItem.ID).Update("quality_status", "Approved").Error; err != nil {
			return fmt.Errorf("update delivery_note_items: %w", err)
		}

		event := map[string]interface{}{
			"event":            "qc_approve",
			"approved_qty":     approvedQty,
			"ng_qty":           ngQty,
			"number_of_defects": numberOfDefects,
			"date_checked":     dateChecked,
			"performed_by":     performedBy,
			"occurred_at":      time.Now().UTC().Format(time.RFC3339),
		}
		if err := appendRoundEventTx(tx, taskID, event); err != nil {
			return err
		}

		dnType, poNumber, err := loadDNTypeAndPONumber(tx, dnItem.IncomingDNID)
		if err != nil {
			return err
		}
		if approvedQty > 0 {
			if err := r.postToInventoryByDNType(tx, dnType, dnItem.ItemUniqCode, approvedQty, dnItem.WeightReceived, dnItem.Uom, performedBy); err != nil {
				return err
			}
			_ = poNumber // reserved for future PO rollup
		}

		return nil
	})
}

func (r *repo) RejectIncoming(ctx context.Context, taskID int64, numberOfDefects int, dateChecked string, performedBy string) error {
	performedBy = normalizeActor(performedBy)

	if numberOfDefects < 0 {
		return apperror.BadRequest("number_of_defects must be >= 0")
	}

	checkedAt, err := time.Parse("2006-01-02", dateChecked)
	if err != nil {
		return apperror.BadRequest("date_checked must be in YYYY-MM-DD format")
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
		if task.IncomingDNItemID == nil || *task.IncomingDNItemID <= 0 {
			return apperror.UnprocessableEntity("incoming_dn_item_id is required for incoming_qc")
		}

		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", *task.IncomingDNItemID).Error; err != nil {
			return apperror.NotFound("incoming_dn_item not found")
		}

		if numberOfDefects > dnItem.QtyReceived {
			return apperror.UnprocessableEntity("number_of_defects must be <= qty_received")
		}

		if err := tx.Model(&qcModels.QCTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":       "rejected",
			"ng_quantity":  numberOfDefects,
			"date_checked": checkedAt,
			"updated_at":   time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("update qc_tasks: %w", err)
		}
		if err := tx.Model(&procModels.IncomingDNItem{}).Where("id = ?", dnItem.ID).Update("quality_status", "Rejected").Error; err != nil {
			return fmt.Errorf("update delivery_note_items: %w", err)
		}

		event := map[string]interface{}{
			"event":             "qc_reject",
			"number_of_defects": numberOfDefects,
			"date_checked":      dateChecked,
			"performed_by":      performedBy,
			"occurred_at":       time.Now().UTC().Format(time.RFC3339),
		}
		return appendRoundEventTx(tx, taskID, event)
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

func loadDNTypeAndPONumber(tx *gorm.DB, dnID int64) (dnType string, poNumber string, err error) {
	if !tableExists(tx, "delivery_notes") {
		return "", "", apperror.UnprocessableEntity("delivery_notes table not found; apply migration 0018_create_delivery_note_up.sql and 0015_dn_feature_up.sql")
	}
	var row struct {
		DnType   string `gorm:"column:type"`
		PoNumber string `gorm:"column:po_number"`
	}
	if err := tx.Table("delivery_notes").Select("type, po_number").Where("id = ?", dnID).Limit(1).Scan(&row).Error; err != nil {
		return "", "", fmt.Errorf("load delivery_notes: %w", err)
	}
	return row.DnType, row.PoNumber, nil
}

func normalizeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}
