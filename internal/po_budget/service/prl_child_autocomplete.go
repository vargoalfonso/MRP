package service

import (
	"encoding/json"
	"strings"

	"github.com/ganasa18/go-template/internal/po_budget/models"
)

type prlChildReference struct {
	UniqCode      string `json:"uniq_code"`
	MaterialGrade string `json:"material_grade"`
}

func shouldAttachPRLChildren(budgetType string) bool {
	switch strings.TrimSpace(strings.ToLower(budgetType)) {
	case "raw_material", "indirect":
		return true
	default:
		return false
	}
}

func parsePRLChildRefs(raw []byte) ([]prlChildReference, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return []prlChildReference{}, nil
	}
	var refs []prlChildReference
	if err := json.Unmarshal(raw, &refs); err != nil {
		return nil, err
	}
	out := make([]prlChildReference, 0, len(refs))
	for _, ref := range refs {
		uniqCode := strings.TrimSpace(ref.UniqCode)
		materialGrade := strings.TrimSpace(ref.MaterialGrade)
		if uniqCode == "" {
			continue
		}
		out = append(out, prlChildReference{
			UniqCode:      uniqCode,
			MaterialGrade: materialGrade,
		})
	}
	return out, nil
}

func extractPRLChildUniqCodes(raw []byte) ([]string, error) {
	refs, err := parsePRLChildRefs(raw)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(refs))
	for _, ref := range refs {
		codes = append(codes, ref.UniqCode)
	}
	return codes, nil
}

func buildPRLAutocompleteChildren(raw []byte, candidates []models.CurrentBomChildRow, parentQty float64) ([]models.PrlForecastChildResponse, error) {
	refs, err := parsePRLChildRefs(raw)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 || len(candidates) == 0 {
		return []models.PrlForecastChildResponse{}, nil
	}

	candidateByUniq := make(map[string]models.CurrentBomChildRow, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(strings.TrimSpace(candidate.UniqCode))
		if key == "" {
			continue
		}
		candidateByUniq[key] = candidate
	}

	children := make([]models.PrlForecastChildResponse, 0, len(refs))
	for _, ref := range refs {
		candidate, ok := candidateByUniq[strings.ToLower(ref.UniqCode)]
		if !ok {
			continue
		}

		materialGrade := ref.MaterialGrade
		if materialGrade == "" && candidate.MaterialSpec.MaterialGrade != nil {
			materialGrade = strings.TrimSpace(*candidate.MaterialSpec.MaterialGrade)
		}
		if materialGrade == "" {
			materialGrade = candidate.UniqCode
		}

		spec := candidate.MaterialSpec
		if spec.MaterialGrade == nil && materialGrade != "" {
			spec.MaterialGrade = strPtr(materialGrade)
		}

		qtyPerUniq := candidate.QtyPerUniq
		if qtyPerUniq <= 0 {
			qtyPerUniq = 1
		}
		childQty := parentQty * qtyPerUniq
		if childQty < 0 {
			childQty = 0
		}

		children = append(children, models.PrlForecastChildResponse{
			Uniq:                materialGrade,
			UniqCode:            candidate.UniqCode,
			PartName:            candidate.PartName,
			PartNumber:          candidate.PartNumber,
			Model:               candidate.Model,
			QtyPerUniq:          candidate.QtyPerUniq,
			WeightKg:            spec.WeightKg,
			Quantity:            childQty,
			ExistingRawMaterial: nil,
			Uom:                 candidate.Uom,
			MaterialSpec:        &spec,
			Suppliers:           []models.PrlForecastChildSupplierResponse{},
		})
	}

	return children, nil
}
