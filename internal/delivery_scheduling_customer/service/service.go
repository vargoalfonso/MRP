package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/models"
	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/repository"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// ─── Interface ────────────────────────────────────────────────────────────────

type IService interface {
	// Schedule
	CreateSchedule(ctx context.Context, req models.CreateScheduleRequest, createdBy string) (*models.CreateScheduleResponse, error)
	GetSchedulesSummary(ctx context.Context, deliveryDate string) (*models.ScheduleSummaryResponse, error)
	GetSchedulesList(ctx context.Context, f models.ScheduleListFilter) (*models.ScheduleListResponse, error)
	GetScheduleDetail(ctx context.Context, scheduleUUID string) (*models.ScheduleDetailResponse, error)

	// Approve
	ApproveSchedule(ctx context.Context, scheduleUUID string, req models.ApproveScheduleRequest, approvedBy string) (*models.ApproveScheduleResponse, error)
	ApproveAll(ctx context.Context, req models.ApproveAllRequest, approvedBy string) (*models.ApproveMultiResponse, error)
	ApprovePartial(ctx context.Context, req models.ApprovePartialRequest, approvedBy string) (*models.ApproveMultiResponse, error)

	// Customer DN (manual / exception flow)
	CreateCustomerDN(ctx context.Context, req models.CreateCustomerDNRequest, createdBy string) (*models.CreateDNResponse, error)
	GetDNList(ctx context.Context, f models.DNListFilter) (*models.DNListResponse, error)
	GetDNDetail(ctx context.Context, dnUUID string) (*models.DNDetailResponse, error)
	ConfirmDN(ctx context.Context, dnUUID string, req models.ConfirmDNRequest) (*models.ConfirmDNResponse, error)

	// Delivery scan
	LookupDeliveryItem(ctx context.Context, dnNumber, itemUniqCode string) (*models.DeliveryLookupResponse, error)
	SubmitDeliveryScan(ctx context.Context, req models.SubmitScanRequest, scannedBy string) (*models.SubmitScanResponse, error)
}

// ─── Implementation ───────────────────────────────────────────────────────────

type service struct {
	repo repository.IRepository
	db   *gorm.DB
}

func New(repo repository.IRepository, db *gorm.DB) IService {
	return &service{repo: repo, db: db}
}

// ─── Schedule ─────────────────────────────────────────────────────────────────

func (s *service) CreateSchedule(ctx context.Context, req models.CreateScheduleRequest, createdBy string) (*models.CreateScheduleResponse, error) {
	var scheduleNumber string

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		year := time.Now().Year()
		prefix := fmt.Sprintf("DS-%d", year)
		last, _ := s.repo.FindLastScheduleNumber(ctx, tx, prefix)
		scheduleNumber = nextNumber(last, prefix)

		deliveryDate, err := parseDate(req.DeliveryDate)
		if err != nil {
			return fmt.Errorf("delivery_date tidak valid: %w", err)
		}

		priority := req.Priority
		if priority == "" {
			priority = "normal"
		}

		// Resolve customer_order_document UUID → integer ID for FK storage
		var coDocID *int64
		if req.CustomerOrderDocumentUUID != "" {
			var resolvedID int64
			if err := s.db.WithContext(ctx).
				Table("customer_order_documents").
				Select("id").
				Where("uuid = ? AND deleted_at IS NULL", req.CustomerOrderDocumentUUID).
				Scan(&resolvedID).Error; err == nil && resolvedID > 0 {
				coDocID = &resolvedID
			}
		}

		sc := &models.ScheduleCustomer{
			UUID:                    uuid.New().String(),
			ScheduleNumber:          scheduleNumber,
			CustomerOrderDocumentID: coDocID,
			CustomerID:              req.CustomerID,
			CustomerNameSnapshot:    strPtr(req.CustomerName),
			ScheduleDate:            deliveryDate,
			Priority:                priority,
			Cycle:                   strPtrIfNotEmpty(req.Cycle),
			TransportCompany:        strPtrIfNotEmpty(req.TransportCompany),
			VehicleNumber:           strPtrIfNotEmpty(req.VehicleNumber),
			DriverName:              strPtrIfNotEmpty(req.DriverName),
			DriverContact:           strPtrIfNotEmpty(req.DriverContact),
			DeliveryInstructions:    strPtrIfNotEmpty(req.DeliveryInstructions),
			Remarks:                 strPtrIfNotEmpty(req.Remarks),
			Status:                  "scheduled",
			ApprovalStatus:          "pending",
			CreatedBy:               strPtrIfNotEmpty(createdBy),
		}

		if req.CustomerOrderReference != "" {
			ref := req.CustomerOrderReference
			sc.CustomerOrderReference = &ref
		}

		if err := parseTimes(req.DepartureAt, req.ArrivalAt, sc); err != nil {
			return err
		}

		if err := s.repo.CreateSchedule(ctx, tx, sc); err != nil {
			return fmt.Errorf("gagal membuat schedule: %w", err)
		}

		items := make([]models.ScheduleItemCustomer, 0, len(req.Items))
		for i, it := range req.Items {
			// Resolve item UUID → integer ID + auto-fill snapshot fields
			var coItemID *int64
			if it.CustomerOrderDocumentItemUUID != "" {
				var snap struct {
					ID         int64   `gorm:"column:id"`
					PartName   string  `gorm:"column:part_name"`
					PartNumber string  `gorm:"column:part_number"`
					Model      *string `gorm:"column:model"`
					UOM        string  `gorm:"column:uom"`
					Quantity   float64 `gorm:"column:quantity"`
				}
				if err := s.db.WithContext(ctx).
					Table("customer_order_document_items").
					Select("id, part_name, part_number, model, uom, quantity").
					Where("uuid = ?", it.CustomerOrderDocumentItemUUID).
					Scan(&snap).Error; err == nil && snap.ID > 0 {
					coItemID = &snap.ID
					if it.PartName == "" {
						it.PartName = snap.PartName
					}
					if it.PartNo == "" {
						it.PartNo = snap.PartNumber
					}
					if it.Model == "" && snap.Model != nil {
						it.Model = *snap.Model
					}
					if it.UOM == "" {
						it.UOM = snap.UOM
					}
					if it.TotalOrder == 0 {
						it.TotalOrder = snap.Quantity
					}
				}
			}

			items = append(items, models.ScheduleItemCustomer{
				UUID:                        uuid.New().String(),
				ScheduleID:                  sc.ID,
				CustomerOrderDocumentItemID: coItemID,
				LineNo:                      i + 1,
				ItemUniqCode:                it.ItemUniqCode,
				Model:                       strPtrIfNotEmpty(it.Model),
				PartName:                    it.PartName,
				PartNumber:                  it.PartNo,
				TotalOrderQty:               it.TotalOrder,
				TotalDeliveryQty:            it.TotalDelivery,
				UOM:                         it.UOM,
				Cycle:                       strPtrIfNotEmpty(req.Cycle),
				Status:                      "scheduled",
				FGReadinessStatus:           "unknown",
			})
		}

		return s.repo.CreateScheduleItems(ctx, tx, items)
	})
	if err != nil {
		return nil, err
	}

	return &models.CreateScheduleResponse{
		ScheduleID: scheduleNumber,
		Status:     "scheduled",
	}, nil
}

func (s *service) GetSchedulesSummary(ctx context.Context, deliveryDate string) (*models.ScheduleSummaryResponse, error) {
	counts, err := s.repo.GetSchedulesSummary(ctx, deliveryDate)
	if err != nil {
		return nil, err
	}

	inTransit := counts["shipped"] + counts["partially_approved"]
	pending := counts["scheduled"]
	dnCreated := counts["dn_created"] + counts["approved"]
	total := 0
	for _, v := range counts {
		total += v
	}

	return &models.ScheduleSummaryResponse{
		TotalDeliveries: total,
		InTransit:       inTransit,
		PendingApproval: pending,
		DNCreated:       dnCreated,
	}, nil
}

func (s *service) GetSchedulesList(ctx context.Context, f models.ScheduleListFilter) (*models.ScheduleListResponse, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}

	schedules, total, err := s.repo.GetSchedulesList(ctx, f)
	if err != nil {
		return nil, err
	}

	// Group by date
	groupMap := map[string]*models.ScheduleGroup{}
	groupOrder := []string{}

	for _, sc := range schedules {
		dateKey := sc.ScheduleDate.Format("2006-01-02")
		if _, ok := groupMap[dateKey]; !ok {
			groupMap[dateKey] = &models.ScheduleGroup{
				DeliveryDate: dateKey,
				Actions: models.GroupActions{
					ApproveAllEnabled:     true,
					ApprovePartialEnabled: true,
				},
				Items: []models.ScheduleRow{},
			}
			groupOrder = append(groupOrder, dateKey)
		}

		for _, item := range sc.Items {
			row := models.ScheduleRow{
				ScheduleID:     sc.ScheduleNumber,
				CustomerName:   derefStr(sc.CustomerNameSnapshot),
				PODNName:       derefStr(sc.CustomerOrderReference),
				ItemUniqCode:   item.ItemUniqCode,
				Model:          derefStr(item.Model),
				PartNo:         item.PartNumber,
				PartName:       item.PartName,
				Quantity:       item.TotalDeliveryQty,
				Cycle:          derefStr(item.Cycle),
				DNNumber:       derefStr(item.DNNumber),
				Status:         sc.Status,
				ApprovalStatus: sc.ApprovalStatus,
			}
			groupMap[dateKey].Items = append(groupMap[dateKey].Items, row)
			groupMap[dateKey].ItemCount++
		}
	}

	groups := make([]models.ScheduleGroup, 0, len(groupOrder))
	for _, k := range groupOrder {
		groups = append(groups, *groupMap[k])
	}

	return &models.ScheduleListResponse{
		Groups:     groups,
		Pagination: repository.BuildPagination(total, f.Page, f.Limit),
	}, nil
}

func (s *service) GetScheduleDetail(ctx context.Context, scheduleID string) (*models.ScheduleDetailResponse, error) {
	sc, err := s.repo.GetScheduleByNumber(ctx, scheduleID)
	if err != nil {
		return nil, errors.New("schedule tidak ditemukan")
	}

	items, err := s.repo.GetScheduleItemsByScheduleID(ctx, sc.ID)
	if err != nil {
		return nil, err
	}

	detailItems := make([]models.ScheduleDetailItem, 0, len(items))
	totalQty := 0.0

	for _, it := range items {
		fgQty, _ := s.repo.GetFGStockQty(ctx, it.ItemUniqCode)
		remaining := 0.0
		if it.TotalDeliveryQty > fgQty {
			remaining = it.TotalDeliveryQty - fgQty
		}

		readiness := "ready"
		if fgQty == 0 {
			readiness = "shortage"
		} else if fgQty < it.TotalDeliveryQty {
			readiness = "partial_ready"
		}

		detailItems = append(detailItems, models.ScheduleDetailItem{
			ItemUniqCode:       it.ItemUniqCode,
			PartName:           it.PartName,
			DNNumber:           derefStr(it.DNNumber),
			Quantity:           it.TotalDeliveryQty,
			UOM:                it.UOM,
			FGAvailable:        fgQty,
			RemainingToPrepare: remaining,
			Readiness:          readiness,
		})
		totalQty += it.TotalDeliveryQty
	}

	dateStr := sc.ScheduleDate.Format("2006-01-02")

	return &models.ScheduleDetailResponse{
		ScheduleID:            sc.ScheduleNumber,
		ScheduleDate:          dateStr,
		DeliveryDate:          dateStr,
		CustomerID:            sc.CustomerID,
		CustomerName:          derefStr(sc.CustomerNameSnapshot),
		PONumber:              derefStr(sc.CustomerOrderReference),
		CustomerContactPerson: derefStr(sc.CustomerContactPerson),
		CustomerPhoneNumber:   derefStr(sc.CustomerPhoneNumber),
		DeliveryAddress:       derefStr(sc.DeliveryAddress),
		TotalItems:            len(detailItems),
		TotalQuantity:         totalQty,
		Priority:              sc.Priority,
		Status:                sc.Status,
		ApprovalStatus:        sc.ApprovalStatus,
		CreatedBy:             derefStr(sc.CreatedBy),
		TransportCompany:      derefStr(sc.TransportCompany),
		VehicleNumber:         derefStr(sc.VehicleNumber),
		DriverName:            derefStr(sc.DriverName),
		DriverContact:         derefStr(sc.DriverContact),
		DepartureAt:           sc.DepartureAt,
		ArrivalAt:             sc.ArrivalAt,
		DeliveryInstructions:  derefStr(sc.DeliveryInstructions),
		Items:                 detailItems,
	}, nil
}

// ─── Approve ──────────────────────────────────────────────────────────────────

func (s *service) ApproveSchedule(ctx context.Context, scheduleID string, req models.ApproveScheduleRequest, approvedBy string) (*models.ApproveScheduleResponse, error) {
	sc, err := s.repo.GetScheduleByNumber(ctx, scheduleID)
	if err != nil {
		return nil, errors.New("schedule tidak ditemukan")
	}
	if sc.Status == "dn_created" || sc.Status == "cancelled" {
		return nil, fmt.Errorf("schedule sudah dalam status %s", sc.Status)
	}

	items, err := s.repo.GetScheduleItemsByScheduleID(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("schedule tidak punya items")
	}

	dn, err := s.buildAndCreateDN(ctx, sc, items, req.ForcePartial, approvedBy)
	if err != nil {
		return nil, err
	}

	return &models.ApproveScheduleResponse{
		ScheduleID: sc.ScheduleNumber,
		DNID:       dn.UUID,
		DNNumber:   dn.DNNumber,
		Status:     "dn_created",
	}, nil
}

func (s *service) ApproveAll(ctx context.Context, req models.ApproveAllRequest, approvedBy string) (*models.ApproveMultiResponse, error) {
	schedules, err := s.repo.GetSchedulesByDateAndCustomer(ctx, req.DeliveryDate, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, errors.New("tidak ada schedule yang bisa diapprove pada tanggal ini")
	}

	approved := 0
	dnCreated := 0
	for _, sc := range schedules {
		items, _ := s.repo.GetScheduleItemsByScheduleID(ctx, sc.ID)
		if len(items) == 0 {
			continue
		}
		if _, err := s.buildAndCreateDN(ctx, &sc, items, req.ForcePartial, approvedBy); err != nil {
			continue // skip invalid, best-effort
		}
		approved++
		dnCreated++
	}

	return &models.ApproveMultiResponse{
		ApprovedCount:  approved,
		DNCreatedCount: dnCreated,
		DeliveryDate:   req.DeliveryDate,
	}, nil
}

func (s *service) ApprovePartial(ctx context.Context, req models.ApprovePartialRequest, approvedBy string) (*models.ApproveMultiResponse, error) {
	schedules, err := s.repo.GetSchedulesByUUIDs(ctx, req.ScheduleIDs)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 0 {
		return nil, errors.New("tidak ada schedule yang ditemukan")
	}

	approved := 0
	dnCreated := 0
	for _, sc := range schedules {
		items, _ := s.repo.GetScheduleItemsByScheduleID(ctx, sc.ID)
		if len(items) == 0 {
			continue
		}
		if _, err := s.buildAndCreateDN(ctx, &sc, items, req.ForcePartial, approvedBy); err != nil {
			continue
		}
		approved++
		dnCreated++
	}

	return &models.ApproveMultiResponse{
		ApprovedCount:  approved,
		DNCreatedCount: dnCreated,
		DeliveryDate:   req.DeliveryDate,
	}, nil
}

// buildAndCreateDN is the shared atomic transaction for approving one schedule → DN.
func (s *service) buildAndCreateDN(ctx context.Context, sc *models.ScheduleCustomer, items []models.ScheduleItemCustomer, forcePartial bool, approvedBy string) (*models.DNCustomer, error) {
	var dn models.DNCustomer

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// FG readiness check
		for _, it := range items {
			fgQty, _ := s.repo.GetFGStockQty(ctx, it.ItemUniqCode)
			if fgQty < it.TotalDeliveryQty && !forcePartial {
				return fmt.Errorf("FG tidak cukup untuk item %s (tersedia: %.4f, dibutuhkan: %.4f)",
					it.ItemUniqCode, fgQty, it.TotalDeliveryQty)
			}
		}

		// Generate DN number
		year := time.Now().Year()
		prefix := fmt.Sprintf("DN-CUST-%d", year)
		last, _ := s.repo.FindLastDNNumber(ctx, tx, prefix)
		dnNumber := nextNumber(last, prefix)

		now := time.Now()
		dn = models.DNCustomer{
			UUID:                  uuid.New().String(),
			DNNumber:              dnNumber,
			ScheduleID:            &sc.ID,
			CustomerID:            sc.CustomerID,
			CustomerNameSnapshot:  sc.CustomerNameSnapshot,
			CustomerContactPerson: sc.CustomerContactPerson,
			CustomerPhoneNumber:   sc.CustomerPhoneNumber,
			DeliveryAddress:       sc.DeliveryAddress,
			DeliveryDate:          sc.ScheduleDate,
			Priority:              sc.Priority,
			TransportCompany:      sc.TransportCompany,
			VehicleNumber:         sc.VehicleNumber,
			DriverName:            sc.DriverName,
			DriverContact:         sc.DriverContact,
			DepartureAt:           sc.DepartureAt,
			ArrivalAt:             sc.ArrivalAt,
			Status:                "created",
			ApprovalStatus:        "approved",
			DeliveryInstructions:  sc.DeliveryInstructions,
			Remarks:               sc.Remarks,
			CreatedBy:             strPtrIfNotEmpty(approvedBy),
			ApprovedBy:            strPtrIfNotEmpty(approvedBy),
			ApprovedAt:            &now,
		}

		if sc.CustomerOrderReference != nil {
			dn.CustomerOrderReference = sc.CustomerOrderReference
		}

		if err := s.repo.CreateDN(ctx, tx, &dn); err != nil {
			return fmt.Errorf("gagal membuat DN: %w", err)
		}

		// Create DN items
		dnItems := make([]models.DNItemCustomer, 0, len(items))
		for i, it := range items {
			packingNumber := fmt.Sprintf("%s-PKG-%04d", dnNumber, i+1)
			dnItems = append(dnItems, models.DNItemCustomer{
				UUID:           uuid.New().String(),
				DNID:           dn.ID,
				ScheduleItemID: &it.ID,
				LineNo:         i + 1,
				ItemUniqCode:   it.ItemUniqCode,
				Model:          it.Model,
				PartName:       it.PartName,
				PartNumber:     it.PartNumber,
				Quantity:       it.TotalDeliveryQty,
				UOM:            it.UOM,
				PackingNumber:  packingNumber,
				ShipmentStatus: "created",
			})
		}

		if err := s.repo.CreateDNItems(ctx, tx, dnItems); err != nil {
			return fmt.Errorf("gagal membuat DN items: %w", err)
		}

		// Generate QR per item
		for _, item := range dnItems {
			qrJSON, _ := json.Marshal(map[string]interface{}{
				"dn_number":      dnNumber,
				"item_uniq_code": item.ItemUniqCode,
				"quantity":       item.Quantity,
				"packing_number": item.PackingNumber,
			})
			qr, err := generateQRBase64(string(qrJSON))
			if err != nil {
				return fmt.Errorf("gagal generate QR: %w", err)
			}
			if err := s.repo.UpdateDNItemQR(ctx, tx, item.ID, qr); err != nil {
				return err
			}
		}

		// Update schedule status
		if err := s.repo.UpdateScheduleStatus(ctx, tx, sc.ID, "dn_created", "approved"); err != nil {
			return err
		}

		// Link DN number to each schedule item
		for _, it := range items {
			_ = s.repo.UpdateScheduleItemDNNumber(ctx, tx, it.ID, dnNumber, "dn_created")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &dn, nil
}

// ─── Customer DN (manual) ─────────────────────────────────────────────────────

func (s *service) CreateCustomerDN(ctx context.Context, req models.CreateCustomerDNRequest, createdBy string) (*models.CreateDNResponse, error) {
	var dn models.DNCustomer
	var dnItems []models.DNItemCustomer

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		year := time.Now().Year()
		prefix := fmt.Sprintf("DN-CUST-%d", year)
		last, _ := s.repo.FindLastDNNumber(ctx, tx, prefix)
		dnNumber := nextNumber(last, prefix)

		deliveryDate, err := parseDate(req.DeliveryDate)
		if err != nil {
			return fmt.Errorf("delivery_date tidak valid: %w", err)
		}

		priority := req.Priority
		if priority == "" {
			priority = "normal"
		}
		status := req.Status
		if status == "" {
			status = "created"
		}
		approvalStatus := req.ApprovalStatus
		if approvalStatus == "" {
			approvalStatus = "pending"
		}

		dn = models.DNCustomer{
			UUID:                  uuid.New().String(),
			DNNumber:              dnNumber,
			CustomerID:            req.CustomerID,
			CustomerNameSnapshot:  strPtr(req.CustomerName),
			CustomerContactPerson: strPtrIfNotEmpty(req.CustomerContactPerson),
			CustomerPhoneNumber:   strPtrIfNotEmpty(req.CustomerPhoneNumber),
			DeliveryAddress:       strPtrIfNotEmpty(req.DeliveryAddress),
			DeliveryDate:          deliveryDate,
			Priority:              priority,
			TransportCompany:      strPtrIfNotEmpty(req.TransportCompany),
			VehicleNumber:         strPtrIfNotEmpty(req.VehicleNumber),
			DriverName:            strPtrIfNotEmpty(req.DriverName),
			DriverContact:         strPtrIfNotEmpty(req.DriverContact),
			Status:                status,
			ApprovalStatus:        approvalStatus,
			DeliveryInstructions:  strPtrIfNotEmpty(req.DeliveryInstructions),
			Remarks:               strPtrIfNotEmpty(req.Remarks),
			CreatedBy:             strPtrIfNotEmpty(createdBy),
		}

		if req.PONumber != "" {
			dn.CustomerOrderReference = &req.PONumber
		}
		if req.ScheduleDate != "" {
			// reference only, schedule_date stored in ScheduleID link
		}

		if err := parseTimes(req.DepartureAt, req.ArrivalAt, &dn); err != nil {
			return err
		}

		if err := s.repo.CreateDN(ctx, tx, &dn); err != nil {
			return fmt.Errorf("gagal membuat DN: %w", err)
		}

		rawItems := make([]models.DNItemCustomer, 0, len(req.Items))
		for i, it := range req.Items {
			packingNumber := fmt.Sprintf("%s-PKG-%04d", dnNumber, i+1)
			rawItems = append(rawItems, models.DNItemCustomer{
				UUID:           uuid.New().String(),
				DNID:           dn.ID,
				LineNo:         i + 1,
				ItemUniqCode:   it.ItemUniqCode,
				Model:          strPtrIfNotEmpty(it.Model),
				PartName:       it.ProductName,
				PartNumber:     it.PartNumber,
				FGLocation:     strPtrIfNotEmpty(it.FGLocation),
				Quantity:       it.Quantity,
				UOM:            it.UOM,
				PackingNumber:  packingNumber,
				ShipmentStatus: "created",
			})
		}

		if err := s.repo.CreateDNItems(ctx, tx, rawItems); err != nil {
			return fmt.Errorf("gagal membuat DN items: %w", err)
		}

		// Generate QR per item
		for _, item := range rawItems {
			qrJSON, _ := json.Marshal(map[string]interface{}{
				"dn_number":      dnNumber,
				"item_uniq_code": item.ItemUniqCode,
				"quantity":       item.Quantity,
				"packing_number": item.PackingNumber,
			})
			qr, err := generateQRBase64(string(qrJSON))
			if err != nil {
				return fmt.Errorf("gagal generate QR: %w", err)
			}
			if err := s.repo.UpdateDNItemQR(ctx, tx, item.ID, qr); err != nil {
				return err
			}
			item.QR = &qr
			dnItems = append(dnItems, item)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	totalQty := 0.0
	respItems := make([]models.CreateDNItemResp, 0, len(dnItems))
	for _, item := range dnItems {
		totalQty += item.Quantity
		respItems = append(respItems, models.CreateDNItemResp{
			DNItemID:      item.UUID,
			DNNumber:      dn.DNNumber,
			ItemUniqCode:  item.ItemUniqCode,
			PartNumber:    item.PartNumber,
			Model:         derefStr(item.Model),
			Quantity:      item.Quantity,
			FGLocation:    derefStr(item.FGLocation),
			PackingNumber: item.PackingNumber,
			QR:            derefStr(item.QR),
		})
	}

	scheduleIDStr := ""
	if req.ScheduleID != "" {
		scheduleIDStr = req.ScheduleID
	}

	return &models.CreateDNResponse{
		DNID:           dn.UUID,
		ScheduleID:     scheduleIDStr,
		DNNumber:       dn.DNNumber,
		TotalItems:     len(respItems),
		TotalQuantity:  totalQty,
		Status:         dn.Status,
		ApprovalStatus: dn.ApprovalStatus,
		CreatedBy:      createdBy,
		PrintedCount:   0,
		Items:          respItems,
	}, nil
}

func (s *service) GetDNList(ctx context.Context, f models.DNListFilter) (*models.DNListResponse, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}

	dns, total, err := s.repo.GetDNList(ctx, f)
	if err != nil {
		return nil, err
	}

	// Build summary from current day
	summary, _ := s.GetSchedulesSummary(ctx, f.DeliveryDate)

	rows := make([]models.DNListRow, 0)
	for _, dn := range dns {
		for _, item := range dn.Items {
			rows = append(rows, models.DNListRow{
				DNID:              dn.UUID,
				DNNumber:          dn.DNNumber,
				DeliveryDate:      dn.DeliveryDate.Format("2006-01-02"),
				CustomerName:      derefStr(dn.CustomerNameSnapshot),
				PONumber:          derefStr(dn.CustomerOrderReference),
				PartName:          item.PartName,
				ItemUniqCode:      item.ItemUniqCode,
				PartNumber:        item.PartNumber,
				Quantity:          item.Quantity,
				FGLocation:        derefStr(item.FGLocation),
				QRCode:            derefStr(item.QR),
				PackingListNumber: item.PackingNumber,
				Status:            dn.Status,
				PrintedCount:      dn.PrintedCount,
			})
		}
	}

	return &models.DNListResponse{
		Summary:    *summary,
		Items:      rows,
		Pagination: repository.BuildPagination(total, f.Page, f.Limit),
	}, nil
}

func (s *service) GetDNDetail(ctx context.Context, dnUUID string) (*models.DNDetailResponse, error) {
	dn, err := s.repo.GetDNByUUID(ctx, dnUUID)
	if err != nil {
		return nil, errors.New("delivery note tidak ditemukan")
	}

	items, err := s.repo.GetDNItemsByDNID(ctx, dn.ID)
	if err != nil {
		return nil, err
	}

	var totalQty float64
	detailItems := make([]models.DNDetailItem, 0, len(items))
	for _, it := range items {
		remaining := it.Quantity - it.QtyShipped
		totalQty += it.Quantity
		detailItems = append(detailItems, models.DNDetailItem{
			DNItemID:      it.UUID,
			ItemUniqCode:  it.ItemUniqCode,
			PartName:      it.PartName,
			Quantity:      it.Quantity,
			UOM:           it.UOM,
			RemainingQty:  remaining,
			PackingNumber: it.PackingNumber,
			QR:            derefStr(it.QR),
		})
	}

	return &models.DNDetailResponse{
		DNID:                  dn.UUID,
		DNNumber:              dn.DNNumber,
		CustomerID:            dn.CustomerID,
		CustomerName:          derefStr(dn.CustomerNameSnapshot),
		PONumber:              derefStr(dn.CustomerOrderReference),
		CustomerContactPerson: derefStr(dn.CustomerContactPerson),
		CustomerPhoneNumber:   derefStr(dn.CustomerPhoneNumber),
		DeliveryAddress:       derefStr(dn.DeliveryAddress),
		DeliveryDate:          dn.DeliveryDate.Format("2006-01-02"),
		Priority:              dn.Priority,
		Status:                dn.Status,
		ApprovalStatus:        dn.ApprovalStatus,
		TransportCompany:      derefStr(dn.TransportCompany),
		VehicleNumber:         derefStr(dn.VehicleNumber),
		DriverName:            derefStr(dn.DriverName),
		DriverContact:         derefStr(dn.DriverContact),
		DepartureAt:           dn.DepartureAt,
		ArrivalAt:             dn.ArrivalAt,
		DeliveryInstructions:  derefStr(dn.DeliveryInstructions),
		TotalItems:            len(items),
		TotalQuantity:         totalQty,
		PrintedCount:          dn.PrintedCount,
		CreatedBy:             derefStr(dn.CreatedBy),
		Items:                 detailItems,
	}, nil
}

func (s *service) ConfirmDN(ctx context.Context, dnUUID string, req models.ConfirmDNRequest) (*models.ConfirmDNResponse, error) {
	dn, err := s.repo.GetDNByUUID(ctx, dnUUID)
	if err != nil {
		return nil, errors.New("delivery note tidak ditemukan")
	}

	if err := s.repo.UpdateDNStatus(ctx, nil, dn.ID, "confirmed"); err != nil {
		return nil, fmt.Errorf("gagal confirm DN: %w", err)
	}

	return &models.ConfirmDNResponse{
		DNID:   dn.UUID,
		Status: "confirmed",
	}, nil
}

// ─── Delivery Scan ────────────────────────────────────────────────────────────

func (s *service) LookupDeliveryItem(ctx context.Context, dnNumber, itemUniqCode string) (*models.DeliveryLookupResponse, error) {
	item, err := s.repo.GetDNItemByDNNumberAndUniqCode(ctx, dnNumber, itemUniqCode)
	if err != nil {
		return nil, errors.New("item DN tidak ditemukan atau sudah selesai")
	}

	var dn models.DNCustomer
	if err := s.db.WithContext(ctx).
		Where("id = ?", item.DNID).
		First(&dn).Error; err != nil {
		return nil, errors.New("delivery note tidak ditemukan")
	}

	// Try to get cycle from linked schedule
	var deliveryCycle string
	if dn.ScheduleID != nil {
		var sc models.ScheduleCustomer
		if err := s.db.WithContext(ctx).Select("cycle").Where("id = ?", *dn.ScheduleID).First(&sc).Error; err == nil {
			deliveryCycle = derefStr(sc.Cycle)
		}
	}

	return &models.DeliveryLookupResponse{
		DNID:          dn.UUID,
		DNItemID:      item.UUID,
		DNNumber:      dn.DNNumber,
		PODNReference: derefStr(dn.CustomerOrderReference),
		ItemUniqCode:  item.ItemUniqCode,
		PartName:      item.PartName,
		Model:         derefStr(item.Model),
		PartNo:        item.PartNumber,
		PackingNumber: item.PackingNumber,
		QuantityOrder: item.Quantity,
		RemainingQty:  item.Quantity - item.QtyShipped,
		UOM:           item.UOM,
		DeliveryDate:  dn.DeliveryDate.Format("2006-01-02"),
		DeliveryCycle: deliveryCycle,
	}, nil
}

func (s *service) SubmitDeliveryScan(ctx context.Context, req models.SubmitScanRequest, scannedBy string) (*models.SubmitScanResponse, error) {
	// Idempotency check
	exists, err := s.repo.IdempotencyKeyExists(ctx, req.ClientEventID)
	if err != nil {
		return nil, err
	}
	if exists {
		// Return existing result (idempotent)
		item, err := s.repo.GetDNItemByDNNumberAndUniqCode(ctx, req.DNNumber, req.ItemUniqCode)
		if err != nil {
			return nil, errors.New("item DN tidak ditemukan")
		}
		var dn models.DNCustomer
		_ = s.db.WithContext(ctx).Where("id = ?", item.DNID).First(&dn).Error
		return &models.SubmitScanResponse{
			DNNumber:     req.DNNumber,
			ItemUniqCode: req.ItemUniqCode,
			DeliveredQty: item.QtyShipped,
			RemainingQty: item.Quantity - item.QtyShipped,
			DNStatus:     dn.Status,
		}, nil
	}

	// Sum scan qty
	totalScan := 0.0
	for _, k := range req.ScannedKanbans {
		totalScan += k.Qty
	}

	var result models.SubmitScanResponse

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := s.repo.GetDNItemByDNNumberAndUniqCode(ctx, req.DNNumber, req.ItemUniqCode)
		if err != nil {
			return errors.New("item DN tidak ditemukan atau sudah selesai")
		}

		remaining := item.Quantity - item.QtyShipped
		if totalScan > remaining {
			return fmt.Errorf("scan qty (%.4f) melebihi sisa DN (%.4f)", totalScan, remaining)
		}

		// Check FG stock
		fgQty, _ := s.repo.GetFGStockQty(ctx, req.ItemUniqCode)
		if totalScan > fgQty {
			return fmt.Errorf("FG stock tidak cukup (tersedia: %.4f, dibutuhkan: %.4f)", fgQty, totalScan)
		}

		// Deduct FG
		if err := s.repo.DeductFGStock(ctx, tx, req.ItemUniqCode, totalScan, req.DNNumber, scannedBy); err != nil {
			return fmt.Errorf("gagal deduct FG: %w", err)
		}

		// Update item qty_shipped + status
		newShipped := item.QtyShipped + totalScan
		itemStatus := "partial"
		if newShipped >= item.Quantity {
			itemStatus = "shipped"
		}
		if err := s.repo.UpdateDNItemShipment(ctx, tx, item.ID, newShipped, itemStatus); err != nil {
			return err
		}

		// Build scan log per kanban
		for _, k := range req.ScannedKanbans {
			key := fmt.Sprintf("%s-%s", req.ClientEventID, k.ProductionKanban)
			log := &models.DNLogCustomer{
				DNID:           item.DNID,
				DNItemID:       item.ID,
				IdempotencyKey: &key,
				ScanRef:        strPtrIfNotEmpty(k.ProductionKanban),
				ItemUniqCode:   req.ItemUniqCode,
				PackingNumber:  &item.PackingNumber,
				ScanType:       "scan_out",
				Qty:            k.Qty,
				ScannedBy:      strPtrIfNotEmpty(scannedBy),
			}
			if err := s.repo.CreateDNLog(ctx, tx, log); err != nil {
				return err
			}
		}

		// Update DN header status
		statuses, _ := s.repo.GetDNItemStatuses(ctx, item.DNID)
		dnStatus := aggregateDNStatus(statuses)
		if dnStatus != "" {
			_ = s.repo.UpdateDNStatus(ctx, tx, item.DNID, dnStatus)
		}

		var dn models.DNCustomer
		_ = s.db.WithContext(ctx).Where("id = ?", item.DNID).First(&dn).Error

		result = models.SubmitScanResponse{
			DNNumber:     req.DNNumber,
			ItemUniqCode: req.ItemUniqCode,
			DeliveredQty: newShipped,
			RemainingQty: item.Quantity - newShipped,
			DNStatus:     dnStatus,
		}
		if result.DNStatus == "" {
			result.DNStatus = dn.Status
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// aggregateDNStatus derives DN header status from all item shipment statuses.
func aggregateDNStatus(statuses []string) string {
	if len(statuses) == 0 {
		return ""
	}
	allShipped := true
	anyShipped := false
	for _, st := range statuses {
		if st == "shipped" {
			anyShipped = true
		} else {
			allShipped = false
		}
	}
	if allShipped {
		return "shipped"
	}
	if anyShipped {
		return "in_transit"
	}
	return ""
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func generateQRBase64(value string) (string, error) {
	png, err := qrcode.Encode(value, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// nextNumber increments the trailing 4-digit sequence in last, or starts at 0001.
func nextNumber(last, prefix string) string {
	if last == "" {
		return fmt.Sprintf("%s-0001", prefix)
	}
	parts := strings.Split(last, "-")
	seq := 0
	fmt.Sscanf(parts[len(parts)-1], "%d", &seq)
	return fmt.Sprintf("%s-%04d", prefix, seq+1)
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("tanggal kosong")
	}
	return time.Parse("2006-01-02", s)
}

func parseTimes(departureStr, arrivalStr string, target interface{}) error {
	parse := func(s string) (*time.Time, error) {
		if s == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("format waktu tidak valid (%s): %w", s, err)
		}
		return &t, nil
	}

	dep, err := parse(departureStr)
	if err != nil {
		return err
	}
	arr, err := parse(arrivalStr)
	if err != nil {
		return err
	}

	switch v := target.(type) {
	case *models.ScheduleCustomer:
		v.DepartureAt = dep
		v.ArrivalAt = arr
	case *models.DNCustomer:
		v.DepartureAt = dep
		v.ArrivalAt = arr
	}
	return nil
}

func strPtr(s string) *string { return &s }

func strPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
