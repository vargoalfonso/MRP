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
	"github.com/ganasa18/go-template/pkg/apperror"
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

	// 🔹 Finish process (scan out)
	ScanOut(ctx context.Context, req dto.ScanOutRequest) error

	// 🔹 QC submit (round 1,2,3)
	QCSubmit(ctx context.Context, req dto.QCSubmitRequest, performedBy string) error

	ListQCTask(ctx context.Context, req dto.ListQCTaskRequest) (map[string]interface{}, error)

	IssueList(ctx context.Context) (map[string]interface{}, error)
	WOList(ctx context.Context, search string) ([]dto.WOListItem, error)
	WODetail(ctx context.Context, woNumber string) (*dto.WODetailResponse, error)
	RawMaterialLookup(ctx context.Context, code string) (*dto.RawMaterialLookupResponse, error)

	ScanMachine(ctx context.Context, req dto.ScanMachineRequest) (*dto.ScanMachineResponse, error)

	ScanOutContext(ctx context.Context, woNumber string) (*dto.ScanOutContextResponse, error)
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
	if item.ScanInCount <= item.ScanOutCount {
		return errors.New("please scan in first")
	}

	if item.LastScannedProcess != currentProcess {
		return errors.New("invalid process sequence")
	}

	// =====================================
	// VALIDASI QC ROUND 2 HARUS PASSED
	// =====================================
	var qcCount int64

	err = s.db.WithContext(ctx).
		Model(&models.QCLog{}).
		Where(`
			wo_item_id = ?
			AND process_name = ?
			AND qc_round = ?
			AND UPPER(status) = ?
		`, item.ID, currentProcess, 2, "APPROVE").
		Count(&qcCount).Error

	if err != nil {
		return err
	}

	if qcCount == 0 {
		return errors.New("scan out blocked: QC round 2 not approved")
	}

	// =====================================
	// CEK SUDAH PERNAH SCAN OUT BELUM
	// =====================================
	if item.Status == "WAITING_FINAL_QC" {
		return errors.New("already scan out, waiting final qc")
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
	item.TotalGoodQty += req.QtyOutput
	item.LastScannedProcess = currentProcess

	// BELUM PINDAH PROCESS
	// MASIH MENUNGGU QC FINAL (ROUND 3)
	item.Status = "WAITING_FINAL_QC"

	return s.repoProduction.UpdateWOItem(ctx, item)
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
			if item.Status != "WAITING_FINAL_QC" {
				return apperror.BadRequest("final qc requires scan out first")
			}
			// task done
			if err := tx.Model(&task).Updates(map[string]interface{}{
				"status":         "done",
				"good_quantity":  int(req.QtyPass),
				"ng_quantity":    int(req.QtyDefect),
				"scrap_quantity": int(req.QtyScrap),
				"updated_at":     now,
			}).Error; err != nil {
				return err
			}

			// pindah process berikutnya / finish
			if err := s.afterFinalQC(ctx, tx, &item, req, performedBy); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) afterFinalQC(ctx context.Context, tx *gorm.DB, item *models.WorkOrderItem, req dto.QCSubmitRequest, performedBy string) error {

	flow := resolveProcessFlow(*item)

	var wo models.WorkOrder
	if err := tx.Where("id = ?", req.WOID).
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

		currentWIP.QtyOut += int(req.QtyPass)
		currentWIP.QtyRemaining =
			currentWIP.QtyIn - currentWIP.QtyOut

		currentWIP.Status = "done"
		currentWIP.UpdatedAt = now

		if err := tx.Save(&currentWIP).Error; err != nil {
			return err
		}

		_ = tx.Create(&models.WIPLog{
			WipItemID: currentWIP.ID,
			Action:    "QC_FINAL_DONE",
			Qty:       int(req.QtyPass),
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

			Stock: int(req.QtyPass),

			QtyIn:        int(req.QtyPass),
			QtyOut:       0,
			QtyRemaining: int(req.QtyPass),

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
			Qty:       int(req.QtyPass),
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
			StockQty:   req.QtyPass,
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
			req.QtyPass,
			req.QtyPass,
			&wo.WONumber,
			stringPtr("QC_FINAL"),
			stringPtr("Create FG from final production process"),
			stringPtr(performedBy),
		); err != nil {
			return err
		}

	} else if err == nil {

		beforeQty := fg.StockQty
		afterQty := beforeQty + req.QtyPass

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
			req.QtyPass,
			afterQty,
			&wo.WONumber,
			stringPtr("QC_FINAL"),
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

func stringPtr(v string) *string {
	return &v
}

func (s *service) createInventoryMovementLog(tx *gorm.DB, category string, movementType string, uniqCode string, entityID *int64, beforeQty float64, changeQty float64, afterQty float64, refNo *string, source *string, remarks *string, user *string) error {

	row := models.InventoryMovementLog{
		MovementCategory: category,
		MovementType:     movementType,
		UniqCode:         uniqCode,
		EntityID:         entityID,

		QtyBefore: beforeQty,
		QtyChange: changeQty,
		QtyAfter:  afterQty,

		ReferenceNo: refNo,
		SourceFlag:  source,
		Remarks:     remarks,

		LoggedBy:  user,
		LoggedAt:  time.Now(),
		CreatedAt: time.Now(),
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

func buildProcessSteps(flow []models.ProcessFlow, currentSeq int) []dto.WODetailProcessStep {
	total := len(flow)
	currentIndex := getCurrentIndex(currentSeq, total)
	steps := make([]dto.WODetailProcessStep, 0, total)
	for i, f := range flow {
		status := "pending"
		if i < currentIndex {
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

	for i, item := range items {
		totalQty += item.Quantity

		// proses uniq berikutnya (untuk "Berikutnya")
		nextProcess := ""
		if i+1 < len(items) {
			nextProcess = items[i+1].ProcessName
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
			ProcessName:    item.ProcessName, // <-- KUNCI: proses uniq ini
			NextProcess:    nextProcess,      // <-- proses uniq berikutnya
			CurrentStep:    i + 1,            // <-- posisi uniq (1/3, 2/3, ...)
			TotalStep:      len(items),       // <-- total uniq
			ScanInCount:    item.ScanInCount,
			ScanOutCount:   item.ScanOutCount,
			MachineScanned: item.MachineID != 0,
			ProcessFlow:    buildWOProcessSteps(items, i),
			SavedQty:       savedQty,
			DandoriTime:    dandori,
			SetupQCTime:    setupQC,
			RawMaterials:   rmList,
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
	for _, rm := range rms {
		if rm.QtyUsed <= 0 {
			continue
		}

		var master models.RawMaterial
		var found bool
		if rm.RMUUID != "" {
			if m, err := s.repoProduction.FindRawMaterialByUUID(ctx, rm.RMUUID); err == nil {
				master, found = m, true
			}
		}
		if !found && rm.UniqCode != "" {
			if m, err := s.repoProduction.FindRawMaterialByCode(ctx, rm.UniqCode); err == nil {
				master, found = m, true
			}
		}

		// 1) catat pemakaian RM
		if err := s.repoProduction.InsertRawMaterial(ctx, models.RawMaterialLog{
			UUID:       uuid.New().String(),
			WOID:       item.WOID,
			WOItemID:   item.ID,
			UniqCode:   rm.UniqCode,
			RMUUID:     master.UUID,
			PartNumber: master.PartNumber,
			PartName:   master.PartName,
			UOM:        master.UOM,
			QtyUsed:    rm.QtyUsed,
			ScannedBy:  scannedBy,
			ScannedAt:  now,
			CreatedAt:  now,
		}); err != nil {
			return err
		}

		if !found {
			continue // RM tidak dikenal → cukup dicatat, stok tak diubah
		}

		// 2) kurangi stok RM
		before, after, err := s.repoProduction.DecreaseRawMaterialStock(ctx, master.ID, rm.QtyUsed)
		if err != nil {
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
			QtyBefore:        before,
			QtyChange:        -rm.QtyUsed,
			QtyAfter:         after,
			ReferenceNo:      &ref,
			SourceFlag:       &src,
			LoggedBy:         &by,
			LoggedAt:         now,
			CreatedAt:        now,
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
	expectedProcess := item.ProcessName
	totalStep := len(items)
	nextProcess := ""
	if currentIdx+1 < totalStep {
		nextProcess = items[currentIdx+1].ProcessName
	}
	steps := buildWOProcessSteps(items, currentIdx)

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

	// VALIDASI: proses mesin harus == proses uniq ini
	if !strings.EqualFold(strings.TrimSpace(scannedProcess), strings.TrimSpace(expectedProcess)) {
		return &dto.ScanMachineResponse{
			Valid: false,
			Message: fmt.Sprintf(
				"Mesin %s untuk proses \"%s\", padahal uniq ini butuh proses \"%s\". Silakan scan ulang mesin yang benar.",
				machine.MachineNumber, fallbackDash(scannedProcess), expectedProcess,
			),
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

	// COCOK → set mesin ke uniq
	if err := s.repoProduction.UpdateWOItemMachine(ctx, item.ID, int(machine.ID), expectedProcess); err != nil {
		return nil, err
	}

	return &dto.ScanMachineResponse{
		Valid:           true,
		Message:         fmt.Sprintf("Mesin %s sesuai untuk proses \"%s\".", machine.MachineNumber, expectedProcess),
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
				TypeLabel:      typeLabel,
				IsWIP:          strings.EqualFold(typeLabel, "WIP"),
				UOM:            uom,
				AvailableStock: avail,
				StockWeightKg:  weight,
				QtyUsed:        l.QtyUsed,
			})
		}

		resp.Items = append(resp.Items, dto.ScanOutContextItem{
			WOItemID:       item.ID,
			Uniq:           item.ItemUniqCode,
			MachineScanned: item.MachineID != 0,
			ScanInCount:    item.ScanInCount,
			ScanOutCount:   item.ScanOutCount,
			RawMaterials:   rms,
		})
	}

	return resp, nil
}

func (s *service) buildItemRawMaterials(ctx context.Context, item models.WorkOrderItem) ([]dto.ScanOutContextRawMaterial, error) {
	logs, err := s.repoProduction.FindProductionRawMaterialLogsByWOItemID(ctx, item.ID)
	if err != nil {
		return nil, err
	}

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