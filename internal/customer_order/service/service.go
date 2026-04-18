package service

import (
	"context"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/customer_order/models"
	"github.com/ganasa18/go-template/internal/customer_order/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
)

type IService interface {
	Create(ctx context.Context, req models.CreateOrderRequest, createdBy string) (*models.DocumentResponse, error)
	GetByUUID(ctx context.Context, id string) (*models.DocumentResponse, error)
	List(ctx context.Context, q models.ListOrderQuery) (*models.ListResponse, error)
	UpdateStatus(ctx context.Context, id string, req models.UpdateStatusRequest) (*models.DocumentResponse, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateOrderRequest, createdBy string) (*models.DocumentResponse, error) {
	if err := validateDeliveryDate(req); err != nil {
		return nil, err
	}

	customerName, err := s.repo.GetCustomerNameByID(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	activePeriode, err := s.repo.GetActivePeriode(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("fetch active periode failed", err)
	}

	doc := &models.CustomerOrderDocument{
		UUID:                 uuid.NewString(),
		DocumentType:         req.DocumentType,
		DocumentDate:         resolveDocumentDate(req),
		PeriodSchedule:       activePeriode,
		CustomerID:           req.CustomerID,
		CustomerNameSnapshot: customerName,
		Status:               "draft",
		CreatedBy:            createdBy,
	}

	if v := strings.TrimSpace(req.ContactPerson); v != "" {
		doc.ContactPerson = &v
	}
	if v := strings.TrimSpace(req.DeliveryAddress); v != "" {
		doc.DeliveryAddress = &v
	}
	if v := strings.TrimSpace(req.Notes); v != "" {
		doc.Notes = &v
	}

	for i, it := range req.Items {
		snap, err := s.repo.GetItemSnapshot(ctx, it.ItemUniqCode)
		if err != nil {
			return nil, err
		}

		item := models.CustomerOrderDocumentItem{
			UUID:         uuid.NewString(),
			LineNo:       i + 1,
			ItemUniqCode: it.ItemUniqCode,
			PartName:     snap.PartName,
			PartNumber:   stringOrEmpty(snap.PartNumber),
			Model:        snap.Model,
			Quantity:     it.Quantity,
		}

		if it.DeliveryDate != "" && req.DocumentType != "DN" {
			t, err := time.Parse("2006-01-02", it.DeliveryDate)
			if err != nil {
				return nil, apperror.BadRequest("item delivery_date format must be YYYY-MM-DD")
			}
			item.DeliveryDate = &t
		}

		doc.Items = append(doc.Items, item)
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}

	resp := models.ToDocumentResponse(doc)
	return &resp, nil
}

func (s *service) GetByUUID(ctx context.Context, id string) (*models.DocumentResponse, error) {
	doc, err := s.repo.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := models.ToDocumentResponse(doc)
	return &resp, nil
}

func (s *service) List(ctx context.Context, q models.ListOrderQuery) (*models.ListResponse, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	docs, total, err := s.repo.List(ctx, models.ListFilters{
		Search:       strings.TrimSpace(q.Search),
		DocumentType: q.DocumentType,
		Status:       q.Status,
		CustomerID:   q.CustomerID,
		Period:       q.Period,
		Page:         page,
		Limit:        limit,
		Offset:       (page - 1) * limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]models.DocumentResponse, 0, len(docs))
	for i := range docs {
		items = append(items, models.ToDocumentResponse(&docs[i]))
	}

	return &models.ListResponse{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) UpdateStatus(ctx context.Context, id string, req models.UpdateStatusRequest) (*models.DocumentResponse, error) {
	doc, err := s.repo.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := validateStatusTransition(doc.Status, req.Status); err != nil {
		return nil, err
	}

	doc.Status = req.Status
	if err := s.repo.UpdateStatus(ctx, doc); err != nil {
		return nil, err
	}

	resp := models.ToDocumentResponse(doc)
	return &resp, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	doc, err := s.repo.FindByUUID(ctx, id)
	if err != nil {
		return err
	}
	if doc.Status != "draft" {
		return apperror.BadRequest("only draft documents can be deleted")
	}
	return s.repo.SoftDelete(ctx, doc)
}

func validateDeliveryDate(req models.CreateOrderRequest) error {
	switch req.DocumentType {
	case "DN":
		if strings.TrimSpace(req.DeliveryDate) == "" {
			return apperror.BadRequest("delivery_date is required for DN (header level)")
		}
		if _, err := time.Parse("2006-01-02", req.DeliveryDate); err != nil {
			return apperror.BadRequest("delivery_date format must be YYYY-MM-DD")
		}
	case "PO", "SO":
		for _, it := range req.Items {
			if strings.TrimSpace(it.DeliveryDate) == "" {
				return apperror.BadRequest("delivery_date is required per item for " + req.DocumentType)
			}
		}
	}
	return nil
}

func resolveDocumentDate(req models.CreateOrderRequest) time.Time {
	if req.DocumentType == "DN" && req.DeliveryDate != "" {
		if t, err := time.Parse("2006-01-02", req.DeliveryDate); err == nil {
			return t
		}
	}
	return time.Now()
}

func validateStatusTransition(current, next string) error {
	allowed := map[string][]string{
		"draft":  {"active", "cancelled"},
		"active": {"completed", "cancelled"},
	}
	for _, s := range allowed[current] {
		if s == next {
			return nil
		}
	}
	return apperror.BadRequest("invalid status transition: " + current + " → " + next)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
