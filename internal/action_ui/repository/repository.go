package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	procModels "github.com/ganasa18/go-template/internal/procurement/models"
	qcModels "github.com/ganasa18/go-template/internal/qc/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IRepository interface {
	// LookupByPackingNumber resolves packing_number → DN context for UI auto-fill.
	// Called when QR is scanned, before the operator fills qty/weight.
	LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error)

	// CreateIncomingScan processes the scan submission. Resolves dn_item internally
	// by packing_number + item_uniq_code — caller does not need bigint IDs.
	CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error)
}

type repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repo{db: db}
}

// LookupByPackingNumber finds a delivery_note_item by dn_number (= packing_number) and returns
// full DN context (po_number, supplier, dn_type, dn_number) for UI auto-fill.
// packing_number in this context = delivery_notes.dn_number.
func (r *repo) LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error) {
	if strings.TrimSpace(packingNumber) == "" {
		return nil, fmt.Errorf("packing_number is required")
	}

	q := r.db.WithContext(ctx).Model(&procModels.IncomingDNItem{}).
		Joins("JOIN delivery_notes dn ON dn.id = delivery_note_items.dn_id").
		Where("dn.dn_number = ?", packingNumber)
	if strings.TrimSpace(itemUniqCode) != "" {
		q = q.Where("delivery_note_items.item_uniq_code = ?", itemUniqCode)
	}

	var dnItem procModels.IncomingDNItem
	if err := q.First(&dnItem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("packing_number not found")
		}
		return nil, fmt.Errorf("lookup packing_number: %w", err)
	}

	return r.buildIncomingScanDNItem(ctx, r.db.WithContext(ctx), dnItem)
}

func (r *repo) CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error) {
	scannedBy = normalizeActor(scannedBy)

	if req.DeltaQty <= 0 {
		return nil, false, fmt.Errorf("delta_qty must be > 0")
	}

	// Idempotency: if the client_event_id already exists, return the existing state.
	var existingScan models.IncomingReceivingScan
	if err := r.db.WithContext(ctx).
		Where("idempotency_key = ?", req.ClientEventID).
		Limit(1).
		Find(&existingScan).Error; err != nil {
		return nil, false, fmt.Errorf("check idempotency: %w", err)
	}
	if existingScan.ID != 0 {
		// Reload DN item from existing scan FK
		var dnItem procModels.IncomingDNItem
		if err := r.db.WithContext(ctx).First(&dnItem, "id = ?", existingScan.IncomingDNItemID).Error; err != nil {
			return nil, true, fmt.Errorf("load dn item: %w", err)
		}
		qcID, err := r.getOrCreateIncomingQCTask(ctx, dnItem.ID)
		if err != nil {
			return nil, true, err
		}
		dnItemResp, _ := r.buildIncomingScanDNItem(ctx, r.db.WithContext(ctx), dnItem)
		return &models.IncomingScanResponse{QCTaskID: qcID, DNItem: *dnItemResp}, true, nil
	}

	// Resolve delivery_note_item by dn_number (= packing_number) + item_uniq_code
	var resolvedItem procModels.IncomingDNItem
	lookupQ := r.db.WithContext(ctx).
		Model(&procModels.IncomingDNItem{}).
		Joins("JOIN delivery_notes dn ON dn.id = delivery_note_items.dn_id").
		Where("dn.dn_number = ? AND delivery_note_items.item_uniq_code = ?", req.PackingNumber, req.ItemUniqCode)
	if err := lookupQ.First(&resolvedItem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, apperror.NotFound(fmt.Sprintf("packing_number '%s' with item '%s' not found", req.PackingNumber, req.ItemUniqCode))
		}
		return nil, false, fmt.Errorf("lookup dn item: %w", err)
	}

	var out *models.IncomingScanResponse
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dnItem procModels.IncomingDNItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dnItem, "id = ?", resolvedItem.ID).Error; err != nil {
			return fmt.Errorf("lock dn item: %w", err)
		}

		// Validate delta_qty does not exceed remaining qty
		remaining := dnItem.OrderQty - dnItem.QtyReceived
		if req.DeltaQty > remaining {
			return apperror.UnprocessableEntity(fmt.Sprintf("delta_qty %d exceeds remaining qty %d (ordered %d, received %d)", req.DeltaQty, remaining, dnItem.OrderQty, dnItem.QtyReceived))
		}

		// Insert receiving scan (append-only)
		idk := req.ClientEventID
		scan := models.IncomingReceivingScan{
			IncomingDNItemID: dnItem.ID,
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
		updates := map[string]interface{}{
			"qty_received": gorm.Expr("COALESCE(qty_received, 0) + ?", req.DeltaQty),
			"received_at":  time.Now(),
			"uom":          req.UOM,
		}
		if req.DeltaWeightKg != nil {
			updates["weight_received"] = gorm.Expr("COALESCE(weight_received, 0) + ?", *req.DeltaWeightKg)
		}
		if err := tx.Model(&procModels.IncomingDNItem{}).
			Where("id = ?", dnItem.ID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update delivery_note_items: %w", err)
		}

		// Reload updated dn item
		if err := tx.First(&dnItem, "id = ?", dnItem.ID).Error; err != nil {
			return fmt.Errorf("reload dn item: %w", err)
		}

		qcID, err := r.getOrCreateIncomingQCTaskTx(ctx, tx, dnItem.ID)
		if err != nil {
			return err
		}

		// Write incoming_scan movement log — audit trail before QC approval
		if err := r.writeScanMovementLogTx(tx, dnItem, req, scannedBy); err != nil {
			return err
		}

		dnItemResp, err := r.buildIncomingScanDNItem(ctx, tx, dnItem)
		if err != nil {
			pn := req.PackingNumber
			dnItemResp = &models.IncomingScanDNItem{
				ID:             dnItem.ID,
				ItemUniqCode:   dnItem.ItemUniqCode,
				PackingNumber:  &pn,
				QtyOrdered:     dnItem.Quantity,
				QtyReceived:    dnItem.QtyReceived,
				QtyRemaining:   dnItem.Quantity - dnItem.QtyReceived,
				WeightKg:       dnItem.Weight,
				WeightReceived: dnItem.WeightReceived,
				QualityStatus:  dnItem.QualityStatus,
				ReceivedAt:     dnItem.ReceivedAt,
				UOM:            dnItem.Uom,
			}
		}

		out = &models.IncomingScanResponse{
			QCTaskID: qcID,
			DNItem:   *dnItemResp,
		}
		return nil
	}); err != nil {
		return nil, false, err
	}

	return out, false, nil
}

func (r *repo) getOrCreateIncomingQCTask(ctx context.Context, dnItemID int64) (int64, error) {
	return r.getOrCreateIncomingQCTaskTx(ctx, r.db.WithContext(ctx), dnItemID)
}

func (r *repo) getOrCreateIncomingQCTaskTx(ctx context.Context, tx *gorm.DB, dnItemID int64) (int64, error) {
	var task qcModels.QCTask
	// Only reuse an open (non-terminal) task — if the previous one was approved/rejected, create a new one.
	if err := tx.Where("task_type = ? AND incoming_dn_item_id = ? AND status IN ?", "incoming_qc", dnItemID, []string{"pending", "in_progress"}).
		Order("id DESC").Limit(1).Find(&task).Error; err != nil {
		return 0, fmt.Errorf("load qc task: %w", err)
	}
	if task.ID != 0 {
		return task.ID, nil
	}

	// Determine next round number — max existing round + 1
	var maxRound int
	tx.Table("qc_tasks").
		Where("task_type = ? AND incoming_dn_item_id = ?", "incoming_qc", dnItemID).
		Select("COALESCE(MAX(round), 0)").
		Scan(&maxRound)

	newTask := qcModels.QCTask{
		TaskType:         "incoming_qc",
		Status:           "pending",
		IncomingDNItemID: &dnItemID,
		Round:            maxRound + 1,
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

// buildIncomingScanDNItem constructs IncomingScanDNItem with PO, supplier, and DN type context.
func (r *repo) buildIncomingScanDNItem(ctx context.Context, tx *gorm.DB, dnItem procModels.IncomingDNItem) (*models.IncomingScanDNItem, error) {
	resp := &models.IncomingScanDNItem{
		ID:             dnItem.ID,
		ItemUniqCode:   dnItem.ItemUniqCode,
		PackingNumber:  dnItem.PackingNumber,
		QtyOrdered:     dnItem.Quantity,
		QtyReceived:    dnItem.QtyReceived,
		QtyRemaining:   dnItem.Quantity - dnItem.QtyReceived,
		WeightKg:       dnItem.Weight,
		WeightReceived: dnItem.WeightReceived,
		QualityStatus:  dnItem.QualityStatus,
		ReceivedAt:     dnItem.ReceivedAt,
		UOM:            dnItem.Uom,
	}

	var dn procModels.IncomingDN
	if err := tx.WithContext(ctx).First(&dn, "id = ?", dnItem.IncomingDNID).Error; err != nil {
		return resp, nil
	}

	resp.PoNumber = &dn.PoNumber
	resp.DnNumber = &dn.DnNumber
	resp.PackingNumber = &dn.DnNumber // packing_number = dn_number (scanned QR/kanban)
	dnTypeLabel := mapDnTypeLabel(dn.DnType)
	resp.RawMaterialType = &dnTypeLabel

	supplierID := dn.SupplierID
	if supplierID == nil && dn.PoNumber != "" {
		// Fallback: delivery_notes.supplier_id may be unset for older DNs — resolve via PO.
		var po procModels.PurchaseOrder
		if err := tx.WithContext(ctx).Select("supplier_id").Where("po_number = ?", dn.PoNumber).First(&po).Error; err == nil {
			supplierID = po.SupplierID
		}
	}
	if supplierID != nil {
		var name string
		tx.WithContext(ctx).Table("suppliers").Select("supplier_name").Where("id = ?", *supplierID).Scan(&name)
		if name != "" {
			resp.SupplierName = &name
		}
	}

	return resp, nil
}

// mapDnTypeLabel converts DB type codes (RM, INDIRECT, SUBCON) to UI-friendly labels.
func mapDnTypeLabel(dnType string) string {
	switch strings.ToUpper(strings.TrimSpace(dnType)) {
	case "RM":
		return "raw_material"
	case "IB":
		return "indirect"
	case "SC":
		return "subcon"
	default:
		return dnType
	}
}

func normalizeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}

// writeScanMovementLogTx inserts an inventory_movement_logs entry (source_flag=incoming_scan)
// when a scan is submitted. This records the scan audit trail before QC approval.
func (r *repo) writeScanMovementLogTx(tx *gorm.DB, dnItem procModels.IncomingDNItem, req models.IncomingScanRequest, scannedBy string) error {
	var dn procModels.IncomingDN
	if err := tx.First(&dn, "id = ?", dnItem.IncomingDNID).Error; err != nil {
		// Non-fatal: if DN not found just skip the log
		return nil
	}
	movCat := dnTypeToMovementCategory(dn.DnType)
	if movCat == "" {
		return nil
	}
	sf := "incoming_scan"
	lb := scannedBy
	return tx.Create(&invModels.InventoryMovementLog{
		MovementCategory: movCat,
		MovementType:     "incoming",
		UniqCode:         dnItem.ItemUniqCode,
		QtyChange:        float64(req.DeltaQty),
		WeightChange:     req.DeltaWeightKg,
		SourceFlag:       &sf,
		DNNumber:         &dn.DnNumber,
		LoggedBy:         &lb,
		LoggedAt:         time.Now(),
	}).Error
}

// dnTypeToMovementCategory maps delivery_notes.type to inventory_movement_logs.movement_category.
func dnTypeToMovementCategory(dnType string) string {
	switch strings.ToUpper(strings.TrimSpace(dnType)) {
	case "RM", "RAW MATERIAL", "RAW_MATERIAL":
		return "raw_material"
	case "IB", "INDIRECT", "INDIRECT RAW MATERIAL", "INDIRECT_RAW_MATERIAL":
		return "indirect_raw_material"
	case "SC", "SUBCON":
		return "subcon"
	}
	return ""
}
