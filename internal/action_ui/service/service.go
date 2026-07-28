package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/internal/action_ui/repository"
	scrapModels "github.com/ganasa18/go-template/internal/scrap_stock/models"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/inventoryconst"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IService interface {
	// =============================
	// Incoming (existing)
	// =============================
	LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error)
	CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error)

	// =============================
	// Production Flow
	// =============================

	// 🔹 Get context after scan QR (auto fill UI)
	ScanContext(ctx context.Context, woNumber string) (*dto.ScanContextResponse, error)
	ScanContextMachine(ctx context.Context, machineID string) (*models.MasterMachine, error)

	// 🔹 Start production (scan in)
	ScanIn(ctx context.Context, req dto.ScanInRequest) error

	ScanOut(ctx context.Context, req dto.ScanOutRequest) error

	CompleteProduction(ctx context.Context, woID int64) error

	QCSubmit(ctx context.Context, req dto.QCSubmitRequest, performedBy string) error

	QCFinish(ctx context.Context, req dto.QCFinishRequest, performedBy string) error

	ListQCTask(ctx context.Context, req dto.ListQCTaskRequest) (map[string]interface{}, error)

	IssueList(ctx context.Context) (map[string]interface{}, error)
	WOList(ctx context.Context, search string) ([]dto.WOListItem, error)
	WODetail(ctx context.Context, woNumber string) (*dto.WODetailResponse, error)
	RawMaterialLookup(ctx context.Context, code string) (*dto.RawMaterialLookupResponse, error)

	ScanMachine(ctx context.Context, req dto.ScanMachineRequest) (*dto.ScanMachineResponse, error)

	ScanOutContext(ctx context.Context, woNumber string) (*dto.ScanOutContextResponse, error)

	// =============================
	// Product Return (BRD)
	// =============================
	ScanReturn(ctx context.Context, req models.ScanReturnRequest) (*models.ScanReturnResponse, error)
	SubmitReturnToQC(ctx context.Context, req models.SubmitReturnToQCRequest, submittedBy string) (*models.ProductReturnRow, error)
	PendingReturnTasks(ctx context.Context) ([]models.PendingReturnTask, error)
	SubmitReturnValidation(ctx context.Context, req models.SubmitReturnValidationRequest, validatedBy string) error
}

type service struct {
	repo           repository.IRepository
	repoProduction repository.IProductionRepository
	repoIncoming   repository.IIncomingRepository
	db             *gorm.DB
}

func New(repo repository.IRepository, repoProduction repository.IProductionRepository, repoIncoming repository.IIncomingRepository, db *gorm.DB) IService {
	return &service{repo: repo, repoProduction: repoProduction, repoIncoming: repoIncoming, db: db}
}

func (s *service) LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error) {
	return s.repo.LookupByPackingNumber(ctx, packingNumber, itemUniqCode)
}

func (s *service) CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error) {
	return s.repo.CreateIncomingScan(ctx, req, scannedBy)
}

func (s *service) ScanContext(ctx context.Context, woNumber string) (*dto.ScanContextResponse, error) {

	// =============================
	// 🔍 1. GET WO
	// =============================
	woItems, err := s.repoProduction.FindWOByKanbanNumber(ctx, woNumber)
	if err != nil {
		return nil, err
	}

	wo, err := s.repoProduction.FindWOByID(ctx, woItems.WOID)
	if err != nil {
		return nil, err
	}
	// =============================
	// 🔍 2. GET ALL ITEMS
	// =============================
	items, err := s.repoProduction.FindWOItemsByWOID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}

	// 👉 ambil item utama (sementara pakai pertama)
	item := items[0]

	// =============================
	// 🏭 MACHINE (OPTIONAL)
	// =============================
	var machineID string
	var productionLine string

	if item.MachineID != 0 {
		m, err := s.repoProduction.FindMachineByID(ctx, item.MachineID)
		if err == nil {
			machineID = strconv.Itoa(m.ID)
			productionLine = m.ProductionLine
		}
	}

	// =============================
	// 🔄 PROCESS FLOW
	// =============================
	flow, err := parseProcessFlow(item.ProcessFlowJSON)
	if err != nil {
		return nil, err
	}

	if err := validateStep(item.CurrentStepSeq, flow); err != nil {
		return nil, err
	}

	var m int64 = 0

	if wo.ID != 0 {
		m, err = s.repoProduction.CountQCLogs(ctx, wo.ID)
	}

	totalStep := len(flow)
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)

	currentProcess := flow[currentIndex].ProcessName

	var nextProcess string
	if currentIndex+1 < totalStep {
		nextProcess = flow[currentIndex+1].ProcessName
	}

	// =============================
	// 🔥 RAW MATERIAL (ALL ITEMS)
	// =============================
	rawMaterials := make([]dto.ScanContextRawMaterial, 0)

	for _, it := range items {
		rawMaterials = append(rawMaterials, dto.ScanContextRawMaterial{
			Uniq:        it.ItemUniqCode,
			PartName:    it.PartName,
			PartNumber:  it.PartNumber,
			UOM:         it.UOM,
			Qty:         it.Quantity,
			ProcessName: it.ProcessName,
		})
	}

	// =============================
	// 🎯 RESPONSE
	// =============================
	return &dto.ScanContextResponse{
		WOID:           wo.ID,
		WONumber:       wo.WONumber,
		Uniq:           item.ItemUniqCode,
		MachineID:      machineID,
		ProductionLine: productionLine,
		ProcessName:    currentProcess,
		NextProcess:    nextProcess,
		CurrentStep:    item.CurrentStepSeq,
		TotalStep:      totalStep,
		CurrentQCStep:  m,
		TotalQCStep:    totalStep * 3,
		PartName:       item.PartName,
		PartNumber:     item.PartNumber,
		KanbanNumber:   item.KanbanNumber,
		UOM:            item.UOM,
		Status:         item.Status,
		// RawMaterials:   rawMaterials,
	}, nil
}

func (s *service) ScanContextMachine(ctx context.Context, machineID string) (*models.MasterMachine, error) {
	var machine models.MasterMachine

	err := s.db.WithContext(ctx).
		Model(&models.MasterMachine{}).
		Where("machine_number = ?", machineID).
		First(&machine).Error
	if err != nil {
		return nil, err
	}

	return &machine, nil
}

func (s *service) ScanIn(ctx context.Context, req dto.ScanInRequest) error {

	item, err := s.repoProduction.FindWOItemByID(ctx, req.WOItemID)
	if err != nil {
		return errors.New("wo item tidak ditemukan")
	}

	// =====================================
	// PROCESS FLOW
	// =====================================
	flow := resolveProcessFlow(item)
	totalStep := len(flow)
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
	currentProcess := flow[currentIndex].ProcessName
	currentStep := flow[currentIndex]

	// =====================================
	// VALIDASI
	// =====================================
	if item.Status == "FINISHED" || item.Status == "DONE" {
		return errors.New("item already finished")
	}

	if item.Status == "WAITING_FINAL_QC" {
		return errors.New("cannot scan in before final qc completed")
	}

	if item.ScanInCount > item.ScanOutCount {
		now := time.Now()
		if err := s.repoProduction.DeleteRawMaterialLogsByWOItemID(ctx, item.ID); err != nil {
			return err
		}
		if err := s.saveRawMaterialLogs(ctx, item, req.RawMaterials, req.ScannedBy, now); err != nil {
			return err
		}
		return nil
	}

	// =====================================
	// MACHINE
	// =====================================
	var machineID int64
	var productionLine string

	if item.MachineID != 0 {
		m, err := s.repoProduction.FindMachineByID(ctx, item.MachineID)
		if err == nil {
			machineID = int64(m.ID)
			productionLine = m.ProductionLine
		}
	}

	now := time.Now()

	// =====================================
	// INSERT SCAN LOG
	// =====================================
	log := models.ProductionScanLog{
		UUID:           uuid.New().String(),
		WOID:           item.WOID,
		WOItemID:       item.ID,
		MachineID:      machineID,
		KanbanNumber:   item.KanbanNumber,
		ProcessName:    currentProcess,
		ProductionLine: productionLine,
		ScanType:       "SCAN_IN",
		QtyInput:       req.Qty,
		Shift:          req.Shift,
		DandoriTime:    req.DandoriTime,
		SetupQCTime:    req.SetupQCTime,
		ScannedBy:      req.ScannedBy,
		ScannedAt:      now,
		CreatedAt:      now,
	}

	if err := s.repoProduction.InsertScanLog(ctx, log); err != nil {
		return err
	}

	if err := s.saveRawMaterialLogs(ctx, item, req.RawMaterials, req.ScannedBy, now); err != nil {
		return err
	}

	// =====================================
	// CREATE WIP HEADER
	// =====================================
	wip, err := s.repoProduction.FindOrCreateWIP(ctx, item.WOID)
	if err != nil {
		return err
	}

	// =====================================
	// FIND QUEUE WIP ITEM (hasil dari process sebelumnya)
	// =====================================
	wipItem, err := s.repoProduction.FindQueueWIPItem(
		ctx,
		wip.ID,
		item.ItemUniqCode,
		currentProcess,
	)

	if err != nil {

		// kalau belum ada, create fresh (process pertama)
		wipItem = models.WIPItem{
			WipID:         wip.ID,
			Uniq:          item.ItemUniqCode,
			PackingNumber: item.KanbanNumber,
			WipType:       "production",

			ProcessName: currentProcess,
			MachineName: derefString(currentStep.MachineName),
			OpSeq:       currentStep.OpSeq,
			Seq:         currentIndex + 1,

			UOM: item.UOM,

			Stock: int(req.Qty),

			QtyIn:        int(req.Qty),
			QtyOut:       0,
			QtyRemaining: int(req.Qty),

			Status: "process",

			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := s.repoProduction.CreateWIPItem(ctx, &wipItem); err != nil {
			return err
		}

	} else {
		// kalau sudah ada queue -> ubah ke process
		wipItem.Status = "process"
		wipItem.UpdatedAt = now

		if err := s.repoProduction.UpdateWIPItem(ctx, &wipItem); err != nil {
			return err
		}
	}

	// =====================================
	// WIP LOG
	// =====================================
	_ = s.repoProduction.CreateWIPLog(ctx, &models.WIPLog{
		WipItemID: wipItem.ID,
		Action:    "SCAN_IN",
		Qty:       int(req.Qty),
		CreatedAt: now,
	})
	// =====================================
	// PRODUCT ISSUE
	// =====================================
	if req.ProductIssue {
		issue := models.ProductionIssue{
			UUID:           uuid.New().String(),
			WOID:           item.WOID,
			WOItemID:       item.ID,
			MachineID:      machineID,
			ProcessName:    currentProcess,
			ProductionLine: productionLine,
			IssueType:      req.ProductIssueType,
			IssueDuration:  req.ProductIssueDuration,
			QtyAffected:    req.Qty,
			ReportedBy:     req.ScannedBy,
			ReportedAt:     now,
			CreatedAt:      now,
		}

		if err := s.repoProduction.InsertProductIssue(ctx, issue); err != nil {
			return err
		}
	}

	// =====================================
	// UPDATE ITEM
	// =====================================
	item.Status = "IN_PROGRESS"
	item.ScanInCount++
	item.LastScannedProcess = currentProcess

	if err := s.createQCTaskIfNeeded(ctx, item, log, req); err != nil {
		return err
	}

	if err := s.repoProduction.UpdateWOStatus(ctx, item.WOID, "In Progress"); err != nil {
		return err
	}

	return nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (s *service) createQCTaskIfNeeded(ctx context.Context, item models.WorkOrderItem, log models.ProductionScanLog, req dto.ScanInRequest) error {

	// =============================
	// Anti duplicate pending task
	// =============================
	exist, err := s.repoProduction.IsQCPendingExist(
		ctx,
		item.ID,
		log.ProcessName,
	)

	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	// =============================
	// Build JSON payload
	// =============================
	payload := map[string]interface{}{
		"uniq":            item.ItemUniqCode,
		"kanban_number":   item.KanbanNumber,
		"process_name":    log.ProcessName,
		"production_line": log.ProductionLine,
		"wo_id":           item.WOID,
		"wo_item_id":      item.ID,
		"qty":             log.QtyInput,
	}

	raw, _ := json.Marshal(payload)

	// =============================
	// Create QC Task
	// =============================
	qc := models.QCTask{
		TaskType: "production_qc",
		Status:   "pending",

		WOID:     &item.WOID,
		WOItemID: &item.ID,

		ProcessName: log.ProcessName,

		Round: 1,

		RoundResults: datatypes.JSON(raw),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.repoProduction.CreateQC(ctx, &qc)
}

func (s *service) ScanOut(ctx context.Context, req dto.ScanOutRequest) error {

	item, err := s.repoProduction.FindWOItemByID(ctx, req.WOItemID)
	if err != nil {
		return errors.New("wo item tidak ditemukan")
	}

	// =====================================
	// PROCESS FLOW
	// =====================================
	flow := resolveProcessFlow(item)
	totalStep := len(flow)
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
	currentProcess := flow[currentIndex].ProcessName

	// =====================================
	// VALIDASI BASIC
	// =====================================
	var scanInLogCount, scanOutLogCount int64

	if err := s.db.WithContext(ctx).
		Model(&models.ProductionScanLog{}).
		Where("wo_item_id = ? AND process_name = ? AND scan_type = ?",
			item.ID, currentProcess, "SCAN_IN").
		Count(&scanInLogCount).Error; err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).
		Model(&models.ProductionScanLog{}).
		Where("wo_item_id = ? AND process_name = ? AND scan_type = ?",
			item.ID, currentProcess, "SCAN_OUT").
		Count(&scanOutLogCount).Error; err != nil {
		return err
	}

	if scanInLogCount <= scanOutLogCount {
		return errors.New("please scan in first")
	}

	_ = currentProcess

	// =====================================
	// [no-qc-round-gate] VALIDASI QC ROUND DIHILANGKAN
	// =====================================
	// Sebelumnya Scan Out diblokir sampai QC round 2 berstatus APPROVE
	// (efeknya di layar: harus sampai / submit round 3 di QC Process).
	// Atas permintaan pengguna, pembatasan itu dihapus agar Scan Out
	// tidak lagi bergantung pada progres QC.
	//
	// Alur QC Process sendiri tidak diubah: task QC tetap dibuat dan
	// round 1/2/3 tetap berjalan seperti biasa.

	// =====================================
	// CEK SUDAH PERNAH SCAN OUT BELUM
	// =====================================
	if item.Status == "FINISHED" {
		return errors.New("already scan out, production finished")
	}

	// =====================================
	// MACHINE OPTIONAL
	// =====================================
	var machineID int64
	var productionLine string

	if item.MachineID != 0 {
		m, err := s.repoProduction.FindMachineByID(ctx, item.MachineID)
		if err == nil {
			machineID = int64(m.ID)
			productionLine = m.ProductionLine
		}
	}

	now := time.Now()

	qtyOut := req.QtyOutput
	if req.TotalProduction > 0 {
		qtyOut = req.TotalProduction
	}

	// =====================================
	// INSERT LOG
	// =====================================
	log := models.ProductionScanLog{
		UUID:           uuid.New().String(),
		WOID:           item.WOID,
		WOItemID:       item.ID,
		MachineID:      machineID,
		KanbanNumber:   item.KanbanNumber,
		ProcessName:    currentProcess,
		ProductionLine: productionLine,
		ScanType:       "SCAN_OUT",

		QtyOutput: qtyOut,

		QtyRMUsed: sumScanOutRM(req.RawMaterials),
		NGMachine: 0,
		NGProcess: 0,
		QtyScrap:  0,
		QtyRework: 0,

		Shift:     req.Shift,
		ScannedBy: req.ScannedBy,
		ScannedAt: now,
		CreatedAt: now,
		Warehouse: req.Warehouse,
	}

	if err := s.repoProduction.InsertScanLog(ctx, log); err != nil {
		return err
	}

	woRef, _ := s.repoProduction.FindWOByID(ctx, item.WOID)
	if err := s.consumeRawMaterials(ctx, item, req.RawMaterials, req.ScannedBy, woRef.WONumber, now); err != nil {
		return err
	}

	// =====================================
	// UPDATE ITEM
	// =====================================
	item.ScanOutCount++
	item.TotalGoodQty = qtyOut
	item.LastScannedProcess = currentProcess

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.advanceProcessAfterScanOut(ctx, tx, &item, qtyOut, req.ScannedBy)
	})
}

func (s *service) CompleteProduction(ctx context.Context, woID int64) error {
	items, err := s.repoProduction.FindWOItemsByWOID(ctx, woID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("work order tidak memiliki item")
	}

	for _, it := range items {
		st := strings.ToUpper(it.Status)
		if st != "FINISHED" && st != "DONE" {
			return errors.New("tidak bisa menyelesaikan WO: masih ada uniq yang belum selesai (uniq " + it.ItemUniqCode + " berstatus " + it.Status + ")")
		}
	}

	// 1) tandai WO selesai
	if err := s.repoProduction.UpdateWOStatus(ctx, woID, "Completed"); err != nil {
		return err
	}

	// 2) 🔥 sinkronkan WIP: semua wip_items milik WO ini ikut "done"
	//    sehingga hilang dari board Work in Progress di Raigine.
	if err := s.repoProduction.MarkWIPDoneByWO(ctx, woID); err != nil {
		return err
	}

	// Pre-processing: when a flagged RM processing WO completes, reduce source RM
	// stock and register the processed target uniq into raw material inventory.
	if err := s.applyRMPreProcessing(ctx, woID); err != nil {
		return err
	}

	return nil
}

func (s *service) applyRMPreProcessing(ctx context.Context, woID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.applyRMProcessingTx(ctx, tx, woID)
	})
}

func (s *service) applyRMProcessingTx(ctx context.Context, tx *gorm.DB, woID int64) error {
	var hdr struct {
		WoNumber   string   `gorm:"column:wo_number"`
		WOKind     string   `gorm:"column:wo_kind"`
		SourceUniq *string  `gorm:"column:source_material_uniq"`
		TargetUniq *string  `gorm:"column:target_material_uniq"`
		InputQty   *float64 `gorm:"column:input_qty"`
		OutputQty  *float64 `gorm:"column:output_qty"`
		OutputUOM  *string  `gorm:"column:output_uom"`
	}
	if err := tx.WithContext(ctx).Table("work_orders").
		Select("wo_number, wo_kind, source_material_uniq, target_material_uniq, input_qty, output_qty, output_uom").
		Where("id = ?", woID).Take(&hdr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(hdr.WOKind), "rm_processing") {
		return nil
	}
	now := time.Now()
	woNumber := strings.TrimSpace(hdr.WoNumber)

	// Idempotency guard: this WO's stock conversion may be triggered from both the
	// scan-out (last process) and the wo-complete endpoint. If we already wrote an
	// rm_processing movement log for this WO number, the stock has been applied —
	// skip so quantities aren't deducted/added twice.
	if woNumber != "" {
		var applied int64
		if err := tx.Table("inventory_movement_logs").
			Where("reference_id = ? AND source_flag = ?", woNumber, string(inventoryconst.SourceRMProcessing)).
			Count(&applied).Error; err != nil {
			return err
		}
		if applied > 0 {
			return nil
		}
	}

	// 1) Source raw material: deduct consumed qty and log the outgoing movement.
	if hdr.SourceUniq != nil && strings.TrimSpace(*hdr.SourceUniq) != "" && hdr.InputQty != nil && *hdr.InputQty > 0 {
		source := strings.TrimSpace(*hdr.SourceUniq)
		var sourceID int64
		if err := tx.Table("raw_materials").Select("id").
			Where("uniq_code = ? AND deleted_at IS NULL", source).
			Limit(1).Scan(&sourceID).Error; err != nil {
			return err
		}
		if err := tx.Table("raw_materials").
			Where("uniq_code = ? AND deleted_at IS NULL", source).
			Updates(map[string]interface{}{
				"stock_qty":  gorm.Expr("stock_qty - ?", *hdr.InputQty),
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		if sourceID > 0 {
			if err := s.createInventoryMovementLog(tx,
				string(inventoryconst.CategoryRawMaterial),
				string(inventoryconst.MovementOutgoing),
				source, &sourceID, 0, -*hdr.InputQty, 0,
				refString(woNumber),
				stringPtr(string(inventoryconst.SourceRMProcessing)),
				stringPtr("RM processing consume (source)"),
				stringPtr("system"),
			); err != nil {
				return err
			}
		}
	}

	if hdr.TargetUniq == nil || strings.TrimSpace(*hdr.TargetUniq) == "" {
		return nil
	}
	target := strings.TrimSpace(*hdr.TargetUniq)
	outQty := 0.0
	if hdr.OutputQty != nil {
		outQty = *hdr.OutputQty
	}

	// 2) Target raw material: add produced qty (create if it doesn't exist yet),
	//    then log the incoming movement.
	var existingID int64
	if err := tx.Table("raw_materials").Select("id").
		Where("uniq_code = ? AND deleted_at IS NULL", target).
		Limit(1).Scan(&existingID).Error; err != nil {
		return err
	}
	targetID := existingID
	if existingID > 0 {
		if err := tx.Table("raw_materials").
			Where("id = ?", existingID).
			Updates(map[string]interface{}{
				"stock_qty":      gorm.Expr("stock_qty + ?", outQty),
				"pre_processing": true,
				"rm_source":      "process",
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}
	} else {
		row := map[string]interface{}{
			"uuid":              uuid.New(),
			"uniq_code":         target,
			"raw_material_type": "others",
			"rm_source":         "process",
			"stock_qty":         outQty,
			"status":            "normal",
			"buy_not_buy":       "not_buy",
			"pre_processing":    true,
			"uom":               hdr.OutputUOM,
			"created_at":        now,
			"updated_at":        now,
		}
		if err := tx.Table("raw_materials").Create(row).Error; err != nil {
			return err
		}
		if err := tx.Table("raw_materials").Select("id").
			Where("uniq_code = ? AND deleted_at IS NULL", target).
			Limit(1).Scan(&targetID).Error; err != nil {
			return err
		}
	}

	if outQty > 0 && targetID > 0 {
		if err := s.createInventoryMovementLog(tx,
			string(inventoryconst.CategoryRawMaterial),
			string(inventoryconst.MovementIncoming),
			target, &targetID, 0, outQty, 0,
			refString(woNumber),
			stringPtr(string(inventoryconst.SourceRMProcessing)),
			stringPtr("RM processing produce (target)"),
			stringPtr("system"),
		); err != nil {
			return err
		}
	}
	return nil
}

// refString returns a *string for a non-empty (trimmed) value, else nil.
func refString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func (s *service) QCSubmit(ctx context.Context, req dto.QCSubmitRequest, performedBy string) error {
	if req.QCTaskID == 0 {
		return apperror.BadRequest("qc_task_id is required")
	}

	if req.WOID == 0 || req.WOItemID == 0 {
		return apperror.BadRequest("wo_id and wo_item_id are required")
	}

	if req.QCRound < 1 || req.QCRound > 3 {
		return apperror.BadRequest("qc_round must be 1 until 3")
	}

	performedBy = strings.TrimSpace(performedBy)
	if performedBy == "" {
		performedBy = "system"
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// =====================================
		// GET QC TASK
		// =====================================
		var task models.QCTask
		if err := tx.
			Where(`
				id = ?
				AND wo_id = ?
				AND wo_item_id = ?
				AND task_type = 'production_qc'
				AND status = 'pending'
			`, req.QCTaskID, req.WOID, req.WOItemID).
			First(&task).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound("pending qc task tidak ditemukan")
			}
			return err
		}

		// =====================================
		// VALIDASI ROUND
		// =====================================
		if task.Round != req.QCRound {
			return apperror.BadRequest(
				fmt.Sprintf("current pending qc round is %d", task.Round),
			)
		}

		// =====================================
		// GET ITEM
		// =====================================
		var item models.WorkOrderItem
		if err := tx.
			Where("id = ? AND wo_id = ?", req.WOItemID, req.WOID).
			First(&item).Error; err != nil {
			return err
		}

		// =====================================
		// GET PAYLOAD JSON
		// =====================================
		var payload struct {
			Uniq        string `json:"uniq"`
			ProcessName string `json:"process_name"`
		}

		if len(task.RoundResults) > 0 {
			_ = json.Unmarshal(task.RoundResults, &payload)
		}

		if payload.Uniq == "" {
			payload.Uniq = item.ItemUniqCode
		}

		if payload.ProcessName == "" {
			payload.ProcessName = item.LastScannedProcess
		}

		now := time.Now()

		// =====================================
		// INSERT QC LOG
		// =====================================
		qc := models.QCLog{
			UUID:        uuid.New().String(),
			WOID:        &item.WOID,
			WOItemID:    &item.ID,
			UniqCode:    payload.Uniq,
			ProcessName: payload.ProcessName,

			QCRound:    req.QCRound,
			QtyChecked: req.QtyChecked,
			QtyPass:    req.QtyPass,
			QtyDefect:  req.QtyDefect,
			QtyScrap:   req.QtyScrap,

			Status:    strings.ToUpper(req.Status),
			CheckedBy: performedBy,
			CheckedAt: now,
			CreatedAt: now,
		}

		if err := tx.Create(&qc).Error; err != nil {
			return err
		}

		// =====================================
		// JIKA REJECT
		// =====================================
		if !strings.EqualFold(req.Status, "APPROVE") &&
			!strings.EqualFold(req.Status, "PASSED") {

			return tx.Model(&task).
				Update("updated_at", now).Error
		}

		// =====================================
		// ROUND 1 -> ROUND 2
		// =====================================
		if req.QCRound == 1 {
			return tx.Model(&task).Updates(map[string]interface{}{
				"round":      2,
				"updated_at": now,
			}).Error
		}

		// =====================================
		// ROUND 2 -> TUNGGU SCAN OUT
		// =====================================
		if req.QCRound == 2 {
			return tx.Model(&task).Updates(map[string]interface{}{
				"round":      3,
				"updated_at": now,
			}).Error
		}

		// =====================================
		// ROUND 3 FINAL QC
		// =====================================
		if req.QCRound == 3 {
			if err := tx.Model(&task).Updates(map[string]interface{}{
				"status":         "done",
				"good_quantity":  int(req.QtyPass),
				"ng_quantity":    int(req.QtyDefect),
				"scrap_quantity": int(req.QtyScrap),
				"updated_at":     now,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) advanceProcessAfterScanOut(ctx context.Context, tx *gorm.DB, item *models.WorkOrderItem, qtyPass float64, performedBy string) error {

	flow := resolveProcessFlow(*item)

	var wo models.WorkOrder
	if err := tx.Where("id = ?", item.WOID).
		First(&wo).Error; err != nil {
		return err
	}

	now := time.Now()

	totalStep := len(flow)
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
	currentStep := flow[currentIndex]

	// =====================================
	// CLOSE CURRENT WIP
	// =====================================
	var currentWIP models.WIPItem

	if err := tx.
		Where(`
			uniq = ?
			AND process_name = ?
			AND status = ?
		`,
			item.ItemUniqCode,
			currentStep.ProcessName,
			"process",
		).
		Order("id desc").
		First(&currentWIP).Error; err == nil {

		currentWIP.QtyOut += int(qtyPass)
		currentWIP.QtyRemaining =
			currentWIP.QtyIn - currentWIP.QtyOut

		currentWIP.Status = "done"
		currentWIP.UpdatedAt = now

		if err := tx.Save(&currentWIP).Error; err != nil {
			return err
		}

		_ = tx.Create(&models.WIPLog{
			WipItemID: currentWIP.ID,
			Action:    "SCAN_OUT_DONE",
			Qty:       int(qtyPass),
			CreatedAt: now,
		}).Error
	}

	// =====================================
	// JIKA MASIH ADA NEXT PROCESS
	// =====================================
	if currentIndex < totalStep-1 {

		nextStep := flow[currentIndex+1]

		nextWIP := models.WIPItem{
			WipID: currentWIP.WipID,

			Uniq:          item.ItemUniqCode,
			PackingNumber: item.KanbanNumber,
			WipType:       "production",

			ProcessName: nextStep.ProcessName,
			MachineName: derefString(nextStep.MachineName),
			OpSeq:       nextStep.OpSeq,
			Seq:         currentIndex + 2, // step ke-2 / ke-3

			UOM: item.UOM,

			Stock:        int(qtyPass),
			QtyIn:        int(qtyPass),
			QtyOut:       0,
			QtyRemaining: int(qtyPass),

			Status: "queue",

			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := tx.Create(&nextWIP).Error; err != nil {
			return err
		}

		_ = tx.Create(&models.WIPLog{
			WipItemID: nextWIP.ID,
			Action:    "TRANSFER_IN",
			Qty:       int(qtyPass),
			CreatedAt: now,
		}).Error

		// =====================================
		// PINDAH STEP
		// pakai 1,2,3
		// =====================================
		item.CurrentStepSeq = currentIndex + 2
		item.Status = "PENDING"
		item.LastScannedProcess = ""

		return tx.Save(item).Error
	}

	// =====================================
	// LAST PROCESS
	// =====================================
	// RM processing outputs a raw material, not a finished good: route the
	// output into raw_materials (target += output, source -= input) with movement
	// logs, and skip the finished-goods path entirely.
	var woKind string
	if err := tx.Table("work_orders").Select("wo_kind").
		Where("id = ?", item.WOID).Take(&woKind).Error; err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(woKind), "rm_processing") {
		if err := s.applyRMProcessingTx(ctx, tx, item.WOID); err != nil {
			return err
		}
		item.Status = "FINISHED"
		item.LastScannedProcess = ""
		return tx.Save(item).Error
	}

	// =====================================
	// LAST PROCESS -> FINISHED GOODS
	// =====================================
	var fg models.FinishedGoods

	err := tx.Where("uniq_code = ?", item.ItemUniqCode).
		First(&fg).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {

		fg = models.FinishedGoods{
			UUID:       uuid.New().String(),
			UniqCode:   item.ItemUniqCode,
			ItemID:     item.ID,
			PartNumber: item.PartNumber,
			PartName:   item.PartName,
			Model:      item.Model,
			WONumber:   wo.WONumber,
			StockQty:   qtyPass,
			UOM:        item.UOM,
			Status:     "AVAILABLE",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := tx.Create(&fg).Error; err != nil {
			return err
		}

		if err := s.createInventoryMovementLog(
			tx,
			"finished_goods",
			"incoming",
			item.ItemUniqCode,
			&fg.ID,
			0,
			qtyPass,
			qtyPass,
			&wo.WONumber,
			stringPtr("SCAN_OUT"),
			stringPtr("Create FG from final production process"),
			stringPtr(performedBy),
		); err != nil {
			return err
		}

	} else if err == nil {

		beforeQty := fg.StockQty
		afterQty := beforeQty + qtyPass

		fg.StockQty = afterQty
		fg.UpdatedAt = now

		if err := tx.Save(&fg).Error; err != nil {
			return err
		}

		if err := s.createInventoryMovementLog(
			tx,
			"finished_goods",
			"incoming",
			item.ItemUniqCode,
			&fg.ID,
			beforeQty,
			qtyPass,
			afterQty,
			&wo.WONumber,
			stringPtr("SCAN_OUT"),
			stringPtr("Add stock from final production process"),
			stringPtr(performedBy),
		); err != nil {
			return err
		}

	} else {
		return err
	}

	item.Status = "FINISHED"
	item.LastScannedProcess = ""

	return tx.Save(item).Error
}

func (s *service) QCFinish(ctx context.Context, req dto.QCFinishRequest, performedBy string) error {
	if req.QCTaskID == 0 {
		return apperror.BadRequest("qc_task_id is required")
	}
	if req.WOID == 0 || req.WOItemID == 0 {
		return apperror.BadRequest("wo_id and wo_item_id are required")
	}

	performedBy = strings.TrimSpace(performedBy)
	if performedBy == "" {
		performedBy = "system"
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// ===== GET QC TASK (production_qc, belum done) =====
		var task models.QCTask
		if err := tx.
			Where(`
				id = ?
				AND wo_id = ?
				AND wo_item_id = ?
				AND task_type = 'production_qc'
				AND LOWER(status) <> 'done'
			`, req.QCTaskID, req.WOID, req.WOItemID).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound("qc task tidak ditemukan / sudah selesai")
			}
			return err
		}

		// ===== GET ITEM + WO =====
		var item models.WorkOrderItem
		if err := tx.Where("id = ? AND wo_id = ?", req.WOItemID, req.WOID).
			First(&item).Error; err != nil {
			return err
		}
		var wo models.WorkOrder
		if err := tx.Where("id = ?", req.WOID).First(&wo).Error; err != nil {
			return err
		}

		// ===== 1. FINISHED GOODS dari Total Production Quantity =====
		if req.TotalProductionQty > 0 {
			var fg models.FinishedGoods
			err := tx.Where("uniq_code = ?", item.ItemUniqCode).First(&fg).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				fg = models.FinishedGoods{
					UUID:       uuid.New().String(),
					UniqCode:   item.ItemUniqCode,
					ItemID:     item.ID,
					PartNumber: item.PartNumber,
					PartName:   item.PartName,
					Model:      item.Model,
					WONumber:   wo.WONumber,
					StockQty:   req.TotalProductionQty,
					UOM:        item.UOM,
					Status:     "AVAILABLE",
					CreatedAt:  now,
					UpdatedAt:  now,
				}
				if err := tx.Create(&fg).Error; err != nil {
					return err
				}
				if err := s.createInventoryMovementLog(tx, "finished_goods", "incoming",
					item.ItemUniqCode, &fg.ID, 0, req.TotalProductionQty, req.TotalProductionQty,
					&wo.WONumber, stringPtr("QC_FINISH"),
					stringPtr("Create FG from QC Finish"), stringPtr(performedBy)); err != nil {
					return err
				}
				if err := s.appendFGMovementLog(tx, fg.ID, item.ItemUniqCode, "incoming_production",
					req.TotalProductionQty, 0, req.TotalProductionQty,
					&wo.WONumber, nil, stringPtr("Masuk FG dari QC Finish (buat baru)"), stringPtr(performedBy)); err != nil {
					return err
				}
			} else if err == nil {
				beforeQty := fg.StockQty
				afterQty := beforeQty + req.TotalProductionQty
				fg.StockQty = afterQty
				fg.UpdatedAt = now
				if err := tx.Save(&fg).Error; err != nil {
					return err
				}
				if err := s.createInventoryMovementLog(tx, "finished_goods", "incoming",
					item.ItemUniqCode, &fg.ID, beforeQty, req.TotalProductionQty, afterQty,
					&wo.WONumber, stringPtr("QC_FINISH"),
					stringPtr("Add FG stock from QC Finish"), stringPtr(performedBy)); err != nil {
					return err
				}
				if err := s.appendFGMovementLog(tx, fg.ID, item.ItemUniqCode, "incoming_production",
					req.TotalProductionQty, beforeQty, afterQty,
					&wo.WONumber, nil, stringPtr("Masuk FG dari QC Finish (tambah stok)"), stringPtr(performedBy)); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// ===== 2. INCOMING SCRAP dari Total Scrap in Scrap Box =====
		if req.TotalScrapInBox > 0 {
			scrapID, err := s.insertIncomingScrap(tx, item, wo.WONumber, req.TotalScrapInBox, performedBy, now)
			if err != nil {
				return err
			}
			if err := s.createInventoryMovementLog(tx, "scrap", "incoming",
				item.ItemUniqCode, &scrapID, 0, req.TotalScrapInBox, req.TotalScrapInBox,
				&wo.WONumber, stringPtr("QC_FINISH"),
				stringPtr("Incoming scrap from QC Finish"), stringPtr(performedBy)); err != nil {
				return err
			}
		}

		// ===== 3. WORK ORDER Re-Work dari NG Defect =====
		if req.NGDefectQty > 0 {
			if err := s.createReworkWorkOrder(tx, item, wo, req.NGDefectQty, performedBy, now); err != nil {
				return err
			}
		}

		// ===== 4. MOVE WIP -> FINISHED GOODS (kurangi WIP + tulis history) =====
		moveWIPQty := req.TotalProductionQty + req.NGDefectQty + req.TotalScrapInBox
		if moveWIPQty > 0 {
			if err := s.reduceWIPToFinishedGoods(tx, req.WOID, item.ItemUniqCode, wo.WONumber, moveWIPQty, performedBy, now); err != nil {
				return err
			}
		}

		if err := tx.Model(&task).Updates(map[string]interface{}{
			"status":         "done",
			"good_quantity":  int(req.TotalProductionQty),
			"ng_quantity":    int(req.NGDefectQty),
			"scrap_quantity": int(req.TotalScrapInBox),
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *service) insertIncomingScrap(tx *gorm.DB, item models.WorkOrderItem, woNumber string, qty float64, performedBy string, now time.Time) (int64, error) {
	sc := scrapModels.ScrapStock{
		UUID:         uuid.New(),
		UniqCode:     item.ItemUniqCode,
		PartNumber:   stringPtr(item.PartNumber),
		PartName:     stringPtr(item.PartName),
		Model:        stringPtr(item.Model),
		WONumber:     stringPtr(woNumber),
		ScrapType:    scrapModels.ScrapTypeProcess, // process_scrap
		Quantity:     qty,
		UOM:          stringPtr(item.UOM),
		DateReceived: &now,
		Validator:    stringPtr(performedBy),
		Status:       scrapModels.StockStatusActive,
		CreatedBy:    stringPtr(performedBy),
		UpdatedBy:    stringPtr(performedBy),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := tx.Create(&sc).Error; err != nil {
		return 0, err
	}
	return sc.ID, nil
}

func (s *service) createReworkWorkOrder(tx *gorm.DB, item models.WorkOrderItem, srcWO models.WorkOrder, qty float64, performedBy string, now time.Time) error {
	prefix := fmt.Sprintf("WO-%d", now.Year())
	woNumber, err := s.nextWONumber(tx, prefix)
	if err != nil {
		return err
	}

	woUUID := uuid.New().String()
	woRow := map[string]interface{}{
		"uuid":            woUUID,
		"wo_number":       woNumber,
		"wo_type":         "Rework", // enum work_orders: New | Assembly | Rework | Addendum
		"wo_kind":         "standard",
		"reference_wo":    srcWO.WONumber,
		"status":          "Draft",
		"approval_status": "Pending",
		"created_date":    now,
		"operator_name":   performedBy,
		"model":           item.Model,
		"notes": fmt.Sprintf("Auto Re-Work dari QC Finish WO %s (UNIQ %s), NG %.4f",
			srcWO.WONumber, item.ItemUniqCode, qty),
		"created_at": now,
		"updated_at": now,
	}
	if err := tx.Table("work_orders").Create(woRow).Error; err != nil {
		return err
	}

	// Ambil id WO yang baru dibuat (map insert tidak mengembalikan PK).
	var woIDs []int64
	if err := tx.Table("work_orders").
		Where("uuid = ?", woUUID).
		Limit(1).
		Pluck("id", &woIDs).Error; err != nil {
		return err
	}
	if len(woIDs) == 0 {
		return apperror.InternalWrap("failed to resolve new rework wo id", errors.New("wo id not found"))
	}
	newWOID := woIDs[0]

	itemRow := map[string]interface{}{
		"uuid":           uuid.New().String(),
		"wo_id":          newWOID,
		"item_uniq_code": item.ItemUniqCode,
		"part_name":      item.PartName,
		"part_number":    item.PartNumber,
		"uom":            item.UOM,
		"quantity":       qty,
		"process_name":   item.ProcessName,
		"kanban_number":  fmt.Sprintf("RW-%s-%s", woNumber, item.ItemUniqCode),
		"status":         "Pending",
		"created_at":     now,
		"updated_at":     now,
	}
	return tx.Table("work_order_items").Create(itemRow).Error
}

func (s *service) nextWONumber(tx *gorm.DB, prefix string) (string, error) {
	var nums []string
	if err := tx.Table("work_orders").
		Where("wo_number LIKE ?", prefix+"-%").
		Order("wo_number DESC").
		Limit(1).
		Pluck("wo_number", &nums).Error; err != nil {
		return "", err
	}
	if len(nums) == 0 || nums[0] == "" {
		return fmt.Sprintf("%s-%06d", prefix, 1), nil
	}
	parts := strings.Split(nums[0], "-")
	seq, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || seq < 0 {
		return fmt.Sprintf("%s-%06d", prefix, 1), nil
	}
	return fmt.Sprintf("%s-%06d", prefix, seq+1), nil
}

func stringPtr(v string) *string {
	return &v
}

func (s *service) createInventoryMovementLog(tx *gorm.DB, category string, movementType string, uniqCode string, entityID *int64, beforeQty float64, changeQty float64, afterQty float64, refNo *string, source *string, remarks *string, user *string) error {

	_ = beforeQty
	_ = afterQty

	row := models.InventoryMovementLog{
		MovementCategory: category,
		MovementType:     movementType,
		UniqCode:         uniqCode,
		EntityID:         entityID,

		QtyChange: changeQty,

		ReferenceID: refNo,
		SourceFlag:  source,
		Notes:       remarks,

		LoggedBy: user,
		LoggedAt: time.Now(),
	}

	return tx.Create(&row).Error
}

func parseProcessFlow(flowJSON string) ([]models.ProcessFlow, error) {
	if flowJSON == "" {
		return nil, errors.New("process flow empty")
	}

	var flow []models.ProcessFlow
	if err := json.Unmarshal([]byte(flowJSON), &flow); err != nil {
		return nil, errors.New("invalid process flow")
	}

	if len(flow) == 0 {
		return nil, errors.New("process flow empty")
	}

	return flow, nil
}

func resolveProcessFlow(item models.WorkOrderItem) []models.ProcessFlow {
	flow, err := parseProcessFlow(item.ProcessFlowJSON)
	if err != nil || len(flow) == 0 {
		fallback := models.ProcessFlow{
			OpSeq:       1,
			ProcessName: item.ProcessName,
		}
		return []models.ProcessFlow{fallback}
	}
	return flow
}

func isUniqFinished(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FINISHED", "DONE", "COMPLETED":
		return true
	}
	return false
}

func buildProcessSteps(flow []models.ProcessFlow, currentSeq int, finished bool) []dto.WODetailProcessStep {
	total := len(flow)
	currentIndex := getCurrentIndex(currentSeq, total)
	steps := make([]dto.WODetailProcessStep, 0, total)
	for i, f := range flow {
		status := "pending"
		if finished {
			status = "done"
		} else if i < currentIndex {
			status = "done"
		} else if i == currentIndex {
			status = "current"
		}
		machineName := ""
		if f.MachineName != nil {
			machineName = *f.MachineName
		}
		steps = append(steps, dto.WODetailProcessStep{
			OpSeq:       f.OpSeq,
			ProcessName: f.ProcessName,
			MachineName: machineName,
			Status:      status,
		})
	}
	return steps
}

func fallbackDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func validateStep(step int, flow []models.ProcessFlow) error {
	if step <= 0 {
		return errors.New("invalid step: must start from 1")
	}
	if step > len(flow) {
		return errors.New("invalid step: overflow")
	}
	return nil
}

func getCurrentIndex(step int, total int) int {
	idx := step - 1

	// 🔒 guard bawah
	if idx < 0 {
		return 0
	}

	// 🔒 guard atas (overflow)
	if idx >= total {
		return total - 1
	}

	return idx
}

func (s *service) ListQCTask(ctx context.Context, req dto.ListQCTaskRequest) (map[string]interface{}, error) {

	if req.Page <= 0 {
		req.Page = 1
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	offset := (req.Page - 1) * req.Limit

	db := s.db.WithContext(ctx).Model(&models.QCTask{})

	// =============================
	// DEFAULT: tampilkan selain done
	// =============================
	db = db.Where("LOWER(status) <> ?", "done")

	// =============================
	// FILTER
	// =============================
	if req.Status != "" {
		db = db.Where("LOWER(status) = LOWER(?)", req.Status)
	}

	if req.TaskType != "" {
		db = db.Where("task_type = ?", req.TaskType)
	}

	if req.Search != "" {
		db = db.Where(`
			CAST(id AS TEXT) ILIKE ?
			OR round_results::text ILIKE ?
		`, "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// =============================
	// COUNT
	// =============================
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// =============================
	// GET DATA
	// =============================
	var rows []models.QCTask
	if err := db.
		Order("id desc").
		Limit(req.Limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]dto.QCTaskListItem, 0)
	woNumberCache := map[int64]string{}

	for _, row := range rows {

		var payload struct {
			Uniq         string  `json:"uniq"`
			KanbanNumber string  `json:"kanban_number"`
			ProcessName  string  `json:"process_name"`
			Qty          float64 `json:"qty"`
		}

		_ = json.Unmarshal(row.RoundResults, &payload)

		// Ambil nomor WO dari WOID (cache agar tidak query berulang).
		woNumber := ""
		if row.WOID != nil {
			if cached, ok := woNumberCache[*row.WOID]; ok {
				woNumber = cached
			} else {
				if wo, err := s.repoProduction.FindWOByID(ctx, *row.WOID); err == nil {
					woNumber = wo.WONumber
				}
				woNumberCache[*row.WOID] = woNumber
			}
		}

		items = append(items, dto.QCTaskListItem{
			ID:           row.ID,
			TaskType:     row.TaskType,
			Status:       row.Status,
			Round:        row.Round,
			WOID:         row.WOID,
			WOItemID:     row.WOItemID,
			WONumber:     woNumber,
			Uniq:         payload.Uniq,
			KanbanNumber: payload.KanbanNumber,
			ProcessName:  payload.ProcessName,
			Qty:          payload.Qty,
			CreatedAt:    row.CreatedAt,
		})
	}

	return map[string]interface{}{
		"page":  req.Page,
		"limit": req.Limit,
		"total": total,
		"items": items,
	}, nil
}

type IssueListResponse struct {
	UUID           string    `json:"uuid"`
	WONumber       string    `json:"wo_number"`
	ItemCode       string    `json:"item_uniq_code"`
	PartName       string    `json:"part_name"`
	PartNumber     string    `json:"part_number"`
	Machine        string    `json:"machine"`
	ProcessName    string    `json:"process_name"`
	ProductionLine string    `json:"production_line"`
	IssueType      string    `json:"issue_type"`
	IssueDuration  int       `json:"issue_duration"`
	QtyAffected    float64   `json:"qty_affected"`
	ReportedBy     string    `json:"reported_by"`
	ReportedAt     time.Time `json:"reported_at"`
}

func (s *service) IssueList(ctx context.Context) (map[string]interface{}, error) {
	var results []IssueListResponse

	err := s.db.WithContext(ctx).
		Table("production_issues pi").
		Select(`
			pi.uuid,
			wo.wo_number,
			woi.item_uniq_code,
			woi.part_name,
   woi.part_number,
			mm.machine_name AS machine,
			pi.process_name,
			pi.production_line,
			pi.issue_type,
			pi.issue_duration,
			pi.qty_affected,
			pi.reported_by,
			pi.reported_at
		`).
		Joins("LEFT JOIN work_orders wo ON wo.id = pi.wo_id").
		Joins("LEFT JOIN work_order_items woi ON woi.id = pi.wo_item_id").
		Joins("LEFT JOIN master_machines mm ON mm.id = pi.machine_id").
		Order("pi.reported_at DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"items": results,
	}, nil
}

// =====================================
// NEW — WO list (dropdown 1.2)
// =====================================
func (s *service) WOList(ctx context.Context, search string) ([]dto.WOListItem, error) {
	rows, err := s.repoProduction.ListWorkOrders(ctx, search, 50)
	if err != nil {
		return nil, err
	}
	out := make([]dto.WOListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.WOListItem{
			WOID:           r.ID,
			WONumber:       r.WONumber,
			Status:         r.Status,
			Model:          r.Model,
			PartName:       r.PartName,
			ProductionLine: r.ProductionLine,
			TotalQty:       r.TotalQty,
			UniqCount:      r.UniqCount,
		})
	}
	return out, nil
}

// =====================================
// NEW — WO detail + semua uniq (1.1 & step form)
// =====================================
func (s *service) WODetail(ctx context.Context, woNumber string) (*dto.WODetailResponse, error) {
	wo, err := s.repoProduction.FindWOByNumber(ctx, strings.TrimSpace(woNumber))
	if err != nil {
		return nil, errors.New("work order tidak ditemukan")
	}

	items, err := s.repoProduction.FindWOItemsByWOID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}

	uniqs := make([]dto.WODetailUniq, 0, len(items))
	var totalQty float64

	for _, item := range items {
		totalQty += item.Quantity

		flow := resolveProcessFlow(item)
		totalStep := len(flow)
		currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
		currentProcess := flow[currentIndex].ProcessName

		// proses berikutnya DI DALAM UNIQ ini (bukan uniq berikutnya)
		nextProcess := ""
		if currentIndex+1 < totalStep {
			nextProcess = flow[currentIndex+1].ProcessName
		}

		machineNumber := ""
		productionLine := ""
		if item.MachineID != 0 {
			if m, e := s.repoProduction.FindMachineByID(ctx, item.MachineID); e == nil {
				machineNumber = m.MachineNumber
				productionLine = m.ProductionLine
			}
		}

		var savedQty float64
		dandori := ""
		setupQC := ""
		if lastIn, e := s.repoProduction.FindLatestScanInLog(ctx, item.ID); e == nil {
			savedQty = lastIn.QtyInput
			dandori = lastIn.DandoriTime
			setupQC = lastIn.SetupQCTime
		}

		rmList, e := s.buildItemRawMaterials(ctx, item)
		if e != nil {
			return nil, e
		}

		bomList, e := s.buildBomMaterials(ctx, item.ItemUniqCode)
		if e != nil {
			return nil, e
		}

		uniqs = append(uniqs, dto.WODetailUniq{
			WOItemID:       item.ID,
			Uniq:           item.ItemUniqCode,
			PartName:       item.PartName,
			PartNumber:     item.PartNumber,
			KanbanNumber:   item.KanbanNumber,
			UOM:            item.UOM,
			Qty:            item.Quantity,
			Status:         item.Status,
			MachineID:      strconv.Itoa(item.MachineID),
			MachineNumber:  machineNumber,
			ProductionLine: productionLine,
			ProcessName:    currentProcess,
			NextProcess:    nextProcess,
			CurrentStep:    currentIndex + 1,
			TotalStep:      totalStep,
			ScanInCount:    item.ScanInCount,
			ScanOutCount:   item.ScanOutCount,
			MachineScanned: item.MachineID != 0,
			ProcessFlow:    buildProcessSteps(flow, item.CurrentStepSeq, isUniqFinished(item.Status)),
			SavedQty:       savedQty,
			DandoriTime:    dandori,
			SetupQCTime:    setupQC,
			RawMaterials:   rmList,
			BomMaterials:   bomList,
		})
	}

	partName := ""
	productionLine := ""
	if len(items) > 0 {
		partName = items[0].PartName
		productionLine = uniqs[0].ProductionLine
	}

	return &dto.WODetailResponse{
		WOID:           wo.ID,
		WONumber:       wo.WONumber,
		Status:         wo.Status,
		Model:          wo.Model,
		PartName:       partName,
		ProductionLine: productionLine,
		TotalQty:       totalQty,
		UniqCount:      len(items),
		Uniqs:          uniqs,
	}, nil
}

// =====================================
// NEW — Lookup RM (Scan RM 2.2)
// =====================================
func (s *service) RawMaterialLookup(ctx context.Context, code string) (*dto.RawMaterialLookupResponse, error) {
	rm, err := s.repoProduction.FindRawMaterialByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	var weight float64
	if rm.StockWeightKg != nil {
		weight = *rm.StockWeightKg
	}
	return &dto.RawMaterialLookupResponse{
		RMID:              rm.ID,
		RMUUID:            rm.UUID,
		UniqCode:          rm.UniqCode,
		PartNumber:        rm.PartNumber,
		PartName:          rm.PartName,
		RawMaterialType:   rm.RawMaterialType,
		TypeLabel:         rawMaterialTypeLabel(rm.RawMaterialType),
		UOM:               rm.UOM,
		AvailableStock:    rm.StockQty,
		StockWeightKg:     weight,
		WarehouseLocation: rm.WarehouseLocation,
	}, nil
}

func rawMaterialTypeLabel(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "wire":
		return "WIRE"
	case "sheet_plate", "sheet":
		return "SHEET"
	case "ssp":
		return "SSP"
	case "wip":
		return "WIP"
	default:
		return "OTHER"
	}
}

// =====================================
// NEW — helper simpan pemakaian RM (scan in: planned)
// =====================================
func (s *service) saveRawMaterialLogs(ctx context.Context, item models.WorkOrderItem, rms []dto.RawMaterialInput, scannedBy string, now time.Time) error {

	if err := s.repoProduction.DeleteRawMaterialLogsByWOItemID(ctx, item.ID); err != nil {
		return err
	}

	for _, rm := range rms {
		if rm.Qty <= 0 {
			continue
		}
		partNumber := rm.UniqCode
		var partName, uom string
		if rm.RMUUID != "" {
			if master, err := s.repoProduction.FindRawMaterialByUUID(ctx, rm.RMUUID); err == nil {
				partNumber = master.PartNumber
				partName = master.PartName
				uom = master.UOM
			}
		}
		if err := s.repoProduction.InsertRawMaterial(ctx, models.RawMaterialLog{
			UUID:       uuid.New().String(),
			WOID:       item.WOID,
			WOItemID:   item.ID,
			UniqCode:   rm.UniqCode,
			RMUUID:     rm.RMUUID,
			PartNumber: partNumber,
			PartName:   partName,
			UOM:        uom,
			QtyUsed:    rm.Qty,
			ScannedBy:  scannedBy,
			ScannedAt:  now,
			CreatedAt:  now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func sumScanOutRM(rms []dto.ScanOutRawMaterial) float64 {
	var t float64
	for _, r := range rms {
		t += r.QtyUsed
	}
	return t
}

func (s *service) consumeRawMaterials(ctx context.Context, item models.WorkOrderItem, rms []dto.ScanOutRawMaterial, scannedBy, woNumber string, now time.Time) error {
	if err := s.repoProduction.DeleteRawMaterialLogsByWOItemID(ctx, item.ID); err != nil {
		return err
	}

	for _, rm := range rms {
		// kode tampilan RM: utamakan packing_list_rm, fallback uniq_code
		code := rm.PackingListRM
		if code == "" {
			code = rm.UniqCode
		}

		var master models.RawMaterial
		var found bool
		if rm.RMUUID != "" {
			if m, err := s.repoProduction.FindRawMaterialByUUID(ctx, rm.RMUUID); err == nil {
				master, found = m, true
			}
		}
		if !found && code != "" {
			if m, err := s.repoProduction.FindRawMaterialByCode(ctx, code); err == nil {
				master, found = m, true
			}
		}

		partNumber := master.PartNumber
		if partNumber == "" {
			partNumber = code
		}

		// 1) catat pemakaian RM — SEMUA RM dicatat (termasuk qty 0) supaya
		//    tetap tampil di done-view sesuai yang benar-benar discan-out.
		if err := s.repoProduction.InsertRawMaterial(ctx, models.RawMaterialLog{
			UUID:       uuid.New().String(),
			WOID:       item.WOID,
			WOItemID:   item.ID,
			UniqCode:   code,
			RMUUID:     rm.RMUUID,
			PartNumber: partNumber,
			PartName:   master.PartName,
			UOM:        master.UOM,
			QtyUsed:    rm.QtyUsed,
			ScannedBy:  scannedBy,
			ScannedAt:  now,
			CreatedAt:  now,
		}); err != nil {
			return err
		}

		// Stok hanya dikurangi kalau qty > 0 dan RM dikenal di master.
		if rm.QtyUsed <= 0 || !found {
			continue
		}

		// 2) kurangi stok RM
		if _, _, err := s.repoProduction.DecreaseRawMaterialStock(ctx, master.ID, rm.QtyUsed); err != nil {
			return err
		}

		// 3) audit trail inventory
		ref := woNumber
		src := "wo_scan"
		entityID := master.ID
		by := scannedBy
		if err := s.repoProduction.InsertInventoryMovementLog(ctx, models.InventoryMovementLog{
			MovementCategory: "raw_material",
			MovementType:     "outgoing",
			UniqCode:         master.UniqCode,
			EntityID:         &entityID,
			QtyChange:        -rm.QtyUsed,
			ReferenceID:      &ref,
			SourceFlag:       &src,
			LoggedBy:         &by,
			LoggedAt:         now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func buildWOProcessSteps(items []models.WorkOrderItem, currentIdx int) []dto.WODetailProcessStep {
	steps := make([]dto.WODetailProcessStep, 0, len(items))
	for i, it := range items {
		status := "pending"
		if i < currentIdx {
			status = "done"
		} else if i == currentIdx {
			status = "current"
		}
		steps = append(steps, dto.WODetailProcessStep{
			OpSeq:       i + 1,
			ProcessName: it.ProcessName,
			MachineName: "",
			Status:      status,
		})
	}
	return steps
}

func (s *service) ScanMachine(ctx context.Context, req dto.ScanMachineRequest) (*dto.ScanMachineResponse, error) {
	items, err := s.repoProduction.FindWOItemsByWOID(ctx, req.WOID)
	if err != nil || len(items) == 0 {
		return nil, errors.New("work order tidak ditemukan")
	}

	// Cari uniq (baris) yang di-scan berdasarkan wo_item_id
	currentIdx := -1
	for i, it := range items {
		if it.ID == req.WOItemID {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 {
		return nil, errors.New("uniq tidak ditemukan di work order ini")
	}

	item := items[currentIdx]
	flow := resolveProcessFlow(item)
	totalStep := len(flow)
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
	expectedProcess := flow[currentIndex].ProcessName
	nextProcess := ""
	if currentIndex+1 < totalStep {
		nextProcess = flow[currentIndex+1].ProcessName
	}
	steps := buildProcessSteps(flow, item.CurrentStepSeq, isUniqFinished(item.Status))

	// Cari mesin yang di-scan
	machine, err := s.repoProduction.FindMachineByNumber(ctx, strings.TrimSpace(req.Machine))
	if err != nil {
		return &dto.ScanMachineResponse{
			Valid:           false,
			Message:         fmt.Sprintf("Mesin \"%s\" tidak ditemukan.", req.Machine),
			ExpectedProcess: expectedProcess,
			CurrentStep:     currentIdx + 1,
			TotalStep:       totalStep,
			NextProcess:     nextProcess,
			ProcessFlow:     steps,
		}, nil
	}

	// Proses milik mesin tsb
	scannedProcess := ""
	if machine.ProcessID != nil {
		if name, e := s.repoProduction.FindProcessNameByID(ctx, *machine.ProcessID); e == nil {
			scannedProcess = name
		}
	}

	// [flex-machine] Validasi "proses mesin harus sama dengan proses uniq" DIHAPUS agar
	// mesin apa pun bisa dipakai untuk proses ini (lebih fleksibel). scannedProcess tetap
	// dihitung & dikembalikan sebagai informasi saja.

	// Set mesin ke uniq (proses tetap = expectedProcess milik uniq ini).
	if err := s.repoProduction.UpdateWOItemMachine(ctx, item.ID, int(machine.ID), expectedProcess); err != nil {
		return nil, err
	}

	return &dto.ScanMachineResponse{
		Valid:           true,
		Message:         fmt.Sprintf("Mesin %s digunakan untuk proses \"%s\".", machine.MachineNumber, expectedProcess),
		MachineID:       strconv.Itoa(int(machine.ID)),
		MachineNumber:   machine.MachineNumber,
		MachineName:     machine.MachineName,
		ProductionLine:  machine.ProductionLine,
		ScannedProcess:  scannedProcess,
		ExpectedProcess: expectedProcess,
		CurrentStep:     currentIdx + 1,
		TotalStep:       totalStep,
		NextProcess:     nextProcess,
		ProcessFlow:     steps,
	}, nil
}

func rmTypeLabel(t string) string {
	switch strings.ToLower(t) {
	case "wire":
		return "WIRE"
	case "sheet_plate":
		return "SHEET"
	case "ssp":
		return "SSP"
	case "wip":
		return "WIP"
	default:
		return "OTHER"
	}
}

func rawMaterialLogKey(l models.RawMaterialLog) string {
	if l.RMUUID != "" {
		return "uuid:" + l.RMUUID
	}
	if l.UniqCode != "" {
		return "code:" + l.UniqCode
	}
	return "part:" + l.PartNumber
}

func dedupRawMaterialLogs(logs []models.RawMaterialLog) []models.RawMaterialLog {
	idxByKey := make(map[string]int) // key -> index di out
	out := make([]models.RawMaterialLog, 0, len(logs))
	for _, l := range logs {
		key := rawMaterialLogKey(l)
		pos, ok := idxByKey[key]
		if !ok {
			idxByKey[key] = len(out)
			out = append(out, l)
			continue
		}
		cur := out[pos]
		better := false
		if l.QtyUsed > 0 && cur.QtyUsed <= 0 {
			better = true // yang lama qty 0, yang baru qty > 0
		} else if (l.QtyUsed > 0) == (cur.QtyUsed > 0) && l.ID >= cur.ID {
			better = true // sama-sama >0 (atau sama-sama 0) → ambil yang terbaru
		}
		if better {
			out[pos] = l
		}
	}
	return out
}

func (s *service) ScanOutContext(
	ctx context.Context, woNumber string,
) (*dto.ScanOutContextResponse, error) {
	wo, err := s.repoProduction.FindWOByNumber(ctx, woNumber)
	if err != nil {
		return nil, errors.New("wo tidak ditemukan")
	}

	items, err := s.repoProduction.FindWOItemsByWOID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}

	resp := &dto.ScanOutContextResponse{
		WOID:  wo.ID,
		Items: make([]dto.ScanOutContextItem, 0, len(items)),
	}

	for _, item := range items {
		logs, err := s.repoProduction.FindProductionRawMaterialLogsByWOItemID(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		logs = dedupRawMaterialLogs(logs)

		uuids := make([]string, 0, len(logs))
		codes := make([]string, 0, len(logs))
		for _, l := range logs {
			if l.RMUUID != "" {
				uuids = append(uuids, l.RMUUID)
			}
			if l.UniqCode != "" {
				codes = append(codes, l.UniqCode)
			}
		}

		metaByUUID := map[string]repository.RawMaterialMeta{}
		metaByCode := map[string]repository.RawMaterialMeta{}
		if len(uuids) > 0 || len(codes) > 0 {
			metas, err := s.repoProduction.FindRawMaterialMetaByKeys(ctx, uuids, codes)
			if err == nil {
				for _, m := range metas {
					if m.UUID != "" {
						metaByUUID[m.UUID] = m
					}
					if m.UniqCode != "" {
						metaByCode[m.UniqCode] = m
					}
				}
			}
		}

		bomMap := s.bomByUniq(ctx, item.ItemUniqCode)
		rms := make([]dto.ScanOutContextRawMaterial, 0, len(logs))
		for _, l := range logs {
			meta, ok := metaByUUID[l.RMUUID]
			if !ok {
				meta, ok = metaByCode[l.UniqCode]
			}

			packing := l.UniqCode
			if packing == "" {
				packing = l.PartNumber
			}
			bm := bomMap[l.UniqCode]
			materialCode := bm.MaterialGrade
			if materialCode == "" {
				materialCode = packing
			}
			form := bm.Form
			specWeight := 0.0
			if bm.WeightKg != nil {
				specWeight = *bm.WeightKg
			}
			uom := l.UOM
			typeLabel := "OTHER"
			var avail, weight float64
			if ok {
				if meta.UOM != "" {
					uom = meta.UOM
				}
				typeLabel = rmTypeLabel(meta.RawMaterialType)
				avail = meta.StockQty
				weight = meta.StockWeightKg
			}

			rms = append(rms, dto.ScanOutContextRawMaterial{
				RMUUID:         l.RMUUID,
				PackingListRM:  packing,
				MaterialCode:   materialCode,
				Form:           form,
				QtyPerUniq:     bm.QtyPerUniq,
				SpecWeightKg:   specWeight,
				TypeLabel:      typeLabel,
				IsWIP:          strings.EqualFold(typeLabel, "WIP"),
				UOM:            uom,
				AvailableStock: avail,
				StockWeightKg:  weight,
				QtyUsed:        l.QtyUsed,
			})
		}

		soFlow := resolveProcessFlow(item)
		soTotalStep := len(soFlow)
		soCurrentIndex := getCurrentIndex(item.CurrentStepSeq, soTotalStep)

		resp.Items = append(resp.Items, dto.ScanOutContextItem{
			WOItemID:       item.ID,
			Uniq:           item.ItemUniqCode,
			MachineScanned: item.MachineID != 0,
			Status:         item.Status,
			CurrentStep:    soCurrentIndex + 1,
			TotalStep:      soTotalStep,
			ScanInCount:    item.ScanInCount,
			ScanOutCount:   item.ScanOutCount,
			TotalOutput:    item.TotalGoodQty,
			RawMaterials:   rms,
		})
	}

	return resp, nil
}

func (s *service) buildBomMaterials(ctx context.Context, rootUniq string) ([]dto.BomMaterial, error) {
	if strings.TrimSpace(rootUniq) == "" {
		return []dto.BomMaterial{}, nil
	}

	rows, err := s.repoProduction.FindBomMaterialsByRootUniq(ctx, rootUniq)
	if err != nil {
		return nil, err
	}

	derefStr := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}
	derefFloat := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}

	out := make([]dto.BomMaterial, 0, len(rows))
	for _, row := range rows {
		// UOM: pakai UOM dari master RM bila ada, lalu UOM di bom_lines, terakhir UOM item.
		uom := deref(row.RMUom)
		if uom == "" {
			uom = deref(row.LineUom)
		}
		if uom == "" {
			uom = strings.TrimSpace(row.ItemUom)
		}

		inInventory := row.RMUUID != nil && strings.TrimSpace(*row.RMUUID) != ""
		rawType := deref(row.RawMaterialType)
		typeLabel := ""
		if inInventory {
			typeLabel = rawMaterialTypeLabel(rawType)
		}

		out = append(out, dto.BomMaterial{
			Uniq:            row.UniqCode,
			PartName:        row.PartName,
			PartNumber:      derefStr(row.PartNumber),
			Level:           row.Level,
			QtyPerUniq:      derefFloat(row.QtyPerUniq),
			UOM:             uom,
			MaterialGrade:   derefStr(row.MaterialGrade),
			Grade:           derefStr(row.Grade),
			TypeMaterial:    derefStr(row.TypeMaterial),
			Form:            derefStr(row.Form),
			WidthMm:         row.WidthMm,
			DiameterMm:      row.DiameterMm,
			ThicknessMm:     row.ThicknessMm,
			LengthMm:        row.LengthMm,
			WeightKg:        row.WeightKg,
			SupplierName:    derefStr(row.SupplierName),
			RMUUID:          deref(row.RMUUID),
			RawMaterialType: rawType,
			TypeLabel:       typeLabel,
			InInventory:     inInventory,
			AvailableStock:  derefFloat(row.StockQty),
			StockWeightKg:   derefFloat(row.StockWeightKg),
		})
	}
	return out, nil
}

func (s *service) bomByUniq(ctx context.Context, rootUniq string) map[string]dto.BomMaterial {
	m := map[string]dto.BomMaterial{}
	bom, err := s.buildBomMaterials(ctx, rootUniq)
	if err != nil {
		return m
	}
	for _, b := range bom {
		m[b.Uniq] = b
	}
	return m
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// =============================
// Product Return (BRD)
// =============================

// ScanReturn resolves an item from a scanned kanban / packing list and returns
// the auto-fill payload for the Product Return form.
func (s *service) ScanReturn(ctx context.Context, req models.ScanReturnRequest) (*models.ScanReturnResponse, error) {
	code := strings.TrimSpace(req.QRCodeValue)
	if code == "" {
		return nil, errors.New("qr_code_value is required")
	}

	resp := &models.ScanReturnResponse{ScrapType: "Product Return"}

	// 1) Resolve packing/kanban -> item uniq code (reuse the incoming lookup).
	uniqCode := code
	if item, err := s.repo.LookupByPackingNumber(ctx, code, ""); err == nil && item != nil {
		if item.ItemUniqCode != "" {
			uniqCode = item.ItemUniqCode
		}
		if item.PackingNumber != nil && *item.PackingNumber != "" {
			resp.PackingNumber = *item.PackingNumber
		}
		if item.UOM != nil && *item.UOM != "" {
			resp.SelectedUnit = *item.UOM
		}
	} else {
		var dnItem struct {
			ItemUniqCode  string `gorm:"column:item_uniq_code"`
			PackingNumber string `gorm:"column:packing_number"`
			UOM           string `gorm:"column:uom"`
		}
		if err := s.db.WithContext(ctx).
			Table("delivery_note_items_customer").
			Select("item_uniq_code, packing_number, uom").
			Where("packing_number = ?", code).
			First(&dnItem).Error; err == nil {
			if dnItem.ItemUniqCode != "" {
				uniqCode = dnItem.ItemUniqCode
			}
			if dnItem.PackingNumber != "" {
				resp.PackingNumber = dnItem.PackingNumber
			}
			if dnItem.UOM != "" {
				resp.SelectedUnit = dnItem.UOM
			}
		}
	}

	resp.Uniq = uniqCode
	resp.UniqID = uniqCode
	if resp.PackingNumber == "" {
		resp.PackingNumber = code
	}

	// 2) Enrich with master item info (part number / name / model).
	var itemRow struct {
		PartNumber string `gorm:"column:part_number"`
		PartName   string `gorm:"column:part_name"`
		Model      string `gorm:"column:model"`
	}
	if err := s.db.WithContext(ctx).
		Table("items").
		Select("part_number, part_name, model").
		Where("uniq_code = ?", uniqCode).
		Limit(1).
		Scan(&itemRow).Error; err == nil {
		resp.PartNumber = itemRow.PartNumber
		resp.PartName = itemRow.PartName
		resp.Model = itemRow.Model
	}

	return resp, nil
}

// SubmitReturnToQC creates a PENDING product_returns row so the item is queued
// for QC validation. Scrap is NOT written here — only after QC validation.
func (s *service) SubmitReturnToQC(ctx context.Context, req models.SubmitReturnToQCRequest, submittedBy string) (*models.ProductReturnRow, error) {
	_ = submittedBy // reserved for audit once created_by exists

	uniq := strings.TrimSpace(req.Uniq)
	if uniq == "" {
		return nil, errors.New("uniq is required")
	}

	// DN fallback: dn_number -> packing_number -> kanban/packing list -> "-".
	dnNumber := strings.TrimSpace(req.DNNumber)
	if dnNumber == "" {
		if p := strings.TrimSpace(req.PackingNumber); p != "" {
			dnNumber = p
		} else if k := strings.TrimSpace(req.KanbanPackingList); k != "" {
			dnNumber = k
		} else {
			dnNumber = "-"
		}
	}

	scrapType := strings.TrimSpace(req.ScrapType)
	if scrapType == "" {
		scrapType = "Product Return"
	}

	if dnNumber != "-" {
		var maxQty float64
		var found bool

		var outItem struct {
			Quantity float64 `gorm:"column:quantity"`
		}
		if err := s.db.WithContext(ctx).Table("delivery_note_items_customer").
			Select("quantity").
			Where("packing_number = ? AND item_uniq_code = ?", dnNumber, uniq).
			First(&outItem).Error; err == nil {
			maxQty = outItem.Quantity
			found = true
		}

		if !found {
			var total struct {
				Total float64
			}
			if err := s.db.WithContext(ctx).Table("delivery_note_items_customer").
				Select("SUM(quantity) as total").
				Joins("JOIN delivery_notes_customer dn ON dn.id = delivery_note_items_customer.dn_id").
				Where("dn.dn_number = ? AND item_uniq_code = ?", dnNumber, uniq).
				Scan(&total).Error; err == nil && total.Total > 0 {
				maxQty = total.Total
				found = true
			}
		}

		if !found {
			var inItem struct {
				Quantity float64 `gorm:"column:quantity"`
			}
			if err := s.db.WithContext(ctx).Table("delivery_note_items").
				Select("quantity").
				Where("packing_number = ? AND item_uniq_code = ?", dnNumber, uniq).
				First(&inItem).Error; err == nil {
				maxQty = inItem.Quantity
				found = true
			}
		}

		if found {
			totalInput := float64(req.QuantityScrap + req.QuantityRework)
			if totalInput > maxQty {
				return nil, apperror.BadRequest(fmt.Sprintf("Kuantitas yang diinput (%v) tidak boleh melebihi kuantitas dari DN (%v)", totalInput, maxQty))
			}
		}
	}

	row := models.ProductReturnRow{
		Uniq:           uniq,
		DNNumber:       dnNumber,
		QuantityScrap:  req.QuantityScrap,
		QuantityRework: req.QuantityRework,
		ScrapType:      scrapType,
		Status:         "PENDING",
	}
	if req.Weight != nil {
		row.Weight = *req.Weight
	}
	if u := strings.TrimSpace(req.UnitMeasurement); u != "" {
		row.Uom = u
	}
	if d := parseReturnDate(req.DateReceived); d != nil {
		row.DateReceived = d
	}

	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// parseReturnDate parses common date formats coming from the UI.
func parseReturnDate(raw string) *time.Time {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, v); err == nil {
			return &t
		}
	}
	return nil
}

// PendingReturnTasks lists product returns awaiting QC validation.
func (s *service) PendingReturnTasks(ctx context.Context) ([]models.PendingReturnTask, error) {
	var rows []struct {
		ID             uint
		Uniq           string
		DNNumber       string
		QuantityScrap  int
		QuantityRework int
		Weight         float64
		Uom            string
		ScrapType      string
		PartNumber     string
		PartName       string
		Model          string
	}

	err := s.db.WithContext(ctx).
		Table("product_returns AS pr").
		Select(`pr.id AS id, pr.uniq AS uniq, pr.dn_number AS dn_number,
			pr.quantity_scrap AS quantity_scrap, pr.quantity_rework AS quantity_rework, pr.weight AS weight, pr.uom AS uom,
			pr.scrap_type AS scrap_type,
			COALESCE(i.part_number, '') AS part_number,
			COALESCE(i.part_name, '') AS part_name,
			COALESCE(i.model, '') AS model`).
		Joins("LEFT JOIN items AS i ON i.uniq_code = pr.uniq").
		Where("pr.status = ?", "PENDING").
		Order("pr.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	tasks := make([]models.PendingReturnTask, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, models.PendingReturnTask{
			ReturnID:          strconv.FormatUint(uint64(r.ID), 10),
			KanbanPackingList: r.DNNumber,
			PackingNumber:     r.DNNumber,
			Uniq:              r.Uniq,
			UniqID:            r.Uniq,
			PartNumber:        r.PartNumber,
			PartName:          r.PartName,
			Model:             r.Model,
			QuantityScrap:     r.QuantityScrap,
			Weight:            r.Weight,
			UnitMeasurement:   r.Uom,
			ScrapType:         r.ScrapType,
			QuantityRework:    r.QuantityRework,
		})
	}
	return tasks, nil
}

// Rule (confirmed vs BRD):
//   - PASS      -> status APPROVED. Scrap is written ONLY when quantity_scrap > 0.
//   - NOT_PASS  -> status REJECTED. No scrap written.
func (s *service) SubmitReturnValidation(ctx context.Context, req models.SubmitReturnValidationRequest, validatedBy string) error {
	returnID := strings.TrimSpace(req.ReturnID)
	if returnID == "" {
		return errors.New("return_id is required")
	}

	id, err := strconv.ParseUint(returnID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid return_id: %w", err)
	}

	newStatus := "APPROVED"
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if action == "NOT_PASS" || status == "rejected" {
		newStatus = "REJECTED"
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) Load the product return row.
		var pr models.ProductReturnRow
		if err := tx.First(&pr, uint(id)).Error; err != nil {
			return err
		}

		// 2) Update its status.
		if err := tx.Model(&pr).Update("status", newStatus).Error; err != nil {
			return err
		}

		// 3) Scrap DB update ONLY when validated PASS and there is scrap qty.
		if newStatus == "APPROVED" && pr.QuantityScrap > 0 {
			// Lookup part info dari tabel items berdasarkan uniq_code
			// (product_returns tidak menyimpan part_number/part_name/model).
			var item struct {
				PartNumber string `gorm:"column:part_number"`
				PartName   string `gorm:"column:part_name"`
				Model      string `gorm:"column:model"`
			}
			_ = tx.Table("items").
				Select("part_number, part_name, model").
				Where("uniq_code = ?", pr.Uniq).
				Take(&item).Error // abaikan error: kalau item tidak ada, field dibiarkan kosong

			scrap := scrapModels.ScrapStock{
				UUID:      uuid.New(),
				UniqCode:  pr.Uniq,
				ScrapType: scrapModels.ScrapTypeProductReturn,
				Quantity:  float64(pr.QuantityScrap),
				Status:    scrapModels.StockStatusActive,
			}

			// Part info hasil lookup
			if strings.TrimSpace(item.PartNumber) != "" {
				pn := item.PartNumber
				scrap.PartNumber = &pn
			}
			if strings.TrimSpace(item.PartName) != "" {
				pnm := item.PartName
				scrap.PartName = &pnm
			}
			if strings.TrimSpace(item.Model) != "" {
				md := item.Model
				scrap.Model = &md
			}
			// Packing number: pakai dn_number yang tersimpan saat submit-to-qc
			if strings.TrimSpace(pr.DNNumber) != "" && pr.DNNumber != "-" {
				pkg := pr.DNNumber
				scrap.PackingNumber = &pkg
			}

			if strings.TrimSpace(pr.Uom) != "" {
				uom := pr.Uom
				scrap.UOM = &uom
			}
			if pr.Weight > 0 {
				w := pr.Weight
				scrap.WeightKg = &w
			}
			if pr.DateReceived != nil {
				scrap.DateReceived = pr.DateReceived
			}
			if strings.TrimSpace(validatedBy) != "" {
				v := validatedBy
				scrap.Validator = &v
				scrap.CreatedBy = &v
			}

			// Cek apakah sudah ada scrap AKTIF dgn uniq_code + tipe yang sama.
			addQty := float64(pr.QuantityScrap)

			var scrapID int64
			var beforeQty float64

			var existing scrapModels.ScrapStock
			findErr := tx.Where(
				"uniq_code = ? AND scrap_type = ? AND status = ?",
				pr.Uniq, scrapModels.ScrapTypeProductReturn, scrapModels.StockStatusActive,
			).First(&existing).Error

			switch {
			case findErr == nil:
				// uniq sudah ada -> akumulasikan quantity
				scrapID = existing.ID
				beforeQty = existing.Quantity
				if err := tx.Model(&scrapModels.ScrapStock{}).
					Where("id = ?", existing.ID).
					Update("quantity", existing.Quantity+addQty).Error; err != nil {
					return err
				}
			case errors.Is(findErr, gorm.ErrRecordNotFound):
				// belum ada -> buat baris baru
				if err := tx.Create(&scrap).Error; err != nil {
					return err
				}
				scrapID = scrap.ID // di-populate GORM setelah Create
				beforeQty = 0
			default:
				return findErr
			}

			// Tulis history log ke inventory_movement_logs supaya muncul di
			// endpoint history-logs (difilter: movement_category='scrap' & entity_id=scrapID).
			var refNo *string
			if strings.TrimSpace(pr.DNNumber) != "" && pr.DNNumber != "-" {
				dn := pr.DNNumber
				refNo = &dn
			}
			if err := s.createInventoryMovementLog(
				tx, "scrap", "incoming",
				pr.Uniq, &scrapID,
				beforeQty, addQty, beforeQty+addQty,
				refNo, stringPtr("PRODUCT_RETURN"),
				stringPtr("Incoming scrap from Product Return QC approval"),
				stringPtr(validatedBy),
			); err != nil {
				return err
			}
		}

		// 4) If newStatus == "APPROVED" and there is rework qty, auto-create Rework WO
		if newStatus == "APPROVED" && pr.QuantityRework > 0 {
			woNumber := fmt.Sprintf("WO-RW-%s-%d", time.Now().Format("20060102"), pr.ID)
			now := time.Now()

			wo := woModels.WorkOrder{
				UUID:           uuid.New(),
				WoNumber:       woNumber,
				WoType:         "Rework",
				WOKind:         "standard",
				Status:         "New",
				ApprovalStatus: "Approved",
				CreatedDate:    now,
			}
			if err := tx.Create(&wo).Error; err != nil {
				return err
			}

			woItem := woModels.WorkOrderItem{
				UUID:         uuid.New(),
				WoID:         wo.ID,
				ItemUniqCode: pr.Uniq,
				Quantity:     float64(pr.QuantityRework),
				Status:       "New",
				KanbanNumber: fmt.Sprintf("KBN-RW-%d", pr.ID),
			}
			if err := tx.Create(&woItem).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) buildItemRawMaterials(ctx context.Context, item models.WorkOrderItem) ([]dto.ScanOutContextRawMaterial, error) {
	logs, err := s.repoProduction.FindProductionRawMaterialLogsByWOItemID(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	logs = dedupRawMaterialLogs(logs)

	uuids := make([]string, 0, len(logs))
	codes := make([]string, 0, len(logs))
	for _, l := range logs {
		if l.RMUUID != "" {
			uuids = append(uuids, l.RMUUID)
		}
		if l.UniqCode != "" {
			codes = append(codes, l.UniqCode)
		}
	}

	metaByUUID := map[string]repository.RawMaterialMeta{}
	metaByCode := map[string]repository.RawMaterialMeta{}
	if len(uuids) > 0 || len(codes) > 0 {
		metas, err := s.repoProduction.FindRawMaterialMetaByKeys(ctx, uuids, codes)
		if err == nil {
			for _, m := range metas {
				if m.UUID != "" {
					metaByUUID[m.UUID] = m
				}
				if m.UniqCode != "" {
					metaByCode[m.UniqCode] = m
				}
			}
		}
	}

	bomMap := s.bomByUniq(ctx, item.ItemUniqCode)
	rms := make([]dto.ScanOutContextRawMaterial, 0, len(logs))
	for _, l := range logs {
		meta, ok := metaByUUID[l.RMUUID]
		if !ok {
			meta, ok = metaByCode[l.UniqCode]
		}

		packing := l.UniqCode
		if packing == "" {
			packing = l.PartNumber
		}
		bm := bomMap[l.UniqCode]
		materialCode := bm.MaterialGrade
		if materialCode == "" {
			materialCode = packing
		}
		form := bm.Form
		specWeight := 0.0
		if bm.WeightKg != nil {
			specWeight = *bm.WeightKg
		}
		uom := l.UOM
		typeLabel := "OTHER"
		var avail, weight float64
		if ok {
			if meta.UOM != "" {
				uom = meta.UOM
			}
			typeLabel = rmTypeLabel(meta.RawMaterialType)
			avail = meta.StockQty
			weight = meta.StockWeightKg
		}
		rms = append(rms, dto.ScanOutContextRawMaterial{
			RMUUID:         l.RMUUID,
			PackingListRM:  packing,
			MaterialCode:   materialCode,
			Form:           form,
			QtyPerUniq:     bm.QtyPerUniq,
			SpecWeightKg:   specWeight,
			TypeLabel:      typeLabel,
			IsWIP:          strings.EqualFold(typeLabel, "WIP"),
			UOM:            uom,
			AvailableStock: avail,
			StockWeightKg:  weight,
			QtyUsed:        l.QtyUsed,
		})
	}
	return rms, nil
}
