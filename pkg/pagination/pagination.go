// Package pagination provides URL query-param parsing helpers for list endpoints.
//
// Supported query params:
//
//	limit=20&page=1&search=LV7&orderBy=created_at&orderDirection=desc
//	filter=status:eq:Active
//
// Filter format: key:op:value  (comma-separated for multiple filters)
// Supported ops: eq (=), neq (!=)
package pagination

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/base/app"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type FilterInput struct {
	Key       string `json:"key"`
	Operation string `json:"operation"` // "=" or "!="
	Value     string `json:"value"`
}

// PaginationInput is the standard pagination struct for most list endpoints.
type PaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"` // "asc" | "desc"
}

// BomPaginationInput extends PaginationInput with BOM-specific filters.
type BomPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"` // searches uniq_code & part_name
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	UniqCode       string        `json:"uniq_code"`
	Status         string        `json:"status"` // Active | Inactive | Obsolete
}

// DateRangePaginationInput adds start/end date filtering.
type DateRangePaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	StartDate      string        `json:"start_date"`
	EndDate        string        `json:"end_date"`
}

// Offset returns the SQL offset value from Page and Limit.
func (p PaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

func (p BomPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

func (p DateRangePaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// ---------------------------------------------------------------------------
// Parsers
// ---------------------------------------------------------------------------

// Pagination parses standard pagination query params from the request URL.
//
//	?limit=20&page=1&search=LV7&orderBy=created_at&orderDirection=desc&filter=status:eq:Active
func Pagination(c *app.Context) PaginationInput {
	limit := 20
	page := 1
	orderBy := "created_at"
	orderDirection := "desc"
	search := ""
	var filter []FilterInput

	for key, values := range c.Request.URL.Query() {
		v := values[len(values)-1]
		switch key {
		case "limit":
			if n, err := strconv.Atoi(v); err == nil {
				limit = clampLimit(n)
			}
		case "page":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		case "search":
			search = v
		case "filter":
			filter = parseFilter(v)
		case "orderBy", "order_by":
			orderBy = v
		case "orderDirection", "order_direction":
			if v == "asc" || v == "desc" {
				orderDirection = v
			}
		}
	}

	return PaginationInput{
		Limit:          limit,
		Page:           page,
		Search:         search,
		Filter:         filter,
		OrderBy:        orderBy,
		OrderDirection: orderDirection,
	}
}

// BomPagination parses BOM-specific pagination params.
//
//	?limit=20&page=1&search=LV7&uniq_code=LV7&status=Active
func BomPagination(c *app.Context) BomPaginationInput {
	base := Pagination(c)

	uniqCode := c.Query("uniq_code")
	status := c.Query("status")

	// search also covers uniq_code if not set separately
	if uniqCode == "" && base.Search != "" {
		uniqCode = base.Search
	}

	return BomPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		UniqCode:       uniqCode,
		Status:         status,
	}
}

// POBudgetPaginationInput extends PaginationInput with PO Budget-specific filters.
type POBudgetPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	UniqCode       string        `json:"uniq_code"`
	CustomerID     int64         `json:"customer_id"`
	Period         string        `json:"period"` // e.g. "October 2025"
	Status         string        `json:"status"` // Draft | Pending | Approved | Rejected
}

func (p POBudgetPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// POBudgetPagination parses PO Budget-specific pagination params.
//
//	?limit=20&page=1&search=RM-001&uniq_code=RM-001&customer_id=5&period=October+2025&status=Draft
func POBudgetPagination(c *app.Context) POBudgetPaginationInput {
	base := Pagination(c)

	var customerID int64
	if v := c.Query("customer_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			customerID = n
		}
	}

	return POBudgetPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		UniqCode:       c.Query("uniq_code"),
		CustomerID:     customerID,
		Period:         c.Query("period"),
		Status:         c.Query("status"),
	}
}

// DateRangePagination parses pagination with start/end date filters.
//
//	?limit=20&page=1&startDate=2025-01-01&endDate=2025-12-31
func DateRangePagination(c *app.Context) DateRangePaginationInput {
	base := Pagination(c)
	return DateRangePaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		StartDate:      c.Query("startDate"),
		EndDate:        c.Query("endDate"),
	}
}

// ---------------------------------------------------------------------------
// Filter parsing
// ---------------------------------------------------------------------------

// parseFilter parses a comma-separated filter string.
//
//	"status:eq:Active"
//	→ [{Key:"status", Operation:"=", Value:"Active"}, ...]
func parseFilter(arg string) []FilterInput {
	var result []FilterInput
	for _, part := range strings.Split(arg, ",") {
		parts := strings.SplitN(part, ":", 3)
		if len(parts) != 3 {
			slog.Warn("pagination: invalid filter format, expected key:op:value", slog.String("input", part))
			continue
		}
		op, ok := mapOp(parts[1])
		if !ok {
			slog.Warn("pagination: unknown filter operation", slog.String("op", parts[1]))
			continue
		}
		result = append(result, FilterInput{
			Key:       parts[0],
			Operation: op,
			Value:     parts[2],
		})
	}
	return result
}

func mapOp(s string) (string, bool) {
	switch s {
	case "eq":
		return "=", true
	case "neq":
		return "!=", true
	case "like":
		return "ILIKE", true
	case "gt":
		return ">", true
	case "gte":
		return ">=", true
	case "lt":
		return "<", true
	case "lte":
		return "<=", true
	}
	return "", false
}

// ---------------------------------------------------------------------------
// Response meta (embed in any list response)
// ---------------------------------------------------------------------------

// Meta carries pagination metadata for API responses.
type Meta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// NewMeta builds pagination meta from count + input.
func NewMeta(total int64, p PaginationInput) Meta {
	pages := 0
	if p.Limit > 0 {
		pages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Meta{
		Total:      total,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalPages: pages,
	}
}

func NewMetaBom(total int64, p BomPaginationInput) Meta {
	pages := 0
	if p.Limit > 0 {
		pages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Meta{
		Total:      total,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalPages: pages,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func clampLimit(n int) int {
	if n < 1 {
		return 20
	}
	if n > 200 {
		return 200
	}
	return n
}
