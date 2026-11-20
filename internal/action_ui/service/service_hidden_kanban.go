package service

import (
	"context"
	"log"
	"strings"

	"github.com/ganasa18/go-template/internal/action_ui/dto"
	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/pkg/assemblystock"
)

const (
	hiddenReasonIndirect  = "indirect"
	hiddenReasonChildPart = "child_part"
)

// [hidden-kanban] hiddenFromProduction menentukan kanban yang TIDAK diproses
// di Scan In/Out dan diabaikan oleh "Selesaikan Produksi":
//   - semua Material Spec milik UNIQ bertipe indirect
//   - WO Assembly: UNIQ adalah child di BOM aktif (hanya parent yang dirakit)
//
// ownBom = Material Spec milik UNIQ; nil = dimuat di sini.
func (s *service) hiddenFromProduction(ctx context.Context, wo models.WorkOrder, item models.WorkOrderItem, ownBom []dto.BomMaterial) (bool, string) {
	if ownBom == nil {
		ownBom, _ = s.buildOwnBomMaterials(ctx, item.ItemUniqCode)
	}
	if allIndirect(ownBom) {
		return true, hiddenReasonIndirect
	}
	if isAssemblyWO(wo.WOType) {
		child, err := assemblystock.IsChildItem(s.db.WithContext(ctx), item.ItemUniqCode)
		if err != nil {
			log.Printf("[hidden-kanban] cek child %s: %v", item.ItemUniqCode, err)
			return false, ""
		}
		if child {
			return true, hiddenReasonChildPart
		}
	}
	return false, ""
}

func allIndirect(specs []dto.BomMaterial) bool {
	if len(specs) == 0 {
		return false
	}
	for _, b := range specs {
		if !strings.EqualFold(strings.TrimSpace(b.TypeMaterial), "indirect") {
			return false
		}
	}
	return true
}
