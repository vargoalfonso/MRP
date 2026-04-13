package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
	deliveryNoteRepo "github.com/ganasa18/go-template/internal/delivery_note/repository"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type IDeliveryNoteService interface {
	Create(ctx context.Context, req models.CreateDNRequest) (*models.DeliveryNote, error)
	GetAll(ctx context.Context) ([]models.DeliveryNote, error)
	GetByID(ctx context.Context, id int64) (*models.DeliveryNote, error)
	ScanAndUpdate(ctx context.Context, packing string) (string, error)
	PreviewDN(ctx context.Context, req models.CreateDNRequest) (*models.PreviewDNResponse, error)
}

// implementation
type deliveryNoteService struct {
	repo deliveryNoteRepo.IDeliveryNoteRepository
	db   *gorm.DB
}

func New(repo deliveryNoteRepo.IDeliveryNoteRepository, db *gorm.DB) IDeliveryNoteService {
	return &deliveryNoteService{
		repo: repo,
		db:   db,
	}
}

// =========================
// CRUD
// =========================

func (s *deliveryNoteService) Create(ctx context.Context, req models.CreateDNRequest) (*models.DeliveryNote, error) {
	var dn models.DeliveryNote

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		year := time.Now().Year()

		// default type
		dnType := req.Type
		if dnType == "" {
			dnType = "RM"
		}

		prefix := fmt.Sprintf("DN-%s-%d", dnType, year)

		last, err := s.repo.FindLastDNNumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		var dnNumber string

		if last == "" {
			dnNumber = fmt.Sprintf("%s-0001", prefix)
		} else {
			dnNumber = generateDNNumber(last, prefix)
		}

		incomingDate, err := time.Parse("02/01/2006", req.IncomingDate)
		if err != nil {
			return fmt.Errorf("invalid date format, use dd/mm/yyyy")
		}

		// 🔥 1. GET PO
		po, err := s.repo.GetPOByPONumber(ctx, req.PONumber)
		if err != nil {
			return fmt.Errorf("PO tidak ditemukan")
		}

		// 🔥 2. GET PO ITEMS
		poItems, err := s.repo.GetPOItemsByPOID(ctx, po.PoID)
		if err != nil {
			return err
		}

		if len(poItems) == 0 {
			return fmt.Errorf("PO tidak memiliki item")
		}

		// 🔥 3. TOTAL PO QTY
		totalQty, err := s.repo.GetTotalQtyByPOID(ctx, po.PoID)
		if err != nil {
			return err
		}

		// 🔥 5. TOTAL DN INCOMING
		totalIncoming, err := s.repo.CountDNIncomingByPONumber(ctx, po.PoNumber)
		if err != nil {
			return err
		}

		// 🔥 CREATE DN
		dn = models.DeliveryNote{
			DNNumber:        dnNumber,
			PONumber:        po.PoNumber,
			CustomerID:      req.CustomerID,
			ContactPerson:   req.ContactPerson,
			Period:          req.Period,
			Type:            dnType,
			Status:          "draft",
			IncomingDate:    incomingDate,
			SupplierID:      po.SupplierID,
			TotalPOQty:      int64(totalQty),
			TotalDNCreated:  int64(len(poItems)),
			TotalDNIncoming: int64(totalIncoming),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.repo.Create(ctx, tx, &dn); err != nil {
			return err
		}

		// 🔥 6. GENERATE ITEMS DARI PO ITEMS
		var items []models.DeliveryNoteItem

		for i, poItem := range poItems {

			// 🔥 ambil kanban
			kanban, err := s.repo.GetKanbanByItemCode(ctx, poItem.ItemUniqCode)
			if err != nil {
				return err
			}

			// qrValue := fmt.Sprintf("%s-%s", dn.DNNumber, poItem.ItemUniqCode)

			// qrImage, err := generateQRBase64(qrValue)
			// if err != nil {
			// 	return err
			// }

			seq := fmt.Sprintf("%04d", i+1)
			dnNumberID := fmt.Sprintf("%04d", dn.ID)

			packingNumber := fmt.Sprintf("DN-%s-PKG-%s", dnNumberID, seq)

			items = append(items, models.DeliveryNoteItem{
				DNID:         dn.ID,
				ItemUniqCode: poItem.ItemUniqCode,
				Quantity:     int64(poItem.OrderedQty),
				UOM:          poItem.UOM,
				Weight:       int64(poItem.WeightKg),
				KanbanID:     kanban.ID,
				// QR:            qrImage,
				OrderQty:      int64(poItem.OrderedQty),
				QtyStated:     int64(poItem.OrderedQty),
				PcsPerKanban:  poItem.PcsPerKanban,
				PackingNumber: packingNumber,
				Check:         "progress",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
		}

		if len(items) > 0 {
			if err := s.repo.CreateItems(ctx, tx, items); err != nil {
				return err
			}
		}

		for _, item := range items {

			qrValue := fmt.Sprintf(
				"http://127.0.0.1:8899/api/v1/delivery-notes/scan?packing=%s",
				item.PackingNumber,
			)

			qrBase64, err := generateQRBase64(qrValue)
			if err != nil {
				return err
			}

			if err := tx.Model(&models.DeliveryNoteItem{}).
				Where("id = ?", item.ID).
				Update("qr", qrBase64).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dn, nil
}

func (s *deliveryNoteService) GetAll(ctx context.Context) ([]models.DeliveryNote, error) {
	var data []models.DeliveryNote

	err := s.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Items").
		Preload("Items.Kanban").
		Find(&data).Error

	return data, err
}

func (s *deliveryNoteService) GetByID(ctx context.Context, id int64) (*models.DeliveryNote, error) {
	var data models.DeliveryNote

	err := s.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Items").
		Preload("Items.Kanban").
		First(&data, id).Error

	if err != nil {
		return nil, err
	}

	return &data, nil
}

func generateDNNumber(last string, prefix string) string {
	if last == "" {
		return fmt.Sprintf("%s-0001", prefix)
	}

	parts := strings.Split(last, "-")
	if len(parts) == 0 {
		return fmt.Sprintf("%s-0001", prefix)
	}

	seqStr := parts[len(parts)-1]

	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		return fmt.Sprintf("%s-0001", prefix)
	}

	seq++

	return fmt.Sprintf("%s-%04d", prefix, seq)
}

// func generateQRBase64(content string) (string, error) {
// 	png, err := qrcode.Encode(content, qrcode.Medium, 256)
// 	if err != nil {
// 		return "", err
// 	}

// 	base64Str := base64.StdEncoding.EncodeToString(png)

// 	return "data:image/png;base64," + base64Str, nil
// }

func generateQRBase64(value string) (string, error) {
	// generate PNG QR (256x256)
	png, err := qrcode.Encode(value, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	// encode ke base64
	base64Str := base64.StdEncoding.EncodeToString(png)

	// optional: prefix biar langsung bisa dipakai di frontend
	return "data:image/png;base64," + base64Str, nil
}

func (s *deliveryNoteService) ScanAndUpdate(ctx context.Context, packing string) (string, error) {
	returnStatus := ""

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var item models.DeliveryNoteItem

		// 🔥 1. cari item
		err := tx.Where("packing_number = ?", packing).
			First(&item).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("item tidak ditemukan")
			}
			return err
		}

		now := time.Now()

		// 🔥 2. kalau sudah pernah scan
		if item.Check == "incoming" || item.Check == "completed" {

			if item.ReceivedAt != nil {

				diff := now.Sub(*item.ReceivedAt).Seconds()

				// ❌ duplicate < 60 detik
				if diff <= 60 {
					return fmt.Errorf("duplicate scan detected, please wait before scanning again")
				}

				// 🔥 > 60 detik → completed
				err = tx.Model(&models.DeliveryNoteItem{}).
					Where("id = ?", item.ID).
					Updates(map[string]interface{}{
						"check":      "completed",
						"updated_at": now,
					}).Error

				if err != nil {
					return err
				}

				returnStatus = "completed"
				return nil
			}
		}

		// 🔥 3. FIRST SCAN → incoming
		err = tx.Model(&models.DeliveryNoteItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"check":           "incoming",
				"qty_received":    item.OrderQty,
				"weight_received": item.Weight,
				"date_incoming":   now,
				"received_at":     now,
				"updated_at":      now,
			}).Error

		if err != nil {
			return err
		}

		// 🔥 4. update DN
		err = tx.Model(&models.DeliveryNote{}).
			Where("id = ?", item.DNID).
			Updates(map[string]interface{}{
				"total_dn_incoming": gorm.Expr("COALESCE(total_dn_incoming, 0) + ?", 1),
				"total_po_incoming": gorm.Expr("COALESCE(total_po_incoming, 0) + ?", item.OrderQty),
			}).Error

		if err != nil {
			return err
		}

		returnStatus = "incoming"
		return nil
	})

	if err != nil {
		return "", err
	}

	return returnStatus, nil
}

func (s *deliveryNoteService) PreviewDN(ctx context.Context, req models.CreateDNRequest) (*models.PreviewDNResponse, error) {

	// 🔥 1. ambil PO
	po, err := s.repo.GetPOByPONumber(ctx, req.PONumber)
	if err != nil {
		return nil, fmt.Errorf("PO tidak ditemukan")
	}

	supplier, err := s.repo.GetSupplierByID(ctx, po.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("Supplier tidak ditemukan")
	}

	// 🔥 2. ambil item PO
	poItems, err := s.repo.GetPOItemsByPOID(ctx, po.PoID)
	if err != nil {
		return nil, err
	}

	// 🔥 3. total qty PO
	totalQty, err := s.repo.GetTotalQtyByPOID(ctx, po.PoID)
	if err != nil {
		return nil, err
	}

	// 🔥 4. total incoming DN
	totalIncoming, err := s.repo.CountDNIncomingByPONumber(ctx, req.PONumber)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("DN-%s", req.Type)

	count, err := s.repo.CountDNByPrefix(ctx, prefix)

	if err != nil {
		return nil, err
	}

	fmt.Printf("count existing DN with prefix %s: %d\n", req.Type, count)
	next := count + 1

	dnNumber := fmt.Sprintf("DN-%s-%04d", req.Type, next)

	// 🔥 5. mapping items
	var items []models.PreviewDNItemResponse

	for i, poItem := range poItems {

		seq := fmt.Sprintf("%04d", i+1)

		packingNumber := fmt.Sprintf("%s-PKG-%s", dnNumber, seq)

		items = append(items, models.PreviewDNItemResponse{
			ItemUniqCode:  poItem.ItemUniqCode,
			MaterialInfo:  poItem.ItemUniqCode, // atau gabung
			TotalQty:      int64(poItem.OrderedQty),
			RemainingQty:  int64(0), // sesuaikan dengan logika bisnis Anda
			UOM:           poItem.UOM,
			OrderQty:      int64(poItem.OrderedQty),
			PcsPerKanban:  poItem.PcsPerKanban,
			PackingNumber: packingNumber,
		})
	}

	return &models.PreviewDNResponse{
		Period:          req.Period,
		PONumber:        po.PoNumber,
		Supplier:        supplier.SupplierName,
		TotalPO:         int64(totalQty),
		TotalIncoming:   int64(totalIncoming),
		TotalDNCreatd:   int64(len(poItems)),
		TotalDNIncoming: int64(totalIncoming),
		DateIncoming:    req.IncomingDate,
		Items:           items,
	}, nil
}
