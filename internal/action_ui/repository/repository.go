package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	procModels "github.com/ganasa18/go-template/internal/procurement/models"
	qcModels "github.com/ganasa18/go-template/internal/qc/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IRepository interface {
	CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error)
}

type repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repo{db: db}
}

func (r *repo) CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error) {
	if req.DeltaQty <= 0 {
		return nil, false, fmt.Errorf("delta_qty must be > 0")
	}

	// Idempotency: if the client_event_id already exists, return the existing state.
	var existingScan models.IncomingReceivingScan
	err := r.db.WithContext(ctx).
		Where("idempotency_key = ?", req.ClientEventID).
		Limit(1).
		Find(&existingScan).Error
	if err != nil {
		return nil, false, fmt.Errorf("check idempotency: %w", err)
	}
	if existingScan.ID != 0 {
		// Return current DN item + QC task id.
		var dnItem procModels.IncomingDNItem
		if err := r.db.WithContext(ctx).First(&dnItem, "id = ?", req.DNItemID).Error; err != nil {
			return nil, true, fmt.Errorf("load dn item: %w", err)
		}

		qcID, err := r.getOrCreateIncomingQCTask(ctx, req.DNItemID)
		if err != nil {
			return nil, true, err
		}

		resp := &models.IncomingScanResponse{
			QCTaskID: qcID,
			DNItem: models.IncomingScanDNItem{
				ID:             dnItem.ID,
				ItemUniqCode:   dnItem.ItemUniqCode,
				PackingNumber:  dnItem.PackingNumber,
				QtyReceived:    dnItem.QtyReceived,
				WeightReceived: dnItem.WeightReceived,
				QualityStatus:  dnItem.QualityStatus,
				ReceivedAt:     dnItem.ReceivedAt,
				UOM:            dnItem.Uom,
			},
		}
		return resp, true, nil
	}

	var out *models.IncomingScanResponse
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", req.DNItemID).Error; err != nil {
			return fmt.Errorf("load dn item: %w", err)
		}
		if req.DNID != "" && dnItem.IncomingDNID != req.DNID {
			return fmt.Errorf("dn_item_id does not belong to dn_id")
		}

		// Insert receiving scan (append-only)
		idk := req.ClientEventID
		scan := models.IncomingReceivingScan{
			IncomingDNItemID: req.DNItemID,
			IdempotencyKey:   &idk,
			ScanRef:          req.PackingNumber,
			Qty:              float64(req.DeltaQty),
			WeightKg:         req.DeltaWeightKg,
			ScannedAt:        time.Now(),
			ScannedBy:        &scannedBy,
		}
		if err := tx.Create(&scan).Error; err != nil {
			return fmt.Errorf("insert incoming_receiving_scans: %w", err)
		}

		// Update aggregate on DN item
		updates := map[string]interface{}{}
		updates["qty_received"] = gorm.Expr("qty_received + ?", req.DeltaQty)
		updates["received_at"] = time.Now()
		if req.PackingNumber != "" {
			updates["packing_number"] = req.PackingNumber
		}
		if req.UOM != "" {
			updates["uom"] = req.UOM
		}
		if req.DeltaWeightKg != nil {
			updates["weight_received"] = gorm.Expr("COALESCE(weight_received, 0) + ?", *req.DeltaWeightKg)
		}
		if err := tx.Model(&procModels.IncomingDNItem{}).
			Where("id = ?", req.DNItemID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update incoming_dn_items: %w", err)
		}

		// Reload updated dn item
		if err := tx.First(&dnItem, "id = ?", req.DNItemID).Error; err != nil {
			return fmt.Errorf("reload dn item: %w", err)
		}

		qcID, err := r.getOrCreateIncomingQCTaskTx(ctx, tx, req.DNItemID)
		if err != nil {
			return err
		}

		out = &models.IncomingScanResponse{
			QCTaskID: qcID,
			DNItem: models.IncomingScanDNItem{
				ID:             dnItem.ID,
				ItemUniqCode:   dnItem.ItemUniqCode,
				PackingNumber:  dnItem.PackingNumber,
				QtyReceived:    dnItem.QtyReceived,
				WeightReceived: dnItem.WeightReceived,
				QualityStatus:  dnItem.QualityStatus,
				ReceivedAt:     dnItem.ReceivedAt,
				UOM:            dnItem.Uom,
			},
		}
		return nil
	}); err != nil {
		return nil, false, err
	}

	return out, false, nil
}

func (r *repo) getOrCreateIncomingQCTask(ctx context.Context, dnItemID string) (int64, error) {
	return r.getOrCreateIncomingQCTaskTx(ctx, r.db.WithContext(ctx), dnItemID)
}

func (r *repo) getOrCreateIncomingQCTaskTx(ctx context.Context, tx *gorm.DB, dnItemID string) (int64, error) {
	var task qcModels.QCTask
	err := tx.Where("task_type = ? AND incoming_dn_item_id = ?", "incoming_qc", dnItemID).Limit(1).Find(&task).Error
	if err != nil {
		return 0, fmt.Errorf("load qc task: %w", err)
	}
	if task.ID != 0 {
		return task.ID, nil
	}

	// Create new task.
	newTask := qcModels.QCTask{
		TaskType:         "incoming_qc",
		Status:           "pending",
		IncomingDNItemID: &dnItemID,
		Round:            1,
		RoundResults:     qcModels.EmptyJSONArray(),
	}
	if err := tx.Create(&newTask).Error; err != nil {
		return 0, fmt.Errorf("create qc task: %w", err)
	}
	if newTask.ID == 0 {
		return 0, errors.New("failed to create qc task")
	}
	return newTask.ID, nil
}
