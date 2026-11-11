// Package bulkimport provides reusable types and helpers for bulk Excel import
// across all modules (BOM, PRL, user, etc.).
package bulkimport

// ImportStatus represents the overall outcome of a bulk import operation.
type ImportStatus string

const (
	StatusSuccess ImportStatus = "success"
	StatusPartial ImportStatus = "partial"
	StatusFailed  ImportStatus = "failed"
)

// RowError records a single row-level validation or DB error.
type RowError struct {
	Sheet   string `json:"sheet"`
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`

	// RawData stores original column values so they can be written back to the
	// error Excel file without re-reading the uploaded file.
	RawData []string `json:"-"`
}

// BulkResult is returned by every import service method.
type BulkResult struct {
	Status       ImportStatus `json:"import_status"`
	Total        int          `json:"total"`
	SuccessCount int          `json:"success_count"`
	FailedCount  int          `json:"failed_count"`
	// ErrorToken is non-empty when Status != StatusSuccess.
	// The handler converts this into a full download_url.
	ErrorToken string `json:"error_token,omitempty"`
	Errors     []RowError
}
