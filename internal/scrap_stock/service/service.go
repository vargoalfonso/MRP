// Package service provides business logic for the Scrap Stock module.
package service

import (
	"context"
	"math"
	"time"

	scrapModels "github.com/ganasa18/go-template/internal/scrap_stock/models"
	"github.com/ganasa18/go-template/internal/scrap_stock/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/creatorresolver"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

type IService interface {
	// Dashboard stats (4 cards)
	GetStats(ctx context.Context) (*scrapModels.ScrapStockStats, error)

	// Scrap Stock (Inventory database)
	ListScrapStocks(ctx context.Context, f repository.ScrapStockFilter) (*scrapModels.ScrapStockListResponse, error)
	GetScrapStockByID(ctx context.Context, id int64) (*scrapModels.ScrapStockItem, error)
	CreateScrapStock(ctx context.Context, req scrapModels.CreateScrapStockRequest, createdBy string) (*scrapModels.ScrapStockItem, error)

	// Incoming Scrap (Action UI scan flow)
	CreateIncomingScrap(ctx context.Context, req scrapModels.IncomingScrapRequest, createdBy string) (*scrapModels.ScrapStockItem, error)

	// Scrap Release (Sell / Dump)
	ListScrapReleases(ctx context.Context, f repository.ScrapReleaseFilter) (*scrapModels.ScrapReleaseListResponse, error)
	GetScrapReleaseByID(ctx context.Context, id int64) (*scrapModels.ScrapReleaseItem, error)
	CreateScrapRelease(ctx context.Context, req scrapModels.CreateScrapReleaseRequest, createdBy string) (*scrapModels.ScrapReleaseItem, error)
	ApproveScrapRelease(ctx context.Context, id int64, req scrapModels.ApproveScrapReleaseRequest, approvedBy string) error
}

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

type service struct {
	repo repository.IRepository
	db   *gorm.DB
}

func New(repo repository.IRepository, db *gorm.DB) IService { return &service{repo: repo, db: db} }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toStockItem(s *scrapModels.ScrapStock) *scrapModels.ScrapStockItem {
	return &scrapModels.ScrapStockItem{
		ID:            s.ID,
		UUID:          s.UUID.String(),
		UniqCode:      s.UniqCode,
		PartNumber:    s.PartNumber,
		PartName:      s.PartName,
		Model:         s.Model,
		PackingNumber: s.PackingNumber,
		WONumber:      s.WONumber,
		ScrapType:     s.ScrapType,
		Quantity:      s.Quantity,
		UOM:           s.UOM,
		WeightKg:      s.WeightKg,
		DateReceived:  s.DateReceived,
		Validator:     s.Validator,
		Remarks:       s.Remarks,
		Status:        s.Status,
		CreatedBy:     s.CreatedBy,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toReleaseItem(r *scrapModels.ScrapRelease) *scrapModels.ScrapReleaseItem {
	return &scrapModels.ScrapReleaseItem{
		ID:             r.ID,
		UUID:           r.UUID.String(),
		ReleaseNumber:  r.ReleaseNumber,
		ScrapStockID:   r.ScrapStockID,
		ReleaseDate:    r.ReleaseDate,
		ReleaseType:    r.ReleaseType,
		ReleaseQty:     r.ReleaseQty,
		WeightReleased: r.WeightReleased,
		CustomerName:   r.CustomerName,
		PricePerUnit:   r.PricePerUnit,
		TotalValue:     r.TotalValue,
		DisposalReason: r.DisposalReason,
		ApprovalStatus: r.ApprovalStatus,
		Validator:      r.Validator,
		Approver:       r.Approver,
		ApprovedBy:     r.ApprovedBy,
		ApprovedAt:     r.ApprovedAt,
		Remarks:        r.Remarks,
		CreatedBy:      r.CreatedBy,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func validateScrapType(t string) error {
	if _, ok := scrapModels.ValidScrapTypes[t]; !ok {
		return apperror.UnprocessableEntity(
			"scrap_type must be one of: setting_machine_scrap, process_scrap, product_return_scrap")
	}
	return nil
}

func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func calcTotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 1
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func (sv *service) GetStats(ctx context.Context) (*scrapModels.ScrapStockStats, error) {
	return sv.repo.GetStats(ctx)
}

// ---------------------------------------------------------------------------
// Scrap Stock
// ---------------------------------------------------------------------------

func (sv *service) ListScrapStocks(ctx context.Context, f repository.ScrapStockFilter) (*scrapModels.ScrapStockListResponse, error) {
	rows, total, err := sv.repo.ListScrapStocks(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]scrapModels.ScrapStockItem, 0, len(rows))
	for i := range rows {
		items = append(items, *toStockItem(&rows[i]))
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	return &scrapModels.ScrapStockListResponse{
		Items: items,
		Pagination: scrapModels.ScrapPagination{
			Total:      total,
			Page:       page,
			Limit:      f.Limit,
			TotalPages: calcTotalPages(total, f.Limit),
		},
	}, nil
}

func (sv *service) GetScrapStockByID(ctx context.Context, id int64) (*scrapModels.ScrapStockItem, error) {
	s, err := sv.repo.GetScrapStockByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toStockItem(s), nil
}

func (sv *service) CreateScrapStock(ctx context.Context, req scrapModels.CreateScrapStockRequest, createdBy string) (*scrapModels.ScrapStockItem, error) {
	if err := validateScrapType(req.ScrapType); err != nil {
		return nil, err
	}

	_, creatorName, err := creatorresolver.Resolve(ctx, sv.db, createdBy)
	if err != nil {
		return nil, err
	}

	s := &scrapModels.ScrapStock{
		UniqCode:      req.UniqCode,
		PartNumber:    req.PartNumber,
		PartName:      req.PartName,
		Model:         req.Model,
		PackingNumber: req.PackingNumber,
		WONumber:      req.WONumber,
		ScrapType:     req.ScrapType,
		Quantity:      req.Quantity,
		UOM:           req.UOM,
		WeightKg:      req.WeightKg,
		DateReceived:  parseDate(req.DateReceived),
		Validator:     creatorName,
		Remarks:       req.Remarks,
		Status:        scrapModels.StockStatusActive,
		CreatedBy:     creatorName,
		UpdatedBy:     creatorName,
	}
	if err := sv.repo.CreateScrapStock(ctx, s); err != nil {
		return nil, err
	}
	return toStockItem(s), nil
}

// ---------------------------------------------------------------------------
// Incoming Scrap (Action UI)
// ---------------------------------------------------------------------------

func (sv *service) CreateIncomingScrap(ctx context.Context, req scrapModels.IncomingScrapRequest, createdBy string) (*scrapModels.ScrapStockItem, error) {
	if err := validateScrapType(req.ScrapType); err != nil {
		return nil, err
	}

	_, creatorName, err := creatorresolver.Resolve(ctx, sv.db, createdBy)
	if err != nil {
		return nil, err
	}
	s := &scrapModels.ScrapStock{
		UniqCode:      req.UniqCode,
		PackingNumber: req.PackingNumber,
		WONumber:      req.WONumber,
		ScrapType:     req.ScrapType,
		Quantity:      req.Quantity,
		UOM:           req.UOM,
		WeightKg:      req.WeightKg,
		DateReceived:  parseDate(req.DateReceived),
		Validator:     creatorName,
		Status:        scrapModels.StockStatusActive,
		CreatedBy:     creatorName,
		UpdatedBy:     creatorName,
	}
	if err := sv.repo.CreateScrapStock(ctx, s); err != nil {
		return nil, err
	}
	return toStockItem(s), nil
}

// ---------------------------------------------------------------------------
// Scrap Release
// ---------------------------------------------------------------------------

func (sv *service) ListScrapReleases(ctx context.Context, f repository.ScrapReleaseFilter) (*scrapModels.ScrapReleaseListResponse, error) {
	rows, total, err := sv.repo.ListScrapReleases(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]scrapModels.ScrapReleaseItem, 0, len(rows))
	for i := range rows {
		items = append(items, *toReleaseItem(&rows[i]))
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	return &scrapModels.ScrapReleaseListResponse{
		Items: items,
		Pagination: scrapModels.ScrapPagination{
			Total:      total,
			Page:       page,
			Limit:      f.Limit,
			TotalPages: calcTotalPages(total, f.Limit),
		},
	}, nil
}

func (sv *service) GetScrapReleaseByID(ctx context.Context, id int64) (*scrapModels.ScrapReleaseItem, error) {
	rel, err := sv.repo.GetScrapReleaseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toReleaseItem(rel), nil
}

func (sv *service) CreateScrapRelease(ctx context.Context, req scrapModels.CreateScrapReleaseRequest, createdBy string) (*scrapModels.ScrapReleaseItem, error) {
	if req.ReleaseType != scrapModels.ReleaseTypeSell && req.ReleaseType != scrapModels.ReleaseTypeDump {
		return nil, apperror.UnprocessableEntity("release_type must be Sell or Dump")
	}

	// Confirm scrap stock exists and has enough qty
	stock, err := sv.repo.GetScrapStockByID(ctx, req.ScrapStockID)
	if err != nil {
		return nil, err
	}
	if req.ReleaseQty > stock.Quantity {
		return nil, apperror.UnprocessableEntity("release_qty exceeds available scrap quantity")
	}

	// Auto-compute total_value for Sell type
	var totalValue *float64
	if req.ReleaseType == scrapModels.ReleaseTypeSell && req.PricePerUnit != nil {
		tv := req.ReleaseQty * *req.PricePerUnit
		totalValue = &tv
	}

	_, creatorName, err := creatorresolver.Resolve(ctx, sv.db, createdBy)
	if err != nil {
		return nil, err
	}

	rel := &scrapModels.ScrapRelease{
		ScrapStockID:   req.ScrapStockID,
		ReleaseDate:    parseDate(req.ReleaseDate),
		ReleaseType:    req.ReleaseType,
		ReleaseQty:     req.ReleaseQty,
		WeightReleased: req.WeightReleased,
		CustomerName:   req.CustomerName,
		PricePerUnit:   req.PricePerUnit,
		TotalValue:     totalValue,
		DisposalReason: req.DisposalReason,
		ApprovalStatus: scrapModels.ApprovalStatusPending,
		Validator:      creatorName,
		Approver:       req.Approver,
		Remarks:        req.Remarks,
		CreatedBy:      creatorName,
		UpdatedBy:      creatorName,
	}
	if err := sv.repo.CreateScrapRelease(ctx, rel); err != nil {
		return nil, err
	}
	return toReleaseItem(rel), nil
}

func (sv *service) ApproveScrapRelease(ctx context.Context, id int64, req scrapModels.ApproveScrapReleaseRequest, approvedBy string) error {
	if req.Action != scrapModels.ApprovalStatusCompleted && req.Action != scrapModels.ApprovalStatusRejected {
		return apperror.UnprocessableEntity("action must be Completed or Rejected")
	}
	return sv.repo.ApproveRelease(ctx, id, req.Action, approvedBy, req.Remarks)
}
