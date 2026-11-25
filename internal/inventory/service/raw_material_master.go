package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
)

func normalizeMasterPart(v *string) string {
	if v == nil {
		return ""
	}
	return strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(*v)), " "))
}

func specKey(req invModels.CreateRawMaterialMasterRequest) string {
	num := func(v *float64, precision int) string {
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.*f", precision, *v)
	}
	return strings.Join([]string{
		normalizeMasterPart(req.MaterialGrade), normalizeMasterPart(req.Form), strings.ToLower(strings.TrimSpace(req.TypeMaterial)),
		num(req.WidthMM, 4), num(req.DiameterMM, 4), num(req.ThicknessMM, 4), num(req.LengthMM, 4), num(req.WeightKG, 6), normalizeMasterPart(req.UOM),
	}, "|")
}

func (s *service) ListRawMaterialMasters(ctx context.Context, search string, page, limit int) (*invModels.RawMaterialMasterList, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 5000 {
		limit = 5000
	}
	items, total, err := s.repo.ListRawMaterialMasters(ctx, search, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	return &invModels.RawMaterialMasterList{Items: items, Total: total}, nil
}

func (s *service) GetRawMaterialMaster(ctx context.Context, id int64) (*invModels.RawMaterialMaster, error) {
	return s.repo.GetRawMaterialMaster(ctx, id)
}

func (s *service) CreateRawMaterialMaster(ctx context.Context, req invModels.CreateRawMaterialMasterRequest, user string) (*invModels.RawMaterialMaster, error) {
	req.MaterialCode = strings.ToUpper(strings.TrimSpace(req.MaterialCode))
	req.MaterialName = strings.TrimSpace(req.MaterialName)
	if req.TypeMaterial == "" {
		req.TypeMaterial = "raw"
	}
	if req.Status == "" {
		req.Status = "Active"
	}
	now := time.Now()
	item := &invModels.RawMaterialMaster{MaterialCode: req.MaterialCode, MaterialName: req.MaterialName, MaterialGrade: req.MaterialGrade, Form: req.Form, TypeMaterial: req.TypeMaterial, WidthMM: req.WidthMM, DiameterMM: req.DiameterMM, ThicknessMM: req.ThicknessMM, LengthMM: req.LengthMM, WeightKG: req.WeightKG, UOM: req.UOM, SpecKey: specKey(req), Status: req.Status, CreatedBy: &user, UpdatedBy: &user, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateRawMaterialMaster(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) UpdateRawMaterialMaster(ctx context.Context, id int64, req invModels.UpdateRawMaterialMasterRequest, user string) (*invModels.RawMaterialMaster, error) {
	current, err := s.repo.GetRawMaterialMaster(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.MaterialCode != nil {
		current.MaterialCode = strings.ToUpper(strings.TrimSpace(*req.MaterialCode))
	}
	if req.MaterialName != nil {
		current.MaterialName = strings.TrimSpace(*req.MaterialName)
	}
	if req.MaterialGrade != nil {
		current.MaterialGrade = req.MaterialGrade
	}
	if req.Form != nil {
		current.Form = req.Form
	}
	if req.TypeMaterial != nil {
		current.TypeMaterial = *req.TypeMaterial
	}
	if req.WidthMM != nil {
		current.WidthMM = req.WidthMM
	}
	if req.DiameterMM != nil {
		current.DiameterMM = req.DiameterMM
	}
	if req.ThicknessMM != nil {
		current.ThicknessMM = req.ThicknessMM
	}
	if req.LengthMM != nil {
		current.LengthMM = req.LengthMM
	}
	if req.WeightKG != nil {
		current.WeightKG = req.WeightKG
	}
	if req.UOM != nil {
		current.UOM = req.UOM
	}
	if req.Status != nil {
		current.Status = *req.Status
	}
	keyReq := invModels.CreateRawMaterialMasterRequest{MaterialGrade: current.MaterialGrade, Form: current.Form, TypeMaterial: current.TypeMaterial, WidthMM: current.WidthMM, DiameterMM: current.DiameterMM, ThicknessMM: current.ThicknessMM, LengthMM: current.LengthMM, WeightKG: current.WeightKG, UOM: current.UOM}
	updates := map[string]interface{}{"material_code": current.MaterialCode, "material_name": current.MaterialName, "material_grade": current.MaterialGrade, "form": current.Form, "type_material": current.TypeMaterial, "width_mm": current.WidthMM, "diameter_mm": current.DiameterMM, "thickness_mm": current.ThicknessMM, "length_mm": current.LengthMM, "weight_kg": current.WeightKG, "uom": current.UOM, "status": current.Status, "spec_key": specKey(keyReq), "updated_by": user, "updated_at": time.Now()}
	return s.repo.UpdateRawMaterialMaster(ctx, id, updates)
}

func (s *service) DeleteRawMaterialMaster(ctx context.Context, id int64, user string) error {
	return s.repo.DeleteRawMaterialMaster(ctx, id, user)
}
func (s *service) ListRawMaterialPlanning(ctx context.Context) ([]invModels.RawMaterialPlanningItem, error) {
	return s.repo.ListRawMaterialPlanning(ctx)
}
