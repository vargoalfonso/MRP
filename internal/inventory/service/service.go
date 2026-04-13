package service

import (
	"context"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	"github.com/ganasa18/go-template/internal/inventory/repository"
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
	now := time.Now()
	rm := invModels.RawMaterial{
		UUID:              uuid.New(),
		UniqCode:          req.UniqCode,
		RawMaterialType:   req.RawMaterialType,
		RMSource:          req.RMSource,
		PartNumber:        req.PartNumber,
		PartName:          req.PartName,
		WarehouseLocation: req.WarehouseLocation,
		UOM:               req.UOM,
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
		items = append(items, invModels.RawMaterial{
			UUID:              uuid.New(),
			UniqCode:          r.UniqCode,
			RawMaterialType:   r.RawMaterialType,
			RMSource:          r.RMSource,
			PartNumber:        r.PartNumber,
			PartName:          r.PartName,
			WarehouseLocation: r.WarehouseLocation,
			UOM:               r.UOM,
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
	now := time.Now()
	statusPtr := "normal"
	irm := invModels.IndirectRawMaterial{
		UUID:              uuid.New(),
		UniqCode:          req.UniqCode,
		PartNumber:        req.PartNumber,
		PartName:          req.PartName,
		WarehouseLocation: req.WarehouseLocation,
		UOM:               req.UOM,
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
		sp := statusPtr
		items = append(items, invModels.IndirectRawMaterial{
			UUID:              uuid.New(),
			UniqCode:          r.UniqCode,
			PartNumber:        r.PartNumber,
			PartName:          r.PartName,
			WarehouseLocation: r.WarehouseLocation,
			UOM:               r.UOM,
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
	now := time.Now()
	si := invModels.SubconInventory{
		UUID:             uuid.New(),
		UniqCode:         req.UniqCode,
		PartNumber:       req.PartNumber,
		PartName:         req.PartName,
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

func (s *service) GetKanbanSummary(ctx context.Context, uniqCode string) (*invModels.KanbanSummary, error) {
	return s.repo.GetKanbanSummary(ctx, uniqCode)
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
		case "qc_approve":
			return "Delivery Notes"
		case "wo_scan":
			return "WO Scan"
		case "incoming_scan":
			return "Scan"
		case "stock_opname":
			return "Stock Opname"
		case "manual":
			return "Manual Adjustment"
		}
	}
	switch movementType {
	case "stock_opname":
		return "Stock Opname"
	case "incoming":
		return "Incoming"
	case "outgoing":
		return "Outgoing"
	case "received_from_vendor":
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

// MovementLogInput is the input struct for writeMovementLog.
// Use this wherever inventory stock changes to keep tracking consistent.
//
// source_flag values:
//   - "manual"       → manual create/adjustment via API
//   - "qc_approve"   → stock masuk dari QC approve
//   - "wo_scan"      → stock keluar karena WO scan production
//   - "stock_opname" → stock opname correction
//   - "production"   → stock keluar ke production
type MovementLogInput struct {
	// Category: "raw_material" | "indirect_raw_material" | "subcon"
	Category string
	// Type: "incoming" | "outgoing" | "adjustment" | "stock_opname" | "received_from_vendor"
	MovementType string
	UniqCode     string
	EntityID     *int64  // ID baris di raw_materials / indirect_raw_materials / subcon_inventories
	QtyChange    float64 // positif = masuk, negatif = keluar
	WeightChange *float64
	SourceFlag   string  // "manual" | "qc_approve" | "wo_scan" | "stock_opname" | "production"
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
