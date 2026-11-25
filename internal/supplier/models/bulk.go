package models

// BulkCreateSupplierRequest is the payload for importing many suppliers at once.
type BulkCreateSupplierRequest struct {
	Items []CreateSupplierRequest `json:"items"`
}

// BulkRowResult describes the outcome of a single row in a bulk import.
type BulkRowResult struct {
	Index        int    `json:"index"`        // 0-based position in the submitted items array
	Row          int    `json:"row"`          // 1-based spreadsheet row (header = row 1, data starts at 2)
	Status       string `json:"status"`       // "success" | "failed"
	ID           string `json:"id,omitempty"` // UUID of the created supplier
	SupplierCode string `json:"supplier_code,omitempty"`
	SupplierName string `json:"supplier_name,omitempty"`
	Message      string `json:"message,omitempty"` // error detail when status = failed
}

// BulkImportResult aggregates the outcome of a bulk import operation.
type BulkImportResult struct {
	Total        int             `json:"total"`
	SuccessCount int             `json:"success_count"`
	FailedCount  int             `json:"failed_count"`
	Results      []BulkRowResult `json:"results"`
}
