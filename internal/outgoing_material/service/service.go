package service

import (
	"context"
	"math"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	outModels "github.com/ganasa18/go-template/internal/outgoing_material/models"
	"github.com/ganasa18/go-template/internal/outgoing_material/repository"
	"github.com/ganasa18/go-template/pkg/pagination"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IService interface {
	List(ctx context.Context, p pagination.OutgoingRMPaginationInput) (*outModels.OutgoingRMListResponse, error)
	GetByID(ctx context.Context, id int64) (*outModels.OutgoingRMItem, error)
	Create(ctx context.Context, req outModels.CreateOutgoingRMRequest, createdBy string) (*outModels.OutgoingRMItem, error)
	// SearchRawMaterials returns raw material options for the create form autocomplete.
	SearchRawMaterials(ctx context.Context, q string, limit int) ([]outModels.FormOptionItem, error)
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type service struct {
	repo repository.IRepository
	db   *gorm.DB
}

func New(repo repository.IRepository, db *gorm.DB) IService { return &service{repo: repo, db: db} }

func (s *service) List(ctx context.Context, p pagination.OutgoingRMPaginationInput) (*outModels.OutgoingRMListResponse, error) {
	f := repository.ListFilter{
		Search:         p.Search,
		DateFrom:       p.DateFrom,
		DateTo:         p.DateTo,
		Reason:         p.Reason,
		Uniq:           p.Uniq,
		TransactionID:  p.TransactionID,
		WorkOrderNo:    p.WorkOrderNo,
		Page:           p.Page,
		Limit:          p.Limit,
		Offset:         p.Offset(),
		OrderBy:        p.OrderBy,
		OrderDirection: p.OrderDirection,
	}
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]outModels.OutgoingRMItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, rowToItem(r))
	}
	totalPages := 0
	if p.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(p.Limit)))
	}
	return &outModels.OutgoingRMListResponse{
		Items: items,
		Pagination: outModels.Pagination{
			Total:      total,
			Page:       p.Page,
			Limit:      p.Limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*outModels.OutgoingRMItem, error) {
	orm, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	item := modelToItem(*orm)
	return &item, nil
}

func (s *service) Create(ctx context.Context, req outModels.CreateOutgoingRMRequest, createdBy string) (*outModels.OutgoingRMItem, error) {
	now := time.Now()
	orm := outModels.OutgoingRawMaterial{
		TransactionDate:     now,
		PackingListRM:       req.PackingListRM,
		Uniq:                req.Uniq,
		Unit:                req.Unit,
		QuantityOut:         req.QuantityOut,
		Reason:              req.Reason,
		Purpose:             req.Purpose,
		WorkOrderNo:         req.WorkOrderNo,
		DestinationLocation: req.DestinationLocation,
		RequestedBy:         req.RequestedBy,
		Remarks:             req.Remarks,
		CreatedBy:           &createdBy,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.repo.ProcessTx(ctx, &orm); err != nil {
		return nil, err
	}
	s.appendMovementLog(ctx, orm)
	item := modelToItem(orm)
	return &item, nil
}

func (s *service) SearchRawMaterials(ctx context.Context, q string, limit int) ([]outModels.FormOptionItem, error) {
	rows, err := s.repo.SearchRawMaterials(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	items := make([]outModels.FormOptionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, outModels.FormOptionItem{
			ID:                r.ID,
			UniqCode:          r.UniqCode,
			PartNumber:        r.PartNumber,
			PartName:          r.PartName,
			UOM:               r.UOM,
			StockQty:          r.StockQty,
			WarehouseLocation: r.WarehouseLocation,
		})
	}
	return items, nil
}

// appendMovementLog writes one row to inventory_movement_logs. Non-fatal — errors are swallowed.
func (s *service) appendMovementLog(ctx context.Context, orm outModels.OutgoingRawMaterial) {
	notes := "reason: " + orm.Reason
	if orm.Purpose != nil {
		notes += ", purpose: " + *orm.Purpose
	}
	if orm.DestinationLocation != nil {
		notes += ", destination: " + *orm.DestinationLocation
	}
	sf := "outgoing_raw_material"
	_ = s.db.WithContext(ctx).Create(&invModels.InventoryMovementLog{
		MovementCategory: "raw_material",
		MovementType:     "outgoing",
		UniqCode:         orm.Uniq,
		EntityID:         orm.RawMaterialID,
		QtyChange:        -orm.QuantityOut,
		SourceFlag:       &sf,
		ReferenceID:      orm.WorkOrderNo,
		Notes:            &notes,
		LoggedBy:         orm.CreatedBy,
		LoggedAt:         time.Now(),
	}).Error
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

func rowToItem(r repository.OutgoingRow) outModels.OutgoingRMItem {
	return outModels.OutgoingRMItem{
		ID:                  r.ID,
		TransactionID:       r.TransactionID,
		TransactionDate:     r.TransactionDate,
		Uniq:                r.Uniq,
		RMName:              r.RMName,
		PackingListRM:       r.PackingListRM,
		Unit:                r.Unit,
		QuantityOut:         r.QuantityOut,
		StockBefore:         r.StockBefore,
		StockAfter:          r.StockAfter,
		Reason:              r.Reason,
		Purpose:             r.Purpose,
		WorkOrderNo:         r.WorkOrderNo,
		DestinationLocation: r.DestinationLocation,
		RequestedBy:         r.RequestedBy,
		Remarks:             r.Remarks,
		CreatedBy:           r.CreatedBy,
		CreatedAt:           r.CreatedAt,
	}
}

func modelToItem(m outModels.OutgoingRawMaterial) outModels.OutgoingRMItem {
	return outModels.OutgoingRMItem{
		ID:                  m.ID,
		TransactionID:       m.TransactionID,
		TransactionDate:     m.TransactionDate,
		Uniq:                m.Uniq,
		RMName:              m.RMName,
		PackingListRM:       m.PackingListRM,
		Unit:                m.Unit,
		QuantityOut:         m.QuantityOut,
		StockBefore:         m.StockBefore,
		StockAfter:          m.StockAfter,
		Reason:              m.Reason,
		Purpose:             m.Purpose,
		WorkOrderNo:         m.WorkOrderNo,
		DestinationLocation: m.DestinationLocation,
		RequestedBy:         m.RequestedBy,
		Remarks:             m.Remarks,
		CreatedBy:           m.CreatedBy,
		CreatedAt:           m.CreatedAt,
	}
}
