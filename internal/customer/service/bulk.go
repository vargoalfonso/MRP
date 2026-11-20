package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/customer/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/validator"
)

// maxBulkCustomerItems caps the number of rows accepted in a single request to
// avoid excessively long-running imports.
const maxBulkCustomerItems = 5000

// BulkCreate imports many customers in a single request. Each row is processed
// independently: a failing row is recorded in the result but does not stop the
// rest of the import. It reuses the existing Create logic so all validation,
// normalization and ID-generation rules stay identical to single creates.
func (s *service) BulkCreate(ctx context.Context, items []models.CreateCustomerRequest) (*models.BulkImportResult, error) {
	if len(items) == 0 {
		return nil, apperror.BadRequest("tidak ada data untuk diimport")
	}
	if len(items) > maxBulkCustomerItems {
		return nil, apperror.BadRequest(fmt.Sprintf("maksimal %d baris per import", maxBulkCustomerItems))
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

		customer, err := s.Create(ctx, item)
		if err != nil {
			row.Status = "failed"
			row.Message = bulkErrorMessage(err)
			result.FailedCount++
			result.Results = append(result.Results, row)
			continue
		}

		row.Status = "success"
		row.ID = customer.UUID
		row.CustomerID = customer.CustomerID
		row.CustomerName = customer.CustomerName
		result.SuccessCount++
		result.Results = append(result.Results, row)
	}

	return result, nil
}

// joinFieldErrors turns validator field errors into a single readable message.
func joinFieldErrors(errs []validator.FieldError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, "; ")
}

// bulkErrorMessage extracts a user-friendly message from an error, preferring
// the AppError message and never leaking internal causes.
func bulkErrorMessage(err error) string {
	if appErr, ok := apperror.As(err); ok {
		if appErr.Message != "" {
			return appErr.Message
		}
	}
	return "gagal menyimpan data"
}
