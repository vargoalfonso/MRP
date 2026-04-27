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

	// 🔹 Start production (scan in)
	ScanIn(ctx context.Context, req dto.ScanInRequest) error

	// 🔹 Finish process (scan out)
	ScanOut(ctx context.Context, req dto.ScanOutRequest) error

	// 🔹 QC submit (round 1,2,3)
	QCSubmit(ctx context.Context, req dto.QCSubmitRequest, performedBy string) error

	ListQCTask(ctx context.Context, req dto.ListQCTaskRequest) (map[string]interface{}, error)

	IssueList(ctx context.Context) (map[string]interface{}, error)
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
		RawMaterials:   rawMaterials,
	}, nil
}

func (s *service) ScanIn(ctx context.Context, req dto.ScanInRequest) error {

	item, err := s.repoProduction.FindWOItemByUniqAndWO(ctx, req.Uniq, req.WOID)
	if err != nil {
		return errors.New("uniq not found")
	}

	// =====================================
	// PROCESS FLOW
	// =====================================
	flow, err := parseProcessFlow(item.ProcessFlowJSON)
	if err != nil {
		return err
	}

	if err := validateStep(item.CurrentStepSeq, flow); err != nil {
		return err
	}

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
		return errors.New("already scan in, please scan out first")
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

	return s.repoProduction.UpdateWOItem(ctx, item)
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

	item, err := s.repoProduction.FindWOItemByUniqAndWO(ctx, req.Uniq, req.WOID)
	if err != nil {
		return errors.New("uniq not found")
	}

	// =====================================
	// PROCESS FLOW
	// =====================================
	flow, err := parseProcessFlow(item.ProcessFlowJSON)
	if err != nil {
		return err
	}

	if err := validateStep(item.CurrentStepSeq, flow); err != nil {
		return err
	}

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

		QtyOutput: req.QtyOutput,

		QtyRMUsed: 0,
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
				return apperror.NotFound("pending qc task not found")
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

	flow, err := parseProcessFlow(item.ProcessFlowJSON)
	if err != nil {
		return err
	}

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

	err = tx.Where("uniq_code = ?", item.ItemUniqCode).
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

	for _, row := range rows {

		var payload struct {
			Uniq         string  `json:"uniq"`
			KanbanNumber string  `json:"kanban_number"`
			ProcessName  string  `json:"process_name"`
			Qty          float64 `json:"qty"`
		}

		_ = json.Unmarshal(row.RoundResults, &payload)

		items = append(items, dto.QCTaskListItem{
			ID:           row.ID,
			TaskType:     row.TaskType,
			Status:       row.Status,
			Round:        row.Round,
			WOID:         row.WOID,
			WOItemID:     row.WOItemID,
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

func (s *service) IssueList(ctx context.Context) (map[string]interface{}, error) {

	type IssueResult struct {
		IssueType string `json:"issue_type"`
	}

	var results []IssueResult

	err := s.db.WithContext(ctx).
		Table("production_issues").
		Select("issue_type").
		Group("issue_type").
		Order("issue_type ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"items": results,
	}, nil
}
