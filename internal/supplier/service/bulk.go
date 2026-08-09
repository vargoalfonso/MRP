package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/validator"
	"github.com/google/uuid"
)

// maxBulkSupplierItems caps the number of rows accepted in a single request.
const maxBulkSupplierItems = 5000

// BulkCreate imports many suppliers in a single request. Each row is processed
// independently: a failing row is recorded but does not stop the rest.
//
// Unlike the single Create, bulk import intentionally SKIPS the welcome email
// so a large import does not fan out into thousands of outbound emails. All
// validation, normalization and code-generation rules stay identical.
func (s *service) BulkCreate(ctx context.Context, items []models.CreateSupplierRequest) (*models.BulkImportResult, error) {
	if len(items) == 0 {
		return nil, apperror.BadRequest("tidak ada data untuk diimport")
	}
	if len(items) > maxBulkSupplierItems {
		return nil, apperror.BadRequest(fmt.Sprintf("maksimal %d baris per import", maxBulkSupplierItems))
	}

	result := &models.BulkImportResult{
		Total:   len(items),
		Results: make([]models.BulkRowResult, 0, len(items)),
	}

	for i, item := range items {
		row := models.BulkRowResult{
			Index: i,
			Row:   i + 2, // spreadsheet row: header is row 1, data starts at row 2
		}

		if errs := validator.Validate(item); errs != nil {
			row.Status = "failed"
			row.Message = joinFieldErrors(errs)
			result.FailedCount++
			result.Results = append(result.Results, row)
			continue
		}

		supplier, err := s.createWithoutEmail(ctx, item)
		if err != nil {
			row.Status = "failed"
			row.Message = bulkErrorMessage(err)
			result.FailedCount++
			result.Results = append(result.Results, row)
			continue
		}

		row.Status = "success"
		row.ID = supplier.UUID
		row.SupplierCode = supplier.SupplierCode
		row.SupplierName = supplier.SupplierName
		result.SuccessCount++
		result.Results = append(result.Results, row)
	}

	return result, nil
}

// createWithoutEmail mirrors service.Create but does not send the welcome email.
// It reuses the same normalization helpers so behaviour stays consistent.
func (s *service) createWithoutEmail(ctx context.Context, req models.CreateSupplierRequest) (*models.Supplier, error) {
	materialCategory, err := normalizeMaterialCategory(req.MaterialCategory)
	if err != nil {
		return nil, err
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return nil, err
	}

	supplierCode := strings.ToUpper(strings.TrimSpace(req.SupplierCode))
	if supplierCode == "" {
		supplierCode = "TMP-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	}

	supplier := &models.Supplier{
		UUID:                 uuid.NewString(),
		SupplierCode:         supplierCode,
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

	return supplier, nil
}

// joinFieldErrors turns validator field errors into a single readable message.
func joinFieldErrors(errs []validator.FieldError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, "; ")
}

// bulkErrorMessage extracts a user-friendly message from an error without
// leaking internal causes.
func bulkErrorMessage(err error) string {
	if appErr, ok := apperror.As(err); ok {
		if appErr.Message != "" {
			return appErr.Message
		}
	}
	return "gagal menyimpan data"
}
