package service

import (
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"gorm.io/gorm"
)

// reduceWIPToFinishedGoods moves produced quantity out of Work In Progress when
// a work order item is finished at QC (QC Finish / round 3). It reduces the WIP
// items for the given WO + item and writes history to wip_logs and
// inventory_movement_logs. It never lets WIP stock go negative.
func (s *service) reduceWIPToFinishedGoods(tx *gorm.DB, woID int64, uniq string, woNumber string, reduceQty float64, performedBy string, now time.Time) error {
	if reduceQty <= 0 {
		return nil
	}
	remaining := int(reduceQty)
	if remaining <= 0 {
		return nil
	}

	var wipIDs []int64
	if err := tx.Table("wips").Where("wo_id = ?", woID).Pluck("id", &wipIDs).Error; err != nil {
		return err
	}
	if len(wipIDs) == 0 {
		return nil
	}

	var items []models.WIPItem
	if err := tx.Where("wip_id IN ? AND uniq = ? AND qty_remaining > 0", wipIDs, uniq).
		Order("op_seq DESC, id DESC").
		Find(&items).Error; err != nil {
		return err
	}

	for i := range items {
		if remaining <= 0 {
			break
		}
		it := items[i]
		take := it.QtyRemaining
		if take > remaining {
			take = remaining
		}
		if take <= 0 {
			continue
		}

		beforeStock := it.Stock
		newStock := it.Stock - take
		if newStock < 0 {
			newStock = 0
		}
		newRemaining := it.QtyRemaining - take
		if newRemaining < 0 {
			newRemaining = 0
		}
		newStatus := it.Status
		if newRemaining == 0 {
			newStatus = "done"
		}

		if err := tx.Model(&models.WIPItem{}).Where("id = ?", it.ID).
			Updates(map[string]interface{}{
				"stock":         newStock,
				"qty_out":       it.QtyOut + take,
				"qty_remaining": newRemaining,
				"status":        newStatus,
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Table("wip_logs").Create(map[string]interface{}{
			"wip_item_id": it.ID,
			"action":      "MOVE_TO_FG",
			"qty":         take,
			"created_at":  now,
		}).Error; err != nil {
			return err
		}

		entityID := it.ID
		if err := s.createInventoryMovementLog(tx, "wip", "outgoing",
			uniq, &entityID, float64(beforeStock), float64(take), float64(newStock),
			&woNumber, stringPtr("QC_FINISH"),
			stringPtr("WIP dipindah ke Finished Goods (QC Finish)"), stringPtr(performedBy)); err != nil {
			return err
		}

		remaining -= take
	}

	return nil
}
