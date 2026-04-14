package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ganasa18/go-template/internal/po_split_setting/models"
	poSplitSettingRepo "github.com/ganasa18/go-template/internal/po_split_setting/repository"
)

type IPOSplitSettingService interface {
	GetAll(ctx context.Context) ([]models.POSplitSetting, error)
	GetByID(ctx context.Context, id int64) (*models.POSplitSetting, error)
	Create(ctx context.Context, req models.CreatePOSplitRequest) (*models.POSplitSetting, error)
	Update(ctx context.Context, id int64, req models.UpdatePOSplitRequest) (*models.POSplitSetting, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo poSplitSettingRepo.IPOSplitSettingRepository
}

func New(repo poSplitSettingRepo.IPOSplitSettingRepository) IPOSplitSettingService {
	return &service{repo: repo}
}

// =========================
// GET
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.POSplitSetting, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.POSplitSetting, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreatePOSplitRequest) (*models.POSplitSetting, error) {

	if req.SplitRule == nil {
		return nil, errors.New("split_rule is required")
	}

	rule := normalizeSplitRule(*req.SplitRule)

	data := models.POSplitSetting{
		SplitRule: rule,
		Status:    "active",
	}

	if req.Status != nil {
		data.Status = normalizeStatus(*req.Status)
	}

	if req.BudgetType != nil {
		data.BudgetType = *req.BudgetType
	}

	if req.Description != nil {
		data.Description = *req.Description
	}

	if req.MinOrderQty != nil {
		data.MinOrderQty = *req.MinOrderQty
	}

	if req.MaxSplitLines != nil {
		data.MaxSplitLines = *req.MaxSplitLines
	}

	// =========================
	// 🔥 HANDLE MODE
	// =========================

	if rule == "percentage" {

		if req.Po1Pct == nil || req.Po2Pct == nil {
			return nil, errors.New("po1_pct and po2_pct required")
		}

		if err := validatePercentage(*req.Po1Pct, *req.Po2Pct); err != nil {
			return nil, err
		}

		data.Po1Pct = *req.Po1Pct
		data.Po2Pct = *req.Po2Pct
		data.PoSplitSUM = data.Po1Pct + data.Po2Pct

	} else {
		// 🔥 HISTORY → FORCE AMAN (NO ERROR)
		data.Po1Pct = 50
		data.Po2Pct = 50
		data.PoSplitSUM = 100
	}

	if err := s.repo.Create(ctx, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdatePOSplitRequest) (*models.POSplitSetting, error) {

	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updateData := map[string]interface{}{}
	currentRule := existing.SplitRule

	// =========================
	// HANDLE SPLIT RULE
	// =========================
	if req.SplitRule != nil {
		currentRule = normalizeSplitRule(*req.SplitRule)
		updateData["split_rule"] = currentRule
	}

	// =========================
	// HANDLE MODE
	// =========================

	if currentRule == "percentage" {

		po1 := existing.Po1Pct
		po2 := existing.Po2Pct

		if req.Po1Pct != nil {
			po1 = *req.Po1Pct
			updateData["po1_pct"] = po1
		}

		if req.Po2Pct != nil {
			po2 = *req.Po2Pct
			updateData["po2_pct"] = po2
		}

		if err := validatePercentage(po1, po2); err != nil {
			return nil, err
		}

		updateData["po_split_pct_sum"] = po1 + po2

	} else {
		// 🔥 HISTORY → FORCE AMAN
		updateData["po1_pct"] = 50
		updateData["po2_pct"] = 50
		updateData["po_split_pct_sum"] = 100
	}

	// =========================
	// COMMON FIELD
	// =========================
	if req.BudgetType != nil {
		updateData["budget_type"] = *req.BudgetType
	}

	if req.Description != nil {
		updateData["description"] = *req.Description
	}

	if req.MinOrderQty != nil {
		updateData["min_order_qty"] = *req.MinOrderQty
	}

	if req.MaxSplitLines != nil {
		updateData["max_split_lines"] = *req.MaxSplitLines
	}

	if req.Status != nil {
		updateData["status"] = normalizeStatus(*req.Status)
	}

	if err := s.repo.Update(ctx, id, updateData); err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

// =========================
// DELETE
// =========================
func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func validateSplitRule(rule string) error {
	switch strings.ToLower(rule) {
	case "percentage", "history":
		return nil
	default:
		return errors.New("invalid split_rule (must be 'percentage' or 'history')")
	}
}

func validatePercentage(po1, po2 float64) error {
	if po1+po2 != 100 {
		return errors.New("po1_pct + po2_pct must equal 100")
	}
	return nil
}

// =========================
// NORMALIZER
// =========================
func normalizeStatus(status string) string {
	status = strings.ToLower(status)

	switch status {
	case "active", "inactive":
		return status
	default:
		return "active"
	}
}

func normalizeSplitRule(rule string) string {
	return strings.ToLower(rule)
}
