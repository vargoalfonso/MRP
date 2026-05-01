package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier/models"
	supplierRepo "github.com/ganasa18/go-template/internal/supplier/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	emailpkg "github.com/ganasa18/go-template/pkg/email"
	"github.com/google/uuid"
)

type SupplierService interface {
	Create(ctx context.Context, req models.CreateSupplierRequest) (*models.Supplier, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Supplier, error)
	List(ctx context.Context, query models.ListSupplierQuery) (*models.SupplierListResult, error)
	Update(ctx context.Context, uuid string, req models.UpdateSupplierRequest) (*models.Supplier, error)
	Delete(ctx context.Context, uuid string) error
}

type service struct {
	repo supplierRepo.IRepository
}

func New(repo supplierRepo.IRepository) SupplierService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateSupplierRequest) (*models.Supplier, error) {
	materialCategory, err := normalizeMaterialCategory(req.MaterialCategory)
	if err != nil {
		return nil, err
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return nil, err
	}

	supplier := &models.Supplier{
		UUID:                 uuid.NewString(),
		SupplierCode:         "TMP-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", "")),
		SupplierName:         models.Trimmed(req.SupplierName),
		ContactPerson:        models.Trimmed(req.ContactPerson),
		ContactNumber:        models.Trimmed(req.ContactNumber),
		EmailAddress:         strings.ToLower(models.Trimmed(req.EmailAddress)),
		MaterialCategory:     materialCategory,
		FullAddress:          models.Trimmed(req.FullAddress),
		City:                 models.Trimmed(req.City),
		Province:             models.Trimmed(req.Province),
		Country:              models.Trimmed(req.Country),
		TaxIDNPWP:            models.Trimmed(req.TaxIDNPWP),
		BankName:             models.Trimmed(req.BankName),
		BankAccountNumber:    models.Trimmed(req.BankAccountNumber),
		BankAccountName:      models.Trimmed(req.BankAccountName),
		PaymentTerms:         models.Trimmed(req.PaymentTerms),
		DeliveryLeadTimeDays: req.DeliveryLeadTimeDays,
		Status:               status,
	}

	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}

	// Send notification email to supplier if email provided (async, non-blocking)
	if strings.TrimSpace(supplier.EmailAddress) != "" {
		go func(to, name string) {
			subject := "Selamat datang sebagai Supplier"
			body := fmt.Sprintf(`<h2>Halo %s 👋</h2><p>Anda telah terdaftar sebagai supplier di sistem kami.</p>`, name)
			if err := emailpkg.SendEmail(to, subject, body); err != nil {
				// Do not fail creation; log to stdout
				fmt.Println("failed send supplier email:", err)
			}
		}(supplier.EmailAddress, supplier.SupplierName)
	}

	return supplier, nil
}

func (s *service) GetByUUID(ctx context.Context, uuid string) (*models.Supplier, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, apperror.BadRequest("supplier id is required")
	}

	return s.repo.FindByUUID(ctx, uuid)
}

func (s *service) List(ctx context.Context, query models.ListSupplierQuery) (*models.SupplierListResult, error) {
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

	status, err := normalizeOptionalStatus(query.Status)
	if err != nil {
		return nil, err
	}

	materialCategory, err := normalizeMaterialCategory(query.MaterialCategory)
	if err != nil {
		return nil, err
	}

	filters := models.SupplierListFilters{
		Search:           models.Trimmed(query.Search),
		Status:           status,
		MaterialCategory: materialCategory,
		Page:             page,
		Limit:            limit,
		Offset:           (page - 1) * limit,
	}

	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &models.SupplierListResult{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) Update(ctx context.Context, uuid string, req models.UpdateSupplierRequest) (*models.Supplier, error) {
	supplier, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	materialCategory, err := normalizeMaterialCategory(req.MaterialCategory)
	if err != nil {
		return nil, err
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return nil, err
	}

	supplier.SupplierName = models.Trimmed(req.SupplierName)
	supplier.ContactPerson = models.Trimmed(req.ContactPerson)
	supplier.ContactNumber = models.Trimmed(req.ContactNumber)
	supplier.EmailAddress = strings.ToLower(models.Trimmed(req.EmailAddress))
	supplier.MaterialCategory = materialCategory
	supplier.FullAddress = models.Trimmed(req.FullAddress)
	supplier.City = models.Trimmed(req.City)
	supplier.Province = models.Trimmed(req.Province)
	supplier.Country = models.Trimmed(req.Country)
	supplier.TaxIDNPWP = models.Trimmed(req.TaxIDNPWP)
	supplier.BankName = models.Trimmed(req.BankName)
	supplier.BankAccountNumber = models.Trimmed(req.BankAccountNumber)
	supplier.BankAccountName = models.Trimmed(req.BankAccountName)
	supplier.PaymentTerms = models.Trimmed(req.PaymentTerms)
	supplier.DeliveryLeadTimeDays = req.DeliveryLeadTimeDays
	supplier.Status = status

	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}

	return supplier, nil
}

func (s *service) Delete(ctx context.Context, uuid string) error {
	supplier, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, supplier)
}

func normalizeMaterialCategory(value string) (*string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" {
		return nil, nil
	}

	switch strings.ToLower(cleaned) {
	case "raw material":
		result := models.MaterialCategoryRawMaterial
		return &result, nil
	case "indirect raw material":
		result := models.MaterialCategoryIndirectRawMaterial
		return &result, nil
	case "subcon":
		result := models.MaterialCategorySubcon
		return &result, nil
	default:
		return nil, apperror.BadRequest("material_category must be empty or one of: Raw Material, Indirect Raw Material, Subcon")
	}
}

func normalizeStatus(value string) (string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" {
		return "", apperror.BadRequest("status is required")
	}

	switch strings.ToLower(cleaned) {
	case "active":
		return models.SupplierStatusActive, nil
	case "inactive":
		return models.SupplierStatusInactive, nil
	default:
		return "", apperror.BadRequest("status must be one of: Active, Inactive")
	}
}

func normalizeOptionalStatus(value string) (*string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	status, err := normalizeStatus(value)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
