package pagination

import (
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
)

// ForecastingPaginationInput extends PaginationInput with forecasting-specific filters.
type ForecastingPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	// Module-specific filters
	Scope      string // global | custom
	Tenant     string
	Uniq       string
	Domain     string // dn | prl
	Status     string // PENDING | RUNNING | SUCCEEDED | FAILED | CANCELLED
	Stage      string // staging | prod
	ItemID     string
}

// Offset returns the SQL offset value from Page and Limit.
func (p ForecastingPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// ForecastingPagination parses forecasting-specific pagination params.
//
//	?scope=custom&tenant=mrp&uniq=138&domain=dn&status=SUCCEEDED&page=1&limit=20
func ForecastingPagination(c *app.Context) ForecastingPaginationInput {
	base := Pagination(c)
	return ForecastingPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		Scope:          c.Query("scope"),
		Tenant:         c.Query("tenant"),
		Uniq:           c.Query("uniq"),
		Domain:         c.Query("domain"),
		Status:         c.Query("status"),
		Stage:          c.Query("stage"),
		ItemID:         c.Query("item_id"),
	}
}

// InferenceResultPaginationInput extends PaginationInput with inference result filters.
type InferenceResultPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	Domain         string
	Tenant         string
	ItemID         string
	Status         string
	ModelVersionID string
}

// Offset returns the SQL offset value from Page and Limit.
func (p InferenceResultPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// InferenceResultPagination parses inference result list pagination params.
//
//	?domain=dn&tenant=mrp&item_id=138&status=SUCCEEDED&page=1&limit=20
func InferenceResultPagination(c *app.Context) InferenceResultPaginationInput {
	base := Pagination(c)
	return InferenceResultPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		Domain:         c.Query("domain"),
		Tenant:         c.Query("tenant"),
		ItemID:         c.Query("item_id"),
		Status:         c.Query("status"),
		ModelVersionID: c.Query("model_version_id"),
	}
}

// InferenceResultCountInput is the count query variant of InferenceResultPaginationInput.
type InferenceResultCountInput struct {
	Domain         string
	Tenant         string
	ItemID         string
	Status         string
	ModelVersionID string
}

// ParseInferenceResultPagination parses inference result list with explicit limit/page.
// Used for GET /inference-results?limit=20&page=1...
func ParseInferenceResultPagination(limitStr, pageStr string) InferenceResultPaginationInput {
	var limit, page int
	if n, err := strconv.Atoi(limitStr); err == nil {
		limit = clampLimit(n)
	} else {
		limit = 20
	}
	if n, err := strconv.Atoi(pageStr); err == nil && n > 0 {
		page = n
	} else {
		page = 1
	}
	return InferenceResultPaginationInput{
		Limit: limit,
		Page:  page,
	}
}
