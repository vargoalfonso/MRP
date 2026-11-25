package service

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier_item/models"
)

const maxBomUsageUniqs = 200

// ListBomUsage: FG / parent tempat uniq-uniq tersebut dipakai (traceability).
func (s *service) ListBomUsage(ctx context.Context, q models.BomUsageQuery) ([]models.BomUsageRow, error) {
	seen := make(map[string]struct{})
	codes := make([]string, 0)
	for _, part := range strings.Split(q.UniqCodes, ",") {
		code := strings.ToUpper(strings.TrimSpace(part))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
		if len(codes) >= maxBomUsageUniqs {
			break
		}
	}
	return s.repo.ListBomUsage(ctx, codes)
}

// ListBomMaterialGrades: material code yang ada di BOM + flag has_master.
func (s *service) ListBomMaterialGrades(ctx context.Context, q models.BomMaterialGradeQuery) ([]models.BomMaterialGrade, error) {
	typeMaterials, err := typeMaterialsFor(q.Type)
	if err != nil {
		return nil, err
	}
	return s.repo.ListBomMaterialGrades(ctx, typeMaterials, q.Search)
}
