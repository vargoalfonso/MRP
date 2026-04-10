package service

import (
	"context"
	"encoding/base64"
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

		// default type kalau kosong
		dnType := req.Type
		if dnType == "" {
			dnType = "RM"
		}

		// prefix: DN-RM-2026
		prefix := fmt.Sprintf("DN-%s-%d", dnType, year)

		last, err := s.repo.FindLastDNNumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		dnNumber := generateDNNumber(last, prefix)

		incomingDate, err := time.Parse("02/01/2006", req.IncomingDate)
		if err != nil {
			return fmt.Errorf("invalid date format, use dd/mm/yyyy")
		}

		dn = models.DeliveryNote{
			DNNumber:      dnNumber,
			PONumber:      req.PONumber,
			CustomerID:    req.CustomerID,
			ContactPerson: req.ContactPerson,
			Period:        req.Period,
			Type:          dnType,
			Status:        "draft",
			IncomingDate:  incomingDate,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := s.repo.Create(ctx, tx, &dn); err != nil {
			return err
		}

		// 🔥 insert items
		var items []models.DeliveryNoteItem

		for _, item := range req.Items {

			qrValue := fmt.Sprintf("%s-%s", dn.DNNumber, item.ItemUniqCode)

			qrImage, err := generateQRBase64(qrValue)
			if err != nil {
				return err
			}

			uomName, err := s.repo.UomByCode(ctx, item.ItemUniqCode)
			if err != nil {
				return err
			}

			items = append(items, models.DeliveryNoteItem{
				DNID:         dn.ID,
				ItemUniqCode: item.ItemUniqCode,
				Quantity:     item.Quantity,
				UOM:          uomName,
				KanbanID:     item.KanbanID,
				QR:           qrImage,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		}

		if len(items) > 0 {
			if err := s.repo.CreateItems(ctx, tx, items); err != nil {
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

func generateQRBase64(content string) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	base64Str := base64.StdEncoding.EncodeToString(png)

	return "data:image/png;base64," + base64Str, nil
}
