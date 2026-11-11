package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/ganasa18/go-template/internal/safety_stock_parameter/constant"
	"github.com/ganasa18/go-template/internal/safety_stock_parameter/models"
	safetyStockRepo "github.com/ganasa18/go-template/internal/safety_stock_parameter/repository"
	"github.com/ganasa18/go-template/pkg/forecastingclient"
)

type ISafetyStockService interface {
	GetAll(ctx context.Context) ([]models.SafetyStockParameter, error)
	GetByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error)
	Create(ctx context.Context, req models.CreateSafetyStockRequest) (*models.SafetyStockParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateSafetyStockRequest) (*models.SafetyStockParameter, error)
	Delete(ctx context.Context, id int64) error
	BulkCreate(ctx context.Context, req models.BulkCreateSafetyStockRequest) error
	Calculate(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, error)
	// CalculateWithNote sama dengan Calculate, tetapi juga mengembalikan catatan
	// bila nilai tidak bisa dihitung penuh (mis. plan forecasting belum ada).
	CalculateWithNote(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, string, error)
}

// InventoryPlanProvider membaca hasil scheduler harian forecasting
// (GET /inventory-plan). *forecastingclient.Client memenuhi interface ini.
type InventoryPlanProvider interface {
	GetInventoryPlan(ctx context.Context, opts forecastingclient.GetInventoryPlanOptions) ([]forecastingclient.InventoryPlanItem, error)
}

// implementation
type service struct {
	repo  safetyStockRepo.ISafetyStockRepository
	plans InventoryPlanProvider // opsional; sumber nilai untuk calculation_type "forecast"
}

func New(repo safetyStockRepo.ISafetyStockRepository) ISafetyStockService {
	return &service{repo: repo}
}

// NewWithForecast: parameter bertipe "forecast" memakai hasil scheduler forecasting.
func NewWithForecast(repo safetyStockRepo.ISafetyStockRepository, plans InventoryPlanProvider) ISafetyStockService {
	return &service{repo: repo, plans: plans}
}

// =========================
// CRUD
// =========================

func (s *service) GetAll(ctx context.Context) ([]models.SafetyStockParameter, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.SafetyStockParameter, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateSafetyStockRequest) (*models.SafetyStockParameter, error) {
	req.CalculationType = normalizeCalculationType(req.CalculationType)

	_, err := s.repo.FindByItemCode(ctx, req.ItemUniqCode)
	if err == nil {
		return nil, errors.New("item sudah ada")
	}

	// 🔥 VALIDATION DI SINI
	if err := validateCalculationType(req.CalculationType); err != nil {
		return nil, err
	}

	status := "Active"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.TrimSpace(*req.Status)
	}

	data := models.SafetyStockParameter{
		InventoryType:   req.InventoryType,
		ItemUniqCode:    req.ItemUniqCode,
		CalculationType: req.CalculationType,
		Constanta:       req.Constanta,
		Status:          status,
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateSafetyStockRequest) (*models.SafetyStockParameter, error) {
	req.CalculationType = normalizeCalculationType(req.CalculationType)

	status := "Active"
	if req.Status != nil {
		status = *req.Status
	}

	err := s.repo.Update(ctx, id, map[string]interface{}{
		"calculation_type": req.CalculationType,
		"constanta":        req.Constanta,
		"status":           status,
	})
	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) BulkCreate(ctx context.Context, req models.BulkCreateSafetyStockRequest) error {
	var data []models.SafetyStockParameter

	for _, item := range req.Items {
		data = append(data, models.SafetyStockParameter{
			InventoryType:   item.InventoryType,
			ItemUniqCode:    item.ItemUniqCode,
			CalculationType: normalizeCalculationType(item.CalculationType),
			Constanta:       item.Constanta,
			Status:          "Active",
		})
	}

	return s.repo.BulkCreate(ctx, data)
}

func (s *service) Calculate(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, error) {
	result, _, err := s.CalculateWithNote(ctx, itemCode, prl, po, workingDays)
	return result, err
}

func (s *service) CalculateWithNote(ctx context.Context, itemCode string, prl, po, workingDays float64) (float64, string, error) {

	param, err := s.repo.FindByItemCode(ctx, itemCode)
	if err != nil {
		return 0, "", errors.New("parameter tidak ditemukan")
	}

	var (
		forecast float64
		note     string
	)

	if param.CalculationType == string(constant.CalcForecast) {
		// Nilai berasal dari job harian forecasting (safety stock = forecast per
		// hari x konstanta hari), bukan dari tabel lokal.
		forecast, note = s.forecastSafetyStock(ctx, param.InventoryType, itemCode)
	}

	result := calculateSafetyStock(
		param.CalculationType,
		prl,
		po,
		workingDays,
		param.Constanta,
		forecast,
	)

	// calculateSafetyStock mengembalikan 0 bila workingDays == 0, termasuk untuk
	// forecast; untuk forecast nilai plan tidak bergantung working days.
	if param.CalculationType == string(constant.CalcForecast) {
		result = forecast
	}

	return result, note, nil
}

// forecastSafetyStock mengambil safety_stock_qty dari plan forecasting terbaru
// untuk uniq tsb. Bila tidak tersedia -> (0, catatan alasan).
func (s *service) forecastSafetyStock(ctx context.Context, inventoryType, itemCode string) (float64, string) {
	if s.plans == nil {
		return 0, "forecasting client belum dikonfigurasi"
	}
	items, err := s.plans.GetInventoryPlan(ctx, forecastingclient.GetInventoryPlanOptions{
		UniqCodes: []string{itemCode},
	})
	if err != nil {
		log.Printf("[safety-stock] inventory plan %s: %v", itemCode, err)
		return 0, "API forecasting tidak dapat dihubungi"
	}

	wantType := normalizeInventoryType(inventoryType)
	var picked *forecastingclient.InventoryPlanItem
	for i := range items {
		it := &items[i]
		if !strings.EqualFold(strings.TrimSpace(it.UniqCode), strings.TrimSpace(itemCode)) {
			continue
		}
		if picked == nil {
			picked = it
		}
		if normalizeInventoryType(it.InventoryType) == wantType {
			picked = it
			break
		}
	}
	if picked == nil {
		return 0, "belum ada hasil scheduler forecasting untuk uniq ini"
	}
	if picked.Note != nil && strings.TrimSpace(*picked.Note) != "" {
		return 0, strings.TrimSpace(*picked.Note)
	}
	if picked.SafetyStockQty == nil {
		return 0, "hasil scheduler forecasting belum memiliki safety_stock_qty"
	}
	return *picked.SafetyStockQty, ""
}

// normalizeInventoryType menyamakan "raw-materials" dengan "raw_material" dst.
func normalizeInventoryType(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	v = strings.NewReplacer("-", "_", " ", "_").Replace(v)
	switch {
	case strings.HasPrefix(v, "raw_material"):
		return "raw_material"
	case strings.HasPrefix(v, "indirect"):
		return "indirect_material"
	case strings.HasPrefix(v, "finished_good"):
		return "finished_goods"
	case strings.HasPrefix(v, "subcon"), strings.HasPrefix(v, "sub_con"):
		return "subcon"
	}
	return v
}

func calculateSafetyStock(calcType string, prl, po, workingDays, constanta, forecast float64) float64 {
	if workingDays == 0 {
		return 0
	}

	daily := (prl + po) / workingDays

	switch calcType {
	case string(constant.CalcDays):
		return daily * constanta

	case string(constant.CalcPercentage):
		return daily * (constanta / 100)

	case string(constant.CalcForecast):
		return forecast
	}

	return 0
}
