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
	ScanAndUpdate(ctx context.Context, id, dnID int64, itemCode string) error
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

		for _, poItem := range poItems {

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

			items = append(items, models.DeliveryNoteItem{
				DNID:         dn.ID,
				ItemUniqCode: poItem.ItemUniqCode,
				Quantity:     int64(poItem.OrderedQty),
				UOM:          poItem.UOM,
				Weight:       int64(poItem.WeightKg),
				KanbanID:     kanban.ID,
				// QR:            qrImage,
				OrderQty:      int64(poItem.OrderedQty),
				PcsPerKanban:  poItem.PcsPerKanban,
				PackingNumber: kanban.KanbanNumber,
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
				"http://192.168.195.83:8899/api/v1/delivery-notes/scan?id=%d&dn_id=%d&item=%s",
				item.ID,
				item.DNID,
				item.ItemUniqCode,
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
		Preload("Items").
		Preload("Items.Kanban").
		Find(&data).Error

	return data, err
}

func (s *deliveryNoteService) GetByID(ctx context.Context, id int64) (*models.DeliveryNote, error) {
	var data models.DeliveryNote

	err := s.db.WithContext(ctx).
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

func (s *deliveryNoteService) ScanAndUpdate(ctx context.Context, id, dnID int64, itemCode string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var item models.DeliveryNoteItem

		// 🔥 1. VALIDASI: item harus ada & sesuai QR
		err := tx.Where("id = ? AND dn_id = ? AND item_uniq_code = ?", id, dnID, itemCode).
			First(&item).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("item tidak ditemukan")
			}
			return err
		}

		// 🔥 2. VALIDASI: jangan double scan
		if item.Check == "incoming" {
			return fmt.Errorf("item sudah di-scan sebelumnya")
		}

		now := time.Now()

		// 🔥 3. UPDATE ITEM STATUS
		err = tx.Model(&models.DeliveryNoteItem{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"check":         "incoming",
				"date_incoming": now,
				"updated_at":    now,
			}).Error

		if err != nil {
			return err
		}

		err = tx.Model(&models.DeliveryNote{}).
			Where("id = ?", dnID).
			Updates(map[string]interface{}{
				"total_dn_incoming": gorm.Expr("total_dn_incoming + ?", 1),
				"total_po_incoming": gorm.Expr("total_po_incoming + ?", item.OrderQty),
			}).Error

		return nil
	})
}
