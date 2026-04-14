package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	invService "github.com/ganasa18/go-template/internal/inventory/service"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	woRepo "github.com/ganasa18/go-template/internal/work_order/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type IService interface {
	Create(ctx context.Context, req woModels.CreateWorkOrderRequest, createdBy string) (*woModels.CreateWorkOrderResponse, error)
	List(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.WorkOrderListResponse, error)
	GetSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error)
	GetDetail(ctx context.Context, woUUID string) (*woModels.WorkOrderDetailResponse, error)
	Approval(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error)
	BulkApproval(ctx context.Context, req woModels.BulkWorkOrderApprovalRequest, performedBy string) (*woModels.BulkWorkOrderApprovalResponse, error)
	GetWorkOrderQR(ctx context.Context, woUUID string, refresh bool) (*woModels.WorkOrderQRResponse, error)
	GetWorkOrderItemQR(ctx context.Context, itemUUID string, refresh bool) (*woModels.WorkOrderItemQRResponse, error)
	ListUniqOptions(ctx context.Context, q string, limit int, sources []string) (*woModels.UniqOptionsResponse, error)
	ListProcessOptions(ctx context.Context) (*woModels.ProcessOptionsResponse, error)
}

type service struct {
	repo   woRepo.IRepository
	db     *gorm.DB
	invSvc invService.IService
}

func New(repo woRepo.IRepository, db *gorm.DB, invSvc invService.IService) IService {
	return &service{repo: repo, db: db, invSvc: invSvc}
}

func (s *service) Create(ctx context.Context, req woModels.CreateWorkOrderRequest, createdBy string) (*woModels.CreateWorkOrderResponse, error) {
	createdDate := time.Now()
	if req.CreatedDate != nil && strings.TrimSpace(*req.CreatedDate) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.CreatedDate))
		if err != nil {
			return nil, apperror.BadRequest("created_date must be YYYY-MM-DD")
		}
		createdDate = t
	}

	var targetDate *time.Time
	if req.TargetDate != nil && strings.TrimSpace(*req.TargetDate) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.TargetDate))
		if err != nil {
			return nil, apperror.BadRequest("target_date must be YYYY-MM-DD")
		}
		targetDate = &t
	}

	var out *woModels.CreateWorkOrderResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		year := time.Now().Year()
		prefix := fmt.Sprintf("WO-%d", year)
		last, err := s.repo.FindLastWONumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		woNumber := generateWONumber(prefix, last)
		woUUID := uuid.New()
		creatorUUID, creatorName, err := s.resolveCreator(ctx, tx, createdBy)
		if err != nil {
			return err
		}
		woQR, err := generateQRDataURL(qrPayloadWO(woNumber))
		if err != nil {
			return apperror.InternalWrap("failed to generate WO QR", err)
		}
		wo := &woModels.WorkOrder{
			UUID:           woUUID,
			WoNumber:       woNumber,
			WoType:         req.WOType,
			ReferenceWO:    req.ReferenceWO,
			Status:         "Draft",
			ApprovalStatus: "Pending",
			CreatedDate:    createdDate,
			TargetDate:     targetDate,
			CreatedBy:      creatorUUID,
			CreatedByName:  creatorName,
			Notes:          req.Notes,
			QRImageBase64:  &woQR,
		}

		if err := s.repo.CreateWorkOrder(ctx, tx, wo); err != nil {
			return err
		}

		kbPrefix := strings.TrimPrefix(woNumber, "WO-") // still used for legacy trace/debug
		seq := 0
		perUniqSeq := map[string]int{}
		items := make([]woModels.WorkOrderItem, 0, 16)
		respItems := make([]woModels.CreateWorkOrderItemBrief, 0, 16)

		// Pre-fetch part metadata for all requested uniq_codes in one query.
		uniqCodes := make([]string, 0, len(req.Items))
		for _, it := range req.Items {
			uniqCodes = append(uniqCodes, it.ItemUniqCode)
		}
		partMeta, err := s.fetchPartMeta(ctx, tx, uniqCodes)
		if err != nil {
			return err
		}

		for _, it := range req.Items {
			kp, err := s.getKanbanParam(ctx, tx, it.ItemUniqCode)
			if err != nil {
				return err
			}
			pcsPerKanban := it.KanbanQty
			if pcsPerKanban <= 0 {
				if kp == nil || kp.KanbanQty <= 0 {
					return apperror.UnprocessableEntity("kanban_qty is required (kanban_parameters not found for item_uniq_code)")
				}
				pcsPerKanban = kp.KanbanQty
			}
			if kp == nil || strings.TrimSpace(kp.KanbanNumber) == "" {
				return apperror.UnprocessableEntity("kanban_parameters not found for item_uniq_code")
			}

			remaining := round4(it.Quantity)
			if remaining <= 0 {
				return apperror.UnprocessableEntity("quantity must be greater than 0")
			}
			perKanban := float64(pcsPerKanban)
			// Round up: WO items cannot be partial; last kanban is full too.
			kanbanCount := int(math.Ceil(remaining / perKanban))
			if kanbanCount <= 0 {
				kanbanCount = 1
			}

			for i := 0; i < kanbanCount; i++ {
				seq++
				perUniqSeq[it.ItemUniqCode]++
				q := round4(perKanban)

				itemUUID := uuid.New()
				_ = kbPrefix
				// Format: <KBN-YYYY-####>-<ITEM_UNIQ_CODE>-<WO_NUMBER>-<NN>
				// WO_NUMBER included to guarantee global uniqueness.
				kanbanNumber := fmt.Sprintf("%s-%s-%s-%02d", kp.KanbanNumber, it.ItemUniqCode, woNumber, perUniqSeq[it.ItemUniqCode])
				kpNum := kp.KanbanNumber
				kpSeq := perUniqSeq[it.ItemUniqCode]
				itemQR, err := generateQRDataURL(qrPayloadWOItem(kanbanNumber))
				if err != nil {
					return apperror.InternalWrap("failed to generate WO item QR", err)
				}
				items = append(items, woModels.WorkOrderItem{
					UUID:              itemUUID,
					WoID:              wo.ID,
					ItemUniqCode:      it.ItemUniqCode,
					PartName:          partMeta[it.ItemUniqCode].PartName,
					PartNumber:        partMeta[it.ItemUniqCode].PartNumber,
					Model:             partMeta[it.ItemUniqCode].Model,
					UOM:               it.UOM,
					ProcessName:       it.ProcessName,
					Quantity:          q,
					KanbanNumber:      kanbanNumber,
					KanbanParamNumber: &kpNum,
					KanbanSeq:         &kpSeq,
					Status:            "Pending",
					QRImageBase64:     &itemQR,
				})
				respItems = append(respItems, woModels.CreateWorkOrderItemBrief{
					ID:           itemUUID.String(),
					WoItemID:     itemUUID.String(),
					KanbanNumber: kanbanNumber,
					ItemUniqCode: it.ItemUniqCode,
					Quantity:     q,
					ProcessName:  it.ProcessName,
					QRDataURL:    &itemQR,
				})
			}
		}

		if err := s.repo.CreateWorkOrderItems(ctx, tx, items); err != nil {
			return err
		}

		out = &woModels.CreateWorkOrderResponse{
			ID:             woUUID.String(),
			WoID:           woUUID.String(),
			WoNumber:       woNumber,
			ApprovalStatus: wo.ApprovalStatus,
			QRDataURL:      &woQR,
			Items:          respItems,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type kanbanParamRow struct {
	KanbanNumber string `gorm:"column:kanban_number"`
	KanbanQty    int    `gorm:"column:kanban_qty"`
}

func (s *service) getKanbanParam(ctx context.Context, tx *gorm.DB, itemUniqCode string) (*kanbanParamRow, error) {
	itemUniqCode = strings.TrimSpace(itemUniqCode)
	if itemUniqCode == "" {
		return nil, nil
	}
	var row kanbanParamRow
	err := tx.WithContext(ctx).
		Table("kanban_parameters").
		Select("kanban_number, kanban_qty").
		Where("item_uniq_code = ? AND status ILIKE 'active'", itemUniqCode).
		Order("id DESC").
		Limit(1).
		Take(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, apperror.InternalWrap("getKanbanParam", err)
	}
	return &row, nil
}

func (s *service) resolveCreator(ctx context.Context, tx *gorm.DB, userUUID string) (*uuid.UUID, *string, error) {
	userUUID = strings.TrimSpace(userUUID)
	if userUUID == "" || userUUID == "system" {
		n := "system"
		return nil, &n, nil
	}
	parsed, err := uuid.Parse(userUUID)
	if err != nil {
		// Keep name as raw uuid string for trace; created_by remains NULL.
		n := userUUID
		return nil, &n, nil
	}

	var username string
	q := tx.WithContext(ctx).Table("users").Select("username").Where("uuid = ?", userUUID).Limit(1)
	if err := q.Take(&username).Error; err != nil {
		// If user not found, still store UUID string as name.
		n := userUUID
		return &parsed, &n, nil
	}
	if strings.TrimSpace(username) == "" {
		n := userUUID
		return &parsed, &n, nil
	}
	n := username
	return &parsed, &n, nil
}

func (s *service) GetWorkOrderQR(ctx context.Context, woUUID string, refresh bool) (*woModels.WorkOrderQRResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUID(ctx, woUUID)
	if err != nil {
		return nil, err
	}
	if !refresh && wo.QRImageBase64 != nil && strings.TrimSpace(*wo.QRImageBase64) != "" {
		payload := qrPayloadWO(wo.WoNumber)
		return &woModels.WorkOrderQRResponse{WoNumber: wo.WoNumber, QRPayload: payload, DataURL: *wo.QRImageBase64}, nil
	}

	payload := qrPayloadWO(wo.WoNumber)
	dataURL, err := generateQRDataURL(payload)
	if err != nil {
		return nil, apperror.InternalWrap("failed to generate QR", err)
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateWorkOrderQR(ctx, tx, wo.ID, dataURL)
	}); err != nil {
		return nil, err
	}

	return &woModels.WorkOrderQRResponse{WoNumber: wo.WoNumber, QRPayload: payload, DataURL: dataURL}, nil
}

func (s *service) GetWorkOrderItemQR(ctx context.Context, itemUUID string, refresh bool) (*woModels.WorkOrderItemQRResponse, error) {
	it, err := s.repo.GetWorkOrderItemByUUID(ctx, itemUUID)
	if err != nil {
		return nil, err
	}
	if !refresh && it.QRImageBase64 != nil && strings.TrimSpace(*it.QRImageBase64) != "" {
		payload := qrPayloadWOItem(it.KanbanNumber)
		return &woModels.WorkOrderItemQRResponse{KanbanNumber: it.KanbanNumber, QRPayload: payload, DataURL: *it.QRImageBase64}, nil
	}

	payload := qrPayloadWOItem(it.KanbanNumber)
	dataURL, err := generateQRDataURL(payload)
	if err != nil {
		return nil, apperror.InternalWrap("failed to generate QR", err)
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateWorkOrderItemQR(ctx, tx, it.ID, dataURL)
	}); err != nil {
		return nil, err
	}

	return &woModels.WorkOrderItemQRResponse{KanbanNumber: it.KanbanNumber, QRPayload: payload, DataURL: dataURL}, nil
}

func (s *service) ListUniqOptions(ctx context.Context, q string, limit int, sources []string) (*woModels.UniqOptionsResponse, error) {
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	allowed := map[string]bool{"items": true, "raw_material": true, "indirect": true, "subcon": true}
	include := map[string]bool{"items": true, "raw_material": true, "indirect": true, "subcon": true}
	if len(sources) > 0 {
		for k := range include {
			include[k] = false
		}
		for _, s0 := range sources {
			s0 = strings.TrimSpace(s0)
			if s0 == "" {
				continue
			}
			// accept a few aliases
			switch s0 {
			case "item", "items":
				s0 = "items"
			case "raw", "raw_material", "raw_materials":
				s0 = "raw_material"
			case "indirect", "indirect_raw_material", "indirect_raw_materials":
				s0 = "indirect"
			case "subcon", "subcon_inventory", "subcon_inventories":
				s0 = "subcon"
			}
			if allowed[s0] {
				include[s0] = true
			}
		}
		// if none valid provided, fallback to all
		any := false
		for _, v := range include {
			if v {
				any = true
				break
			}
		}
		if !any {
			for k := range include {
				include[k] = true
			}
		}
	}

	type row struct {
		UniqCode     string   `gorm:"column:uniq_code"`
		PartName     *string  `gorm:"column:part_name"`
		PartNumber   *string  `gorm:"column:part_number"`
		UOM          *string  `gorm:"column:uom"`
		AvailableQty *float64 `gorm:"column:available_qty"`
		KanbanQty    *int     `gorm:"column:kanban_qty"`
		KanbanNumber *string  `gorm:"column:kanban_number"`
		Sources      string   `gorm:"column:sources"`
	}

	var unions []string
	if include["items"] {
		unions = append(unions, "SELECT uniq_code, part_name, part_number, uom, NULL::numeric AS qty, 'items' AS source FROM items WHERE deleted_at IS NULL")
	}
	if include["raw_material"] {
		unions = append(unions, "SELECT uniq_code, part_name, part_number, uom, stock_qty AS qty, 'raw_material' AS source FROM raw_materials WHERE deleted_at IS NULL")
	}
	if include["indirect"] {
		unions = append(unions, "SELECT uniq_code, part_name, part_number, uom, stock_qty AS qty, 'indirect' AS source FROM indirect_raw_materials WHERE deleted_at IS NULL")
	}
	if include["subcon"] {
		unions = append(unions, "SELECT uniq_code, part_name, part_number, NULL::text AS uom, stock_at_vendor_qty AS qty, 'subcon' AS source FROM subcon_inventories WHERE deleted_at IS NULL")
	}
	if len(unions) == 0 {
		return &woModels.UniqOptionsResponse{Items: []woModels.UniqOptionItem{}}, nil
	}

	sql := "SELECT t.uniq_code, MAX(t.part_name) AS part_name, MAX(t.part_number) AS part_number, MAX(t.uom) AS uom, " +
		"SUM(COALESCE(t.qty,0))::numeric AS available_qty, " +
		"MAX(kp.kanban_qty) AS kanban_qty, MAX(kp.kanban_number) AS kanban_number, " +
		"string_agg(DISTINCT t.source, ',') AS sources " +
		"FROM (" + strings.Join(unions, " UNION ALL ") + ") t " +
		"LEFT JOIN kanban_parameters kp ON kp.item_uniq_code = t.uniq_code AND kp.status ILIKE 'active' " +
		"WHERE ($1 = '' OR t.uniq_code ILIKE '%' || $1 || '%' OR COALESCE(t.part_name,'') ILIKE '%' || $1 || '%' OR COALESCE(t.part_number,'') ILIKE '%' || $1 || '%') " +
		"GROUP BY t.uniq_code ORDER BY t.uniq_code LIMIT $2"

	rows := make([]row, 0, limit)
	if err := s.db.WithContext(ctx).Raw(sql, q, limit).Scan(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("failed to list uniq options", err)
	}

	out := make([]woModels.UniqOptionItem, 0, len(rows))
	for _, r := range rows {
		var src []string
		if strings.TrimSpace(r.Sources) != "" {
			for _, s0 := range strings.Split(r.Sources, ",") {
				s0 = strings.TrimSpace(s0)
				if s0 != "" {
					src = append(src, s0)
				}
			}
		}
		out = append(out, woModels.UniqOptionItem{
			UniqCode:     r.UniqCode,
			PartName:     r.PartName,
			PartNumber:   r.PartNumber,
			UOM:          r.UOM,
			AvailableQty: r.AvailableQty,
			KanbanQty:    r.KanbanQty,
			KanbanNumber: r.KanbanNumber,
			Sources:      src,
		})
	}

	return &woModels.UniqOptionsResponse{Items: out}, nil
}

func (s *service) ListProcessOptions(ctx context.Context) (*woModels.ProcessOptionsResponse, error) {
	type row struct {
		ProcessCode string `gorm:"column:process_code"`
		ProcessName string `gorm:"column:process_name"`
		Sequence    int    `gorm:"column:sequence"`
		Status      string `gorm:"column:status"`
	}
	var rows []row
	err := s.db.WithContext(ctx).
		Table("process_parameters").
		Select("process_code, process_name, sequence, status").
		Where("status ILIKE 'active'").
		Order("sequence ASC, process_name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to list process options", err)
	}
	out := make([]woModels.ProcessOptionItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, woModels.ProcessOptionItem{ProcessCode: r.ProcessCode, ProcessName: r.ProcessName})
	}
	return &woModels.ProcessOptionsResponse{Items: out}, nil
}

func (s *service) List(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.WorkOrderListResponse, error) {
	f := woRepo.ListFilter{
		Search:         p.Search,
		Status:         p.Status,
		ApprovalStatus: p.ApprovalStatus,
		WOType:         p.WOType,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	}
	rows, total, err := s.repo.ListWorkOrders(ctx, f)
	if err != nil {
		return nil, err
	}

	// Batch-load items for all returned WOs in one query.
	woIDs := make([]int64, 0, len(rows))
	woIDIndex := make(map[int64]int, len(rows)) // woID → index in items slice
	for i, r := range rows {
		woIDs = append(woIDs, r.ID)
		woIDIndex[r.ID] = i
	}
	itemRows, err := s.repo.GetItemsByWOIDs(ctx, woIDs)
	if err != nil {
		return nil, err
	}
	// Group items by wo_id.
	itemsByWO := make(map[int64][]woModels.WorkOrderListItemDetail, len(rows))
	for _, it := range itemRows {
		itemsByWO[it.WoID] = append(itemsByWO[it.WoID], woModels.WorkOrderListItemDetail{
			ID:           it.UUID.String(),
			ItemUniqCode: it.ItemUniqCode,
			PartName:     it.PartName,
			PartNumber:   it.PartNumber,
			Model:        it.Model,
			Quantity:     it.Quantity,
			UOM:          it.UOM,
			Status:       it.Status,
		})
	}

	items := make([]woModels.WorkOrderListItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if r.ItemCount > 0 {
			pct = math.Round(float64(r.ClosedCount)/float64(r.ItemCount)*100*100) / 100
		}
		woItems := itemsByWO[r.ID]
		if woItems == nil {
			woItems = []woModels.WorkOrderListItemDetail{}
		}
		items = append(items, woModels.WorkOrderListItem{
			ID:             r.UUID,
			WoNumber:       r.WoNumber,
			WoType:         r.WoType,
			ReferenceWO:    r.ReferenceWO,
			Status:         r.Status,
			ApprovalStatus: r.ApprovalStatus,
			CreatedDate:    r.CreatedDate,
			TargetDate:     r.TargetDate,
			CreatedByName:  r.CreatedByName,
			UniqCount:      r.UniqCount,
			ItemCount:      r.ItemCount,
			ClosedCount:    r.ClosedCount,
			ProgressPct:    pct,
			AgingDays:      r.AgingDays,
			Items:          woItems,
		})
	}

	return &woModels.WorkOrderListResponse{
		Items:      items,
		Pagination: pagination.NewMeta(total, p.PaginationInput),
	}, nil
}

func (s *service) GetSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error) {
	row, err := s.repo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}
	return &woModels.WorkOrderSummaryResponse{
		ActiveWOs:  row.ActiveWOs,
		Completed:  row.Completed,
		PendingWOs: row.PendingWOs,
		TotalUniqs: row.TotalUniqs,
	}, nil
}

func (s *service) GetDetail(ctx context.Context, woUUID string) (*woModels.WorkOrderDetailResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUID(ctx, woUUID)
	if err != nil {
		return nil, err
	}
	itemRows, err := s.repo.GetWorkOrderItemsByWOID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}

	items := make([]woModels.WorkOrderDetailItem, 0, len(itemRows))
	for _, it := range itemRows {
		var itemQR *string
		if it.QRImageBase64 != nil && strings.TrimSpace(*it.QRImageBase64) != "" {
			itemQR = it.QRImageBase64
		} else {
			if qr, err := generateQRDataURL(qrPayloadWOItem(it.KanbanNumber)); err == nil {
				itemQR = &qr
			}
		}
		items = append(items, woModels.WorkOrderDetailItem{
			ID:           it.UUID.String(),
			WoItemID:     it.UUID.String(),
			KanbanNumber: it.KanbanNumber,
			ItemUniqCode: it.ItemUniqCode,
			Quantity:     it.Quantity,
			UOM:          it.UOM,
			ProcessName:  it.ProcessName,
			Status:       it.Status,
			QRDataURL:    itemQR,
		})
	}

	createdDate := wo.CreatedDate.Format("2006-01-02")
	var targetDate *string
	if wo.TargetDate != nil {
		s := wo.TargetDate.Format("2006-01-02")
		targetDate = &s
	}

	var woQR *string
	if wo.QRImageBase64 != nil && strings.TrimSpace(*wo.QRImageBase64) != "" {
		woQR = wo.QRImageBase64
	} else {
		if qr, err := generateQRDataURL(qrPayloadWO(wo.WoNumber)); err == nil {
			woQR = &qr
		}
	}

	return &woModels.WorkOrderDetailResponse{
		ID:             wo.UUID.String(),
		WoNumber:       wo.WoNumber,
		WoType:         wo.WoType,
		ReferenceWO:    wo.ReferenceWO,
		Status:         wo.Status,
		ApprovalStatus: wo.ApprovalStatus,
		CreatedDate:    createdDate,
		TargetDate:     targetDate,
		CreatedByName:  wo.CreatedByName,
		Notes:          wo.Notes,
		QRDataURL:      woQR,
		Items:          items,
	}, nil
}

func (s *service) Approval(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUID(ctx, woUUID)
	if err != nil {
		return nil, err
	}

	newStatus := ""
	switch req.Decision {
	case "approve":
		newStatus = "Approved"
	case "reject":
		newStatus = "Rejected"
	default:
		return nil, apperror.BadRequest("decision must be approve or reject")
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateWorkOrderApprovalStatus(ctx, tx, wo.ID, newStatus)
	}); err != nil {
		return nil, err
	}

	// Deduct inventory stock and write movement logs when approved.
	if req.Decision == "approve" {
		items, err := s.repo.GetWorkOrderItemsByWOID(ctx, wo.ID)
		if err == nil && len(items) > 0 {
			consumeItems := make([]invService.ConsumeItem, 0, len(items))
			for _, it := range items {
				consumeItems = append(consumeItems, invService.ConsumeItem{
					UniqCode: it.ItemUniqCode,
					Qty:      it.Quantity,
				})
			}
			_ = s.invSvc.ConsumeStockForWorkOrder(ctx, consumeItems, wo.WoNumber, performedBy)
		}
	}

	return &woModels.WorkOrderApprovalResponse{
		ID:             wo.UUID.String(),
		WoNumber:       wo.WoNumber,
		ApprovalStatus: newStatus,
	}, nil
}

func (s *service) BulkApproval(ctx context.Context, req woModels.BulkWorkOrderApprovalRequest, performedBy string) (*woModels.BulkWorkOrderApprovalResponse, error) {
	newStatus := ""
	switch req.Decision {
	case "approve":
		newStatus = "Approved"
	case "reject":
		newStatus = "Rejected"
	default:
		return nil, apperror.BadRequest("decision must be approve or reject")
	}

	found, err := s.repo.FindWorkOrdersByWONumbers(ctx, req.WONumbers)
	if err != nil {
		return nil, err
	}

	updated := make([]woModels.BulkApprovalResultItem, 0)
	failed := make([]woModels.BulkApprovalFailedItem, 0)

	for _, woNumber := range req.WONumbers {
		wo, ok := found[woNumber]
		if !ok {
			failed = append(failed, woModels.BulkApprovalFailedItem{WoNumber: woNumber, Reason: "not found"})
			continue
		}
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return s.repo.UpdateWorkOrderApprovalStatus(ctx, tx, wo.ID, newStatus)
		})
		if err != nil {
			failed = append(failed, woModels.BulkApprovalFailedItem{WoNumber: woNumber, Reason: err.Error()})
			continue
		}

		// Deduct inventory stock when approved.
		if req.Decision == "approve" {
			items, iErr := s.repo.GetWorkOrderItemsByWOID(ctx, wo.ID)
			if iErr == nil && len(items) > 0 {
				consumeItems := make([]invService.ConsumeItem, 0, len(items))
				for _, it := range items {
					consumeItems = append(consumeItems, invService.ConsumeItem{
						UniqCode: it.ItemUniqCode,
						Qty:      it.Quantity,
					})
				}
				_ = s.invSvc.ConsumeStockForWorkOrder(ctx, consumeItems, wo.WoNumber, performedBy)
			}
		}

		updated = append(updated, woModels.BulkApprovalResultItem{WoNumber: woNumber, ApprovalStatus: newStatus})
	}

	return &woModels.BulkWorkOrderApprovalResponse{
		Decision:       req.Decision,
		TotalRequested: len(req.WONumbers),
		TotalUpdated:   len(updated),
		Updated:        updated,
		Failed:         failed,
	}, nil
}

func generateWONumber(prefix string, last string) string {
	if last == "" {
		return fmt.Sprintf("%s-%06d", prefix, 1)
	}
	parts := strings.Split(last, "-")
	if len(parts) == 0 {
		return fmt.Sprintf("%s-%06d", prefix, 1)
	}
	seqStr := parts[len(parts)-1]
	seq, err := strconv.Atoi(seqStr)
	if err != nil || seq < 0 {
		return fmt.Sprintf("%s-%06d", prefix, 1)
	}
	return fmt.Sprintf("%s-%06d", prefix, seq+1)
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func strPtr(s string) *string { return &s }

func generateQRDataURL(value string) (string, error) {
	png, err := qrcode.Encode(value, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	base64Str := base64.StdEncoding.EncodeToString(png)
	return "data:image/png;base64," + base64Str, nil
}

func qrPayloadWO(woNumber string) string {
	return fmt.Sprintf(`{"t":"wo","wo":%s}`, strconv.Quote(woNumber))
}

func qrPayloadWOItem(kanbanNumber string) string {
	return fmt.Sprintf(`{"t":"wo_item","kb":%s}`, strconv.Quote(kanbanNumber))
}

// partMetaRow holds denormalized item metadata fetched from items / raw_materials / indirect_raw_materials.
type partMetaRow struct {
	PartName   *string
	PartNumber *string
	Model      *string
}

// fetchPartMeta queries part metadata from all three item-master tables in a single UNION ALL
// and returns a map keyed by uniq_code.  Any field absent in a particular table is coalesced
// to NULL so the caller can decide a safe default.
func (s *service) fetchPartMeta(ctx context.Context, tx *gorm.DB, uniqCodes []string) (map[string]partMetaRow, error) {
	if len(uniqCodes) == 0 {
		return map[string]partMetaRow{}, nil
	}
	type row struct {
		UniqCode   string  `gorm:"column:uniq_code"`
		PartName   *string `gorm:"column:part_name"`
		PartNumber *string `gorm:"column:part_number"`
		Model      *string `gorm:"column:model"`
	}
	rawSQL := `
		SELECT t.uniq_code,
		       MAX(t.part_name)   AS part_name,
		       MAX(t.part_number) AS part_number,
		       MAX(t.model)       AS model
		FROM (
		    SELECT uniq_code,
		           part_name,
		           part_number,
		           model
		    FROM   items
		    WHERE  deleted_at IS NULL
		    UNION ALL
		    SELECT uniq_code,
		           COALESCE(part_name, '')  AS part_name,
		           part_number,
		           NULL::text              AS model
		    FROM   raw_materials
		    WHERE  deleted_at IS NULL
		    UNION ALL
		    SELECT uniq_code,
		           COALESCE(part_name, '')  AS part_name,
		           part_number,
		           NULL::text              AS model
		    FROM   indirect_raw_materials
		    WHERE  deleted_at IS NULL
		) t
		WHERE t.uniq_code IN ?
		GROUP BY t.uniq_code`

	var rows []row
	db := s.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	if err := db.Raw(rawSQL, uniqCodes).Scan(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("failed to fetch part metadata", err)
	}
	out := make(map[string]partMetaRow, len(rows))
	for _, r := range rows {
		out[r.UniqCode] = partMetaRow{
			PartName:   r.PartName,
			PartNumber: r.PartNumber,
			Model:      r.Model,
		}
	}
	return out, nil
}
