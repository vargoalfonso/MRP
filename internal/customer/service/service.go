package service

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/customer/models"
	customerRepo "github.com/ganasa18/go-template/internal/customer/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
)

type CustomerService interface {
	Create(ctx context.Context, req models.CreateCustomerRequest) (*models.Customer, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Customer, error)
	List(ctx context.Context, query models.ListCustomerQuery) (*models.CustomerListResult, error)
	Update(ctx context.Context, uuid string, req models.UpdateCustomerRequest) (*models.Customer, error)
	Delete(ctx context.Context, uuid string) error
}

type service struct {
	repo customerRepo.IRepository
}

func New(repo customerRepo.IRepository) CustomerService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateCustomerRequest) (*models.Customer, error) {
	billingAddress, err := resolveBillingAddress(req.ShippingAddress, req.BillingAddress, req.BillingSameAsShipping)
	if err != nil {
		return nil, err
	}

	bankAccount := normalizeOptionalString(req.BankAccount)
	bankAccountNumber := normalizeOptionalString(req.BankAccountNumber)

	customer := &models.Customer{
		UUID:                  uuid.NewString(),
		CustomerID:            "PENDING",
		CustomerName:          models.Trimmed(req.CustomerName),
		PhoneNumber:           models.Trimmed(req.PhoneNumber),
		ShippingAddress:       models.Trimmed(req.ShippingAddress),
		BillingAddress:        billingAddress,
		BillingSameAsShipping: req.BillingSameAsShipping,
		BankAccount:           bankAccount,
		BankAccountNumber:     bankAccountNumber,
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *service) GetByUUID(ctx context.Context, uuid string) (*models.Customer, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, apperror.BadRequest("customer id is required")
	}

	return s.repo.FindByUUID(ctx, uuid)
}

func (s *service) List(ctx context.Context, query models.ListCustomerQuery) (*models.CustomerListResult, error) {
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

	filters := models.CustomerListFilters{
		Search: models.Trimmed(query.Search),
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &models.CustomerListResult{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) Update(ctx context.Context, uuid string, req models.UpdateCustomerRequest) (*models.Customer, error) {
	customer, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	billingAddress, err := resolveBillingAddress(req.ShippingAddress, req.BillingAddress, req.BillingSameAsShipping)
	if err != nil {
		return nil, err
	}

	customer.CustomerName = models.Trimmed(req.CustomerName)
	customer.PhoneNumber = models.Trimmed(req.PhoneNumber)
	customer.ShippingAddress = models.Trimmed(req.ShippingAddress)
	customer.BillingAddress = billingAddress
	customer.BillingSameAsShipping = req.BillingSameAsShipping
	customer.BankAccount = normalizeOptionalString(req.BankAccount)
	customer.BankAccountNumber = normalizeOptionalString(req.BankAccountNumber)

	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *service) Delete(ctx context.Context, uuid string) error {
	customer, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, customer)
}

func resolveBillingAddress(shippingAddress, billingAddress string, sameAsShipping bool) (string, error) {
	shipping := models.Trimmed(shippingAddress)
	billing := models.Trimmed(billingAddress)

	if sameAsShipping {
		if shipping == "" {
			return "", apperror.BadRequest("shipping_address is required")
		}
		return shipping, nil
	}

	if billing == "" {
		return "", apperror.BadRequest("billing_address is required when billing_same_as_shipping is false")
	}

	return billing, nil
}

func normalizeOptionalString(value string) *string {
	cleaned := models.Trimmed(value)
	if cleaned == "" {
		return nil
	}

	return &cleaned
}
