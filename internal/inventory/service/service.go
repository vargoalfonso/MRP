package service

import (
	"context"
	"strings"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	"github.com/ganasa18/go-template/internal/inventory/repository"
	"github.com/ganasa18/go-template/pkg/inventoryconst"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IService interface {
	// Raw Material
	ListRawMaterials(ctx context.Context, p pagination.InventoryRMPaginationInput) (*invModels.RawMaterialListResponse, error)
	GetRawMaterialByID(ctx context.Context, id int64) (*invModels.RawMaterial, error)
	CreateRawMaterial(ctx context.Context, req invModels.CreateRawMaterialRequest, createdBy string) (*invModels.RawMaterialItem, error)
	BulkCreateRawMaterials(ctx context.Context, req invModels.BulkCreateRawMaterialRequest, createdBy string) (int, error)
	UpdateRawMaterial(ctx context.Context, id int64, req invModels.UpdateRawMaterialRequest, updatedBy string) (*invModels.RawMaterial, error)
	DeleteRawMaterial(ctx context.Context, id int64, deletedBy string) error
	GetRawMaterialHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error)

	// Indirect Raw Material
	ListIndirectMaterials(ctx context.Context, p pagination.InventoryIndirectPaginationInput) (*invModels.IndirectMaterialListResponse, error)
	GetIndirectByID(ctx context.Context, id int64) (*invModels.IndirectRawMaterial, error)
	CreateIndirectMaterial(ctx context.Context, req invModels.CreateIndirectMaterialRequest, createdBy string) (*invModels.IndirectMaterialItem, error)
	BulkCreateIndirectMaterials(ctx context.Context, req invModels.BulkCreateIndirectMaterialRequest, createdBy string) (int, error)
	UpdateIndirectMaterial(ctx context.Context, id int64, req invModels.UpdateIndirectMaterialRequest, updatedBy string) (*invModels.IndirectRawMaterial, error)
	DeleteIndirectMaterial(ctx context.Context, id int64, deletedBy string) error
	GetIndirectHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error)

	// Subcon Inventory
	ListSubconInventory(ctx context.Context, p pagination.InventorySubconPaginationInput) (*invModels.SubconInventoryListResponse, error)
	GetSubconByID(ctx context.Context, id int64) (*invModels.SubconInventory, error)
	CreateSubconInventory(ctx context.Context, req invModels.CreateSubconInventoryRequest, createdBy string) (*invModels.SubconInventoryItem, error)
	UpdateSubconInventory(ctx context.Context, id int64, req invModels.UpdateSubconInventoryRequest, updatedBy string) (*invModels.SubconInventory, error)
	DeleteSubconInventory(ctx context.Context, id int64, deletedBy string) error
	GetSubconHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error)

	// Incoming scans (tab view - shared)
	ListIncoming(ctx context.Context, dnType string, p pagination.InventoryIncomingPaginationInput) (*invModels.IncomingListResponse, error)

	// Kanban summary — per item_uniq_code, called async per row by frontend
	GetKanbanSummary(ctx context.Context, uniqCode string) (*invModels.KanbanSummary, error)

	// Work Order consumption — deducts stock and writes outgoing movement logs for each WO item.
	ConsumeStockForWorkOrder(ctx context.Context, items []ConsumeItem, woNumber string, performedBy string) error
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService { return &service{repo: repo} }

// ---------------------------------------------------------------------------
// Raw Material
// ---------------------------------------------------------------------------

func (s *service) ListRawMaterials(ctx context.Context, p pagination.InventoryRMPaginationInput) (*invModels.RawMaterialListResponse, error) {
	f := repository.ListFilter{
		Search:         p.Search,
		RMType:         p.RMType,
		RMSource:       p.RMSource,
		Status:         p.Status,
		BuyNotBuy:      p.BuyNotBuy,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	}

	rows, total, err := s.repo.ListRawMaterials(ctx, f)
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.GetRawMaterialStats(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]invModels.RawMaterialItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, rawMaterialRowToItem(r))
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &invModels.RawMaterialListResponse{
		Stats: invModels.InventoryStats{
			TotalItems:         stats.TotalItems,
			BuyRecommendations: stats.BuyRecommendations,
			LowStockItems:      stats.LowStockItems,
		},
		Items: items,
		Pagination: invModels.InventoryPagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetRawMaterialByID(ctx context.Context, id int64) (*invModels.RawMaterial, error) {
	return s.repo.GetRawMaterialByID(ctx, id)
}

func (s *service) CreateRawMaterial(ctx context.Context, req invModels.CreateRawMaterialRequest, createdBy string) (*invModels.RawMaterialItem, error) {
	// Auto-fill part fields from items table when missing.
	lookupKey := strings.TrimSpace(req.UniqCode)
	if req.ItemUniqCode != nil && strings.TrimSpace(*req.ItemUniqCode) != "" {
		lookupKey = strings.TrimSpace(*req.ItemUniqCode)
	}
	itemRow, err := s.repo.FindItemByUniqCode(ctx, lookupKey)
	if err != nil {
		return nil, err
	}
	partNumber := req.PartNumber
	partName := req.PartName
	uom := req.UOM
	var itemID *int64
	if itemRow != nil {
		itemID = &itemRow.ID
		if partNumber == nil {
			partNumber = itemRow.PartNumber
		}
		if partName == nil {
			partName = itemRow.PartName
		}
		if uom == nil {
			uom = itemRow.UOM
		}
	}

	now := time.Now()
	rm := invModels.RawMaterial{
		UUID:              uuid.New(),
		UniqCode:          req.UniqCode,
		RawMaterialType:   req.RawMaterialType,
		RMSource:          req.RMSource,
		PartNumber:        partNumber,
		PartName:          partName,
		WarehouseLocation: req.WarehouseLocation,
		UOM:               uom,
		ItemID:            itemID,
		StockQty:          req.StockQty,
		StockWeightKg:     req.StockWeightKg,
		KanbanCount:       req.KanbanCount,
		KanbanStandardQty: req.KanbanStandardQty,
		SafetyStockQty:    req.SafetyStockQty,
		DailyUsageQty:     req.DailyUsageQty,
		Status:            "normal",
		BuyNotBuy:         "not_buy",
		CreatedBy:         &createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateRawMaterial(ctx, &rm); err != nil {
		return nil, err
	}
	s.writeMovementLog(ctx, MovementLogInput{
		Category:     "raw_material",
		MovementType: "incoming",
		UniqCode:     rm.UniqCode,
		EntityID:     &rm.ID,
		QtyChange:    rm.StockQty,
		WeightChange: rm.StockWeightKg,
		SourceFlag:   "manual",
		LoggedBy:     createdBy,
	})
	item := rawMaterialModelToItem(rm)
	return &item, nil
}

func (s *service) BulkCreateRawMaterials(ctx context.Context, req invModels.BulkCreateRawMaterialRequest, createdBy string) (int, error) {
	now := time.Now()
	items := make([]invModels.RawMaterial, 0, len(req.Items))
	for _, r := range req.Items {
		lookupKey := strings.TrimSpace(r.UniqCode)
		if r.ItemUniqCode != nil && strings.TrimSpace(*r.ItemUniqCode) != "" {
			lookupKey = strings.TrimSpace(*r.ItemUniqCode)
		}
		itemRow, err := s.repo.FindItemByUniqCode(ctx, lookupKey)
		if err != nil {
			return 0, err
		}
		partNumber := r.PartNumber
		partName := r.PartName
		uom := r.UOM
		var itemID *int64
		if itemRow != nil {
			itemID = &itemRow.ID
			if partNumber == nil {
				partNumber = itemRow.PartNumber
			}
			if partName == nil {
				partName = itemRow.PartName
			}
			if uom == nil {
				uom = itemRow.UOM
			}
		}

		items = append(items, invModels.RawMaterial{
			UUID:              uuid.New(),
			UniqCode:          r.UniqCode,
			RawMaterialType:   r.RawMaterialType,
			RMSource:          r.RMSource,
			PartNumber:        partNumber,
			PartName:          partName,
			WarehouseLocation: r.WarehouseLocation,
			UOM:               uom,
			ItemID:            itemID,
			StockQty:          r.StockQty,
			StockWeightKg:     r.StockWeightKg,
			KanbanCount:       r.KanbanCount,
			KanbanStandardQty: r.KanbanStandardQty,
			SafetyStockQty:    r.SafetyStockQty,
			DailyUsageQty:     r.DailyUsageQty,
			Status:            "normal",
			BuyNotBuy:         "not_buy",
			CreatedBy:         &createdBy,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	if err := s.repo.BulkCreateRawMaterials(ctx, items); err != nil {
		return 0, err
	}
	for _, rm := range items {
		rmCopy := rm
		s.writeMovementLog(ctx, MovementLogInput{
			Category:     "raw_material",
			MovementType: "incoming",
			UniqCode:     rmCopy.UniqCode,
			EntityID:     &rmCopy.ID,
			QtyChange:    rmCopy.StockQty,
			WeightChange: rmCopy.StockWeightKg,
			SourceFlag:   "manual",
			LoggedBy:     createdBy,
		})
	}
	return len(items), nil
}

func (s *service) UpdateRawMaterial(ctx context.Context, id int64, req invModels.UpdateRawMaterialRequest, updatedBy string) (*invModels.RawMaterial, error) {
	updates := map[string]interface{}{"updated_by": &updatedBy, "updated_at": time.Now()}
	if req.RawMaterialType != nil {
		updates["raw_material_type"] = *req.RawMaterialType
	}
	if req.RMSource != nil {
		updates["rm_source"] = *req.RMSource
	}
	if req.PartNumber != nil {
		updates["part_number"] = *req.PartNumber
	}
	if req.PartName != nil {
		updates["part_name"] = *req.PartName
	}
	if req.WarehouseLocation != nil {
		updates["warehouse_location"] = *req.WarehouseLocation
	}
	if req.UOM != nil {
		updates["uom"] = *req.UOM
	}
	if req.StockQty != nil {
		updates["stock_qty"] = *req.StockQty
	}
	if req.StockWeightKg != nil {
		updates["stock_weight_kg"] = *req.StockWeightKg
	}
	if req.KanbanCount != nil {
		updates["kanban_count"] = *req.KanbanCount
	}
	if req.KanbanStandardQty != nil {
		updates["kanban_standard_qty"] = *req.KanbanStandardQty
	}
	if req.SafetyStockQty != nil {
		updates["safety_stock_qty"] = *req.SafetyStockQty
	}
	if req.DailyUsageQty != nil {
		updates["daily_usage_qty"] = *req.DailyUsageQty
	}
	return s.repo.UpdateRawMaterial(ctx, id, updates)
}

func (s *service) DeleteRawMaterial(ctx context.Context, id int64, deletedBy string) error {
	return s.repo.SoftDeleteRawMaterial(ctx, id, deletedBy)
}

func (s *service) GetRawMaterialHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error) {
	rm, err := s.repo.GetRawMaterialByID(ctx, id)
	if err != nil {
		return nil, err
	}
	f := repository.ListFilter{
		Limit:  p.Limit,
		Offset: p.Offset(),
	}
	rows, total, err := s.repo.GetMovementHistory(ctx, "raw_material", rm.UniqCode, f)
	if err != nil {
		return nil, err
	}
	return buildHistoryResponse(rows, total, p), nil
}

// ---------------------------------------------------------------------------
// Indirect Raw Material
// ---------------------------------------------------------------------------

func (s *service) ListIndirectMaterials(ctx context.Context, p pagination.InventoryIndirectPaginationInput) (*invModels.IndirectMaterialListResponse, error) {
	f := repository.ListFilter{
		Search:         p.Search,
		Status:         p.Status,
		BuyNotBuy:      p.BuyNotBuy,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	}

	rows, total, err := s.repo.ListIndirectMaterials(ctx, f)
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.GetIndirectStats(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]invModels.IndirectMaterialItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, indirectRowToItem(r))
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &invModels.IndirectMaterialListResponse{
		Stats: invModels.InventoryStats{
			TotalItems:         stats.TotalItems,
			BuyRecommendations: stats.BuyRecommendations,
			LowStockItems:      stats.LowStockItems,
		},
		Items: items,
		Pagination: invModels.InventoryPagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetIndirectByID(ctx context.Context, id int64) (*invModels.IndirectRawMaterial, error) {
	return s.repo.GetIndirectByID(ctx, id)
}

func (s *service) CreateIndirectMaterial(ctx context.Context, req invModels.CreateIndirectMaterialRequest, createdBy string) (*invModels.IndirectMaterialItem, error) {
	lookupKey := strings.TrimSpace(req.UniqCode)
	if req.ItemUniqCode != nil && strings.TrimSpace(*req.ItemUniqCode) != "" {
		lookupKey = strings.TrimSpace(*req.ItemUniqCode)
	}
	itemRow, err := s.repo.FindItemByUniqCode(ctx, lookupKey)
	if err != nil {
		return nil, err
	}
	partNumber := req.PartNumber
	partName := req.PartName
	uom := req.UOM
	var itemID *int64
	if itemRow != nil {
		itemID = &itemRow.ID
		if partNumber == nil {
			partNumber = itemRow.PartNumber
		}
		if partName == nil {
			partName = itemRow.PartName
		}
		if uom == nil {
			uom = itemRow.UOM
		}
	}

	now := time.Now()
	statusPtr := "normal"
	irm := invModels.IndirectRawMaterial{
		UUID:              uuid.New(),
		UniqCode:          req.UniqCode,
		PartNumber:        partNumber,
		PartName:          partName,
		WarehouseLocation: req.WarehouseLocation,
		UOM:               uom,
		ItemID:            itemID,
		StockQty:          req.StockQty,
		StockWeightKg:     req.StockWeightKg,
		KanbanCount:       req.KanbanCount,
		KanbanStandardQty: req.KanbanStandardQty,
		SafetyStockQty:    req.SafetyStockQty,
		DailyUsageQty:     req.DailyUsageQty,
		Status:            &statusPtr,
		BuyNotBuy:         "not_buy",
		CreatedBy:         &createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.CreateIndirectMaterial(ctx, &irm); err != nil {
		return nil, err
	}
	s.writeMovementLog(ctx, MovementLogInput{
		Category:     "indirect_raw_material",
		MovementType: "incoming",
		UniqCode:     irm.UniqCode,
		EntityID:     &irm.ID,
		QtyChange:    irm.StockQty,
		WeightChange: irm.StockWeightKg,
		SourceFlag:   "manual",
		LoggedBy:     createdBy,
	})
	item := indirectModelToItem(irm)
	return &item, nil
}

func (s *service) BulkCreateIndirectMaterials(ctx context.Context, req invModels.BulkCreateIndirectMaterialRequest, createdBy string) (int, error) {
	now := time.Now()
	statusPtr := "normal"
	items := make([]invModels.IndirectRawMaterial, 0, len(req.Items))
	for _, r := range req.Items {
		lookupKey := strings.TrimSpace(r.UniqCode)
		if r.ItemUniqCode != nil && strings.TrimSpace(*r.ItemUniqCode) != "" {
			lookupKey = strings.TrimSpace(*r.ItemUniqCode)
		}
		itemRow, err := s.repo.FindItemByUniqCode(ctx, lookupKey)
		if err != nil {
			return 0, err
		}
		partNumber := r.PartNumber
		partName := r.PartName
		uom := r.UOM
		var itemID *int64
		if itemRow != nil {
			itemID = &itemRow.ID
			if partNumber == nil {
				partNumber = itemRow.PartNumber
			}
			if partName == nil {
				partName = itemRow.PartName
			}
			if uom == nil {
				uom = itemRow.UOM
			}
		}

		sp := statusPtr
		items = append(items, invModels.IndirectRawMaterial{
			UUID:              uuid.New(),
			UniqCode:          r.UniqCode,
			PartNumber:        partNumber,
			PartName:          partName,
			WarehouseLocation: r.WarehouseLocation,
			UOM:               uom,
			ItemID:            itemID,
			StockQty:          r.StockQty,
			StockWeightKg:     r.StockWeightKg,
			KanbanCount:       r.KanbanCount,
			KanbanStandardQty: r.KanbanStandardQty,
			SafetyStockQty:    r.SafetyStockQty,
			DailyUsageQty:     r.DailyUsageQty,
			Status:            &sp,
			BuyNotBuy:         "not_buy",
			CreatedBy:         &createdBy,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}
	if err := s.repo.BulkCreateIndirectMaterials(ctx, items); err != nil {
		return 0, err
	}
	for _, irm := range items {
		irmCopy := irm
		s.writeMovementLog(ctx, MovementLogInput{
			Category:     "indirect_raw_material",
			MovementType: "incoming",
			UniqCode:     irmCopy.UniqCode,
			EntityID:     &irmCopy.ID,
			QtyChange:    irmCopy.StockQty,
			WeightChange: irmCopy.StockWeightKg,
			SourceFlag:   "manual",
			LoggedBy:     createdBy,
		})
	}
	return len(items), nil
}

func (s *service) UpdateIndirectMaterial(ctx context.Context, id int64, req invModels.UpdateIndirectMaterialRequest, updatedBy string) (*invModels.IndirectRawMaterial, error) {
	updates := map[string]interface{}{"updated_by": &updatedBy, "updated_at": time.Now()}
	if req.PartNumber != nil {
		updates["part_number"] = *req.PartNumber
	}
	if req.PartName != nil {
		updates["part_name"] = *req.PartName
	}
	if req.WarehouseLocation != nil {
		updates["warehouse_location"] = *req.WarehouseLocation
	}
	if req.UOM != nil {
		updates["uom"] = *req.UOM
	}
	if req.StockQty != nil {
		updates["stock_qty"] = *req.StockQty
	}
	if req.StockWeightKg != nil {
		updates["stock_weight_kg"] = *req.StockWeightKg
	}
	if req.KanbanCount != nil {
		updates["kanban_count"] = *req.KanbanCount
	}
	if req.KanbanStandardQty != nil {
		updates["kanban_standard_qty"] = *req.KanbanStandardQty
	}
	if req.SafetyStockQty != nil {
		updates["safety_stock_qty"] = *req.SafetyStockQty
	}
	if req.DailyUsageQty != nil {
		updates["daily_usage_qty"] = *req.DailyUsageQty
	}
	return s.repo.UpdateIndirectMaterial(ctx, id, updates)
}

func (s *service) DeleteIndirectMaterial(ctx context.Context, id int64, deletedBy string) error {
	return s.repo.SoftDeleteIndirectMaterial(ctx, id, deletedBy)
}

func (s *service) GetIndirectHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error) {
	irm, err := s.repo.GetIndirectByID(ctx, id)
	if err != nil {
		return nil, err
	}
	f := repository.ListFilter{Limit: p.Limit, Offset: p.Offset()}
	rows, total, err := s.repo.GetMovementHistory(ctx, "indirect_raw_material", irm.UniqCode, f)
	if err != nil {
		return nil, err
	}
	return buildHistoryResponse(rows, total, p), nil
}

// ---------------------------------------------------------------------------
// Subcon Inventory
// ---------------------------------------------------------------------------

func (s *service) ListSubconInventory(ctx context.Context, p pagination.InventorySubconPaginationInput) (*invModels.SubconInventoryListResponse, error) {
	f := repository.SubconListFilter{
		Search:         p.Search,
		PONumber:       p.PONumber,
		SupplierID:     p.SupplierID,
		Period:         p.Period,
		Status:         p.Status,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	}

	rows, total, err := s.repo.ListSubconInventory(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]invModels.SubconInventoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, subconRowToItem(r))
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &invModels.SubconInventoryListResponse{
		Items: items,
		Pagination: invModels.InventoryPagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetSubconByID(ctx context.Context, id int64) (*invModels.SubconInventory, error) {
	return s.repo.GetSubconByID(ctx, id)
}

func (s *service) CreateSubconInventory(ctx context.Context, req invModels.CreateSubconInventoryRequest, createdBy string) (*invModels.SubconInventoryItem, error) {
	lookupKey := strings.TrimSpace(req.UniqCode)
	if req.ItemUniqCode != nil && strings.TrimSpace(*req.ItemUniqCode) != "" {
		lookupKey = strings.TrimSpace(*req.ItemUniqCode)
	}
	itemRow, err := s.repo.FindItemByUniqCode(ctx, lookupKey)
	if err != nil {
		return nil, err
	}
	partNumber := req.PartNumber
	partName := req.PartName
	if itemRow != nil {
		if partNumber == nil {
			partNumber = itemRow.PartNumber
		}
		if partName == nil {
			partName = itemRow.PartName
		}
	}

	now := time.Now()
	si := invModels.SubconInventory{
		UUID:             uuid.New(),
		UniqCode:         req.UniqCode,
		PartNumber:       partNumber,
		PartName:         partName,
		PONumber:         req.PONumber,
		POPeriod:         req.POPeriod,
		SubconVendorID:   req.SubconVendorID,
		SubconVendorName: req.SubconVendorName,
		StockAtVendorQty: req.StockAtVendorQty,
		TotalPOQty:       req.TotalPOQty,
		SafetyStockQty:   req.SafetyStockQty,
		DateDelivery:     req.DateDelivery,
		Status:           "normal",
		CreatedBy:        &createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateSubconInventory(ctx, &si); err != nil {
		return nil, err
	}
	s.writeMovementLog(ctx, MovementLogInput{
		Category:     "subcon",
		MovementType: "received_from_vendor",
		UniqCode:     si.UniqCode,
		EntityID:     &si.ID,
		QtyChange:    si.StockAtVendorQty,
		SourceFlag:   "manual",
		LoggedBy:     createdBy,
	})
	item := subconModelToItem(si)
	return &item, nil
}

func (s *service) UpdateSubconInventory(ctx context.Context, id int64, req invModels.UpdateSubconInventoryRequest, updatedBy string) (*invModels.SubconInventory, error) {
	updates := map[string]interface{}{"updated_by": &updatedBy, "updated_at": time.Now()}
	if req.PartNumber != nil {
		updates["part_number"] = *req.PartNumber
	}
	if req.PartName != nil {
		updates["part_name"] = *req.PartName
	}
	if req.PONumber != nil {
		updates["po_number"] = *req.PONumber
	}
	if req.POPeriod != nil {
		updates["po_period"] = *req.POPeriod
	}
	if req.SubconVendorID != nil {
		updates["subcon_vendor_id"] = *req.SubconVendorID
	}
	if req.SubconVendorName != nil {
		updates["subcon_vendor_name"] = *req.SubconVendorName
	}
	if req.StockAtVendorQty != nil {
		updates["stock_at_vendor_qty"] = *req.StockAtVendorQty
	}
	if req.TotalPOQty != nil {
		updates["total_po_qty"] = *req.TotalPOQty
	}
	if req.SafetyStockQty != nil {
		updates["safety_stock_qty"] = *req.SafetyStockQty
	}
	if req.DateDelivery != nil {
		updates["date_delivery"] = *req.DateDelivery
	}
	return s.repo.UpdateSubconInventory(ctx, id, updates)
}

func (s *service) DeleteSubconInventory(ctx context.Context, id int64, deletedBy string) error {
	return s.repo.SoftDeleteSubconInventory(ctx, id, deletedBy)
}

func (s *service) GetSubconHistory(ctx context.Context, id int64, p pagination.PaginationInput) (*invModels.HistoryLogResponse, error) {
	si, err := s.repo.GetSubconByID(ctx, id)
	if err != nil {
		return nil, err
	}
	f := repository.ListFilter{Limit: p.Limit, Offset: p.Offset()}
	rows, total, err := s.repo.GetMovementHistory(ctx, "subcon", si.UniqCode, f)
	if err != nil {
		return nil, err
	}
	return buildHistoryResponse(rows, total, p), nil
}

// ---------------------------------------------------------------------------
// Incoming Scans
// ---------------------------------------------------------------------------

func (s *service) ListIncoming(ctx context.Context, dnType string, p pagination.InventoryIncomingPaginationInput) (*invModels.IncomingListResponse, error) {
	// Allow caller to override dn_type from query; default to the path-injected dnType
	effectiveDNType := dnType
	if p.DNType != "" {
		effectiveDNType = p.DNType
	}

	f := repository.IncomingListFilter{
		Search:     p.Search,
		DNType:     effectiveDNType,
		PONumber:   p.PONumber,
		Status:     p.Status,
		SupplierID: p.SupplierID,
		Page:       p.Page,
		Limit:      p.Limit,
		Offset:     p.Offset(),
	}

	rows, total, err := s.repo.ListIncoming(ctx, f)
	if err != nil {
		return nil, err
	}

	items := make([]invModels.IncomingItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, invModels.IncomingItem{
			ScanID:          r.ScanID,
			UniqCode:        r.ItemUniqCode,
			IncomingQty:     r.IncomingQty,
			Warehouse:       r.Warehouse,
			ScanDate:        r.ScanDate,
			SupplierName:    r.SupplierName,
			PONumber:        r.PONumber,
			DNNumber:        r.DNNumber,
			QCStatus:        r.QCStatus,
			QCStatusDisplay: qcStatusDisplay(r.QCStatus),
			UOM:             r.UOM,
			DNType:          r.DNType,
		})
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	return &invModels.IncomingListResponse{
		Items: items,
		Pagination: invModels.InventoryPagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Kanban Summary
// ---------------------------------------------------------------------------

const (
	kanbanDefaultPkgQty = 10  // pcs per package until supplier package table is ready
	kanbanHighRatio     = 2.0 // overstock threshold = safety_stock * highRatio
)

func (s *service) GetKanbanSummary(ctx context.Context, uniqCode string) (*invModels.KanbanSummary, error) {
	// 1. Load only the persistent stock data from raw_materials.
	//    Derived/computed columns (safety_stock_qty, kanban_standard_qty, daily_usage_qty,
	//    status, stock_days, buy_not_buy) are intentionally NOT read here because they
	//    will be dropped — all values are computed from the parameter tables below.
	rm, err := s.repo.GetRawMaterialByUniqCode(ctx, uniqCode)
	if err != nil {
		// Item not found — return a zero summary so the frontend row still renders.
		return &invModels.KanbanSummary{
			UniqCode:  uniqCode,
			Status:    "normal",
			BuyNotBuy: "n/a",
		}, nil
	}

	stockQty := rm.StockQty

	// 2. Kanban pcs per package — from supplier_item.pcs_per_kanban, fallback to 50.
	kanbanPkgQty := kanbanDefaultPkgQty
	if pkgQty, err := s.repo.GetKanbanPkgQty(ctx, uniqCode); err == nil && pkgQty > 0 {
		kanbanPkgQty = pkgQty
	}
	kanbanPkgF := float64(kanbanPkgQty)

	// 3. Resolve daily_usage and safety_stock — three paths, pick one:
	//
	//   PATH A — safety_stock_parameter is active for this item:
	//     daily_usage  = (PRL + PO) / working_days
	//     safety_stock = daily_usage × constanta
	//
	//   PATH B — no parameter configured:
	//     daily_usage  = forecasted_usage_per_day from forecast_results (historical)
	//     safety_stock = 0  (no threshold → no buy recommendation yet)
	//
	//   PATH C — no parameter and no forecast:
	//     daily_usage  = daily_usage_qty manually set on the raw_material record
	//     safety_stock = 0

	safetyStockQty := 0.0
	var dailyUsage float64

	param, paramErr := s.repo.GetSafetyStockParam(ctx, uniqCode)
	if paramErr == nil && param != nil {
		// PATH A: parameter active — derive daily usage from PRL + PO demand.
		workingDays, _ := s.repo.GetActiveWorkingDays(ctx)
		if workingDays <= 0 {
			workingDays = 20
		}
		prl, _ := s.repo.GetPRLQtyByUniqCode(ctx, uniqCode)
		po, _ := s.repo.GetActivePOQtyByUniqCode(ctx, uniqCode)
		dailyUsage = (prl + po) / float64(workingDays)
		safetyStockQty = dailyUsage * param.Constanta
	} else {
		// PATH B: no parameter — fall back to historical forecast per day.
		dailyUsage, _ = s.repo.GetForecastDailyUsage(ctx, uniqCode)
		// PATH C: no forecast either — use manually set daily_usage_qty from the record.
		if dailyUsage <= 0 && rm.DailyUsageQty != nil && *rm.DailyUsageQty > 0 {
			dailyUsage = *rm.DailyUsageQty
		}
	}

	// 4. Kanban metrics.
	totalKanban := int64(stockQty / kanbanPkgF)

	deficit := safetyStockQty - stockQty
	var kanbansNeeded int64
	var stockToComplete float64
	if deficit > 0 {
		kanbansNeeded = int64((deficit + kanbanPkgF - 1) / kanbanPkgF) // ceil
		stockToComplete = float64(kanbansNeeded) * kanbanPkgF
	}

	// 5. Status.
	status := "normal"
	if stockQty < safetyStockQty {
		status = "low_on_stock"
	} else if safetyStockQty > 0 && stockQty > safetyStockQty*kanbanHighRatio {
		status = "overstock"
	}

	// 6. Buy / Not Buy.
	buyNotBuy := "not_buy"
	if rm.RawMaterialType == "ssp" {
		buyNotBuy = "n/a"
	} else if stockQty < safetyStockQty {
		buyNotBuy = "buy"
	}

	// 7. Stock days = floor(stock / daily_usage_for_days).
	//    Prefer the manually set daily_usage_qty from the record (source of truth for days);
	//    fall back to the computed dailyUsage (from PRL+PO or forecast) if not set.
	var stockDays *int
	dailyUsageForDays := dailyUsage
	if rm.DailyUsageQty != nil && *rm.DailyUsageQty > 0 {
		dailyUsageForDays = *rm.DailyUsageQty
	}
	if dailyUsageForDays > 0 {
		d := int(stockQty / dailyUsageForDays)
		stockDays = &d
	}

	return &invModels.KanbanSummary{
		UniqCode:        uniqCode,
		StockQty:        stockQty,
		TotalKanban:     totalKanban,
		KanbansNeeded:   kanbansNeeded,
		StockToComplete: stockToComplete,
		KanbanPkgQty:    kanbanPkgQty,
		SafetyStockQty:  safetyStockQty,
		Status:          status,
		BuyNotBuy:       buyNotBuy,
		StockDays:       stockDays,
	}, nil
}

// ---------------------------------------------------------------------------
// Private mapping helpers
// ---------------------------------------------------------------------------

func rawMaterialRowToItem(r repository.RawMaterialRow) invModels.RawMaterialItem {
	return invModels.RawMaterialItem{
		ID:                    r.ID,
		UniqCode:              r.UniqCode,
		PartNumber:            r.PartNumber,
		PartName:              r.PartName,
		RawMaterialType:       r.RawMaterialType,
		RMSource:              r.RMSource,
		WarehouseLocation:     r.WarehouseLocation,
		UOM:                   r.UOM,
		StockQty:              r.StockQty,
		StockWeightKg:         r.StockWeightKg,
		KanbanCount:           r.KanbanCount,
		KanbanStandardQty:     r.KanbanStandardQty,
		SafetyStockQty:        r.SafetyStockQty,
		DailyUsageQty:         r.DailyUsageQty,
		Status:                r.Status,
		StockDays:             r.StockDays,
		BuyNotBuy:             r.BuyNotBuy,
		StockToCompleteKanban: r.StockToCompleteKanban,
		CreatedBy:             r.CreatedBy,
		CreatedAt:             r.CreatedAt,
		UpdatedBy:             r.UpdatedBy,
		UpdatedAt:             r.UpdatedAt,
	}
}

func indirectRowToItem(r repository.IndirectRow) invModels.IndirectMaterialItem {
	return invModels.IndirectMaterialItem{
		ID:                    r.ID,
		UniqCode:              r.UniqCode,
		PartNumber:            r.PartNumber,
		PartName:              r.PartName,
		WarehouseLocation:     r.WarehouseLocation,
		UOM:                   r.UOM,
		StockQty:              r.StockQty,
		StockWeightKg:         r.StockWeightKg,
		KanbanCount:           r.KanbanCount,
		KanbanStandardQty:     r.KanbanStandardQty,
		SafetyStockQty:        r.SafetyStockQty,
		DailyUsageQty:         r.DailyUsageQty,
		Status:                r.Status,
		StockDays:             r.StockDays,
		BuyNotBuy:             r.BuyNotBuy,
		StockToCompleteKanban: r.StockToCompleteKanban,
		CreatedBy:             r.CreatedBy,
		CreatedAt:             r.CreatedAt,
		UpdatedBy:             r.UpdatedBy,
		UpdatedAt:             r.UpdatedAt,
	}
}

func subconRowToItem(r repository.SubconRow) invModels.SubconInventoryItem {
	return invModels.SubconInventoryItem{
		ID:               r.ID,
		UniqCode:         r.UniqCode,
		PartNumber:       r.PartNumber,
		PartName:         r.PartName,
		PONumber:         r.PONumber,
		POPeriod:         r.POPeriod,
		SubconVendorID:   r.SubconVendorID,
		SubconVendorName: r.SubconVendorName,
		StockAtVendorQty: r.StockAtVendorQty,
		TotalPOQty:       r.TotalPOQty,
		TotalReceivedQty: r.TotalReceivedQty,
		DeltaPO:          r.DeltaPO,
		SafetyStockQty:   r.SafetyStockQty,
		DateDelivery:     r.DateDelivery,
		Status:           r.Status,
		CreatedBy:        r.CreatedBy,
		CreatedAt:        r.CreatedAt,
		UpdatedBy:        r.UpdatedBy,
		UpdatedAt:        r.UpdatedAt,
	}
}

// model-to-item mappers (for Create responses — model has no json tags, response struct does)
func rawMaterialModelToItem(m invModels.RawMaterial) invModels.RawMaterialItem {
	return invModels.RawMaterialItem{
		ID:                    m.ID,
		UniqCode:              m.UniqCode,
		PartNumber:            m.PartNumber,
		PartName:              m.PartName,
		RawMaterialType:       m.RawMaterialType,
		RMSource:              m.RMSource,
		WarehouseLocation:     m.WarehouseLocation,
		UOM:                   m.UOM,
		StockQty:              m.StockQty,
		StockWeightKg:         m.StockWeightKg,
		KanbanCount:           m.KanbanCount,
		KanbanStandardQty:     m.KanbanStandardQty,
		SafetyStockQty:        m.SafetyStockQty,
		DailyUsageQty:         m.DailyUsageQty,
		Status:                m.Status,
		StockDays:             m.StockDays,
		BuyNotBuy:             m.BuyNotBuy,
		StockToCompleteKanban: m.StockToCompleteKanban,
		CreatedBy:             m.CreatedBy,
		CreatedAt:             m.CreatedAt,
		UpdatedBy:             m.UpdatedBy,
		UpdatedAt:             m.UpdatedAt,
	}
}

func indirectModelToItem(m invModels.IndirectRawMaterial) invModels.IndirectMaterialItem {
	return invModels.IndirectMaterialItem{
		ID:                    m.ID,
		UniqCode:              m.UniqCode,
		PartNumber:            m.PartNumber,
		PartName:              m.PartName,
		WarehouseLocation:     m.WarehouseLocation,
		UOM:                   m.UOM,
		StockQty:              m.StockQty,
		StockWeightKg:         m.StockWeightKg,
		KanbanCount:           m.KanbanCount,
		KanbanStandardQty:     m.KanbanStandardQty,
		SafetyStockQty:        m.SafetyStockQty,
		DailyUsageQty:         m.DailyUsageQty,
		Status:                m.Status,
		StockDays:             m.StockDays,
		BuyNotBuy:             m.BuyNotBuy,
		StockToCompleteKanban: m.StockToCompleteKanban,
		CreatedBy:             m.CreatedBy,
		CreatedAt:             m.CreatedAt,
		UpdatedBy:             m.UpdatedBy,
		UpdatedAt:             m.UpdatedAt,
	}
}

func subconModelToItem(m invModels.SubconInventory) invModels.SubconInventoryItem {
	return invModels.SubconInventoryItem{
		ID:               m.ID,
		UniqCode:         m.UniqCode,
		PartNumber:       m.PartNumber,
		PartName:         m.PartName,
		PONumber:         m.PONumber,
		POPeriod:         m.POPeriod,
		SubconVendorID:   m.SubconVendorID,
		SubconVendorName: m.SubconVendorName,
		StockAtVendorQty: m.StockAtVendorQty,
		TotalPOQty:       m.TotalPOQty,
		SafetyStockQty:   m.SafetyStockQty,
		DateDelivery:     m.DateDelivery,
		Status:           m.Status,
		CreatedBy:        m.CreatedBy,
		CreatedAt:        m.CreatedAt,
		UpdatedBy:        m.UpdatedBy,
		UpdatedAt:        m.UpdatedAt,
	}
}

func buildHistoryResponse(rows []repository.HistoryRow, total int64, p pagination.PaginationInput) *invModels.HistoryLogResponse {
	totalPages := 0
	if p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}

	items := make([]invModels.HistoryLogItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, invModels.HistoryLogItem{
			ID:            r.ID,
			UniqCode:      r.UniqCode,
			KanbanPacking: r.DNNumber,
			QtyChange:     r.QtyChange,
			WeightChange:  r.WeightChange,
			MovementType:  r.MovementType,
			Reason:        sourceFlagToReason(r.SourceFlag, r.MovementType),
			LogStatus:     r.LogStatus,
			Notes:         r.Notes,
			LoggedBy:      r.LoggedBy,
			LoggedAt:      r.LoggedAt,
		})
	}

	return &invModels.HistoryLogResponse{
		Items: items,
		Pagination: invModels.InventoryPagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}
}

// sourceFlagToReason maps source_flag/movement_type to the human-readable "Reason" column.
func sourceFlagToReason(sourceFlag *string, movementType string) string {
	if sourceFlag != nil {
		switch *sourceFlag {
		case string(inventoryconst.SourceQCApprove):
			return "Delivery Notes"
		case string(inventoryconst.SourceWOApprove):
			return "WO Approved"
		case string(inventoryconst.SourceWOReserve):
			return "WO Reserved"
		case string(inventoryconst.SourceWOConsumeActual):
			return "WO Consume (Actual)"
		case string(inventoryconst.SourceProductionReject):
			return "Production Reject"
		case string(inventoryconst.SourceWOScan):
			return "WO Scan"
		case string(inventoryconst.SourceIncomingScan):
			return "Scan"
		case string(inventoryconst.SourceStockOpname):
			return "Stock Opname"
		case string(inventoryconst.SourceManual):
			return "Manual Adjustment"
		}
	}
	switch movementType {
	case string(inventoryconst.MovementStockOpname):
		return "Stock Opname"
	case string(inventoryconst.MovementIncoming):
		return "Incoming"
	case string(inventoryconst.MovementOutgoing):
		return "Outgoing"
	case string(inventoryconst.MovementReceivedFromVendor):
		return "Received from Vendor"
	}
	return movementType
}

// qcStatusDisplay maps qc_tasks.status → display string shown on UI.
func qcStatusDisplay(status string) string {
	switch status {
	case "approved":
		return "Received"
	case "rejected":
		return "Rejected"
	case "in_progress":
		return "In Progress"
	default:
		return "Pending Approval"
	}
}

// ---------------------------------------------------------------------------
// Movement Log Helper
// ---------------------------------------------------------------------------

// ConsumeItem represents one work order item line to deduct from inventory.
type ConsumeItem struct {
	UniqCode string
	Qty      float64
}

// ConsumeStockForWorkOrder atomically deducts stock for each WO item and writes an outgoing
// movement log entry. Items not found in either raw_materials or indirect_raw_materials are
// skipped (they may be finished-goods tracked elsewhere).
func (s *service) ConsumeStockForWorkOrder(ctx context.Context, items []ConsumeItem, woNumber string, performedBy string) error {
	for _, item := range items {
		if item.Qty <= 0 {
			continue
		}

		// Try raw_material first.
		rmID, err := s.repo.DeductRawMaterialByUniqCode(ctx, item.UniqCode, item.Qty, performedBy)
		if err != nil {
			return err
		}
		if rmID > 0 {
			s.writeMovementLog(ctx, MovementLogInput{
				Category:     "raw_material",
				MovementType: "outgoing",
				UniqCode:     item.UniqCode,
				EntityID:     &rmID,
				QtyChange:    -item.Qty,
				SourceFlag:   string(inventoryconst.SourceWOApprove),
				ReferenceID:  &woNumber,
				LoggedBy:     performedBy,
			})
			continue
		}

		// Try indirect_raw_material.
		irmID, err := s.repo.DeductIndirectMaterialByUniqCode(ctx, item.UniqCode, item.Qty, performedBy)
		if err != nil {
			return err
		}
		if irmID > 0 {
			s.writeMovementLog(ctx, MovementLogInput{
				Category:     "indirect_raw_material",
				MovementType: "outgoing",
				UniqCode:     item.UniqCode,
				EntityID:     &irmID,
				QtyChange:    -item.Qty,
				SourceFlag:   string(inventoryconst.SourceWOApprove),
				ReferenceID:  &woNumber,
				LoggedBy:     performedBy,
			})
		}
		// If neither found, silently skip (item not tracked in these tables).
	}
	return nil
}

// MovementLogInput is the input struct for writeMovementLog.
// Use this wherever inventory stock changes to keep tracking consistent.
//
// source_flag values (standard, non-exhaustive):
//   - "manual"            → manual create/adjustment via API
//   - "incoming_scan"     → scan incoming (pre-QC)
//   - "qc_approve"        → stock masuk dari QC approve
//   - "wo_approve"        → stock keluar untuk WO yang di-approve (reserve/issue)
//   - "production_reject" → stock masuk kembali karena QC/production reject (future)
//   - "stock_opname"      → stock opname correction
//
// Legacy flags kept for backward compatibility:
//   - "wo_scan" | "production"
type MovementLogInput struct {
	// Category: "raw_material" | "indirect_raw_material" | "subcon"
	Category string
	// Type: "incoming" | "outgoing" | "adjustment" | "stock_opname" | "received_from_vendor"
	MovementType string
	UniqCode     string
	EntityID     *int64  // ID baris di raw_materials / indirect_raw_materials / subcon_inventories
	QtyChange    float64 // positif = masuk, negatif = keluar
	WeightChange *float64
	SourceFlag   string  // see list above
	ReferenceID  *string // PO number, DN number, WO number, dll
	DNNumber     *string
	Notes        *string
	LoggedBy     string
}

// writeMovementLog inserts one row to inventory_movement_logs. Non-fatal: error only logged, not returned.
// Call this after any stock change (create, update qty, production consume, etc.).
func (s *service) writeMovementLog(ctx context.Context, input MovementLogInput) {
	sf := input.SourceFlag
	lb := input.LoggedBy
	_ = s.repo.CreateMovementLog(ctx, &invModels.InventoryMovementLog{
		MovementCategory: input.Category,
		MovementType:     input.MovementType,
		UniqCode:         input.UniqCode,
		EntityID:         input.EntityID,
		QtyChange:        input.QtyChange,
		WeightChange:     input.WeightChange,
		SourceFlag:       &sf,
		ReferenceID:      input.ReferenceID,
		DNNumber:         input.DNNumber,
		Notes:            input.Notes,
		LoggedBy:         &lb,
	})
}
