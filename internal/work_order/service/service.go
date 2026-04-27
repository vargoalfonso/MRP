package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	invService "github.com/ganasa18/go-template/internal/inventory/service"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	woRepo "github.com/ganasa18/go-template/internal/work_order/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/creatorresolver"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IService interface {
	Create(ctx context.Context, req woModels.CreateWorkOrderRequest, createdBy string) (*woModels.CreateWorkOrderResponse, error)
	CreateBulk(ctx context.Context, req woModels.CreateBulkWorkOrderRequest, createdBy string) (*woModels.CreateBulkWorkOrderResponse, error)
	CreateRMProcessing(ctx context.Context, req woModels.CreateRMProcessingWorkOrderRequest, createdBy string) (*woModels.RMProcessingWorkOrderCreateResponse, error)
	Preview(ctx context.Context, req woModels.CreateWorkOrderRequest) (*woModels.WorkOrderPreviewResponse, error)
	List(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.WorkOrderListResponse, error)
	ListBulk(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.WorkOrderListResponse, error)
	ListRMProcessing(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.RMProcessingWorkOrderListResponse, error)
	GetSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error)
	GetBulkSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error)
	GetRMProcessingSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error)
	GetDetail(ctx context.Context, woUUID string) (*woModels.WorkOrderDetailResponse, error)
	GetBulkDetail(ctx context.Context, woUUID string) (*woModels.WorkOrderDetailResponse, error)
	GetRMProcessingDetail(ctx context.Context, woUUID string) (*woModels.RMProcessingWorkOrderDetailResponse, error)
	Approval(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error)
	ApprovalBulk(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error)
	ApprovalRMProcessing(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error)
	BulkApproval(ctx context.Context, req woModels.BulkWorkOrderApprovalRequest, performedBy string) (*woModels.BulkWorkOrderApprovalResponse, error)
	BulkApprovalBulk(ctx context.Context, req woModels.BulkWorkOrderApprovalRequest, performedBy string) (*woModels.BulkWorkOrderApprovalResponse, error)
	GetWorkOrderQR(ctx context.Context, woUUID string, refresh bool) (*woModels.WorkOrderQRResponse, error)
	GetWorkOrderItemQR(ctx context.Context, itemUUID string, refresh bool) (*woModels.WorkOrderItemQRResponse, error)
	ListUniqOptions(ctx context.Context, q string, limit int, sources []string) (*woModels.UniqOptionsResponse, error)
	ListProcessOptions(ctx context.Context) (*woModels.ProcessOptionsResponse, error)
	ListBulkSourceDocuments(ctx context.Context, documentType, q, targetDate string, limit int) (*woModels.BulkDocumentOptionsResponse, error)
	ListBulkSourceDocumentItems(ctx context.Context, documentID, documentType string) (*woModels.BulkDocumentItemsResponse, error)
}

type service struct {
	repo   woRepo.IRepository
	db     *gorm.DB
	invSvc invService.IService
}

const (
	workOrderKindStandard       = "standard"
	workOrderPrefixStandard     = "WO"
	workOrderKindBulk           = "bulk"
	workOrderPrefixBulk         = "BULK-WO"
	workOrderKindRMProcessing   = "rm_processing"
	workOrderPrefixRMProcessing = "RM-WO"
	workOrderTypeRMProcessing   = "RM Processing"
)

func New(repo woRepo.IRepository, db *gorm.DB, invSvc invService.IService) IService {
	return &service{repo: repo, db: db, invSvc: invSvc}
}

func (s *service) ListBulkSourceDocuments(ctx context.Context, documentType, q, targetDate string, limit int) (*woModels.BulkDocumentOptionsResponse, error) {
	documentType = strings.ToUpper(strings.TrimSpace(documentType))
	if documentType != "" && documentType != "SO" && documentType != "PO" && documentType != "DN" {
		return nil, apperror.BadRequest("document_type must be one of SO, PO, DN")
	}
	rows, err := s.repo.ListBulkSourceDocuments(ctx, documentType, strings.TrimSpace(q), strings.TrimSpace(targetDate), limit)
	if err != nil {
		return nil, err
	}
	out := make([]woModels.BulkDocumentOptionItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, woModels.BulkDocumentOptionItem{
			DocumentID:     row.DocumentUUID,
			DocumentNumber: row.DocumentNumber,
			DocumentType:   row.DocumentType,
			DocumentDate:   row.DocumentDate,
			CustomerName:   row.CustomerName,
			ItemCount:      row.ItemCount,
		})
	}
	return &woModels.BulkDocumentOptionsResponse{Items: out}, nil
}

func (s *service) ListBulkSourceDocumentItems(ctx context.Context, documentID, documentType string) (*woModels.BulkDocumentItemsResponse, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, apperror.BadRequest("document_id is required")
	}
	header, err := s.repo.GetBulkSourceDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListBulkSourceDocumentItems(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	items := make([]woModels.BulkDocumentItem, 0, len(rows))
	for _, row := range rows {
		kp, err := s.getKanbanParam(ctx, s.db, row.ItemUniqCode)
		if err != nil {
			return nil, err
		}
		kanbanCount := 0
		var kanbanQty *int
		if kp != nil && kp.KanbanQty > 0 {
			kanbanQty = &kp.KanbanQty
			kanbanCount = int(math.Ceil(round4(row.Quantity) / float64(kp.KanbanQty)))
			if kanbanCount <= 0 {
				kanbanCount = 1
			}
		}
		items = append(items, woModels.BulkDocumentItem{
			SourceLineID: row.SourceLineUUID,
			ItemUniqCode: row.ItemUniqCode,
			PartName:     row.PartName,
			PartNumber:   row.PartNumber,
			UOM:          row.UOM,
			Quantity:     round4(row.Quantity),
			KanbanQty:    kanbanQty,
			KanbanCount:  kanbanCount,
			TargetDate:   row.TargetDate,
		})
	}
	return &woModels.BulkDocumentItemsResponse{
		Document: woModels.BulkDocumentMeta{
			DocumentID:     header.DocumentUUID,
			DocumentNumber: header.DocumentNumber,
			DocumentType:   header.DocumentType,
			DocumentDate:   header.DocumentDate,
			CustomerName:   header.CustomerName,
		},
		Items: items,
	}, nil
}

func (s *service) CreateBulk(ctx context.Context, req woModels.CreateBulkWorkOrderRequest, createdBy string) (*woModels.CreateBulkWorkOrderResponse, error) {
	createdDate := time.Now()
	if req.CreatedDate != nil && strings.TrimSpace(*req.CreatedDate) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.CreatedDate))
		if err != nil {
			return nil, apperror.BadRequest("created_date must be YYYY-MM-DD")
		}
		createdDate = t
	}

	var headerTargetDate *time.Time
	if req.TargetDate != nil && strings.TrimSpace(*req.TargetDate) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.TargetDate))
		if err != nil {
			return nil, apperror.BadRequest("target_date must be YYYY-MM-DD")
		}
		headerTargetDate = &t
	}

	documentType := strings.ToUpper(strings.TrimSpace(req.SourceDocumentType))
	if documentType != "SO" && documentType != "PO" && documentType != "DN" {
		return nil, apperror.BadRequest("source_document_type must be one of SO, PO, DN")
	}

	header, err := s.repo.GetBulkSourceDocument(ctx, strings.TrimSpace(req.SourceDocumentID))
	if err != nil {
		return nil, err
	}
	if documentType != header.DocumentType {
		return nil, apperror.BadRequest("source_document_type does not match source document")
	}
	sourceRows, err := s.repo.ListBulkSourceDocumentItems(ctx, strings.TrimSpace(req.SourceDocumentID))
	if err != nil {
		return nil, err
	}
	sourceByLine := make(map[string]woRepo.BulkSourceDocumentItemRow, len(sourceRows))
	for _, row := range sourceRows {
		sourceByLine[row.SourceLineUUID] = row
	}
	if headerTargetDate == nil {
		for _, it := range req.Items {
			if it.TargetDate != nil && strings.TrimSpace(*it.TargetDate) != "" {
				parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*it.TargetDate))
				if err != nil {
					return nil, apperror.BadRequest("item target_date must be YYYY-MM-DD")
				}
				headerTargetDate = &parsed
				break
			}
			src, ok := sourceByLine[strings.TrimSpace(it.SourceLineID)]
			if ok && src.TargetDate != nil && strings.TrimSpace(*src.TargetDate) != "" {
				parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*src.TargetDate))
				if err == nil {
					headerTargetDate = &parsed
					break
				}
			}
		}
	}

	var out *woModels.CreateBulkWorkOrderResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		year := createdDate.Year()
		prefix := fmt.Sprintf("%s-%d", workOrderPrefixBulk, year)
		last, err := s.repo.FindLastWONumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		woNumber := generateWONumber(prefix, last)
		woUUID := uuid.New()
		creatorUUID, creatorName, err := creatorresolver.Resolve(ctx, tx, createdBy)
		if err != nil {
			return err
		}
		woQR, err := generateQRDataURL(qrPayloadWO(woNumber))
		if err != nil {
			return apperror.InternalWrap("failed to generate WO QR", err)
		}
		notes := req.Notes
		if notes == nil || strings.TrimSpace(*notes) == "" {
			autoNote := fmt.Sprintf("Bulk WO source: %s %s", documentType, strings.TrimSpace(req.SourceDocumentID))
			notes = &autoNote
		}
		wo := &woModels.WorkOrder{
			UUID:           woUUID,
			WoNumber:       woNumber,
			WoType:         req.WOType,
			WOKind:         workOrderKindBulk,
			Status:         "Draft",
			ApprovalStatus: "Pending",
			CreatedDate:    createdDate,
			TargetDate:     headerTargetDate,
			CreatedBy:      creatorUUID,
			CreatedByName:  creatorName,
			Notes:          notes,
			QRImageBase64:  &woQR,
		}
		if err := s.repo.CreateWorkOrder(ctx, tx, wo); err != nil {
			return err
		}

		uniqCodes := make([]string, 0, len(req.Items))
		for _, it := range req.Items {
			uniqCodes = append(uniqCodes, it.ItemUniqCode)
		}
		partMeta, err := s.fetchPartMeta(ctx, tx, uniqCodes)
		if err != nil {
			return err
		}
		processFlows, err := s.fetchProcessFlows(ctx, tx, uniqCodes)
		if err != nil {
			return err
		}

		perUniqSeq := map[string]int{}
		items := make([]woModels.WorkOrderItem, 0, 16)
		respItems := make([]woModels.CreateWorkOrderItemBrief, 0, 16)
		for _, it := range req.Items {
			src, ok := sourceByLine[strings.TrimSpace(it.SourceLineID)]
			if !ok {
				return apperror.UnprocessableEntity("source_line_id does not belong to source document")
			}
			if strings.TrimSpace(src.ItemUniqCode) != strings.TrimSpace(it.ItemUniqCode) {
				return apperror.UnprocessableEntity("item_uniq_code does not match source document item")
			}
			kp, err := s.getKanbanParam(ctx, tx, it.ItemUniqCode)
			if err != nil {
				return err
			}
			pcsPerKanban := it.KanbanQty
			if pcsPerKanban <= 0 {
				if kp == nil || kp.KanbanQty <= 0 {
					return apperror.UnprocessableEntity("kanban_parameters not found for item_uniq_code")
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
			kanbanCount := int(math.Ceil(remaining / perKanban))
			if kanbanCount <= 0 {
				kanbanCount = 1
			}
			for i := 0; i < kanbanCount; i++ {
				perUniqSeq[it.ItemUniqCode]++
				seq := perUniqSeq[it.ItemUniqCode]
				itemUUID := uuid.New()
				kanbanNumber := fmt.Sprintf("%s-%s-%s-%02d", kp.KanbanNumber, it.ItemUniqCode, woNumber, seq)
				kpNum := kp.KanbanNumber
				kpSeq := seq
				itemQR, err := generateQRDataURL(qrPayloadWOItem(kanbanNumber))
				if err != nil {
					return apperror.InternalWrap("failed to generate WO item QR", err)
				}
				uom := it.UOM
				if uom == nil || strings.TrimSpace(*uom) == "" {
					uom = strPtr(src.UOM)
				}
				items = append(items, woModels.WorkOrderItem{
					UUID:              itemUUID,
					WoID:              wo.ID,
					ItemUniqCode:      it.ItemUniqCode,
					PartName:          partMeta[it.ItemUniqCode].PartName,
					PartNumber:        partMeta[it.ItemUniqCode].PartNumber,
					Model:             partMeta[it.ItemUniqCode].Model,
					UOM:               uom,
					ProcessName:       it.ProcessName,
					Quantity:          round4(perKanban),
					KanbanNumber:      kanbanNumber,
					ProcessFlowJSON:   processFlows[it.ItemUniqCode],
					CurrentStepSeq:    1,
					KanbanParamNumber: &kpNum,
					KanbanSeq:         &kpSeq,
					Status:            "Pending",
					QRImageBase64:     &itemQR,
				})
				respItems = append(respItems, woModels.CreateWorkOrderItemBrief{
					ID:              itemUUID.String(),
					WoItemID:        itemUUID.String(),
					KanbanNumber:    kanbanNumber,
					ItemUniqCode:    it.ItemUniqCode,
					Quantity:        round4(perKanban),
					ProcessName:     it.ProcessName,
					ProcessFlowJSON: json.RawMessage(processFlows[it.ItemUniqCode]),
					QRDataURL:       &itemQR,
				})
			}
		}
		if err := s.repo.CreateWorkOrderItems(ctx, tx, items); err != nil {
			return err
		}
		out = &woModels.CreateBulkWorkOrderResponse{
			ID:                 woUUID.String(),
			WoID:               woUUID.String(),
			WoNumber:           woNumber,
			WOKind:             workOrderKindBulk,
			ApprovalStatus:     wo.ApprovalStatus,
			SourceDocumentID:   strings.TrimSpace(req.SourceDocumentID),
			SourceDocumentType: documentType,
			QRDataURL:          &woQR,
			Items:              respItems,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *service) CreateRMProcessing(ctx context.Context, req woModels.CreateRMProcessingWorkOrderRequest, createdBy string) (*woModels.RMProcessingWorkOrderCreateResponse, error) {
	if strings.EqualFold(strings.TrimSpace(req.SourceMaterialUniq), strings.TrimSpace(req.TargetMaterialUniq)) {
		return nil, apperror.BadRequest("source_material_uniq and target_material_uniq must be different")
	}

	dateIssued := time.Now()
	if req.DateIssued != nil && strings.TrimSpace(*req.DateIssued) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DateIssued))
		if err != nil {
			return nil, apperror.BadRequest("date_issued must be YYYY-MM-DD")
		}
		dateIssued = t
	}

	var out *woModels.RMProcessingWorkOrderCreateResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		year := dateIssued.Year()
		prefix := fmt.Sprintf("%s-%d", workOrderPrefixRMProcessing, year)
		last, err := s.repo.FindLastWONumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		woNumber := generateWONumber(prefix, last)
		woUUID := uuid.New()
		creatorUUID, creatorName, err := creatorresolver.Resolve(ctx, tx, createdBy)
		if err != nil {
			return err
		}
		woQR, err := generateQRDataURL(qrPayloadWO(woNumber))
		if err != nil {
			return apperror.InternalWrap("failed to generate WO QR", err)
		}
		wo := &woModels.WorkOrder{
			UUID:               woUUID,
			WoNumber:           woNumber,
			WoType:             workOrderTypeRMProcessing,
			WOKind:             workOrderKindRMProcessing,
			Status:             "Draft",
			ApprovalStatus:     "Pending",
			CreatedDate:        dateIssued,
			DateIssued:         &dateIssued,
			CreatedBy:          creatorUUID,
			CreatedByName:      creatorName,
			Notes:              req.Remarks,
			SourceMaterialUniq: strPtr(strings.TrimSpace(req.SourceMaterialUniq)),
			TargetMaterialUniq: strPtr(strings.TrimSpace(req.TargetMaterialUniq)),
			Model:              req.Model,
			GradeSize:          req.GradeSize,
			InputQty:           floatPtr(round4(req.InputQty)),
			InputUOM:           strPtr(strings.TrimSpace(req.InputUOM)),
			OutputQty:          floatPtr(round4(req.OutputQty)),
			OutputUOM:          strPtr(strings.TrimSpace(req.OutputUOM)),
			Remarks:            req.Remarks,
			QRImageBase64:      &woQR,
		}
		if err := s.repo.CreateWorkOrder(ctx, tx, wo); err != nil {
			return err
		}

		out = &woModels.RMProcessingWorkOrderCreateResponse{
			ID:                 woUUID.String(),
			WoID:               woUUID.String(),
			WoNumber:           woNumber,
			WoType:             wo.WoType,
			WOKind:             wo.WOKind,
			Status:             wo.Status,
			ApprovalStatus:     wo.ApprovalStatus,
			SourceMaterialUniq: strings.TrimSpace(req.SourceMaterialUniq),
			TargetMaterialUniq: strings.TrimSpace(req.TargetMaterialUniq),
			Model:              req.Model,
			GradeSize:          req.GradeSize,
			InputQty:           round4(req.InputQty),
			InputUOM:           strings.TrimSpace(req.InputUOM),
			OutputQty:          round4(req.OutputQty),
			OutputUOM:          strings.TrimSpace(req.OutputUOM),
			DateIssued:         dateIssued.Format("2006-01-02"),
			Remarks:            req.Remarks,
			QRDataURL:          &woQR,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
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
		prefix := fmt.Sprintf("%s-%d", workOrderPrefixStandard, year)
		last, err := s.repo.FindLastWONumber(ctx, tx, prefix)
		if err != nil {
			return err
		}

		woNumber := generateWONumber(prefix, last)
		woUUID := uuid.New()
		if req.WOID != nil && strings.TrimSpace(*req.WOID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*req.WOID))
			if err != nil {
				return apperror.BadRequest("wo_id must be a valid UUID")
			}
			woUUID = parsed
		}
		creatorUUID, creatorName, err := creatorresolver.Resolve(ctx, tx, createdBy)
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
			WOKind:         workOrderKindStandard,
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
		// Pre-fetch process flows (poka-yoke routing snapshot) from BOM.
		processFlows, err := s.fetchProcessFlows(ctx, tx, uniqCodes)
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
					ProcessFlowJSON:   processFlows[it.ItemUniqCode],
					CurrentStepSeq:    1,
					KanbanParamNumber: &kpNum,
					KanbanSeq:         &kpSeq,
					Status:            "Pending",
					QRImageBase64:     &itemQR,
				})
				respItems = append(respItems, woModels.CreateWorkOrderItemBrief{
					ID:              itemUUID.String(),
					WoItemID:        itemUUID.String(),
					KanbanNumber:    kanbanNumber,
					ItemUniqCode:    it.ItemUniqCode,
					Quantity:        q,
					ProcessName:     it.ProcessName,
					ProcessFlowJSON: json.RawMessage(processFlows[it.ItemUniqCode]),
					QRDataURL:       &itemQR,
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

func (s *service) Preview(ctx context.Context, req woModels.CreateWorkOrderRequest) (*woModels.WorkOrderPreviewResponse, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("%s-%d", workOrderPrefixStandard, year)
	last, err := s.repo.FindLastWONumber(ctx, nil, prefix)
	if err != nil {
		return nil, err
	}

	woNumber := generateWONumber(prefix, last)
	// wo_id is returned so FE can reuse it in Create request.
	// Note: wo_number is still best-effort and may change.
	woID := uuid.New()
	perUniqSeq := map[string]int{}
	outItems := make([]woModels.WorkOrderPreviewItem, 0, 32)

	for _, it := range req.Items {
		kp, err := s.getKanbanParam(ctx, s.db, it.ItemUniqCode)
		if err != nil {
			return nil, err
		}
		pcsPerKanban := it.KanbanQty
		if pcsPerKanban <= 0 {
			if kp == nil || kp.KanbanQty <= 0 {
				return nil, apperror.UnprocessableEntity("kanban_qty is required (kanban_parameters not found for item_uniq_code)")
			}
			pcsPerKanban = kp.KanbanQty
		}
		if kp == nil || strings.TrimSpace(kp.KanbanNumber) == "" {
			return nil, apperror.UnprocessableEntity("kanban_parameters not found for item_uniq_code")
		}

		qty := round4(it.Quantity)
		if qty <= 0 {
			return nil, apperror.UnprocessableEntity("quantity must be greater than 0")
		}
		perKanban := float64(pcsPerKanban)
		kanbanCount := int(math.Ceil(qty / perKanban))
		if kanbanCount <= 0 {
			kanbanCount = 1
		}

		for i := 0; i < kanbanCount; i++ {
			perUniqSeq[it.ItemUniqCode]++
			seq := perUniqSeq[it.ItemUniqCode]
			kanbanNumber := fmt.Sprintf("%s-%s-%s-%02d", kp.KanbanNumber, it.ItemUniqCode, woNumber, seq)
			kpNum := kp.KanbanNumber
			kpSeq := seq
			outItems = append(outItems, woModels.WorkOrderPreviewItem{
				ItemUniqCode:      it.ItemUniqCode,
				UOM:               it.UOM,
				ProcessName:       it.ProcessName,
				Quantity:          round4(perKanban),
				KanbanNumber:      kanbanNumber,
				KanbanParamNumber: &kpNum,
				KanbanSeq:         &kpSeq,
			})
		}
	}

	n := "preview only; wo_number and kanban_number may change on save"
	return &woModels.WorkOrderPreviewResponse{
		PreviewID: woID.String(),
		WoID:      woID.String(),
		WoNumber:  woNumber,
		Items:     outItems,
		Notes:     &n,
	}, nil
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
		WOKind:         workOrderKindStandard,
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
			ID:              it.UUID.String(),
			ItemUniqCode:    it.ItemUniqCode,
			PartName:        it.PartName,
			PartNumber:      it.PartNumber,
			Model:           it.Model,
			Quantity:        it.Quantity,
			UOM:             it.UOM,
			Status:          it.Status,
			ProcessFlowJSON: jsonRawOrEmpty(it.ProcessFlowJSON),
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
			WOKind:         r.WOKind,
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

func (s *service) ListBulk(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.WorkOrderListResponse, error) {
	f := woRepo.ListFilter{
		Search:         p.Search,
		Status:         p.Status,
		ApprovalStatus: p.ApprovalStatus,
		WOType:         p.WOType,
		WOKind:         workOrderKindBulk,
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
	woIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		woIDs = append(woIDs, r.ID)
	}
	itemRows, err := s.repo.GetItemsByWOIDs(ctx, woIDs)
	if err != nil {
		return nil, err
	}
	itemsByWO := make(map[int64][]woModels.WorkOrderListItemDetail, len(rows))
	for _, it := range itemRows {
		itemsByWO[it.WoID] = append(itemsByWO[it.WoID], woModels.WorkOrderListItemDetail{
			ID:              it.UUID.String(),
			ItemUniqCode:    it.ItemUniqCode,
			PartName:        it.PartName,
			PartNumber:      it.PartNumber,
			Model:           it.Model,
			Quantity:        it.Quantity,
			UOM:             it.UOM,
			Status:          it.Status,
			ProcessFlowJSON: jsonRawOrEmpty(it.ProcessFlowJSON),
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
			WOKind:         r.WOKind,
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
	return &woModels.WorkOrderListResponse{Items: items, Pagination: pagination.NewMeta(total, p.PaginationInput)}, nil
}

func (s *service) ListRMProcessing(ctx context.Context, p pagination.WorkOrderPaginationInput) (*woModels.RMProcessingWorkOrderListResponse, error) {
	f := woRepo.ListFilter{
		Search:         p.Search,
		Status:         p.Status,
		ApprovalStatus: p.ApprovalStatus,
		WOKind:         workOrderKindRMProcessing,
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

	items := make([]woModels.RMProcessingWorkOrderListItem, 0, len(rows))
	for _, r := range rows {
		var qrDataURL *string
		if woQR, err := generateQRDataURL(qrPayloadWO(r.WoNumber)); err == nil {
			qrDataURL = &woQR
		}
		source := ""
		target := ""
		inputUOM := ""
		outputUOM := ""
		inputQty := 0.0
		outputQty := 0.0
		if r.SourceMaterialUniq != nil {
			source = *r.SourceMaterialUniq
		}
		if r.TargetMaterialUniq != nil {
			target = *r.TargetMaterialUniq
		}
		if r.InputUOM != nil {
			inputUOM = *r.InputUOM
		}
		if r.OutputUOM != nil {
			outputUOM = *r.OutputUOM
		}
		if r.InputQty != nil {
			inputQty = *r.InputQty
		}
		if r.OutputQty != nil {
			outputQty = *r.OutputQty
		}
		dateIssued := r.CreatedDate
		if r.DateIssued != nil && strings.TrimSpace(*r.DateIssued) != "" {
			dateIssued = *r.DateIssued
		}
		items = append(items, woModels.RMProcessingWorkOrderListItem{
			ID:                 r.UUID,
			WoNumber:           r.WoNumber,
			WoType:             r.WoType,
			WOKind:             r.WOKind,
			Status:             r.Status,
			ApprovalStatus:     r.ApprovalStatus,
			CreatedDate:        r.CreatedDate,
			CreatedByName:      r.CreatedByName,
			SourceMaterialUniq: source,
			TargetMaterialUniq: target,
			Model:              r.Model,
			GradeSize:          r.GradeSize,
			InputQty:           inputQty,
			InputUOM:           inputUOM,
			OutputQty:          outputQty,
			OutputUOM:          outputUOM,
			DateIssued:         dateIssued,
			DateCompleted:      r.DateCompleted,
			CycleTimeDays:      r.CycleTimeDays,
			Remarks:            r.Remarks,
			AgingDays:          r.AgingDays,
			QRDataURL:          qrDataURL,
		})
	}

	return &woModels.RMProcessingWorkOrderListResponse{
		Items:      items,
		Pagination: pagination.NewMeta(total, p.PaginationInput),
	}, nil
}

func (s *service) GetSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error) {
	row, err := s.repo.GetSummaryByKind(ctx, workOrderKindStandard)
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

func (s *service) GetBulkSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error) {
	row, err := s.repo.GetSummaryByKind(ctx, workOrderKindBulk)
	if err != nil {
		return nil, err
	}
	return &woModels.WorkOrderSummaryResponse{ActiveWOs: row.ActiveWOs, Completed: row.Completed, PendingWOs: row.PendingWOs, TotalUniqs: row.TotalUniqs}, nil
}

func (s *service) GetRMProcessingSummary(ctx context.Context) (*woModels.WorkOrderSummaryResponse, error) {
	row, err := s.repo.GetSummaryByKind(ctx, workOrderKindRMProcessing)
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
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindStandard)
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
			ID:              it.UUID.String(),
			WoItemID:        it.UUID.String(),
			KanbanNumber:    it.KanbanNumber,
			ItemUniqCode:    it.ItemUniqCode,
			Quantity:        it.Quantity,
			UOM:             it.UOM,
			ProcessName:     it.ProcessName,
			Status:          it.Status,
			ProcessFlowJSON: jsonRawOrEmpty(it.ProcessFlowJSON),
			QRDataURL:       itemQR,
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

func (s *service) GetBulkDetail(ctx context.Context, woUUID string) (*woModels.WorkOrderDetailResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindBulk)
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
		} else if qr, err := generateQRDataURL(qrPayloadWOItem(it.KanbanNumber)); err == nil {
			itemQR = &qr
		}
		items = append(items, woModels.WorkOrderDetailItem{ID: it.UUID.String(), WoItemID: it.UUID.String(), KanbanNumber: it.KanbanNumber, ItemUniqCode: it.ItemUniqCode, Quantity: it.Quantity, UOM: it.UOM, ProcessName: it.ProcessName, Status: it.Status, ProcessFlowJSON: jsonRawOrEmpty(it.ProcessFlowJSON), QRDataURL: itemQR})
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
	} else if qr, err := generateQRDataURL(qrPayloadWO(wo.WoNumber)); err == nil {
		woQR = &qr
	}
	return &woModels.WorkOrderDetailResponse{ID: wo.UUID.String(), WoNumber: wo.WoNumber, WoType: wo.WoType, ReferenceWO: wo.ReferenceWO, Status: wo.Status, ApprovalStatus: wo.ApprovalStatus, CreatedDate: createdDate, TargetDate: targetDate, CreatedByName: wo.CreatedByName, Notes: wo.Notes, QRDataURL: woQR, Items: items}, nil
}

func (s *service) GetRMProcessingDetail(ctx context.Context, woUUID string) (*woModels.RMProcessingWorkOrderDetailResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindRMProcessing)
	if err != nil {
		return nil, err
	}

	createdDate := wo.CreatedDate.Format("2006-01-02")
	var dateCompleted *string
	if wo.DateCompleted != nil {
		s := wo.DateCompleted.Format("2006-01-02")
		dateCompleted = &s
	}
	dateIssued := createdDate
	if wo.DateIssued != nil {
		dateIssued = wo.DateIssued.Format("2006-01-02")
	}
	source := ""
	target := ""
	inputUOM := ""
	outputUOM := ""
	inputQty := 0.0
	outputQty := 0.0
	if wo.SourceMaterialUniq != nil {
		source = *wo.SourceMaterialUniq
	}
	if wo.TargetMaterialUniq != nil {
		target = *wo.TargetMaterialUniq
	}
	if wo.InputUOM != nil {
		inputUOM = *wo.InputUOM
	}
	if wo.OutputUOM != nil {
		outputUOM = *wo.OutputUOM
	}
	if wo.InputQty != nil {
		inputQty = *wo.InputQty
	}
	if wo.OutputQty != nil {
		outputQty = *wo.OutputQty
	}

	var woQR *string
	if wo.QRImageBase64 != nil && strings.TrimSpace(*wo.QRImageBase64) != "" {
		woQR = wo.QRImageBase64
	} else {
		if qr, err := generateQRDataURL(qrPayloadWO(wo.WoNumber)); err == nil {
			woQR = &qr
		}
	}

	return &woModels.RMProcessingWorkOrderDetailResponse{
		ID:                 wo.UUID.String(),
		WoNumber:           wo.WoNumber,
		WoType:             wo.WoType,
		WOKind:             wo.WOKind,
		Status:             wo.Status,
		ApprovalStatus:     wo.ApprovalStatus,
		CreatedDate:        createdDate,
		CreatedByName:      wo.CreatedByName,
		SourceMaterialUniq: source,
		TargetMaterialUniq: target,
		Model:              wo.Model,
		GradeSize:          wo.GradeSize,
		InputQty:           inputQty,
		InputUOM:           inputUOM,
		OutputQty:          outputQty,
		OutputUOM:          outputUOM,
		DateIssued:         dateIssued,
		DateCompleted:      dateCompleted,
		CycleTimeDays:      wo.CycleTimeDays,
		Remarks:            wo.Remarks,
		Notes:              wo.Notes,
		QRDataURL:          woQR,
	}, nil
}

func (s *service) Approval(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindStandard)
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

func (s *service) ApprovalBulk(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindBulk)
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
	if req.Decision == "approve" {
		items, err := s.repo.GetWorkOrderItemsByWOID(ctx, wo.ID)
		if err == nil && len(items) > 0 {
			consumeItems := make([]invService.ConsumeItem, 0, len(items))
			for _, it := range items {
				consumeItems = append(consumeItems, invService.ConsumeItem{UniqCode: it.ItemUniqCode, Qty: it.Quantity})
			}
			_ = s.invSvc.ConsumeStockForWorkOrder(ctx, consumeItems, wo.WoNumber, performedBy)
		}
	}
	return &woModels.WorkOrderApprovalResponse{ID: wo.UUID.String(), WoNumber: wo.WoNumber, ApprovalStatus: newStatus}, nil
}

func (s *service) ApprovalRMProcessing(ctx context.Context, woUUID string, req woModels.WorkOrderApprovalRequest, performedBy string) (*woModels.WorkOrderApprovalResponse, error) {
	wo, err := s.repo.GetWorkOrderByUUIDAndKind(ctx, woUUID, workOrderKindRMProcessing)
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

	_ = performedBy
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

	found, err := s.repo.FindWorkOrdersByWONumbersAndKind(ctx, req.WONumbers, workOrderKindStandard)
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

func (s *service) BulkApprovalBulk(ctx context.Context, req woModels.BulkWorkOrderApprovalRequest, performedBy string) (*woModels.BulkWorkOrderApprovalResponse, error) {
	newStatus := ""
	switch req.Decision {
	case "approve":
		newStatus = "Approved"
	case "reject":
		newStatus = "Rejected"
	default:
		return nil, apperror.BadRequest("decision must be approve or reject")
	}
	found, err := s.repo.FindWorkOrdersByWONumbersAndKind(ctx, req.WONumbers, workOrderKindBulk)
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
		if req.Decision == "approve" {
			items, iErr := s.repo.GetWorkOrderItemsByWOID(ctx, wo.ID)
			if iErr == nil && len(items) > 0 {
				consumeItems := make([]invService.ConsumeItem, 0, len(items))
				for _, it := range items {
					consumeItems = append(consumeItems, invService.ConsumeItem{UniqCode: it.ItemUniqCode, Qty: it.Quantity})
				}
				_ = s.invSvc.ConsumeStockForWorkOrder(ctx, consumeItems, wo.WoNumber, performedBy)
			}
		}
		updated = append(updated, woModels.BulkApprovalResultItem{WoNumber: woNumber, ApprovalStatus: newStatus})
	}
	return &woModels.BulkWorkOrderApprovalResponse{Decision: req.Decision, TotalRequested: len(req.WONumbers), TotalUpdated: len(updated), Updated: updated, Failed: failed}, nil
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

func strPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	v = strings.TrimSpace(v)
	return &v
}

func floatPtr(v float64) *float64 {
	v = round4(v)
	return &v
}

func jsonRawOrEmpty(v datatypes.JSON) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage("[]")
	}
	trimmed := strings.TrimSpace(string(v))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return json.RawMessage("[]")
	}
	return json.RawMessage(v)
}

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

// processFlowStep is a lightweight snapshot of one routing step written into
// work_order_items.process_flow_json at WO creation time.
type processFlowStep struct {
	OpSeq        int      `json:"op_seq"`
	ProcessName  string   `json:"process_name"`
	MachineName  *string  `json:"machine_name"`
	CycleTimeSec *float64 `json:"cycle_time_sec"`
	SetupTimeMin *float64 `json:"setup_time_min"`
}

// fetchProcessFlows returns a map from uniq_code → marshalled JSON array of
// processFlowStep, sourced from items → routing_headers → routing_operations
// → process_parameters.  If no routing is found for an item, the value is []byte("[]").
func (s *service) fetchProcessFlows(ctx context.Context, tx *gorm.DB, uniqCodes []string) (map[string]datatypes.JSON, error) {
	out := make(map[string]datatypes.JSON, len(uniqCodes))
	for _, c := range uniqCodes {
		out[c] = datatypes.JSON([]byte("[]"))
	}
	if len(uniqCodes) == 0 {
		return out, nil
	}

	type opRow struct {
		UniqCode     string   `gorm:"column:uniq_code"`
		OpSeq        int      `gorm:"column:op_seq"`
		ProcessName  string   `gorm:"column:process_name"`
		MachineName  *string  `gorm:"column:machine_name"`
		CycleTimeSec *float64 `gorm:"column:cycle_time_sec"`
		SetupTimeMin *float64 `gorm:"column:setup_time_min"`
	}

	rawSQL := `
		SELECT DISTINCT ON (i.uniq_code, ro.op_seq)
		       i.uniq_code,
		       ro.op_seq,
		       pp.process_name,
		       mm.machine_name     AS machine_name,
		       ro.cycle_time_sec,
		       ro.setup_time_min
		FROM   items i
		JOIN   routing_headers rh ON rh.item_id = i.id
		JOIN   routing_operations ro ON ro.routing_header_id = rh.id
		JOIN   process_parameters pp ON pp.id = ro.process_id
		LEFT   JOIN master_machines mm ON mm.id = ro.machine_id
		WHERE  i.uniq_code IN ?
		  AND  i.deleted_at IS NULL
		ORDER  BY i.uniq_code, ro.op_seq, rh.id DESC`

	var opRows []opRow
	db := s.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	if err := db.Raw(rawSQL, uniqCodes).Scan(&opRows).Error; err != nil {
		return nil, apperror.InternalWrap("fetchProcessFlows", err)
	}

	grouped := make(map[string][]processFlowStep, len(uniqCodes))
	for _, r := range opRows {
		grouped[r.UniqCode] = append(grouped[r.UniqCode], processFlowStep{
			OpSeq:        r.OpSeq,
			ProcessName:  r.ProcessName,
			MachineName:  r.MachineName,
			CycleTimeSec: r.CycleTimeSec,
			SetupTimeMin: r.SetupTimeMin,
		})
	}
	for uniq, steps := range grouped {
		b, err := json.Marshal(steps)
		if err != nil {
			return nil, apperror.InternalWrap("fetchProcessFlows marshal", err)
		}
		out[uniq] = datatypes.JSON(b)
	}
	return out, nil
}
