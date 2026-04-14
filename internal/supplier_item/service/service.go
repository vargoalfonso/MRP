package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier_item/models"
	supplierItemRepo "github.com/ganasa18/go-template/internal/supplier_item/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
)

type SupplierItemService interface {
	Create(ctx context.Context, req models.CreateSupplierItemRequest) (*models.SupplierItem, error)
	GetByUUID(ctx context.Context, uuid string) (*models.SupplierItem, error)
	List(ctx context.Context, query models.ListSupplierItemQuery) (*models.SupplierItemListResult, error)
	Update(ctx context.Context, uuid string, req models.UpdateSupplierItemRequest) (*models.SupplierItem, error)
	Delete(ctx context.Context, uuid string) error
}

type service struct {
	repo supplierItemRepo.IRepository
}

func New(repo supplierItemRepo.IRepository) SupplierItemService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateSupplierItemRequest) (*models.SupplierItem, error) {
	supplier, err := s.repo.FindSupplierByUUID(ctx, req.SupplierUUID)
	if err != nil {
		return nil, err
	}

	quantity, err := parseRequiredInt64(req.Quantity, "quantity")
	if err != nil {
		return nil, err
	}
	weight, err := parseOptionalFloat64(req.Weight, "weight")
	if err != nil {
		return nil, err
	}
	pcsPerKanban, err := parseOptionalInt64(req.PcsPerKanban, "pcs_per_kanban")
	if err != nil {
		return nil, err
	}

	item := &models.SupplierItem{
		UUID:          uuid.NewString(),
		SupplierUUID:  supplier.UUID,
		SupplierName:  supplier.SupplierName,
		SebangoCode:   strings.ToUpper(models.Trimmed(req.SebangoCode)),
		UniqCode:      strings.ToUpper(models.Trimmed(req.UniqCode)),
		Type:          normalizeType(req.Type),
		Description:   models.NormalizeOptionalString(toOptionalString(req.Description)),
		Quantity:      quantity,
		UOM:           models.NormalizeOptionalString(toOptionalString(req.UOM)),
		Weight:        weight,
		PcsPerKanban:  pcsPerKanban,
		CustomerCycle: models.NormalizeOptionalString(toOptionalString(req.CustomerCycle)),
		Status:        normalizeStatus(req.Status),
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) GetByUUID(ctx context.Context, uuid string) (*models.SupplierItem, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, apperror.BadRequest("supplier item id is required")
	}
	return s.repo.FindByUUID(ctx, uuid)
}

func (s *service) List(ctx context.Context, query models.ListSupplierItemQuery) (*models.SupplierItemListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filters := models.SupplierItemListFilters{
		Search: strings.TrimSpace(query.Search),
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
	if strings.TrimSpace(query.SupplierUUID) != "" {
		supplierUUID := strings.TrimSpace(query.SupplierUUID)
		filters.SupplierUUID = &supplierUUID
	}
	if strings.TrimSpace(query.Type) != "" {
		typeValue := normalizeType(query.Type)
		filters.Type = &typeValue
	}
	if strings.TrimSpace(query.Status) != "" {
		statusValue := normalizeStatus(query.Status)
		filters.Status = &statusValue
	}

	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}
	return &models.SupplierItemListResult{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) Update(ctx context.Context, uuid string, req models.UpdateSupplierItemRequest) (*models.SupplierItem, error) {
	item, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	supplier, err := s.repo.FindSupplierByUUID(ctx, req.SupplierUUID)
	if err != nil {
		return nil, err
	}

	quantity, err := parseRequiredInt64(req.Quantity, "quantity")
	if err != nil {
		return nil, err
	}
	weight, err := parseOptionalFloat64(req.Weight, "weight")
	if err != nil {
		return nil, err
	}
	pcsPerKanban, err := parseOptionalInt64(req.PcsPerKanban, "pcs_per_kanban")
	if err != nil {
		return nil, err
	}

	item.SupplierUUID = supplier.UUID
	item.SupplierName = supplier.SupplierName
	item.SebangoCode = strings.ToUpper(models.Trimmed(req.SebangoCode))
	item.UniqCode = strings.ToUpper(models.Trimmed(req.UniqCode))
	item.Type = normalizeType(req.Type)
	item.Description = models.NormalizeOptionalString(toOptionalString(req.Description))
	item.Quantity = quantity
	item.UOM = models.NormalizeOptionalString(toOptionalString(req.UOM))
	item.Weight = weight
	item.PcsPerKanban = pcsPerKanban
	item.CustomerCycle = models.NormalizeOptionalString(toOptionalString(req.CustomerCycle))
	item.Status = normalizeStatus(req.Status)

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) Delete(ctx context.Context, uuid string) error {
	item, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, item)
}

func normalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func toOptionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseRequiredInt64(value string, fieldName string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, apperror.BadRequest(fmt.Sprintf("%s must be a valid integer string", fieldName))
	}
	if parsed < 0 {
		return 0, apperror.BadRequest(fmt.Sprintf("%s must be greater than or equal to 0", fieldName))
	}
	return parsed, nil
}

func parseOptionalInt64(value string, fieldName string) (*int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil, apperror.BadRequest(fmt.Sprintf("%s must be a valid integer string", fieldName))
	}
	if parsed < 0 {
		return nil, apperror.BadRequest(fmt.Sprintf("%s must be greater than or equal to 0", fieldName))
	}
	return &parsed, nil
}

func parseOptionalFloat64(value string, fieldName string) (*float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil, apperror.BadRequest(fmt.Sprintf("%s must be a valid number string", fieldName))
	}
	if parsed < 0 {
		return nil, apperror.BadRequest(fmt.Sprintf("%s must be greater than or equal to 0", fieldName))
	}
	return &parsed, nil
}
