package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/internal/action_ui/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	QCSubmit(ctx context.Context, req dto.QCSubmitRequest) error
}

type service struct {
	repo           repository.IRepository
	repoProduction repository.IProductionRepository
	repoIncoming   repository.IIncomingRepository
}

func New(repo repository.IRepository, repoProduction repository.IProductionRepository, repoIncoming repository.IIncomingRepository) IService {
	return &service{repo: repo, repoProduction: repoProduction, repoIncoming: repoIncoming}
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

	// anti double scan in
	if item.ScanInCount > item.ScanOutCount {
		return errors.New("already scan in, please scan out first")
	}

	// =============================
	// 🏭 MACHINE (optional)
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

	// =============================
	// 📝 INSERT LOG
	// =============================
	log := models.ProductionScanLog{
		UUID:         uuid.New().String(),
		WOID:         item.WOID,
		WOItemID:     item.ID,
		MachineID:    machineID,
		KanbanNumber: item.KanbanNumber,

		// 🔥 FIX INI
		RawMaterialID: nil,

		ProcessName:    currentProcess,
		ProductionLine: productionLine,
		ScanType:       "SCAN_IN",
		QtyInput:       req.Qty,
		Shift:          req.Shift,
		DandoriTime:    req.DandoriTime,
		SetupQCTime:    req.SetupQCTime,
		ScannedBy:      req.ScannedBy,
		ScannedAt:      time.Now(),
		CreatedAt:      time.Now(),
	}

	if err := s.repoProduction.InsertScanLog(ctx, log); err != nil {
		return err
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

	// 🔒 cek sudah ada QC pending belum (anti duplicate)
	exist, err := s.repoProduction.IsQCPendingExist(ctx, item.ID, log.ProcessName)
	if err != nil {
		return err
	}
	if exist {
		return nil // sudah ada, skip
	}

	qc := models.QCTask{
		TaskType: "production_qc",
		Status:   "pending",

		// optional kalau mau link ke WO item
		// IncomingDNItemID: nil,

		Round: 1,

		// 🔥 simpan info tambahan ke JSON
		RoundResults: datatypes.JSON([]byte(fmt.Sprintf(`{
			"kanban_number": "%s",
			"process_name": "%s",
			"wo_id": %d,
			"qty": %f
		}`, item.KanbanNumber, log.ProcessName, item.WOID, log.QtyInput))),

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

	// harus sudah scan in
	if item.ScanInCount <= item.ScanOutCount {
		return errors.New("please scan in first")
	}

	// process harus match
	if item.LastScannedProcess != currentProcess {
		return errors.New("invalid process sequence")
	}

	// =============================
	// 🏭 MACHINE (optional)
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
	// =============================
	// 📝 INSERT LOG
	// =============================
	log := models.ProductionScanLog{
		UUID:           uuid.New().String(),
		WOID:           item.WOID,
		WOItemID:       item.ID,
		MachineID:      machineID,
		KanbanNumber:   item.KanbanNumber,
		ProcessName:    currentProcess,
		ProductionLine: productionLine,
		ScanType:       "SCAN_OUT",
		QtyOutput:      req.QtyOutput,
		QtyRMUsed:      req.QtyRMUsed,
		NGMachine:      req.NGMachine,
		NGProcess:      req.NGProcess,
		QtyScrap:       req.QtyScrap,
		QtyRework:      req.QtyRework,
		Shift:          "1",
		ScannedBy:      req.ScannedBy,
		ScannedAt:      time.Now(),
		CreatedAt:      time.Now(),
	}

	if err := s.repoProduction.InsertScanLog(ctx, log); err != nil {
		return err
	}

	// =============================
	// 📊 UPDATE AGGREGATE
	// =============================
	item.ScanOutCount++
	item.TotalGoodQty += req.QtyOutput
	item.TotalNGQty += req.NGMachine + req.NGProcess
	item.TotalScrapQty += req.QtyScrap

	// =============================
	// 🔥 STEP TRANSITION
	// =============================
	if item.CurrentStepSeq < totalStep {
		item.CurrentStepSeq++
	} else {
		item.Status = "FINISHED"
	}

	return s.repoProduction.UpdateWOItem(ctx, item)
}

func (s *service) QCSubmit(ctx context.Context, req dto.QCSubmitRequest) error {

	item, err := s.repoProduction.FindWOItemByUuid(ctx, req.UUID)
	if err != nil {
		return err
	}

	// ✅ INSERT QC LOG
	err = s.repoProduction.InsertQCLog(ctx, models.QCLog{
		UUID:       uuid.New().String(),
		WOID:       item.WOID,
		WOItemID:   item.ID,
		UniqCode:   req.Uniq,
		QCRound:    req.QCRound,
		QtyChecked: req.QtyChecked,
		QtyPass:    req.QtyPass,
		QtyDefect:  req.QtyDefect,
		QtyScrap:   req.QtyScrap,
		Status:     req.Status,
		CheckedAt:  time.Now(),
		CreatedAt:  time.Now(),
	})
	if err != nil {
		return err
	}

	// 🔥 ONLY ROUND 3 → MASUK INVENTORY
	if req.QCRound == 3 && req.Status == "PASSED" {

		err = s.insertFinishedGoods(ctx, item, req)
		if err != nil {
			return err
		}

		item.Status = "DONE"
		return s.repoProduction.UpdateWOItem(ctx, item)
	}

	return nil
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
