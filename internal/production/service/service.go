package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/tuotoo/qrcode"

	"github.com/ganasa18/go-template/internal/production/models"
	productionRepo "github.com/ganasa18/go-template/internal/production/repository"
)

type IProductionService interface {
	GetScanData(ctx context.Context, qr string) (*models.ProductionScanResponse, error)

	ScanIn(ctx context.Context, req models.ProductionScanLog) error
	ScanOut(ctx context.Context, req models.ProductionScanLog) error
}

type productionService struct {
	repo productionRepo.IProductionRepository
}

func New(repo productionRepo.IProductionRepository) IProductionService {
	return &productionService{repo}
}

func (s *productionService) GetScanData(ctx context.Context, qr string) (*models.ProductionScanResponse, error) {

	if qr == "" {
		return nil, errors.New("QR tidak boleh kosong")
	}

	var payload models.QRPayload

	// =====================================================
	// 🔥 CASE 1: JSON STRING
	// contoh:
	// "{\"t\":\"wo_item\",\"kb\":\"KBN-...\"}"
	// =====================================================
	if strings.HasPrefix(strings.TrimSpace(qr), "{") {

		if err := json.Unmarshal([]byte(qr), &payload); err != nil {
			return nil, errors.New("format QR JSON tidak valid")
		}

		return s.handlePayload(ctx, payload)
	}

	// =====================================================
	// 🔥 CASE 2: BASE64 IMAGE
	// =====================================================
	if strings.Contains(qr, "base64,") {

		parts := strings.Split(qr, "base64,")
		if len(parts) != 2 {
			return nil, errors.New("format base64 tidak valid")
		}
		qr = parts[1]
	}

	// decode base64
	imgBytes, err := base64.StdEncoding.DecodeString(qr)
	if err != nil {
		return nil, errors.New("gagal decode base64")
	}

	// baca QR dari image
	qrCode, err := qrcode.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, errors.New("gagal membaca QR")
	}

	content := qrCode.Content
	// contoh content:
	// {"t":"wo_item","kb":"KBN-..."}

	// parse JSON dari QR
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, errors.New("QR bukan format JSON valid")
	}

	return s.handlePayload(ctx, payload)
}

func (s *productionService) handlePayload(ctx context.Context, payload models.QRPayload) (*models.ProductionScanResponse, error) {

	if payload.T != "wo_item" {
		return nil, errors.New("QR bukan WO")
	}

	if payload.KB == "" {
		return nil, errors.New("kanban kosong")
	}

	data, err := s.repo.GetWOItemDetail(ctx, payload.KB)
	if err != nil {
		return nil, errors.New("data WO tidak ditemukan")
	}

	if data == nil {
		return nil, errors.New("kanban tidak ditemukan")
	}

	return &models.ProductionScanResponse{
		WOID:           data.WOID,
		WONumber:       data.WONumber,
		ProcessName:    data.ProcessName,
		ProductionLine: data.ProductionLine,
		MachineNumber:  data.MachineNumber,
		PackingNumber:  data.PackingNumber,
		KanbanNumber:   data.KanbanNumber,
		ProductName:    data.ProductName,
		QtyPlan:        data.QtyPlan,
		Unit:           data.Unit,
	}, nil
}

func (s *productionService) ScanIn(ctx context.Context, req models.ProductionScanLog) error {

	if req.WOID == 0 {
		return errors.New("WO wajib ada")
	}

	if req.KanbanNumber == "" {
		return errors.New("kanban wajib ada")
	}

	if req.ProcessName == "" {
		return errors.New("process wajib ada")
	}

	// 🔥 CEK SUDAH PERNAH SCAN?
	last, err := s.repo.GetLastByKanban(ctx, req.KanbanNumber)
	if err == nil && last != nil {
		if last.ScanType == models.ScanTypeIn {
			return errors.New("kanban sudah scan IN")
		}
	}

	req.ScanType = models.ScanTypeIn

	// default 1 uniq
	if req.QtyInput == 0 {
		req.QtyInput = 1
	}

	req.ScannedAt = time.Now()

	return s.repo.Create(ctx, &req)
}

func (s *productionService) ScanOut(ctx context.Context, req models.ProductionScanLog) error {

	if req.WOID == 0 {
		return errors.New("WO wajib ada")
	}

	if req.KanbanNumber == "" {
		return errors.New("kanban wajib ada")
	}

	if req.QtyOutput <= 0 {
		return errors.New("output harus lebih dari 0")
	}

	// 🔥 VALIDASI HARUS SUDAH SCAN IN
	last, err := s.repo.GetLastByKanban(ctx, req.KanbanNumber)
	if err != nil {
		return errors.New("belum pernah scan IN")
	}

	if last.ScanType != models.ScanTypeIn {
		return errors.New("flow tidak valid, harus scan IN dulu")
	}

	// 🔥 VALIDASI TOTAL
	totalNG := req.NGMachine + req.NGProcess
	total := req.QtyOutput + totalNG + req.QtyScrap

	if total <= 0 {
		return errors.New("total produksi tidak valid")
	}

	// optional: validasi against plan
	if req.QtyOutput > last.QtyInput {
		return errors.New("output melebihi input")
	}

	req.ScanType = models.ScanTypeOut
	req.ScannedAt = time.Now()

	return s.repo.Create(ctx, &req)
}
