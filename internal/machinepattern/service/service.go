// Package service implements business logic for the Machine Pattern module.
package service

import (
	"context"
	"fmt"
	"math"

	"github.com/ganasa18/go-template/internal/machinepattern/models"
	"github.com/ganasa18/go-template/internal/machinepattern/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
)

type IService interface {
	Create(ctx context.Context, req models.CreateMachinePatternRequest) (*models.MachinePatternResponse, error)
	BulkCreate(ctx context.Context, req models.BulkCreateRequest) (*models.BulkCreateResponse, error)
	Update(ctx context.Context, id int64, req models.UpdateMachinePatternRequest) (*models.MachinePatternResponse, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*models.MachinePatternResponse, error)
	List(ctx context.Context, q models.ListQuery) (*models.ListMachinePatternResponse, error)
	Calculate(ctx context.Context, req models.CalculateRequest) (*models.CalculateResult, error)
	GetParams(ctx context.Context) (*models.MachinePatternParam, error)
	UpdateParams(ctx context.Context, req models.UpdateParamRequest) (*models.MachinePatternParam, error)
	ListSafetyStock(ctx context.Context) ([]models.SafetyStockOutput, error)
}

type service struct{ repo repository.IRepository }

func New(repo repository.IRepository) IService { return &service{repo: repo} }

// ---------------------------------------------------------------------------
// Core CRUD
// ---------------------------------------------------------------------------

func (s *service) Create(ctx context.Context, req models.CreateMachinePatternRequest) (*models.MachinePatternResponse, error) {
	// Duplicate guard: one pattern per UNIQ+machine pair
	existing, err := s.repo.GetByUniqAndMachine(ctx, req.UniqCode, req.MachineID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.Conflict(fmt.Sprintf("pattern for uniq '%s' and machine_id %d already exists", req.UniqCode, req.MachineID))
	}

	params, err := s.repo.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	calc := calculate(req.PrlReference, float64(req.WorkingDays), req.CycleTimeSec, params)

	status := req.Status
	if status == "" {
		status = "Active"
	}

	mp := &models.MachinePattern{
		UniqCode:     req.UniqCode,
		MachineID:    req.MachineID,
		CycleTimeSec: req.CycleTimeSec,
		PrlReference: req.PrlReference,
		PatternValue: calc.PatternValue,
		WorkingDays:  req.WorkingDays,
		MovingType:   calc.MovingType,
		MinOutput:    req.MinOutput,
		Status:       status,
	}
	if err := s.repo.Create(ctx, mp); err != nil {
		return nil, err
	}
	return s.enrichOne(ctx, mp)
}

func (s *service) BulkCreate(ctx context.Context, req models.BulkCreateRequest) (*models.BulkCreateResponse, error) {
	params, err := s.repo.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	result := &models.BulkCreateResponse{}

	for i, item := range req.Items {
		existing, err := s.repo.GetByUniqAndMachine(ctx, item.UniqCode, item.MachineID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+1, err.Error()))
			continue
		}
		if existing != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: duplicate uniq_code '%s' + machine_id %d", i+1, item.UniqCode, item.MachineID))
			continue
		}

		calc := calculate(item.PrlReference, float64(item.WorkingDays), item.CycleTimeSec, params)
		status := item.Status
		if status == "" {
			status = "Active"
		}
		mp := &models.MachinePattern{
			UniqCode:     item.UniqCode,
			MachineID:    item.MachineID,
			CycleTimeSec: item.CycleTimeSec,
			PrlReference: item.PrlReference,
			PatternValue: calc.PatternValue,
			WorkingDays:  item.WorkingDays,
			MovingType:   calc.MovingType,
			MinOutput:    item.MinOutput,
			Status:       status,
		}
		if err := s.repo.Create(ctx, mp); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+1, err.Error()))
			continue
		}
		if resp, err := s.enrichOne(ctx, mp); err == nil {
			result.Items = append(result.Items, *resp)
			result.Created++
		}
	}
	return result, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateMachinePatternRequest) (*models.MachinePatternResponse, error) {
	mp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.CycleTimeSec != nil {
		mp.CycleTimeSec = *req.CycleTimeSec
	}
	if req.PrlReference != nil {
		mp.PrlReference = *req.PrlReference
	}
	if req.WorkingDays != nil {
		mp.WorkingDays = *req.WorkingDays
	}
	if req.MinOutput != nil {
		mp.MinOutput = *req.MinOutput
	}
	if req.Status != nil {
		mp.Status = *req.Status
	}

	// Recalculate derived fields
	params, err := s.repo.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	calc := calculate(mp.PrlReference, float64(mp.WorkingDays), mp.CycleTimeSec, params)
	mp.MovingType = calc.MovingType
	mp.PatternValue = calc.PatternValue

	if err := s.repo.Update(ctx, mp); err != nil {
		return nil, err
	}
	return s.enrichOne(ctx, mp)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.MachinePatternResponse, error) {
	mp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.enrichOne(ctx, mp)
}

func (s *service) List(ctx context.Context, q models.ListQuery) (*models.ListMachinePatternResponse, error) {
	if q.Limit < 1 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Page < 1 {
		q.Page = 1
	}

	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}

	machineIDSet := make(map[int64]struct{})
	for _, item := range items {
		machineIDSet[item.MachineID] = struct{}{}
	}
	machineIDs := make([]int64, 0, len(machineIDSet))
	for id := range machineIDSet {
		machineIDs = append(machineIDs, id)
	}
	machineNames, err := s.repo.GetMachineNamesByIDs(ctx, machineIDs)
	if err != nil {
		return nil, err
	}

	resps := make([]models.MachinePatternResponse, 0, len(items))
	for _, item := range items {
		resps = append(resps, toResponse(item, machineNames[item.MachineID]))
	}

	return &models.ListMachinePatternResponse{
		Total: total,
		Page:  q.Page,
		Limit: q.Limit,
		Items: resps,
	}, nil
}

// ---------------------------------------------------------------------------
// Calculate (preview only — no DB write)
// ---------------------------------------------------------------------------

func (s *service) Calculate(ctx context.Context, req models.CalculateRequest) (*models.CalculateResult, error) {
	if req.WorkingDays <= 0 {
		req.WorkingDays = 25
	}
	params, err := s.repo.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	calc := calculate(req.PrlReference, float64(req.WorkingDays), req.CycleTimeSec, params)
	return &calc, nil
}

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

func (s *service) GetParams(ctx context.Context) (*models.MachinePatternParam, error) {
	return s.repo.GetParams(ctx)
}

func (s *service) UpdateParams(ctx context.Context, req models.UpdateParamRequest) (*models.MachinePatternParam, error) {
	p, err := s.repo.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if req.FastMovingThreshold != nil {
		p.FastMovingThreshold = *req.FastMovingThreshold
	}
	if req.SlowMovingThreshold != nil {
		p.SlowMovingThreshold = *req.SlowMovingThreshold
	}
	if req.PatternMinMinutes != nil {
		p.PatternMinMinutes = *req.PatternMinMinutes
	}
	if req.DefaultWorkingDays != nil {
		p.DefaultWorkingDays = *req.DefaultWorkingDays
	}
	if err := s.repo.UpsertParams(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// Safety Stock output
// ---------------------------------------------------------------------------

func (s *service) ListSafetyStock(ctx context.Context) ([]models.SafetyStockOutput, error) {
	items, _, err := s.repo.List(ctx, models.ListQuery{Page: 1, Limit: 200, Status: "Active"})
	if err != nil {
		return nil, err
	}

	machineIDSet := make(map[int64]struct{})
	for _, item := range items {
		machineIDSet[item.MachineID] = struct{}{}
	}
	machineIDs := make([]int64, 0, len(machineIDSet))
	for id := range machineIDSet {
		machineIDs = append(machineIDs, id)
	}
	machineNames, err := s.repo.GetMachineNamesByIDs(ctx, machineIDs)
	if err != nil {
		return nil, err
	}

	result := make([]models.SafetyStockOutput, 0, len(items))
	for _, item := range items {
		// daily_output = min_output (already = daily_req * cycle_time * pattern)
		safetyStock := item.MinOutput * float64(item.WorkingDays)
		result = append(result, models.SafetyStockOutput{
			UniqCode:       item.UniqCode,
			MachineName:    machineNames[item.MachineID],
			DailyOutput:    item.MinOutput,
			PatternValue:   item.PatternValue,
			MovingType:     item.MovingType,
			SafetyStockQty: safetyStock,
		})
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Core calculation logic (parameterized)
// ---------------------------------------------------------------------------

// calculate derives MovingType, PatternValue, and MinOutput from raw inputs.
//
// Logic (per spec):
//  1. daily_req = prl / working_days
//  2. Fast Moving  : daily_req > fast_threshold  (default C=1000)
//     Slow Moving  : daily_req < slow_threshold  (default C=1000)
//     Normal       : daily_req == threshold (edge case)
//  3. Pattern calc : (daily_req * cycle_time_sec) >= pattern_min_seconds (default 48 min)
//     → pattern = 1 (one run fills the time window)
//     else          → pattern = floor(pattern_min_sec / (daily_req * cycle_time_sec)), min 1
//  4. min_output   = daily_req * cycle_time_sec * pattern_value
func calculate(prl, workingDays, cycleTimeSec float64, params *models.MachinePatternParam) models.CalculateResult {
	if workingDays <= 0 {
		workingDays = float64(params.DefaultWorkingDays)
	}

	dailyReq := prl / workingDays

	// Moving type
	var movingType string
	switch {
	case dailyReq > params.FastMovingThreshold:
		movingType = "Fast Moving"
	case dailyReq < params.SlowMovingThreshold:
		movingType = "Slow Moving"
	default:
		movingType = "Normal"
	}

	// Pattern value
	patternMinSec := params.PatternMinMinutes * 60 // convert minutes → seconds
	dailyLoadSec := dailyReq * cycleTimeSec

	var patternValue float64
	if dailyLoadSec <= 0 {
		patternValue = 1
	} else if dailyLoadSec >= patternMinSec {
		// Total load already fills the time window → 1 production run
		patternValue = 1
	} else {
		// Fit as many patterns as the time window allows
		patternValue = math.Floor(patternMinSec / dailyLoadSec)
		if patternValue < 1 {
			patternValue = 1
		}
	}

	// min_output = daily_req * cycle_time_sec * pattern
	minOutput := dailyReq * cycleTimeSec * patternValue

	return models.CalculateResult{
		DailyRequirement: dailyReq,
		MovingType:       movingType,
		PatternValue:     patternValue,
		MinOutput:        minOutput,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *service) enrichOne(ctx context.Context, mp *models.MachinePattern) (*models.MachinePatternResponse, error) {
	machineNames, err := s.repo.GetMachineNamesByIDs(ctx, []int64{mp.MachineID})
	if err != nil {
		return nil, err
	}
	resp := toResponse(*mp, machineNames[mp.MachineID])
	return &resp, nil
}

func toResponse(mp models.MachinePattern, machineName string) models.MachinePatternResponse {
	return models.MachinePatternResponse{
		ID:           mp.ID,
		UniqCode:     mp.UniqCode,
		MachineName:  machineName,
		MachineID:    mp.MachineID,
		CycleTimeSec: mp.CycleTimeSec,
		PrlReference: mp.PrlReference,
		PatternValue: mp.PatternValue,
		WorkingDays:  mp.WorkingDays,
		MovingType:   mp.MovingType,
		MinOutput:    mp.MinOutput,
		Status:       mp.Status,
		CreatedAt:    mp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    mp.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
