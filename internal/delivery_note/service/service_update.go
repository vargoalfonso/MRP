package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
	"gorm.io/gorm"
)

var (
	// ErrDNNotFound is returned when the delivery note does not exist.
	ErrDNNotFound = errors.New("delivery note tidak ditemukan")
	// ErrDNLocked is returned when the delivery note was already processed (scan in/out).
	ErrDNLocked = errors.New("DN sudah diproses (ada scan incoming/outgoing), tidak bisa diedit")
)

// dnItemProcessed reports whether an item was already scanned in or out.
func dnItemProcessed(it models.DeliveryNoteItem) bool {
	if it.QtyReceived != 0 || it.QtySent != 0 || it.ReceivedAt != nil {
		return true
	}
	check := strings.ToLower(strings.TrimSpace(it.Check))
	return check != "" && check != "progress"
}

// Update edits an unprocessed delivery note: header (wo_number, delivery_to,
// delivery_total) and, for existing items, quantity and incoming date.
//
// Rules (same as Create, so an edit can never exceed what Create would accept):
//   - qty > 0 and qty <= ordered qty of the PO item;
//   - qty already used by other DN items of the same uniq + new qty <= ordered qty.
//
// Packing numbers and QR codes are not touched, so printed labels stay valid.
func (s *deliveryNoteService) Update(ctx context.Context, id int64, req models.UpdateDNRequest) (*models.DeliveryNote, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dn models.DeliveryNote
		if err := tx.Preload("Items").Where("id = ?", id).First(&dn).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDNNotFound
			}
			return err
		}

		if !strings.EqualFold(strings.TrimSpace(dn.Status), "active") {
			return ErrDNLocked
		}
		for _, it := range dn.Items {
			if dnItemProcessed(it) {
				return ErrDNLocked
			}
		}

		itemByID := make(map[int64]models.DeliveryNoteItem, len(dn.Items))
		for _, it := range dn.Items {
			itemByID[it.ID] = it
		}

		if len(req.Items) > 0 {
			po, err := s.repo.GetPOByPONumber(ctx, dn.PONumber)
			if err != nil {
				return fmt.Errorf("PO tidak ditemukan")
			}
			poItems, err := s.repo.GetPOItemsByPOID(ctx, po.PoID)
			if err != nil {
				return err
			}
			poMap := make(map[string]models.PurchaseOrderItem, len(poItems))
			for _, item := range poItems {
				poMap[item.ItemUniqCode] = item
			}

			seen := make(map[int64]bool, len(req.Items))
			for _, in := range req.Items {
				cur, ok := itemByID[in.ID]
				if !ok {
					return fmt.Errorf("item %d bukan bagian dari DN ini", in.ID)
				}
				if seen[in.ID] {
					return fmt.Errorf("duplicate item %d", in.ID)
				}
				seen[in.ID] = true

				if in.Qty <= 0 {
					return fmt.Errorf("qty harus > 0")
				}

				poItem, ok := poMap[cur.ItemUniqCode]
				if !ok {
					return fmt.Errorf("item %s tidak ada di PO", cur.ItemUniqCode)
				}
				if in.Qty > int64(poItem.OrderedQty) {
					return fmt.Errorf("qty melebihi PO")
				}

				used, err := s.repo.GetUsedQtyByItem(ctx, cur.ItemUniqCode)
				if err != nil {
					return err
				}
				// "used" already includes this item's current quantity.
				if used-cur.Quantity+in.Qty > int64(poItem.OrderedQty) {
					return fmt.Errorf("qty over %s", cur.ItemUniqCode)
				}

				fields := map[string]interface{}{
					"quantity":   in.Qty,
					"updated_at": time.Now(),
				}
				if strings.TrimSpace(in.IncomingDate) != "" {
					date, err := time.Parse("02/01/2006", strings.TrimSpace(in.IncomingDate))
					if err != nil {
						return fmt.Errorf("incoming_date tidak valid (format dd/MM/yyyy): %w", err)
					}
					fields["date_incoming"] = date
				}

				if err := tx.Model(&models.DeliveryNoteItem{}).
					Where("id = ? AND dn_id = ?", cur.ID, dn.ID).
					Updates(fields).Error; err != nil {
					return err
				}
			}
		}

		header := map[string]interface{}{"updated_at": time.Now()}
		if req.WONumber != nil {
			header["wo_number"] = strings.TrimSpace(*req.WONumber)
		}
		if req.DeliveryTo != nil {
			if *req.DeliveryTo < 0 {
				return fmt.Errorf("delivery_to tidak boleh negatif")
			}
			header["delivery_to"] = *req.DeliveryTo
		}
		if req.DeliveryTotal != nil {
			if *req.DeliveryTotal < 0 {
				return fmt.Errorf("delivery_total tidak boleh negatif")
			}
			header["delivery_total"] = *req.DeliveryTotal
		}
		return tx.Model(&models.DeliveryNote{}).Where("id = ?", dn.ID).Updates(header).Error
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}
