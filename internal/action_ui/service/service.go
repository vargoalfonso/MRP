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
	wo, err := s.repoProduction.FindWOByNumber(ctx, woNumber)
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

	// =============================
	// 🔄 PROCESS FLOW
	// =============================
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

	// =============================
	// 🚫 VALIDASI
	// =============================
	if item.Status == "FINISHED" || item.Status == "DONE" {
		return errors.New("item already finished")
	}

	if item.ScanInCount > item.ScanOutCount {
		return errors.New("already scan in, please scan out first")
	}

	// =============================
	// 🏭 MACHINE
	// =============================
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

	// =============================
	// 📝 INSERT SCAN LOG
	// =============================
	log := models.ProductionScanLog{
		UUID:           uuid.New().String(),
		WOID:           item.WOID,
		WOItemID:       item.ID,
		MachineID:      machineID,
		KanbanNumber:   item.KanbanNumber,
		RawMaterialID:  nil,
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

	// =============================
	// 🚨 INSERT PRODUCT ISSUE
	// =============================
	if req.ProductIssue {

		issue := models.ProductionIssue{
			UUID:           uuid.New().String(),
			WOID:           item.WOID,
			WOItemID:       item.ID,
			MachineID:      machineID,
			ProcessName:    currentProcess,
			ProductionLine: productionLine,
			IssueType:      req.ProductIssueType,
			IssueDuration:  int64(req.ProductIssueDuration), // menit
			QtyAffected:    float64(req.Qty),
			ReportedBy:     req.ScannedBy,
			ReportedAt:     now,
			CreatedAt:      now,
		}

		if err := s.repoProduction.InsertProductIssue(ctx, issue); err != nil {
			return err
		}
	}

	// =============================
	// 🔄 UPDATE ITEM
	// =============================
	item.Status = "IN_PROGRESS"
	item.ScanInCount++
	item.LastScannedProcess = currentProcess

	err = s.createQCTaskIfNeeded(ctx, item, log, req)
	if err != nil {
		return err
	}

	return s.repoProduction.UpdateWOItem(ctx, item)
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
	// VALIDASI QC ROUND 3 HARUS PASSED
	// =====================================
	var qcCount int64

	err = s.db.WithContext(ctx).
		Model(&models.QCLog{}).
		Where(`
			wo_item_id = ?
			AND process_name = ?
			AND qc_round = ?
			AND UPPER(status) = ?
		`, item.ID, currentProcess, 3, "PASSED").
		Count(&qcCount).Error

	if err != nil {
		return err
	}

	if qcCount == 0 {
		return errors.New("scan out blocked: QC round 3 not passed")
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

	if currentIndex < totalStep-1 {
		item.CurrentStepSeq = flow[currentIndex+1].OpSeq
		item.Status = "PENDING"
	} else {
		item.Status = "FINISHED"
	}

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
		// AMBIL PAYLOAD TASK JSON
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
			payload.ProcessName = item.ProcessName
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
		// UPDATE TASK
		// =====================================
		if strings.EqualFold(req.Status, "PASSED") {

			// ROUND 1 -> 2 / ROUND 2 -> 3
			if req.QCRound < 3 {
				if err := tx.Model(&task).Updates(map[string]interface{}{
					"round":      req.QCRound + 1,
					"updated_at": now,
				}).Error; err != nil {
					return err
				}
			} else {
				// ROUND 3 DONE
				if err := tx.Model(&task).Updates(map[string]interface{}{
					"status":     "done",
					"updated_at": now,
				}).Error; err != nil {
					return err
				}

				// unlock scan out
				if err := tx.Model(&item).Update("status", "QC_PASSED").Error; err != nil {
					return err
				}
			}

		} else {
			// gagal = tetap round yg sama
			if err := tx.Model(&task).Update("updated_at", now).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func mapQCSourceToScrapType(defectSource string) string {
	if defectSource == "setting_machine" {
		return scrapModels.ScrapTypeSettingMachine
	}
	return scrapModels.ScrapTypeProcess
}

func stringPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func (s *service) insertFinishedGoods(ctx context.Context, item models.WorkOrderItem, qc dto.QCSubmitRequest) error {

	fg := models.FinishedGoods{
		UUID:       uuid.New().String(),
		UniqCode:   item.ItemUniqCode,
		ItemID:     item.ID,
		PartNumber: item.PartNumber,
		PartName:   item.PartName,
		Model:      item.Model,
		WONumber:   "",
		StockQty:   qc.QtyPass,
		UOM:        item.UOM,
		Status:     "AVAILABLE",
		CreatedAt:  time.Now(),
	}

	return s.repoProduction.InsertFinishedGoods(ctx, fg)
}

func (s *service) isLastStep(item models.WorkOrderItem) bool {
	var flow []models.ProcessFlow

	err := json.Unmarshal([]byte(item.ProcessFlowJSON), &flow)
	if err != nil || len(flow) == 0 {
		return false
	}

	return item.CurrentStepSeq >= len(flow)-1
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

// func (s *service) ScanIncoming(ctx context.Context, req dto.IncomingScanRequest) error {

// 	// =============================
// 	// 🔒 IDEMPOTENCY CHECK
// 	// =============================
// 	exist, err := s.repoIncoming.IsIdempotentExist(ctx, req.IdempotencyKey)
// 	if err != nil {
// 		return err
// 	}
// 	if exist {
// 		return nil // already processed
// 	}

// 	// =============================
// 	// 🔍 VALIDATE DN ITEM
// 	// =============================
// 	_, err = s.repoIncoming.GetDNItem(ctx, req.DNItemID)
// 	if err != nil {
// 		return errors.New("DN item not found")
// 	}

// 	// =============================
// 	// 📝 INSERT SCAN
// 	// =============================
// 	scan := models.IncomingReceivingScan{
// 		IncomingDNItemID:  req.DNItemID,
// 		IdempotencyKey:    req.IdempotencyKey,
// 		ScanRef:           req.ScanRef,
// 		Qty:               req.Qty,
// 		WeightKg:          req.WeightKg,
// 		ScannedAt:         time.Now(),
// 		ScannedBy:         req.ScannedBy,
// 		WarehouseLocation: req.Warehouse,
// 		Status:            "pending",
// 	}

// 	if err := s.repoIncoming.InsertScan(ctx, &scan); err != nil {
// 		return err
// 	}

// 	// =============================
// 	// 🧪 CREATE QC TASK
// 	// =============================
// 	qc := models.QCTask{
// 		TaskType:         "incoming_qc",
// 		Status:           "pending",
// 		IncomingDNItemID: &req.DNItemID,
// 		CreatedAt:        time.Now(),
// 		UpdatedAt:        time.Now(),
// 	}

// 	if err := s.repoIncoming.CreateQCTask(ctx, &qc); err != nil {
// 		return err
// 	}

// 	// =============================
// 	// 🔗 LINK QC → SCAN
// 	// =============================
// 	if err := s.repoIncoming.AttachQCToScan(ctx, scan.ID, qc.ID); err != nil {
// 		return err
// 	}

// 	return nil
// }
