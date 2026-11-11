package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/pkg/assemblystock"
	"gorm.io/gorm"
)

// finalOutputGoesToWIP menentukan tujuan hasil proses TERAKHIR sebuah WO item:
//   - WO Assembly                              -> Finished Goods
//   - item child di BOM aktif (child part /
//     item di tengah BOM)                      -> Inventory WIP
//   - selain itu (parent tanpa layer)          -> Finished Goods
func (s *service) finalOutputGoesToWIP(tx *gorm.DB, uniq string, wo models.WorkOrder) (bool, error) {
	if strings.EqualFold(strings.TrimSpace(wo.WOType), "Assembly") {
		return false, nil
	}
	return assemblystock.IsChildItem(tx, uniq)
}

// postFinalOutputToWIP menambah stok wip_inventories dari hasil proses terakhir
// dan menandai WO item selesai (padanan cabang Finished Goods).
func (s *service) postFinalOutputToWIP(tx *gorm.DB, item *models.WorkOrderItem, wo models.WorkOrder, lastProcess string, qty float64, performedBy string) error {
	notes := fmt.Sprintf("Hasil produksi kanban %s (Scan Out proses terakhir: %s)", item.KanbanNumber, lastProcess)
	if _, _, _, err := assemblystock.AdjustWIP(tx, assemblystock.WIPPost{
		UniqCode:     item.ItemUniqCode,
		PartNumber:   item.PartNumber,
		PartName:     item.PartName,
		Model:        item.Model,
		UOM:          item.UOM,
		WONumber:     wo.WONumber,
		KanbanNumber: item.KanbanNumber,
		MovementType: "production_in",
		Notes:        notes,
		By:           performedBy,
		Delta:        qty,
	}); err != nil {
		return err
	}
	item.Status = "FINISHED"
	item.LastScannedProcess = ""
	return tx.Save(item).Error
}

// applyWIPMoves menerapkan koreksi QC Round 3 (NG/scrap/adjustment) ke stok WIP.
// Padanan applyFGMoves untuk item yang hasilnya masuk Inventory WIP.
func (s *service) applyWIPMoves(tx *gorm.DB, item models.WorkOrderItem, wo models.WorkOrder, performedBy string, moves []fgMove) error {
	var total float64
	var notes []string
	for _, m := range moves {
		total += m.qty
		if n := strings.TrimSpace(m.notes); n != "" {
			notes = append(notes, n)
		}
	}
	if total > -1e-6 && total < 1e-6 {
		return nil
	}
	_, _, _, err := assemblystock.AdjustWIP(tx, assemblystock.WIPPost{
		UniqCode:     item.ItemUniqCode,
		WONumber:     wo.WONumber,
		KanbanNumber: item.KanbanNumber,
		MovementType: "qc_adjustment",
		Notes:        strings.Join(notes, "; "),
		By:           performedBy,
		Delta:        total,
	})
	return err
}

// consumeAssemblyComponents mengurangi stok komponen (yang sudah di-lock saat WO
// Assembly dibuat) pada Scan Out proses assembly (is_assembly = true). Bila routing
// tidak punya proses assembly, pengurangan dilakukan di Scan Out proses terakhir
// supaya lock tidak menggantung.
func (s *service) consumeAssemblyComponents(tx *gorm.DB, item models.WorkOrderItem, flow []models.ProcessFlow, currentIndex int, qty float64, performedBy string, now time.Time) error {
	if qty <= 0 || currentIndex < 0 || currentIndex >= len(flow) {
		return nil
	}

	type woRow struct {
		ID       int64  `gorm:"column:id"`
		WoNumber string `gorm:"column:wo_number"`
		WoType   string `gorm:"column:wo_type"`
	}
	var wo woRow
	if err := tx.Raw(`SELECT id, wo_number, wo_type FROM work_orders WHERE id = ?`, item.WOID).Scan(&wo).Error; err != nil {
		return err
	}
	if wo.ID == 0 || !strings.EqualFold(strings.TrimSpace(wo.WoType), "Assembly") {
		return nil
	}

	anyAssembly := false
	for _, st := range flow {
		if st.IsAssembly {
			anyAssembly = true
			break
		}
	}
	isAssemblyStep := flow[currentIndex].IsAssembly
	isLastStep := currentIndex == len(flow)-1
	if !(isAssemblyStep || (!anyAssembly && isLastStep)) {
		return nil
	}

	consumed, err := assemblystock.Consume(tx, wo.ID, wo.WoNumber, item.ItemUniqCode, qty, performedBy)
	if err != nil {
		return err
	}
	for _, c := range consumed {
		if c.Source == assemblystock.SourceWIP {
			continue // movement WIP sudah dicatat di wip_inventory_movements
		}
		category := "raw_material"
		if c.Source == assemblystock.SourceIRM {
			category = "indirect_raw_material"
		}
		entityID := c.EntityID
		notes := fmt.Sprintf("Komponen WO Assembly %s untuk %s (kanban %s)", wo.WoNumber, item.ItemUniqCode, item.KanbanNumber)
		if err := s.createInventoryMovementLog(tx, category, "outgoing",
			c.ComponentUniq, &entityID, c.Before, -c.Qty, c.After,
			stringPtr(wo.WoNumber), stringPtr("ASSEMBLY_CONSUME"),
			stringPtr(notes), stringPtr(performedBy)); err != nil {
			return err
		}
	}
	_ = now
	return nil
}
