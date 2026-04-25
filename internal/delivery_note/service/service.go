package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	workflowModels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	approvalRepo "github.com/ganasa18/go-template/internal/approval_workflow/repository"
	"github.com/ganasa18/go-template/internal/delivery_note/models"
	deliveryNoteRepo "github.com/ganasa18/go-template/internal/delivery_note/repository"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IDeliveryNoteService interface {
	Create(ctx context.Context, req models.CreateDNRequest) (*models.DeliveryNote, error)
	GetAll(ctx context.Context, page, limit int) ([]models.DeliveryNote, models.Pagination, error)
	GetByID(ctx context.Context, id int64) (*models.DeliveryNote, error)
	Scan(ctx context.Context, req models.QRPayload) (string, error)
	PreviewDN(ctx context.Context, req models.CreateDNRequest) (*models.PreviewDNResponse, error)
	PreviewItem(ctx context.Context, req models.PreviewDNItem) (*models.PreviewDNItemRespons, error)
	ScanDelivery(ctx context.Context, req models.ScanDeliveryRequest) error
	SubmitDelivery(ctx context.Context, req models.SubmitDeliveryRequest) error
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
			Period:          req.Period,
			Type:            req.Type,
			Status:          "active",
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
				return fmt.Errorf("Uniq tersebut belum terdaftar pada kanban.")
			}

			date, err := time.Parse("02/01/2006", it.IncomingDate)
			if err != nil {
				return err
			}

			qtyStart, err := s.repo.CheckItemExistsInDN(ctx, it.ItemUniqCode)
			if err != nil {
				return err
			}

			packing := fmt.Sprintf("DN-%04d-PKG-%04d", dn.ID, i+1)

			items = append(items, models.DeliveryNoteItem{
				DNID:          dn.ID,
				ItemUniqCode:  it.ItemUniqCode,
				Quantity:      it.Qty,
				OrderQty:      int64(poItem.OrderedQty),
				QtyStated:     qtyStart,
				UOM:           poItem.UOM,
				Weight:        int64(poItem.WeightKg),
				KanbanID:      kanban.ID,
				PcsPerKanban:  poItem.PcsPerKanban,
				PackingNumber: packing,
				DateIncoming:  &date,
				Check:         "progress",
				QtySent:       0,
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

			payload := models.QRPayload{
				Packing: item.PackingNumber,
			}

			qrBytes, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			qrValue := string(qrBytes)

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

func (s *deliveryNoteService) GetAll(ctx context.Context, page, limit int) ([]models.DeliveryNote, models.Pagination, error) {
	var data []models.DeliveryNote
	var total int64

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// total data
	if err := s.db.WithContext(ctx).
		Model(&models.DeliveryNote{}).
		Count(&total).Error; err != nil {
		return nil, models.Pagination{}, err
	}

	// ambil data
	err := s.db.WithContext(ctx).
		Preload("Supplier").
		Preload("Items").
		Preload("Items.Kanban").
		Limit(limit).
		Offset(offset).
		Order("id DESC").
		Find(&data).Error

	if err != nil {
		return nil, models.Pagination{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	pagination := models.Pagination{
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	return data, pagination, nil
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

func (s *deliveryNoteService) Scan(ctx context.Context, req models.QRPayload) (string, error) {

	var result string

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if req.Quantity <= 0 {
			return fmt.Errorf("qty harus lebih dari 0")
		}

		var item models.DeliveryNoteItem

		// 🔒 lock row
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("packing_number = ?", req.Packing).
			First(&item).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("item tidak ditemukan")
			}
			return err
		}

		var dn models.DeliveryNote
		if err := tx.Where("id = ?", item.DNID).
			First(&dn).Error; err != nil {
			return err
		}

		if dn.Status != "active" {
			return fmt.Errorf("DN tidak aktif")
		}

		qty := int64(req.Quantity)
		now := time.Now()

		var scanType, fromLoc, toLoc, status string

		// ======================================================
		// 🟢 RM / IRM
		// ======================================================
		if dn.Type == "RM" || dn.Type == "IRM" {

			remaining := item.Quantity - item.QtyReceived

			if remaining <= 0 {
				return fmt.Errorf("item sudah selesai, tidak bisa scan lagi")
			}

			if qty > remaining {
				return fmt.Errorf("qty melebihi sisa. sisa saat ini: %d", remaining)
			}

			newQty := item.QtyReceived + qty

			// 🔥 status logic
			if item.QtyReceived == 0 {
				status = "incoming"
			} else if newQty < item.Quantity {
				status = "remaining"
			} else {
				status = "completed"
			}

			if err := tx.Model(&models.DeliveryNoteItem{}).
				Where("id = ?", item.ID).
				Updates(map[string]interface{}{
					"qty_received": gorm.Expr("COALESCE(qty_received,0) + ?", qty),
					"check":        status,
				}).Error; err != nil {
				return err
			}

			scanType = "incoming"
			fromLoc = "supplier"
			toLoc = "warehouse"
			result = status
		}

		// ======================================================
		// 🔴 SUBCON
		// ======================================================
		if dn.Type == "SUBCON" {

			// ===============================
			// 🔴 OUTGOING
			// ===============================
			if item.QtySent < item.Quantity {

				remaining := item.Quantity - item.QtySent

				if remaining <= 0 {
					return fmt.Errorf("item sudah selesai, tidak bisa scan lagi")
				}

				if qty > remaining {
					return fmt.Errorf("qty melebihi sisa kirim. sisa saat ini: %d", remaining)
				}

				newQty := item.QtySent + qty

				if item.QtySent == 0 {
					status = "outgoing"
				} else if newQty < item.Quantity {
					status = "remaining"
				} else {
					status = "remaining" // belum selesai, masih nunggu return
				}

				if err := tx.Model(&models.DeliveryNoteItem{}).
					Where("id = ?", item.ID).
					Updates(map[string]interface{}{
						"qty_sent": gorm.Expr("COALESCE(qty_sent,0) + ?", qty),
						"check":    status,
					}).Error; err != nil {
					return err
				}

				scanType = "outgoing"
				fromLoc = "warehouse"
				toLoc = "vendor"
				result = status

			} else {

				// ===============================
				// 🟢 INCOMING (RETURN)
				// ===============================
				remaining := item.QtySent - item.QtyReceived

				if qty > remaining {
					return fmt.Errorf("qty melebihi sisa return")
				}

				newQty := item.QtyReceived + qty

				if item.QtyReceived == 0 {
					status = "incoming"
				} else if newQty < item.QtySent {
					status = "remaining"
				} else {
					status = "completed"
				}

				if err := tx.Model(&models.DeliveryNoteItem{}).
					Where("id = ?", item.ID).
					Updates(map[string]interface{}{
						"qty_received": gorm.Expr("COALESCE(qty_received,0) + ?", qty),
						"check":        status,
					}).Error; err != nil {
					return err
				}

				scanType = "incoming"
				fromLoc = "vendor"
				toLoc = "warehouse"
				result = status
			}
		}

		// ======================================================
		// 🧾 LOG
		// ======================================================
		log := models.DeliveryNoteLog{
			DNID:          item.DNID,
			DNItemID:      item.ID,
			ItemUniqCode:  item.ItemUniqCode,
			PackingNumber: req.Packing,
			ScanType:      scanType,
			Qty:           float64(qty),
			FromLocation:  fromLoc,
			ToLocation:    toLoc,
			CreatedAt:     now,
		}

		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return result, nil
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

func (s *deliveryNoteService) PreviewItem(ctx context.Context, req models.PreviewDNItem) (*models.PreviewDNItemRespons, error) {

	// 🔥 1. ambil item dari PO
	poItem, err := s.repo.GetPOItemByPackingNumber(ctx, req.Packing)
	if err != nil {
		return nil, fmt.Errorf("item tidak ditemukan")
	}

	dn, err := s.repo.GetDNByID(ctx, poItem.DNID)
	if err != nil {
		return nil, fmt.Errorf("dn tidak ditemukan")
	}

	supplier, err := s.repo.GetSupplierByID(ctx, dn.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("supplier tidak ditemukan")
	}
	// 🔥 3. build response
	res := &models.PreviewDNItemRespons{
		DNNumber:      dn.DNNumber,
		PackingNumber: poItem.PackingNumber,
		PONumber:      dn.PONumber,
		Supplier:      supplier.SupplierName,
		ItemUniqCode:  poItem.ItemUniqCode,
		MaterialInfo:  poItem.ItemUniqCode,
		Weight:        poItem.Weight,
		TotalQty:      int64(poItem.Quantity),
		RemainingQty:  int64(poItem.QtyReceived),
		UOM:           poItem.UOM,
		OrderQty:      int64(poItem.QtyReceived),
		PcsPerKanban:  poItem.PcsPerKanban,
	}
	return res, nil
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

func (s *deliveryNoteService) ScanDelivery(ctx context.Context, req models.ScanDeliveryRequest) error {

	if req.KanbanNumber == "" {
		return errors.New("kanban wajib diisi")
	}

	if req.DNNumber == "" {
		return errors.New("DN wajib diisi")
	}

	if req.Qty <= 0 {
		return errors.New("qty harus > 0")
	}

	return s.repo.WithTx(ctx, func(tx *gorm.DB) error {

		poItem, err := s.repo.GetPOItemByPackingNumber(ctx, req.KanbanNumber)
		if err != nil {
			return errors.New("item tidak ditemukan")
		}
		// =============================
		// 🔍 GET STOCK (LOCK)
		// =============================
		fg, err := s.repo.GetFinishedGoodsForUpdate(ctx, tx, poItem.ItemUniqCode)
		if err != nil {
			return errors.New("kanban tidak ditemukan")
		}

		if fg.StockQty <= 0 {
			return errors.New("stock kosong")
		}

		if fg.StockQty < req.Qty {
			return errors.New("stock tidak cukup")
		}

		// =============================
		// 🔎 CEK DN SUDAH ADA?
		// =============================
		dn, err := s.repo.FindDNByNumber(ctx, tx, req.DNNumber)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// =============================
		// 🆕 CREATE HEADER (kalau belum ada)
		// =============================
		if dn == nil || dn.ID == 0 {
			now := time.Now()

			newDN := models.DeliveryNoteSupplier{
				DNNumber:     req.DNNumber,
				KanbanNumber: req.KanbanNumber,
				Status:       "active",
				ScannedBy:    req.ScannedBy,
				ScannedAt:    &now,
				TotalQty:     req.Qty,
			}

			if err := s.repo.CreateDN(ctx, tx, &newDN); err != nil {
				return err
			}

			dn = &newDN
		} else {
			// =============================
			// ➕ UPDATE TOTAL QTY
			// =============================
			if err := s.repo.AddDNQty(ctx, tx, dn.ID, req.Qty); err != nil {
				return err
			}
		}

		// =============================
		// 📝 INSERT ITEM
		// =============================
		item := models.DeliveryNoteSupplierItem{
			DNID:         dn.ID,
			KanbanNumber: req.KanbanNumber,
			Qty:          req.Qty,
		}

		if err := s.repo.InsertDNItem(ctx, tx, &item); err != nil {
			return err
		}

		// =============================
		// 📉 REDUCE STOCK
		// =============================
		if err := s.repo.ReduceStockTx(ctx, tx, fg.ID, req.Qty); err != nil {
			return err
		}

		return nil
	})
}

func (s *deliveryNoteService) SubmitDelivery(ctx context.Context, req models.SubmitDeliveryRequest) error {

	if len(req.Items) == 0 {
		return errors.New("items tidak boleh kosong")
	}

	parsedDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return errors.New("format date salah (YYYY-MM-DD)")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		priority := req.Priority

		if priority == "" {
			priority = "normal" // default aman
		}
		// =============================
		// 🧾 CREATE HEADER
		// =============================
		header := models.DeliveryScheduleCustomer{
			UUID:           uuid.NewString(),
			ScheduleNumber: generateScheduleNumber(), // nanti helper
			CustomerID:     req.CustomerID,
			ScheduleDate:   parsedDate,
			Cycle:          req.Cycle,
			Priority:       priority,
			Status:         "scheduled",
			ApprovalStatus: "pending",
			CreatedBy:      req.CreatedBy,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := s.repo.InsertHeaderTx(tx, &header); err != nil {
			return err
		}

		// =============================
		// 🔁 LOOP ITEMS
		// =============================
		for i, item := range req.Items {

			// 🔒 LOCK FG
			fg, err := s.repo.GetFGForUpdate(tx, item.ItemUniqCode)
			if err != nil {
				return errors.New("finished goods tidak ditemukan")
			}

			// =============================
			// ❗ VALIDASI STOCK
			// =============================
			if fg.StockQty < item.Qty {
				return errors.New("stock tidak cukup untuk item " + item.ItemUniqCode)
			}

			// =============================
			// 📦 INSERT ITEM
			// =============================
			detail := models.DeliveryScheduleItemCustomer{
				UUID:              uuid.NewString(),
				ScheduleID:        header.ID,
				LineNo:            i + 1,
				ItemUniqCode:      item.ItemUniqCode,
				PartName:          fg.PartName,
				PartNumber:        fg.PartNumber,
				TotalOrderQty:     fg.StockQty, // optional
				TotalDeliveryQty:  item.Qty,
				UOM:               item.UOM,
				Status:            "scheduled",
				FGReadinessStatus: "ready",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}

			if err := s.repo.InsertItemTx(tx, &detail); err != nil {
				return err
			}

			// =============================
			// 📉 REDUCE STOCK
			// =============================
			if err := s.repo.ReduceFGStock(tx, fg.ID, item.Qty); err != nil {
				return err
			}
		}

		return nil
	})
}

func generateScheduleNumber() string {
	return "SCH-" + time.Now().Format("20060102150405")
}
