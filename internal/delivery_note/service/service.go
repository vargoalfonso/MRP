package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	workflowModels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	approvalRepo "github.com/ganasa18/go-template/internal/approval_workflow/repository"
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
	repo         deliveryNoteRepo.IDeliveryNoteRepository
	db           *gorm.DB
	approvalRepo approvalRepo.IApprovalWorkflowRepository
}

func New(repo deliveryNoteRepo.IDeliveryNoteRepository, db *gorm.DB, approvalRepo approvalRepo.IApprovalWorkflowRepository) IDeliveryNoteService {
	return &deliveryNoteService{
		repo:         repo,
		db:           db,
		approvalRepo: approvalRepo,
	}
}

// =========================
// CRUD
// =========================

func (s *deliveryNoteService) Create(ctx context.Context, req models.CreateDNRequest) (*models.DeliveryNote, error) {
	var dn models.DeliveryNote

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// ==============================
		// 🔥 GENERATE DN NUMBER
		// ==============================
		year := time.Now().Year()
		prefix := fmt.Sprintf("DN-%s-%d", req.Type, year)

		last, _ := s.repo.FindLastDNNumber(ctx, tx, prefix)

		if last == "" {
			dn.DNNumber = fmt.Sprintf("%s-0001", prefix)
		} else {
			dn.DNNumber = generateDNNumber(last, prefix)
		}

		// ==============================
		// 🔥 GET PO
		// ==============================
		po, err := s.repo.GetPOByPONumber(ctx, req.PONumber)
		if err != nil {
			return fmt.Errorf("PO tidak ditemukan")
		}

		// ==============================
		// 🔥 GET PO ITEMS
		// ==============================
		poItems, err := s.repo.GetPOItemsByPOID(ctx, po.PoID)
		if err != nil {
			return err
		}

		poMap := make(map[string]models.PurchaseOrderItem)
		for _, item := range poItems {
			poMap[item.ItemUniqCode] = item
		}

		// ==============================
		// 🔥 SUMMARY
		// ==============================
		totalQty, _ := s.repo.GetTotalQtyByPOID(ctx, po.PoID)
		summary, _ := s.repo.GetDNSummaryByPO(ctx, po.PoNumber)

		if summary == nil {
			summary = &deliveryNoteRepo.DNCountSummary{
				Total:    1,
				Incoming: 0,
			}
		}
		// ==============================
		// 🔥 CREATE HEADER
		// ==============================
		dn = models.DeliveryNote{
			DNNumber:        dn.DNNumber,
			PONumber:        po.PoNumber,
			CustomerID:      req.CustomerID,
			ContactPerson:   req.ContactPerson,
			Period:          req.Period,
			Type:            req.Type,
			Status:          "draft",
			SupplierID:      po.SupplierID,
			TotalPOQty:      totalQty,
			TotalDNCreated:  summary.Total,
			TotalDNIncoming: summary.Incoming,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.repo.Create(ctx, tx, &dn); err != nil {
			return err
		}

		workflow, err := s.approvalRepo.FindByActionName(ctx, "Delivery Note")
		if err != nil {
			return fmt.Errorf("workflow belum dibuat oleh operator")
		}

		// ==============================
		// 🔥 BUILD APPROVAL
		// ==============================
		progress, maxLevel := BuildApprovalProgress(*workflow)

		// ==============================
		// 🔥 CREATE APPROVAL INSTANCE
		// ==============================
		instance := workflowModels.ApprovalInstance{
			ActionName:         workflow.ActionName,
			ReferenceTable:     "delivery_notes",
			ReferenceID:        dn.ID,
			ApprovalWorkflowID: workflow.ID,
			CurrentLevel:       1,
			MaxLevel:           maxLevel,
			Status:             "pending",
			SubmittedBy:        "system",
			ApprovalProgress:   progress,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := s.approvalRepo.CreateInstance(ctx, tx, &instance); err != nil {
			return err
		}

		// ==============================
		// 🔥 VALIDASI ITEMS
		// ==============================
		seen := make(map[string]bool)
		var items []models.DeliveryNoteItem

		for i, it := range req.Items {

			poItem, ok := poMap[it.ItemUniqCode]
			if !ok {
				return fmt.Errorf("item %s tidak ada di PO", it.ItemUniqCode)
			}

			if seen[it.ItemUniqCode] {
				return fmt.Errorf("duplicate item %s", it.ItemUniqCode)
			}
			seen[it.ItemUniqCode] = true

			if it.Qty <= 0 {
				return fmt.Errorf("qty harus > 0")
			}

			if it.Qty > int64(poItem.OrderedQty) {
				return fmt.Errorf("qty melebihi PO")
			}

			used, _ := s.repo.GetUsedQtyByItem(ctx, it.ItemUniqCode)
			if used+it.Qty > int64(poItem.OrderedQty) {
				return fmt.Errorf("qty over %s", it.ItemUniqCode)
			}

			kanban, err := s.repo.GetKanbanByItemCode(ctx, it.ItemUniqCode)
			if err != nil {
				return err
			}

			date, err := time.Parse("02/01/2006", it.IncomingDate)
			if err != nil {
				return err
			}

			packing := fmt.Sprintf("DN-%04d-PKG-%04d", dn.ID, i+1)

			items = append(items, models.DeliveryNoteItem{
				DNID:          dn.ID,
				ItemUniqCode:  it.ItemUniqCode,
				Quantity:      it.Qty,
				OrderQty:      int64(poItem.OrderedQty),
				QtyStated:     it.Qty,
				UOM:           poItem.UOM,
				Weight:        int64(poItem.WeightKg),
				KanbanID:      kanban.ID,
				PcsPerKanban:  poItem.PcsPerKanban,
				PackingNumber: packing,
				DateIncoming:  &date,
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

		var dn models.DeliveryNote

		err = tx.Where("id = ?", item.DNID).
			First(&dn).Error

		if err != nil {
			return err
		}

		if dn.Status == "completed" {
			return fmt.Errorf("delivery note sudah completed, tidak bisa scan")
		}

		if dn.Status != "draft" && dn.Status != "incoming" && dn.Status != "waiting" {
			return fmt.Errorf("delivery note tidak aktif")
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
			DateIncoming:  time.Now().Format("02/01/2006"),
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
		Items:           items,
	}, nil
}

func BuildApprovalProgress(workflow workflowModels.ApprovalWorkflow) (workflowModels.ApprovalProgress, int) {

	roles := []string{
		workflow.Level1Role,
		workflow.Level2Role,
		workflow.Level3Role,
		workflow.Level4Role,
	}

	var levels []workflowModels.ApprovalLevel
	maxLevel := 0

	for i, role := range roles {
		level := i + 1

		status := "pending"
		if role == "" {
			status = "skipped"
		} else {
			maxLevel = level
		}

		levels = append(levels, workflowModels.ApprovalLevel{
			Note:       "",
			Role:       role,
			Level:      level,
			Status:     status,
			ApprovedAt: "",
			ApprovedBy: "",
		})
	}

	return workflowModels.ApprovalProgress{Levels: levels}, maxLevel
}
