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

	// [overflow-topup] rencana penempatan kelebihan (utk modal konfirmasi FE)
	PreviewQCOverflow(ctx context.Context, qcTaskID int64, tpq float64) (dto.QCOverflowPreview, error)

	ListQCTask(ctx context.Context, req dto.ListQCTaskRequest) (map[string]interface{}, error)

	// [qc-round-db] ambil ronde tersubmit per qc_task_id (unik per kanban) dari DB
	ListQCRounds(ctx context.Context, qcTaskID int64) (map[string]interface{}, error)

	IssueList(ctx context.Context) (map[string]interface{}, error)
	WOList(ctx context.Context, search string) ([]dto.WOListItem, error)
	WODetail(ctx context.Context, woNumber string) (*dto.WODetailResponse, error)
	RawMaterialLookup(ctx context.Context, code string) (*dto.RawMaterialLookupResponse, error)

	ScanMachine(ctx context.Context, req dto.ScanMachineRequest) (*dto.ScanMachineResponse, error)

	ScanOutContext(ctx context.Context, woNumber string) (*dto.ScanOutContextResponse, error)

	// [repacking] list packing/kanban milik satu Raw Material (buat modal Repacking).
	RMPackingList(ctx context.Context, rmUUID, code string) (*dto.RMPackingListResponse, error)
	// [repack-sisa] pindahkan sisa material ke packing list RM lain.
	RMRepack(ctx context.Context, req dto.RMRepackRequest) (*dto.RMRepackResponse, error)
	// [scanin-draft-db] draft scan-in (seed) bersama lintas gadget.
	ListScanInDrafts(ctx context.Context, woID int64, currentStep int) (*dto.ListScanInDraftsResponse, error)
	UpsertScanInDraft(ctx context.Context, req dto.UpsertScanInDraftRequest, updatedBy string) error
	DeleteScanInDraft(ctx context.Context, req dto.DeleteScanInDraftRequest) error

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

	fmt.Printf("DEBUG SCAN_IN -> ID: %d | ScanInCount: %d | ScanOutCount: %d\n", item.ID, item.ScanInCount, item.ScanOutCount)

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
		// [proc-scope] jangan hapus log RM milik proses lain.
		if err := s.repoProduction.DeleteRawMaterialLogsByWOItemStep(ctx, item.ID, currentStepSeqOf(item)); err != nil {
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
	wipItem, err := s.repoProduction.FindQueueWIPItem(
		ctx,
		wip.ID,
		item.ItemUniqCode,
		currentProcess,
		currentIndex+1,
	)

	if err != nil {
		// Cek apakah sudah ada status 'process' untuk seq ini
		var existingWIP models.WIPItem
		errExist := s.db.WithContext(ctx).
			Where("wip_id = ? AND uniq = ? AND process_name = ? AND seq = ? AND status = ?",
				wip.ID, item.ItemUniqCode, currentProcess, currentIndex+1, "process").
			First(&existingWIP).Error

		if errExist == nil {
			wipItem = existingWIP
		} else {
			// kalau benar-benar belum ada (hanya boleh di step pertama), baru create fresh
			if currentIndex > 0 {
				return errors.New("WIP item tidak ditemukan untuk proses ini")
			}

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
			// [rm-source] detail bebas untuk jenis issue "Lainnya"
			IssueNote:     strings.TrimSpace(req.ProductIssueNote),
			IssueDuration: req.ProductIssueDuration,
			QtyAffected:   req.Qty,
			ReportedBy:    req.ScannedBy,
			ReportedAt:    now,
			CreatedAt:     now,
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

	if err := s.repoProduction.UpdateWOItem(ctx, item); err != nil {
		return err
	}

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
		// [overflow-qc] Pemecahan kanban kelebihan TIDAK lagi dilakukan di Scan Out.
		// Seluruh qty scan-out diteruskan (ke FG pada proses terakhir / ke WIP proses
		// berikutnya). Pembuatan kanban baru untuk kelebihan kini terjadi di QC
		// Process Round 3 (QCFinish) berdasarkan Total Production Quantity.
		item.TotalGoodQty = qtyOut

		if err := s.advanceProcessAfterScanOut(ctx, tx, &item, qtyOut, req.ScannedBy); err != nil {
			return err
		}
		return nil
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
			UUID: uuid.New().String(),
			// [qc-round-db] scope ronde ke qc_task_id (unik per kanban)
			QCTaskID:    &req.QCTaskID,
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

			// [qc-issue-table] tipe + ringkasan issue dari round 1 / 2
			IssueType: qcIssuePrimaryType(req),
			IssueNote: qcTrim255(qcIssueSummary(req)),
			CheckedAt: now,
			CreatedAt: now,
		}

		if err := tx.Create(&qc).Error; err != nil {
			return err
		}

		// =====================================
		// JIKA REJECT
		// =====================================
		// [qc-reason] simpan keterangan defect / scrap round ini
		if err := s.saveQCReasons(tx, qc, item, req, performedBy, now); err != nil {
			return err
		}

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

// [overflow-kanban] createOverflowKanban membuat kanban baru untuk kelebihan
// produksi (Total Production > qty rencana) pada PROSES TERAKHIR. Kanban ini
// langsung FINISHED, TIDAK dibuatkan QC task (skip QC Process), dan qty-nya
// langsung masuk Finished Goods lewat advanceProcessAfterQCFinish.
func (s *service) createOverflowKanban(ctx context.Context, tx *gorm.DB, base models.WorkOrderItem, wo models.WorkOrder, excess float64, performedBy string, now time.Time) error {
	if excess <= 0 {
		return nil
	}

	newKanban := s.nextOverflowKanbanNumber(tx, base)

	newItem := models.WorkOrderItem{
		UUID:               uuid.New().String(),
		WOID:               base.WOID,
		ItemUniqCode:       base.ItemUniqCode,
		PartName:           base.PartName,
		PartNumber:         base.PartNumber,
		UOM:                base.UOM,
		Quantity:           excess,
		ProcessName:        base.ProcessName,
		KanbanNumber:       newKanban,
		ProcessFlowJSON:    base.ProcessFlowJSON,
		CurrentStepSeq:     base.CurrentStepSeq,
		Status:             "PENDING",
		LastScannedProcess: "",
		ScanInCount:        0,
		ScanOutCount:       1,
		TotalGoodQty:       excess,
		Model:              base.Model,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := tx.Create(&newItem).Error; err != nil {
		return err
	}

	// Catat scan-out untuk jejak audit. Tidak ada createQCTaskIfNeeded -> skip QC.
	if err := tx.Create(&models.ProductionScanLog{
		UUID:         uuid.New().String(),
		WOID:         base.WOID,
		WOItemID:     newItem.ID,
		KanbanNumber: newKanban,
		ProcessName:  currentProcessNameOf(base),
		ScanType:     "SCAN_OUT",
		QtyOutput:    excess,
		ScannedBy:    performedBy,
		ScannedAt:    now,
		CreatedAt:    now,
	}).Error; err != nil {
		return err
	}

	// Masukkan kelebihan langsung ke Finished Goods (proses terakhir).
	return s.advanceProcessAfterQCFinish(ctx, tx, &newItem, wo, excess, performedBy, now)
}

// [overflow-kanban] nextOverflowKanbanNumber menghasilkan nomor kanban unik
// untuk kanban kelebihan, pola "<kanban asli>-OF01", "-OF02", dst.
func (s *service) nextOverflowKanbanNumber(tx *gorm.DB, base models.WorkOrderItem) string {
	var cnt int64
	_ = tx.Model(&models.WorkOrderItem{}).
		Where("wo_id = ? AND item_uniq_code = ? AND kanban_number LIKE ?",
			base.WOID, base.ItemUniqCode, base.KanbanNumber+"-OF%").
		Count(&cnt).Error
	return fmt.Sprintf("%s-OF%02d", base.KanbanNumber, cnt+1)
}

func (s *service) advanceProcessAfterScanOut(ctx context.Context, tx *gorm.DB, item *models.WorkOrderItem, qtyPass float64, performedBy string) error {
	_ = qtyPass
	_ = performedBy

	// [bug4] Scan Out TIDAK lagi memindahkan barang ke WIP / Finished Goods
	// maupun menaikkan step proses. Perpindahan ke WIP (proses berikutnya) atau
	// ke Finished Goods (proses terakhir) dilakukan di QC Process Round 3
	// (QCFinish) berdasarkan Total Production Quantity.
	//
	// Pengecualian: WO rm_processing tidak melalui QC Round 3, jadi tetap
	// diselesaikan langsung di sini (output berupa raw material).
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

	// [REVERT bug4] Scan Out KEMBALI memindahkan barang ke WIP / Finished Goods
	// dan menaikkan step proses. User ingin bisa lanjut proses di POKA YOKE
	// segera setelah Scan Out (tanpa menunggu QC Round 3 selesai).
	var wo models.WorkOrder
	if err := tx.Where("id = ?", item.WOID).First(&wo).Error; err != nil {
		return err
	}

	return s.advanceProcessAfterQCFinish(ctx, tx, item, wo, qtyPass, performedBy, time.Now())
}

// advanceProcessAfterQCFinish dijalankan saat QC Process Round 3 (QCFinish)
// selesai. Berdasarkan Total Production Quantity, ia memutuskan tujuan barang:
//   - jika MASIH ADA proses berikutnya -> qty masuk ke WIP (queue) untuk proses
//     berikutnya, step dinaikkan, item kembali PENDING (siap Scan In lagi).
//   - jika ini proses TERAKHIR          -> qty masuk ke Finished Goods.
//
// WIP proses saat ini (status "process") ditutup (status "done") di sini.
func (s *service) advanceProcessAfterQCFinish(ctx context.Context, tx *gorm.DB, item *models.WorkOrderItem, wo models.WorkOrder, qtyProduced float64, performedBy string, now time.Time) error {
	if qtyProduced <= 0 {
		return nil
	}

	flow := resolveProcessFlow(*item)
	totalStep := len(flow)
	if totalStep == 0 {
		return nil
	}
	currentIndex := getCurrentIndex(item.CurrentStepSeq, totalStep)
	currentStep := flow[currentIndex]

	// Tutup WIP proses saat ini.
	var currentWIP models.WIPItem
	haveCurrentWIP := false
	// [wip-scope] WAJIB dibatasi ke WO ini. Tanpa filter wo_id, UNIQ yang sama
	// di WO lain bisa ikut tertutup dan WIP proses berikutnya dibuat di header
	// WO yang salah -> material WIP proses 2 tidak ketemu.
	if err := tx.
		Where(`wip_id IN (SELECT id FROM wips WHERE wo_id = ?)
			AND uniq = ? AND process_name = ? AND status = ? AND seq = ?`,
			item.WOID, item.ItemUniqCode, currentStep.ProcessName, "process", currentIndex+1).
		Order("id desc").
		First(&currentWIP).Error; err == nil {
		haveCurrentWIP = true
		currentWIP.QtyOut += int(qtyProduced)
		currentWIP.QtyRemaining = currentWIP.QtyIn - currentWIP.QtyOut
		if currentWIP.QtyRemaining < 0 {
			currentWIP.QtyRemaining = 0
		}
		currentWIP.Status = "done"
		currentWIP.UpdatedAt = now
		if err := tx.Save(&currentWIP).Error; err != nil {
			return err
		}
		_ = tx.Create(&models.WIPLog{
			WipItemID: currentWIP.ID,
			Action:    "SCAN_OUT_DONE",
			Qty:       int(qtyProduced),
			CreatedAt: now,
		}).Error
	}

	// Masih ada proses berikutnya -> masuk WIP (queue).
	if currentIndex < totalStep-1 {
		nextStep := flow[currentIndex+1]

		wipHeaderID := int64(0)
		if haveCurrentWIP {
			wipHeaderID = currentWIP.WipID
		} else {
			wip, err := s.repoProduction.FindOrCreateWIP(ctx, item.WOID)
			if err != nil {
				return err
			}
			wipHeaderID = wip.ID
		}

		nextWIP := models.WIPItem{
			WipID:         wipHeaderID,
			Uniq:          item.ItemUniqCode,
			PackingNumber: item.KanbanNumber,
			WipType:       "production",
			ProcessName:   nextStep.ProcessName,
			// [wip-scope] stok ini HASIL proses sekarang, belum diproses oleh
			// proses berikutnya. Dipakai untuk tampilan daftar WIP.
			FromProcess:  currentStep.ProcessName,
			MachineName:  derefString(nextStep.MachineName),
			OpSeq:        nextStep.OpSeq,
			Seq:          currentIndex + 2,
			UOM:          item.UOM,
			Stock:        int(qtyProduced),
			QtyIn:        int(qtyProduced),
			QtyOut:       0,
			QtyRemaining: int(qtyProduced),
			Status:       "queue",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&nextWIP).Error; err != nil {
			return err
		}
		_ = tx.Create(&models.WIPLog{
			WipItemID: nextWIP.ID,
			Action:    "TRANSFER_IN",
			Qty:       int(qtyProduced),
			CreatedAt: now,
		}).Error

		item.CurrentStepSeq = currentIndex + 2
		item.Status = "PENDING"
		item.LastScannedProcess = ""
		return tx.Save(item).Error
	}

	// Proses terakhir -> Finished Goods.
	var fg models.FinishedGoods
	errFG := tx.Where("uniq_code = ?", item.ItemUniqCode).First(&fg).Error
	if errors.Is(errFG, gorm.ErrRecordNotFound) {
		fg = models.FinishedGoods{
			UUID:       uuid.New().String(),
			UniqCode:   item.ItemUniqCode,
			ItemID:     item.ID,
			PartNumber: item.PartNumber,
			PartName:   item.PartName,
			Model:      item.Model,
			WONumber:   wo.WONumber,
			StockQty:   qtyProduced,
			UOM:        item.UOM,
			Status:     "AVAILABLE",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&fg).Error; err != nil {
			return err
		}
		if err := s.createInventoryMovementLog(tx, "finished_goods", "incoming",
			item.ItemUniqCode, &fg.ID, 0, qtyProduced, qtyProduced,
			&wo.WONumber, stringPtr("QC_FINISH"),
			stringPtr("Create FG from QC Finish (proses terakhir)"), stringPtr(performedBy)); err != nil {
			return err
		}
		if err := s.appendFGMovementLog(tx, fg.ID, item.ItemUniqCode, "incoming_production",
			qtyProduced, 0, qtyProduced,
			&wo.WONumber, nil, stringPtr("Masuk FG dari QC Finish (buat baru)"), stringPtr(performedBy)); err != nil {
			return err
		}
	} else if errFG == nil {
		beforeQty := fg.StockQty
		afterQty := beforeQty + qtyProduced
		fg.StockQty = afterQty
		fg.UpdatedAt = now
		if err := tx.Save(&fg).Error; err != nil {
			return err
		}
		if err := s.createInventoryMovementLog(tx, "finished_goods", "incoming",
			item.ItemUniqCode, &fg.ID, beforeQty, qtyProduced, afterQty,
			&wo.WONumber, stringPtr("QC_FINISH"),
			stringPtr("Add FG stock from QC Finish (proses terakhir)"), stringPtr(performedBy)); err != nil {
			return err
		}
		if err := s.appendFGMovementLog(tx, fg.ID, item.ItemUniqCode, "incoming_production",
			qtyProduced, beforeQty, afterQty,
			&wo.WONumber, nil, stringPtr("Masuk FG dari QC Finish (tambah stok)"), stringPtr(performedBy)); err != nil {
			return err
		}
	} else {
		return errFG
	}

	item.Status = "FINISHED"
	item.LastScannedProcess = ""
	return tx.Save(item).Error
}

// [qc-reason] jaga panjang teks supaya aman untuk kolom VARCHAR(255)
func qcTrim255(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 255 {
		return strings.TrimSpace(s[:255])
	}
	return s
}

// [qc-reason] simpan keterangan defect / scrap round 1 & 2 ke qc_defect_items
func (s *service) saveQCReasons(
	tx *gorm.DB,
	qc models.QCLog,
	item models.WorkOrderItem,
	req dto.QCSubmitRequest,
	performedBy string,
	now time.Time,
) error {
	issueType := strings.TrimSpace(req.IssueType)
	issueNote := qcTrim255(req.IssueNote)
	defectNote := qcTrim255(req.DefectReason)
	scrapNote := qcTrim255(req.ScrapReason)

	if defectNote == "" && issueNote != "" {
		defectNote = issueNote
	}

	reasonCode := issueType
	if reasonCode == "" {
		reasonCode = "defect"
	}

	add := func(code, text string, qtyDefect, qtyScrap float64) error {
		if qtyDefect <= 0 && qtyScrap <= 0 && text == "" {
			return nil
		}
		taskID := req.QCTaskID
		row := models.QCDefectItem{
			QCLogID:          qc.ID,
			QCTaskID:         &taskID,
			WOID:             &item.WOID,
			WOItemID:         &item.ID,
			UniqCode:         qc.UniqCode,
			DefectSource:     "process",
			DefectReasonCode: code,
			DefectReasonText: text,
			QtyDefect:        qtyDefect,
			QtyScrap:         qtyScrap,
			ProcessName:      qc.ProcessName,
			ReportedBy:       performedBy,
			ReportedAt:       now,
		}
		return tx.Create(&row).Error
	}

	if err := add(reasonCode, defectNote, req.QtyDefect, 0); err != nil {
		return err
	}
	if err := add("scrap", scrapNote, 0, req.QtyScrap); err != nil {
		return err
	}

	// [qc-issue-table] simpan tiap baris issue (round 1/2) ke qc_defect_items
	taskID2 := req.QCTaskID
	for _, it := range req.Issues {
		code := strings.TrimSpace(it.Issue)
		text := qcTrim255(strings.TrimSpace(it.Detail))
		if code == "" && text == "" && it.Qty <= 0 {
			continue
		}
		if code == "" {
			code = "issue"
		}
		row := models.QCDefectItem{
			QCLogID:          qc.ID,
			QCTaskID:         &taskID2,
			WOID:             &item.WOID,
			WOItemID:         &item.ID,
			UniqCode:         qc.UniqCode,
			DefectSource:     "issue",
			DefectReasonCode: code,
			DefectReasonText: text,
			Qty:              it.Qty,
			ProcessName:      qc.ProcessName,
			ReportedBy:       performedBy,
			ReportedAt:       now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// [qc-issue-table] tipe issue utama (baris pertama) untuk kolom ringkas qc_logs
func qcIssuePrimaryType(req dto.QCSubmitRequest) string {
	if len(req.Issues) > 0 {
		return strings.TrimSpace(req.Issues[0].Issue)
	}
	return strings.TrimSpace(req.IssueType)
}

// [qc-issue-table] ringkasan semua issue -> "issue: detail (qty); ..."
func qcIssueSummary(req dto.QCSubmitRequest) string {
	if len(req.Issues) == 0 {
		return req.IssueNote
	}
	parts := make([]string, 0, len(req.Issues))
	for _, it := range req.Issues {
		seg := strings.TrimSpace(it.Issue)
		detail := strings.TrimSpace(it.Detail)
		if detail != "" {
			if seg != "" {
				seg += ": "
			}
			seg += detail
		}
		seg = strings.TrimSpace(fmt.Sprintf("%s (%g)", seg, it.Qty))
		parts = append(parts, seg)
	}
	return strings.Join(parts, "; ")
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

		// ===== 1. ROUTING PRODUKSI (QC Round 3): WIP proses berikutnya / Finished Goods =====
		// [REVERT bug4] Perpindahan barang ke WIP / FG KINI dilakukan di saat Scan Out
		// (advanceProcessAfterScanOut), BUKAN lagi di sini, agar user tidak perlu
		// menunggu QC untuk lanjut ke proses selanjutnya.
		//
		// if req.TotalProductionQty > 0 {
		// 	if err := s.advanceProcessAfterQCFinish(ctx, tx, &item, wo, req.TotalProductionQty, performedBy, now); err != nil {
		// 		return err
		// 	}
		// }

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

		// ===== 4. (dihapus) MOVE WIP -> FG kini ditangani advanceProcessAfterQCFinish =====
		// [bug4] Perpindahan WIP -> Finished Goods (atau ke WIP proses berikutnya)
		// sudah ditangani di advanceProcessAfterQCFinish (section 1), jadi tidak ada
		// lagi reduceWIPToFinishedGoods di sini.

		// [qc-reason] keterangan defect (NG) / scrap dari Final Inspection.
		// Hanya ditulis kalau memang diisi, supaya alur lama tetap jalan.
		if dr, sr := qcTrim255(req.DefectReason), qcTrim255(req.ScrapReason); dr != "" || sr != "" {
			if err := tx.Model(&task).Updates(map[string]interface{}{
				"defect_reason": dr,
				"scrap_reason":  sr,
			}).Error; err != nil {
				return err
			}
		}

		// [qc-issue-table] Round 3: simpan Reason/Info NG & Scrap ke qc_defect_items
		if len(req.NGReasons) > 0 || len(req.ScrapReasons) > 0 {
			taskID3 := req.QCTaskID
			qc3 := models.QCLog{
				UUID:       uuid.New().String(),
				QCTaskID:   &taskID3,
				WOID:       &item.WOID,
				WOItemID:   &item.ID,
				UniqCode:   item.ItemUniqCode,
				QCRound:    3,
				QtyChecked: req.TotalProductionQty + req.NGDefectQty + req.TotalScrapInBox,
				QtyDefect:  req.NGDefectQty,
				QtyScrap:   req.TotalScrapInBox,
				Status:     "FINISH",
				CheckedBy:  performedBy,
				CheckedAt:  now,
				CreatedAt:  now,
			}
			if err := tx.Create(&qc3).Error; err != nil {
				return err
			}
			saveReason := func(source string, rows []dto.QCReasonInput) error {
				for _, r := range rows {
					text := qcTrim255(strings.TrimSpace(r.Info))
					if text == "" && r.Qty <= 0 {
						continue
					}
					row := models.QCDefectItem{
						QCLogID:          qc3.ID,
						QCTaskID:         &taskID3,
						WOID:             &item.WOID,
						WOItemID:         &item.ID,
						UniqCode:         item.ItemUniqCode,
						DefectSource:     source,
						DefectReasonText: text,
						Qty:              r.Qty,
						ReportedBy:       performedBy,
						ReportedAt:       now,
					}
					if err := tx.Create(&row).Error; err != nil {
						return err
					}
				}
				return nil
			}
			if err := saveReason("ng_reason", req.NGReasons); err != nil {
				return err
			}
			if err := saveReason("scrap_reason", req.ScrapReasons); err != nil {
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

		// [overflow-qc] Buat kanban baru bila Total Production Quantity (TPQ) melebihi
		// qty rencana kanban (maks kanban). Kelebihannya jadi kanban baru murni good
		// yang otomatis ter-QC (mengikuti kanban utama). Kanban utama dibatasi good =
		// min(TPQ, maksKanban); NG & scrap tetap di kanban utama.
		maxKanban := item.Quantity
		tpq := req.TotalProductionQty
		mainGood := tpq
		if maxKanban > 0 && mainGood > maxKanban {
			mainGood = maxKanban
		}
		if err := tx.Model(&models.WorkOrderItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"total_good_qty":  mainGood,
				"total_ng_qty":    req.NGDefectQty,
				"total_scrap_qty": req.TotalScrapInBox,
				"updated_at":      now,
			}).Error; err != nil {
			return err
		}
		if maxKanban > 0 && tpq > maxKanban {
			excess := tpq - maxKanban
			// [overflow-topup] isi slot kanban overflow existing dulu, sisanya kanban baru
			if err := s.distributeQCOverflow(ctx, tx, item, wo, excess, maxKanban, req.SkipOverflowTopUp, performedBy, now); err != nil {
				return err
			}
		}

		return nil
	})
}

// [overflow-qc] createQCOverflowKanban membuat kanban baru dari kelebihan Total
// Production Quantity saat QC Finish. Kanban ini MURNI good, langsung FINISHED,
// mewarisi data Step-1 (mesin, dandori, setup QC) & hasil QC (pass) dari kanban
// sumber. TIDAK menambah Finished Goods (qty sudah masuk FG saat Scan Out);
// ini hanya pemecahan record kanban. Idempoten via overflow_source_item_id.
func (s *service) createQCOverflowKanban(ctx context.Context, tx *gorm.DB, base models.WorkOrderItem, wo models.WorkOrder, excess float64, performedBy string, now time.Time) error {
	if excess <= 0 {
		return nil
	}

	var existing int64
	if err := tx.Model(&models.WorkOrderItem{}).
		Where("overflow_source_item_id = ?", base.ID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	// [overflow-qc-split] Pecah kelebihan (excess) menjadi BEBERAPA kanban, tiap
	// kanban maksimal = kapasitas kanban (base.Quantity). Sebelumnya seluruh
	// kelebihan dijadikan SATU kanban sehingga qty-nya bisa melebihi maks kanban
	// (mis. maks 10 tapi kanban overflow terisi 40). Kini 40 -> 4 kanban @ 10.
	maxPer := base.Quantity
	if maxPer <= 0 {
		maxPer = excess
	}

	remaining := excess
	for remaining > 1e-9 {
		chunk := remaining
		if chunk > maxPer {
			chunk = maxPer
		}
		remaining -= chunk

		newKanban := s.nextOverflowKanbanNumber(tx, base)
		srcID := base.ID
		newItem := models.WorkOrderItem{
			OverflowSourceItemID: &srcID,
			UUID:                 uuid.New().String(),
			WOID:                 base.WOID,
			ItemUniqCode:         base.ItemUniqCode,
			PartName:             base.PartName,
			PartNumber:           base.PartNumber,
			UOM:                  base.UOM,
			Quantity:             chunk,
			ProcessName:          base.ProcessName,
			KanbanNumber:         newKanban,
			MachineID:            base.MachineID,
			ProcessFlowJSON:      base.ProcessFlowJSON,
			CurrentStepSeq:       base.CurrentStepSeq,
			Status:               "FINISHED",
			LastScannedProcess:   "",
			ScanInCount:          1,
			ScanOutCount:         1,
			TotalGoodQty:         chunk,
			Model:                base.Model,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := tx.Create(&newItem).Error; err != nil {
			return err
		}

		// Bawa data Step-1 (dandori & setup QC): WODetail membaca dari SCAN_IN log
		// terakhir. Buat SCAN_IN tiruan dari snapshot scan-in kanban sumber.
		if srcIn, e := s.repoProduction.FindLatestScanInLog(ctx, base.ID); e == nil {
			if err := tx.Create(&models.ProductionScanLog{
				UUID:           uuid.New().String(),
				WOID:           base.WOID,
				WOItemID:       newItem.ID,
				MachineID:      srcIn.MachineID,
				KanbanNumber:   newKanban,
				ProcessName:    currentProcessNameOf(base),
				ProductionLine: srcIn.ProductionLine,
				ScanType:       "SCAN_IN",
				QtyInput:       chunk,
				Shift:          srcIn.Shift,
				DandoriTime:    srcIn.DandoriTime,
				SetupQCTime:    srcIn.SetupQCTime,
				ScannedBy:      performedBy,
				ScannedAt:      now,
				CreatedAt:      now,
			}).Error; err != nil {
				return err
			}
		}

		// SCAN_OUT log untuk jejak audit kanban baru.
		if err := tx.Create(&models.ProductionScanLog{
			UUID:         uuid.New().String(),
			WOID:         base.WOID,
			WOItemID:     newItem.ID,
			MachineID:    int64(base.MachineID),
			KanbanNumber: newKanban,
			ProcessName:  currentProcessNameOf(base),
			ScanType:     "SCAN_OUT",
			QtyOutput:    chunk,
			ScannedBy:    performedBy,
			ScannedAt:    now,
			CreatedAt:    now,
		}).Error; err != nil {
			return err
		}

		// Mirror QC (pass) ke kanban baru: QCTask done + QCLog round 3 FINISH,
		// good = chunk, tanpa NG/scrap.
		gq := int(chunk)
		zero := 0
		// [overflow-qc-rr] qc_tasks.round_results NOT NULL -> isi JSON ringkas.
		rrOF := fmt.Sprintf(`{"uniq":%q,"kanban_number":%q,"process_name":%q,"wo_id":%d,"wo_item_id":%d,"qty":%g,"auto_overflow":true}`,
			newItem.ItemUniqCode, newKanban, currentProcessNameOf(base), newItem.WOID, newItem.ID, chunk)
		mirrorTask := models.QCTask{
			TaskType:      "production_qc",
			Status:        "done",
			RoundResults:  datatypes.JSON([]byte(rrOF)),
			WOID:          &newItem.WOID,
			WOItemID:      &newItem.ID,
			ProcessName:   currentProcessNameOf(base),
			Round:         3,
			GoodQuantity:  &gq,
			NgQuantity:    &zero,
			ScrapQuantity: &zero,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := tx.Create(&mirrorTask).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.QCLog{
			UUID:        uuid.New().String(),
			QCTaskID:    &mirrorTask.ID,
			WOID:        &newItem.WOID,
			WOItemID:    &newItem.ID,
			UniqCode:    newItem.ItemUniqCode,
			QCRound:     3,
			QtyChecked:  chunk,
			QtyPass:     chunk,
			QtyDefect:   0,
			QtyScrap:    0,
			Status:      "FINISH",
			ProcessName: currentProcessNameOf(base),
			CheckedBy:   performedBy,
			CheckedAt:   now,
			CreatedAt:   now,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

// [overflow-topup] overflowSourceEntry satu catatan sumber kelebihan.
type overflowSourceEntry struct {
	SourceWOItemID int64     `json:"source_wo_item_id"`
	SourceKanban   string    `json:"source_kanban"`
	WOID           int64     `json:"wo_id"`
	WONumber       string    `json:"wo_number"`
	Qty            float64   `json:"qty"`
	CreatedAt      time.Time `json:"created_at"`
}

// [overflow-topup] findOverflowSlots: kanban overflow (FG/uniq sama, lintas
// SEMUA WO) yang masih punya slot kosong (quantity < maxKanban), urut terlama.
func (s *service) findOverflowSlots(tx *gorm.DB, uniq string, maxKanban float64, excludeItemID int64) ([]models.WorkOrderItem, error) {
	var rows []models.WorkOrderItem
	q := tx.Model(&models.WorkOrderItem{}).
		Where("item_uniq_code = ?", uniq).
		Where("overflow_source_item_id IS NOT NULL").
		Where("quantity < ?", maxKanban).
		Order("created_at ASC, id ASC")
	if excludeItemID > 0 {
		q = q.Where("id <> ?", excludeItemID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// [overflow-topup] appendOverflowSource menambah 1 entry sumber ke JSON array.
func appendOverflowSource(existing datatypes.JSON, entry overflowSourceEntry) (datatypes.JSON, error) {
	var arr []overflowSourceEntry
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &arr)
	}
	arr = append(arr, entry)
	b, err := json.Marshal(arr)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

// [overflow-topup] distributeQCOverflow membagi kelebihan (excess) ke kanban
// overflow existing yang masih ada slot (top-up) lalu sisanya jadi kanban baru.
// Bila skipTopUp true (user pilih Batal di modal) lewati top-up.
func (s *service) distributeQCOverflow(ctx context.Context, tx *gorm.DB, base models.WorkOrderItem, wo models.WorkOrder, excess, maxKanban float64, skipTopUp bool, performedBy string, now time.Time) error {
	if excess <= 0 {
		return nil
	}
	remaining := excess
	if !skipTopUp && maxKanban > 0 {
		slots, err := s.findOverflowSlots(tx, base.ItemUniqCode, maxKanban, base.ID)
		if err != nil {
			return err
		}
		for i := range slots {
			if remaining <= 0 {
				break
			}
			free := maxKanban - slots[i].Quantity
			if free <= 0 {
				continue
			}
			fill := free
			if fill > remaining {
				fill = remaining
			}
			nextSrc, err := appendOverflowSource(slots[i].OverflowSources, overflowSourceEntry{
				SourceWOItemID: base.ID,
				SourceKanban:   base.KanbanNumber,
				WOID:           base.WOID,
				WONumber:       wo.WONumber,
				Qty:            fill,
				CreatedAt:      now,
			})
			if err != nil {
				return err
			}
			if err := tx.Model(&models.WorkOrderItem{}).
				Where("id = ?", slots[i].ID).
				Updates(map[string]interface{}{
					"quantity":         slots[i].Quantity + fill,
					"total_good_qty":   slots[i].TotalGoodQty + fill,
					"overflow_sources": nextSrc,
					"updated_at":       now,
				}).Error; err != nil {
				return err
			}
			remaining -= fill
		}
	}
	if remaining <= 0 {
		return nil
	}
	return s.createQCOverflowKanban(ctx, tx, base, wo, remaining, performedBy, now)
}

// [overflow-topup] PreviewQCOverflow menghitung rencana penempatan kelebihan
// Total Production Quantity saat QC Round 3 finish TANPA menulis apa pun.
func (s *service) PreviewQCOverflow(ctx context.Context, qcTaskID int64, tpq float64) (dto.QCOverflowPreview, error) {
	var out dto.QCOverflowPreview
	if qcTaskID == 0 {
		return out, apperror.BadRequest("qc_task_id is required")
	}
	var task models.QCTask
	if err := s.db.WithContext(ctx).Where("id = ?", qcTaskID).First(&task).Error; err != nil {
		return out, err
	}
	if task.WOItemID == nil {
		return out, apperror.BadRequest("qc task tidak punya wo_item")
	}
	var item models.WorkOrderItem
	if err := s.db.WithContext(ctx).Where("id = ?", *task.WOItemID).First(&item).Error; err != nil {
		return out, err
	}
	maxKanban := item.Quantity
	out.MaxKanban = maxKanban
	out.MainGood = tpq
	if maxKanban > 0 && out.MainGood > maxKanban {
		out.MainGood = maxKanban
	}
	if maxKanban <= 0 || tpq <= maxKanban {
		return out, nil
	}
	excess := tpq - maxKanban
	out.Excess = excess
	slots, err := s.findOverflowSlots(s.db.WithContext(ctx), item.ItemUniqCode, maxKanban, item.ID)
	if err != nil {
		return out, err
	}
	remaining := excess
	for i := range slots {
		if remaining <= 0 {
			break
		}
		free := maxKanban - slots[i].Quantity
		if free <= 0 {
			continue
		}
		fill := free
		if fill > remaining {
			fill = remaining
		}
		var woNumber string
		_ = s.db.WithContext(ctx).Table("work_orders").Select("wo_number").Where("id = ?", slots[i].WOID).Take(&woNumber).Error
		out.TopUps = append(out.TopUps, dto.QCOverflowTopUp{
			WOItemID:     slots[i].ID,
			KanbanNumber: slots[i].KanbanNumber,
			WONumber:     woNumber,
			FreeBefore:   free,
			Fill:         fill,
		})
		remaining -= fill
	}
	out.HasTopUp = len(out.TopUps) > 0
	out.NewKanbanQty = remaining
	return out, nil
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
		"uuid":         woUUID,
		"wo_number":    woNumber,
		"wo_type":      "Rework", // enum work_orders: New | Assembly | Rework | Addendum
		"wo_kind":      "standard",
		"reference_wo": srcWO.WONumber,
		// [wo-defect-reason-scope] simpan wo_item SUMBER (kanban spesifik) agar detail WO
		// rework hanya menampilkan Reason/Info Defect milik kanban ini, bukan gabungan
		// semua kanban dengan uniq yang sama.
		"reference_wo_item_id": item.ID,
		"status":               "Draft",
		"approval_status":      "Pending",
		"created_date":         now,
		"operator_name":        performedBy,
		"model":                item.Model,
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

// [qc-round-db] ListQCRounds mengambil Round 1 & 2 yang sudah tersubmit untuk
// satu wo_item, direkonstruksi dari qc_logs (angka) + qc_defect_items (reason &
// issue). Dipakai frontend QC Process agar data ronde tetap tampil lintas
// gadget/browser tanpa bergantung pada localStorage.
func (s *service) ListQCRounds(ctx context.Context, qcTaskID int64) (map[string]interface{}, error) {
	if qcTaskID == 0 {
		return nil, apperror.BadRequest("qc_task_id is required")
	}

	// [qc-round-db] status scan-out diambil dari DB (WorkOrderItem milik task),
	// bukan localStorage, supaya gate Round 3 konsisten lintas gadget.
	scanOutCompleted := false
	totalProduction := float64(0)
	var task models.QCTask
	if err := s.db.WithContext(ctx).
		Where(&models.QCTask{ID: qcTaskID}).
		First(&task).Error; err != nil {
		return nil, err
	}
	if task.WOItemID != nil {
		// [qc-scanout] Sumber kebenaran scan-out = ProductionScanLog (scan_type SCAN_OUT)
		// untuk wo_item + proses task ini. Counter ScanInCount/ScanOutCount TIDAK dipakai
		// karena ScanInCount tidak selalu tersimpan (ScanIn tidak mem-persist counter),
		// sehingga kondisi lama selalu false walau uniq sudah scan out.
		var scanOutLogCount int64
		q := s.db.WithContext(ctx).
			Model(&models.ProductionScanLog{}).
			Where("wo_item_id = ? AND scan_type = ?", *task.WOItemID, "SCAN_OUT")
		if task.ProcessName != "" {
			q = q.Where("process_name = ?", task.ProcessName)
		}
		if err := q.Count(&scanOutLogCount).Error; err == nil {
			scanOutCompleted = scanOutLogCount > 0
		}
		if item, err := s.repoProduction.FindWOItemByID(ctx, *task.WOItemID); err == nil {
			totalProduction = item.TotalGoodQty
		}
	}

	// [qc-round-db] ronde tersubmit di-scope ke qc_task_id (unik per kanban),
	// bukan wo_item_id yang bisa sama antar kanban dalam satu WO item.
	var logs []models.QCLog
	if err := s.db.WithContext(ctx).
		Where(&models.QCLog{QCTaskID: &qcTaskID}).
		Order("id asc").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	// Ambil log terbaru per round (slice sudah urut id asc, jadi entri terakhir
	// menang bila ada submit berulang untuk round yang sama).
	latest := map[int]models.QCLog{}
	for _, lg := range logs {
		if lg.QCRound == 1 || lg.QCRound == 2 {
			latest[lg.QCRound] = lg
		}
	}

	rounds := make([]dto.QCRoundItem, 0, 2)
	for _, rnd := range []int{1, 2} {
		lg, ok := latest[rnd]
		if !ok {
			continue
		}

		item := dto.QCRoundItem{
			Round:      lg.QCRound,
			QtyChecked: lg.QtyChecked,
			QtyPass:    lg.QtyPass,
			QtyDefect:  lg.QtyDefect,
			QtyScrap:   lg.QtyScrap,
			Issues:     make([]dto.QCRoundIssue, 0),
		}

		var defItems []models.QCDefectItem
		if err := s.db.WithContext(ctx).
			Where(&models.QCDefectItem{QCLogID: lg.ID}).
			Order("id asc").
			Find(&defItems).Error; err != nil {
			return nil, err
		}

		for _, di := range defItems {
			switch di.DefectSource {
			case "process":
				if item.DefectReason == "" {
					item.DefectReason = di.DefectReasonText
				}
			case "scrap":
				if item.ScrapReason == "" {
					item.ScrapReason = di.DefectReasonText
				}
			case "issue":
				item.Issues = append(item.Issues, dto.QCRoundIssue{
					Issue:  di.DefectReasonCode,
					Detail: di.DefectReasonText,
					Qty:    di.Qty,
				})
			}
		}

		rounds = append(rounds, item)
	}

	return map[string]interface{}{
		"rounds":             rounds,
		"scan_out_completed": scanOutCompleted,
		"total_production":   totalProduction,
	}, nil
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
			WOType:         r.WOType,
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

		// [wip-source] Untuk process >= 2, sediakan WIP UNIQ dari proses sebelumnya
		// sebagai material input (menggantikan daftar BOM yang fix).
		var wipMaterial *dto.WIPMaterialDTO
		if currentIndex >= 1 {
			prevProcess := flow[currentIndex-1].ProcessName
			if wi, e := s.repoProduction.FindIncomingWIPForItem(ctx, item.WOID, item.ItemUniqCode, currentProcess); e == nil {
				qtyAvail := float64(wi.QtyRemaining)
				if qtyAvail <= 0 {
					qtyAvail = float64(wi.QtyIn)
				}
				wipMaterial = &dto.WIPMaterialDTO{
					Uniq:         item.ItemUniqCode,
					ProcessName:  currentProcess,
					PrevProcess:  prevProcess,
					OpSeq:        wi.OpSeq,
					QtyAvailable: qtyAvail,
					UOM:          item.UOM,
					PartName:     item.PartName,
				}
			}
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

		// [rm-per-uniq] Daftar RM Step 1 memakai Material Spec milik UNIQ
		// sendiri (bukan flatten anak/cucu). buildBomMaterials lama tetap
		// dipakai oleh bomByUniq untuk enrichment Scan Out.
		bomList, e := s.buildOwnBomMaterials(ctx, item.ItemUniqCode)
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
			WIPMaterial:    wipMaterial,
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
		WOType:         wo.WOType,
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
	codeStr := strings.TrimSpace(code)

	derefStr := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	// 1. Try to find as WO Number (for wip type assembly)
	if wo, err := s.repoProduction.FindWOByNumber(ctx, codeStr); err == nil && wo.ID != 0 {
		items, err := s.repoProduction.FindWOItemsByWOID(ctx, wo.ID)
		if err == nil && len(items) > 0 {
			firstItem := items[0]
			return &dto.RawMaterialLookupResponse{
				RMID:              0,
				RMUUID:            "",
				UniqCode:          firstItem.ItemUniqCode,
				PartNumber:        firstItem.PartNumber,
				PartName:          firstItem.PartName,
				RawMaterialType:   "wip",
				TypeLabel:         "WIP",
				UOM:               firstItem.UOM,
				AvailableStock:    firstItem.TotalGoodQty,
				StockWeightKg:     0,
				WarehouseLocation: "",
			}, nil
		}
	}

	searchCode := codeStr
	// 2. Try to find by packing number
	if uniq, err := s.repoProduction.LookupItemUniqByPacking(ctx, codeStr); err == nil && uniq != "" {
		searchCode = uniq
	}

	// 3. Try to find as master RM
	rm, err := s.repoProduction.FindRawMaterialByCode(ctx, searchCode)
	if err == nil {
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

	// 4. If not found in RM, try to find as Finished Goods / Item
	item, errItem := s.repoProduction.FindItemByUniq(ctx, searchCode)
	if errItem == nil {
		return &dto.RawMaterialLookupResponse{
			RMID:              item.ID,
			RMUUID:            "",
			UniqCode:          item.UniqCode,
			PartNumber:        derefStr(item.PartNumber),
			PartName:          derefStr(item.PartName),
			RawMaterialType:   "wip", // Label as WIP so frontend treats it correctly
			TypeLabel:         "WIP",
			UOM:               derefStr(item.UOM),
			AvailableStock:    item.StockQty,
			StockWeightKg:     0,
			WarehouseLocation: "",
		}, nil
	}

	// Return original error if not found anywhere
	return nil, err
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
// [proc-scope] Nomor step proses yang sedang berjalan pada satu WO item.
func currentStepSeqOf(item models.WorkOrderItem) int {
	flow := resolveProcessFlow(item)
	if len(flow) == 0 {
		return 1
	}
	return getCurrentIndex(item.CurrentStepSeq, len(flow)) + 1
}

// [proc-scope] Nama proses yang sedang berjalan pada satu WO item.
func currentProcessNameOf(item models.WorkOrderItem) string {
	flow := resolveProcessFlow(item)
	if len(flow) == 0 {
		return ""
	}
	return flow[getCurrentIndex(item.CurrentStepSeq, len(flow))].ProcessName
}

func (s *service) saveRawMaterialLogs(ctx context.Context, item models.WorkOrderItem, rms []dto.RawMaterialInput, scannedBy string, now time.Time) error {
	// [proc-scope] Hanya log milik step proses ini yang ditimpa.
	stepSeq := currentStepSeqOf(item)
	procName := currentProcessNameOf(item)

	if err := s.repoProduction.DeleteRawMaterialLogsByWOItemStep(ctx, item.ID, stepSeq); err != nil {
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

		packingsJSON := ""
		if len(rm.Packings) > 0 {
			if b, err := json.Marshal(rm.Packings); err == nil {
				packingsJSON = string(b)
			}
		}

		if err := s.repoProduction.InsertRawMaterial(ctx, models.RawMaterialLog{
			UUID:        uuid.New().String(),
			WOID:        item.WOID,
			WOItemID:    item.ID,
			UniqCode:    rm.UniqCode,
			RMUUID:      rm.RMUUID,
			PartNumber:  partNumber,
			PartName:    partName,
			UOM:         uom,
			QtyUsed:     rm.Qty,
			ProcessName: procName,
			StepSeq:     stepSeq,
			ScannedBy:   scannedBy,
			ScannedAt:   now,
			CreatedAt:   now,
			Packings:    packingsJSON,
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
	// [proc-scope] Log RM dipisah per step proses.
	stepSeq := currentStepSeqOf(item)
	procName := currentProcessNameOf(item)

	if err := s.repoProduction.DeleteRawMaterialLogsByWOItemStep(ctx, item.ID, stepSeq); err != nil {
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

		// [qty-tersedia-scanout] simpan alokasi packing pada log scan out juga,
		// agar done-view bisa menghitung Qty Tersedia dari stok berjalan packing
		// (bukan stok master). Tanpa ini, log hasil scan out kehilangan packings.
		packingsJSON := ""
		if len(rm.Packings) > 0 {
			alloc := make([]dto.RawMaterialPackingAllocation, 0, len(rm.Packings))
			for _, p := range rm.Packings {
				if strings.TrimSpace(p.PackingNumber) == "" {
					continue
				}
				q := p.FinalQty
				if p.DeductOnly {
					q = p.DeductQty
				}
				alloc = append(alloc, dto.RawMaterialPackingAllocation{PackingNumber: p.PackingNumber, Qty: q})
			}
			if len(alloc) > 0 {
				if b, err := json.Marshal(alloc); err == nil {
					packingsJSON = string(b)
				}
			}
		}

		// 1) catat pemakaian RM — SEMUA RM dicatat (termasuk qty 0) supaya
		//    tetap tampil di done-view sesuai yang benar-benar discan-out.
		if err := s.repoProduction.InsertRawMaterial(ctx, models.RawMaterialLog{
			UUID:        uuid.New().String(),
			WOID:        item.WOID,
			WOItemID:    item.ID,
			UniqCode:    code,
			RMUUID:      rm.RMUUID,
			PartNumber:  partNumber,
			PartName:    master.PartName,
			UOM:         master.UOM,
			QtyUsed:     rm.QtyUsed,
			ProcessName: procName,
			StepSeq:     stepSeq,
			ScannedBy:   scannedBy,
			ScannedAt:   now,
			CreatedAt:   now,
			Packings:    packingsJSON,
		}); err != nil {
			return err
		}

		// [repacking] Terapkan penyesuaian qty per packing/kanban dari UI Repacking.
		// FinalQty = qty_current packing setelah repack (nilai absolut).
		if len(rm.Packings) > 0 {
			uniqToDeduct := code
			if found && master.UniqCode != "" {
				uniqToDeduct = master.UniqCode
			}
			for _, p := range rm.Packings {
				if strings.TrimSpace(p.PackingNumber) == "" {
					continue
				}
				// [packing-deduct] DeductOnly = packing dipakai apa adanya dari
				// alokasi Step 1 (tanpa Repacking): qty packing dikurangi
				// relatif terhadap qty berjalan saat ini.
				if p.DeductOnly {
					if p.DeductQty <= 0 {
						continue
					}
					if err := s.repoProduction.DeductPackingQty(ctx, uniqToDeduct, p.PackingNumber, p.DeductQty, now); err != nil {
						return err
					}
					continue
				}
				if err := s.repoProduction.ApplyPackingQtyOpname(ctx, uniqToDeduct, p.PackingNumber, p.FinalQty, now); err != nil {
					return err
				}
			}
		}

		// Stok hanya dikurangi kalau qty > 0.
		// [wip-source] Material WIP tidak memotong stok.
		if rm.QtyUsed <= 0 || rm.IsWIP {
			continue
		}

		if found {
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
		} else if code != "" {
			// FG fallback: jika material tidak ditemukan di master RM, asumsikan Finished Goods
			var fg struct {
				ID       int64
				StockQty float64
			}
			err := s.db.WithContext(ctx).Raw("SELECT id, stock_qty FROM finished_goods WHERE uniq_code = ?", code).Scan(&fg).Error
			if err == nil && fg.ID > 0 {
				before := fg.StockQty
				after := before - rm.QtyUsed
				if after < 0 {
					after = 0
				}
				if err := s.db.WithContext(ctx).Exec(`
					UPDATE finished_goods
					SET stock_qty = ?, updated_at = ?
					WHERE id = ?
				`, after, now, fg.ID).Error; err != nil {
					return err
				}
				ref := woNumber
				src := "wo_scan"
				by := scannedBy
				notes := "Used in production scan out"

				// 1. Tulis ke inventory_movement_logs
				if err := s.repoProduction.InsertInventoryMovementLog(ctx, models.InventoryMovementLog{
					MovementCategory: "finished_goods",
					MovementType:     "outgoing",
					UniqCode:         code,
					EntityID:         &fg.ID,
					QtyChange:        -rm.QtyUsed,
					ReferenceID:      &ref,
					SourceFlag:       &src,
					LoggedBy:         &by,
					LoggedAt:         now,
				}); err != nil {
					return err
				}

				// 2. Tulis ke fg_movement_logs
				if err := s.appendFGMovementLog(s.db.WithContext(ctx), fg.ID, code, "outgoing", -rm.QtyUsed, before, after, &ref, nil, &notes, &by); err != nil {
					return err
				}
			}
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
		// [proc-scope] Log RM proses sebelumnya tidak boleh muncul di Step 2
		// proses berikutnya (sebelum "Mulai Produksi" ditekan).
		logs, err := s.repoProduction.FindProductionRawMaterialLogsByWOItemStep(ctx, item.ID, currentStepSeqOf(item))
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
			} else if l.UniqCode != "" {
				if fgItem, err := s.repoProduction.FindItemByUniq(ctx, l.UniqCode); err == nil {
					if fgItem.UOM != nil && *fgItem.UOM != "" {
						uom = *fgItem.UOM
					}
					avail = fgItem.StockQty
				}
			}

			// [wip-source] Material hasil proses sebelumnya memakai UNIQ yang
			// sama dengan item WO ini, tetapi type_label-nya bisa "OTHER".
			isWIP := strings.EqualFold(typeLabel, "WIP") ||
				(currentStepSeqOf(item) > 1 && strings.EqualFold(l.UniqCode, item.ItemUniqCode))
			if isWIP {
				typeLabel = "WIP"
				flow := resolveProcessFlow(item)
				idx := getCurrentIndex(item.CurrentStepSeq, len(flow))
				if idx >= 0 && idx < len(flow) {
					avail = s.getWIPAvailableStock(ctx, item.WOID, l.UniqCode, flow[idx].ProcessName, item.CurrentStepSeq)
				}
			}

			var packingsAlloc []dto.RawMaterialPackingAllocation
			if l.Packings != "" {
				_ = json.Unmarshal([]byte(l.Packings), &packingsAlloc)
			}

			// [qty-tersedia-scanout] Qty Tersedia mengikuti stok BERJALAN packing/
			// kanban (qty_opname) yang sudah berkurang setelah scan out, bukan alokasi
			// Step 1 maupun stok master RM. Nilainya sama dengan "Qty saat ini" pada
			// detail Raw Material di ERP.
			if !isWIP && len(packingsAlloc) > 0 {
				pkNumbers := make([]string, 0, len(packingsAlloc))
				for _, pa := range packingsAlloc {
					pkNumbers = append(pkNumbers, pa.PackingNumber)
				}
				rmUniq := l.UniqCode
				if ok && meta.UniqCode != "" {
					rmUniq = meta.UniqCode
				}
				if live, found, e := s.repoProduction.SumLivePackingQty(ctx, rmUniq, pkNumbers); e == nil && found {
					avail = live
				}
			}

			rms = append(rms, dto.ScanOutContextRawMaterial{
				RMUUID:         l.RMUUID,
				PackingListRM:  packing,
				MaterialCode:   materialCode,
				Form:           form,
				QtyPerUniq:     bm.QtyPerUniq,
				SpecWeightKg:   specWeight,
				TypeLabel:      typeLabel,
				IsWIP:          isWIP,
				UOM:            uom,
				AvailableStock: avail,
				StockWeightKg:  weight,
				QtyUsed:        l.QtyUsed,
				Packings:       packingsAlloc,
			})
		}

		soFlow := resolveProcessFlow(item)
		soTotalStep := len(soFlow)
		soCurrentIndex := getCurrentIndex(item.CurrentStepSeq, soTotalStep)

		// [overflow-qc-so-total] Total Produksi di layar Scan Out = qty saat kejadian
		// SCAN_OUT (fakta scan-out), BUKAN total_good_qty yang bisa berubah setelah QC
		// Round 3 memecah kelebihan ke kanban overflow. Ini menjaga Total Produksi tetap
		// match dengan Qty Aktual Dipakai (RM). Fallback ke TotalGoodQty bila log SCAN_OUT
		// belum ada (mis. proses saat ini belum di-scan-out).
		soTotalOutput := item.TotalGoodQty
		soProcName := ""
		if soCurrentIndex >= 0 && soCurrentIndex < len(soFlow) {
			soProcName = soFlow[soCurrentIndex].ProcessName
		}
		{
			soQ := s.db.WithContext(ctx).
				Model(&models.ProductionScanLog{}).
				Where("wo_item_id = ? AND scan_type = ?", item.ID, "SCAN_OUT")
			if soProcName != "" {
				soQ = soQ.Where("process_name = ?", soProcName)
			}
			var soLog models.ProductionScanLog
			if err := soQ.Order("id DESC").First(&soLog).Error; err == nil {
				soTotalOutput = soLog.QtyOutput
			}
		}

		resp.Items = append(resp.Items, dto.ScanOutContextItem{
			WOItemID:       item.ID,
			Uniq:           item.ItemUniqCode,
			MachineScanned: item.MachineID != 0,
			Status:         item.Status,
			CurrentStep:    soCurrentIndex + 1,
			TotalStep:      soTotalStep,
			ScanInCount:    item.ScanInCount,
			ScanOutCount:   item.ScanOutCount,
			TotalOutput:    soTotalOutput,
			RawMaterials:   rms,
		})
	}

	return resp, nil
}

// [rm-per-uniq] buildOwnBomMaterials menyusun daftar Raw Material yang benar-
// benar milik UNIQ tersebut (Material Spec node itu sendiri), BUKAN hasil
// flatten anak/cucu BOM. Node tanpa Material Spec sendiri (mis. parent
// perakitan murni) menghasilkan daftar KOSONG. Setiap baris ditandai
// IsOwnMaterial=true agar FE dapat memilih baris tanpa bergantung pada level.
func (s *service) buildOwnBomMaterials(ctx context.Context, uniq string) ([]dto.BomMaterial, error) {
	if strings.TrimSpace(uniq) == "" {
		return []dto.BomMaterial{}, nil
	}

	rows, err := s.repoProduction.FindOwnBomMaterialsByUniq(ctx, uniq)
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
	// hasOwnSpec: node punya Material Spec sendiri bila ada salah satu penanda
	// spesifikasi (grade/type/form/dimensi/berat) atau sudah ter-resolve ke
	// master Raw Material. Tanpa ini, node perakitan tanpa spec ikut tampil.
	hasOwnSpec := func(row repository.BomMaterialRow) bool {
		if derefStr(row.MaterialGrade) != "" || derefStr(row.Grade) != "" ||
			derefStr(row.TypeMaterial) != "" || derefStr(row.Form) != "" {
			return true
		}
		if row.WidthMm != nil || row.DiameterMm != nil || row.ThicknessMm != nil ||
			row.LengthMm != nil || row.WeightKg != nil {
			return true
		}
		if row.RMUUID != nil && strings.TrimSpace(*row.RMUUID) != "" {
			return true
		}
		return false
	}

	out := make([]dto.BomMaterial, 0, len(rows))
	for _, row := range rows {
		if !hasOwnSpec(row) {
			continue
		}
		// UOM: pakai UOM master RM bila ada, lalu UOM di bom_lines, lalu UOM item.
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
			RMUniqCode:      derefStr(row.RMUniqCode),
			RawMaterialType: rawType,
			TypeLabel:       typeLabel,
			InInventory:     inInventory,
			AvailableStock:  derefFloat(row.StockQty),
			StockWeightKg:   derefFloat(row.StockWeightKg),
			IsOwnMaterial:   true,
		})
	}
	return out, nil
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
			RMUniqCode:      derefStr(row.RMUniqCode),
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

	// [rm-spec] Log pemakaian RM menyimpan uniq master Raw Material
	// (mis. BR50), sedangkan map di atas hanya berkunci uniq item BOM
	// (mis. M19). Tanpa alias di bawah, Form / Weight / QPU dari
	// spesifikasi BOM tidak ketemu sehingga perhitungan otomatis di
	// scan out kehilangan berat dan menampilkan "Berat: -".
	for _, b := range bom {
		for _, alias := range []string{b.RMUniqCode, b.MaterialGrade} {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if _, exists := m[alias]; exists {
				continue
			}
			m[alias] = b
		}
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

func (s *service) getWIPAvailableStock(ctx context.Context, woID int64, uniqCode string, processName string, seq int) float64 {
	var stock float64
	s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(wi.qty_remaining), 0)
		FROM wip_items wi
		JOIN wips w ON w.id = wi.wip_id
		WHERE w.wo_id = ? AND wi.uniq = ? AND wi.process_name = ? AND wi.seq = ? AND wi.status IN ('queue', 'process')
	`, woID, uniqCode, processName, seq).Scan(&stock)
	return stock
}

func (s *service) buildItemRawMaterials(ctx context.Context, item models.WorkOrderItem) ([]dto.ScanOutContextRawMaterial, error) {
	// [proc-scope] Ambil hanya log RM milik step proses yang sedang berjalan,
	// supaya daftar material proses 2 tidak diisi RM sisa proses 1.
	logs, err := s.repoProduction.FindProductionRawMaterialLogsByWOItemStep(ctx, item.ID, currentStepSeqOf(item))
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
		isWIP := strings.EqualFold(typeLabel, "WIP") || (currentStepSeqOf(item) > 1 && strings.EqualFold(l.UniqCode, item.ItemUniqCode))
		if isWIP {
			typeLabel = "WIP"
			flow := resolveProcessFlow(item)
			idx := getCurrentIndex(item.CurrentStepSeq, len(flow))
			if idx >= 0 && idx < len(flow) {
				avail = s.getWIPAvailableStock(ctx, item.WOID, l.UniqCode, flow[idx].ProcessName, item.CurrentStepSeq)
			}
		}

		var packingsAlloc []dto.RawMaterialPackingAllocation
		if l.Packings != "" {
			_ = json.Unmarshal([]byte(l.Packings), &packingsAlloc)
		}

		// [qty-tersedia-step1] Qty Tersedia di Step 1 (WODetail/Scan In) juga
		// mengikuti stok BERJALAN packing/kanban (qty_opname) untuk packing yang
		// benar-benar discan, BUKAN stok master RM yang menjumlahkan SEMUA
		// packing. Tanpa ini, setelah scan out lalu kembali ke Step 1, Qty
		// Tersedia salah menjadi total seluruh packing (mis. 600) alih-alih
		// packing yang bersangkutan (mis. 100). Mirror perbaikan ScanOutContext.
		if !isWIP && len(packingsAlloc) > 0 {
			pkNumbers := make([]string, 0, len(packingsAlloc))
			for _, pa := range packingsAlloc {
				pkNumbers = append(pkNumbers, pa.PackingNumber)
			}
			rmUniq := l.UniqCode
			if ok && meta.UniqCode != "" {
				rmUniq = meta.UniqCode
			}
			if live, found, e := s.repoProduction.SumLivePackingQty(ctx, rmUniq, pkNumbers); e == nil && found {
				avail = live
			}
		}

		rms = append(rms, dto.ScanOutContextRawMaterial{
			RMUUID:         l.RMUUID,
			PackingListRM:  packing,
			MaterialCode:   materialCode,
			Form:           form,
			QtyPerUniq:     bm.QtyPerUniq,
			SpecWeightKg:   specWeight,
			TypeLabel:      typeLabel,
			IsWIP:          isWIP,
			UOM:            uom,
			AvailableStock: avail,
			StockWeightKg:  weight,
			QtyUsed:        l.QtyUsed,
			Packings:       packingsAlloc,
		})
	}
	return rms, nil
}

// RMPackingList mengembalikan daftar packing/kanban milik satu Raw Material.
// rmUUID diprioritaskan; kalau kosong pakai code (packing_list_rm / material code)
// untuk resolve master RM lalu ambil uniq_code-nya.
func (s *service) RMPackingList(ctx context.Context, rmUUID, code string) (*dto.RMPackingListResponse, error) {
	uniqCode := ""
	if rmUUID != "" {
		if m, err := s.repoProduction.FindRawMaterialByUUID(ctx, rmUUID); err == nil && m.UniqCode != "" {
			uniqCode = m.UniqCode
		}
	}
	if uniqCode == "" && strings.TrimSpace(code) != "" {
		if m, err := s.repoProduction.FindRawMaterialByCode(ctx, strings.TrimSpace(code)); err == nil && m.UniqCode != "" {
			uniqCode = m.UniqCode
		}
	}
	if uniqCode == "" {
		uniqCode = strings.TrimSpace(code)
	}
	if uniqCode == "" {
		return &dto.RMPackingListResponse{Items: []dto.RMPackingItem{}}, nil
	}

	rows, err := s.repoProduction.ListRMPackingList(ctx, uniqCode)
	if err != nil {
		return nil, err
	}

	items := make([]dto.RMPackingItem, 0, len(rows))
	var totalCurrent, totalMax float64
	for _, row := range rows {
		progress := 0
		if row.QtyMax > 0 {
			p := int((row.QtyCurrent / row.QtyMax) * 100)
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			progress = p
		}
		totalCurrent += row.QtyCurrent
		totalMax += row.QtyMax
		items = append(items, dto.RMPackingItem{
			DNNumber:      row.DNNumber,
			PackingNumber: row.PackingNumber,
			Quantity:      row.Quantity,
			QtyCurrent:    row.QtyCurrent,
			QtyMax:        row.QtyMax,
			Progress:      progress,
			Status:        row.Status,
			WONumber:      row.WONumber,
			Source:        row.Source,
		})
	}

	return &dto.RMPackingListResponse{
		UniqCode:        uniqCode,
		Items:           items,
		TotalPacking:    len(items),
		TotalQtyCurrent: totalCurrent,
		TotalQtyMax:     totalMax,
	}, nil
}

// [repack-sisa] resolveRMUniqCode mencari uniq_code master Raw Material dari
// rm_uuid, atau dari code (packing_list_rm / material code).
func (s *service) resolveRMUniqCode(ctx context.Context, rmUUID, code string) string {
	if strings.TrimSpace(rmUUID) != "" {
		if m, err := s.repoProduction.FindRawMaterialByUUID(ctx, strings.TrimSpace(rmUUID)); err == nil && m.UniqCode != "" {
			return m.UniqCode
		}
	}
	if strings.TrimSpace(code) != "" {
		if m, err := s.repoProduction.FindRawMaterialByCode(ctx, strings.TrimSpace(code)); err == nil && m.UniqCode != "" {
			return m.UniqCode
		}
	}
	return strings.TrimSpace(code)
}

// RMRepack memindahkan sisa material dari packing asal ke satu/lebih packing
// tujuan yang masih punya slot.
//
// Contoh: packing 001 sisa 100/500, packing 002 sekarang 400/500.
// Moves = [{002, 100}] -> 002 jadi 500/500, 001 jadi 0/500.
//
// Qty yang melebihi slot packing tujuan ditolak; frontend menawarkan opsi
// scan packing lain atau menyimpan sisanya di packing asal.
func (s *service) RMRepack(ctx context.Context, req dto.RMRepackRequest) (*dto.RMRepackResponse, error) {
	type packSlot struct {
		current float64
		max     float64
	}

	const eps = 0.000001

	source := strings.TrimSpace(req.SourcePackingNumber)
	if source == "" {
		return nil, apperror.BadRequest("source_packing_number wajib diisi")
	}
	if len(req.Moves) == 0 {
		return nil, apperror.BadRequest("moves (packing tujuan) wajib diisi")
	}

	uniqCode := s.resolveRMUniqCode(ctx, req.RMUUID, req.Code)
	if uniqCode == "" {
		return nil, apperror.BadRequest("raw material tidak ditemukan")
	}

	rows, err := s.repoProduction.ListRMPackingList(ctx, uniqCode)
	if err != nil {
		return nil, err
	}

	slots := make(map[string]packSlot, len(rows))
	for _, row := range rows {
		slots[strings.TrimSpace(row.PackingNumber)] = packSlot{current: row.QtyCurrent, max: row.QtyMax}
	}

	src, ok := slots[source]
	if !ok {
		return nil, apperror.BadRequest("packing asal " + source + " tidak ditemukan pada raw material ini")
	}

	// Validasi semua tujuan dulu supaya tidak ada perubahan setengah jalan.
	total := 0.0
	seen := make(map[string]bool, len(req.Moves))
	for _, mv := range req.Moves {
		dest := strings.TrimSpace(mv.PackingNumber)
		if dest == "" || mv.Qty <= 0 {
			return nil, apperror.BadRequest("packing tujuan & qty repack wajib diisi")
		}
		if dest == source {
			return nil, apperror.BadRequest("packing tujuan tidak boleh sama dengan packing asal")
		}
		if seen[dest] {
			return nil, apperror.BadRequest("packing tujuan " + dest + " dikirim lebih dari sekali")
		}
		seen[dest] = true

		row, found := slots[dest]
		if !found {
			return nil, apperror.BadRequest("packing tujuan " + dest + " tidak ditemukan pada raw material ini")
		}
		free := row.max - row.current
		if free < 0 {
			free = 0
		}
		if mv.Qty > free+eps {
			return nil, apperror.BadRequest(fmt.Sprintf(
				"qty %.2f melebihi slot packing %s (tersisa %.2f). Simpan di packing asal atau scan packing lain yang masih available.",
				mv.Qty, dest, free))
		}
		total += mv.Qty
	}
	if total > src.current+eps {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"total qty repack %.2f melebihi sisa packing %s (%.2f)", total, source, src.current))
	}

	now := time.Now()
	for _, mv := range req.Moves {
		if err := s.repoProduction.AddPackingQty(ctx, uniqCode, strings.TrimSpace(mv.PackingNumber), mv.Qty, now); err != nil {
			return nil, err
		}
	}
	if err := s.repoProduction.DeductPackingQty(ctx, uniqCode, source, total, now); err != nil {
		return nil, err
	}

	fresh, err := s.RMPackingList(ctx, req.RMUUID, req.Code)
	if err != nil {
		return nil, err
	}

	return &dto.RMRepackResponse{
		UniqCode: uniqCode,
		Moved:    total,
		Items:    fresh.Items,
	}, nil
}

// ================================
// [scanin-draft-db] Draft Scan-In (seed) bersama lintas gadget
// ================================

func (s *service) ListScanInDrafts(ctx context.Context, woID int64, currentStep int) (*dto.ListScanInDraftsResponse, error) {
	rows, err := s.repoProduction.ListScanInDrafts(ctx, woID, currentStep)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ScanInDraftItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, dto.ScanInDraftItem{
			WOItemID:    r.WOItemID,
			CurrentStep: r.CurrentStep,
			Payload:     json.RawMessage(r.Payload),
			UpdatedBy:   r.UpdatedBy,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	return &dto.ListScanInDraftsResponse{WOID: woID, Items: items}, nil
}

func (s *service) UpsertScanInDraft(ctx context.Context, req dto.UpsertScanInDraftRequest, updatedBy string) error {
	if len(req.Payload) == 0 || !json.Valid(req.Payload) {
		return apperror.BadRequest("payload draft tidak valid")
	}
	step := req.CurrentStep
	if step <= 0 {
		step = 1
	}
	var by *string
	if strings.TrimSpace(updatedBy) != "" {
		v := updatedBy
		by = &v
	}
	return s.repoProduction.UpsertScanInDraft(ctx, models.ProductionScaninDraft{
		WOID:        req.WOID,
		WOItemID:    req.WOItemID,
		CurrentStep: step,
		Payload:     datatypes.JSON(req.Payload),
		UpdatedBy:   by,
	})
}

func (s *service) DeleteScanInDraft(ctx context.Context, req dto.DeleteScanInDraftRequest) error {
	return s.repoProduction.DeleteScanInDraft(ctx, req.WOID, req.WOItemID, req.CurrentStep)
}
