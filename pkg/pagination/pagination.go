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

// WorkOrderPaginationInput extends PaginationInput with WO board filters.
//
// Supported query params (in addition to Pagination):
//
//	?status=Pending&approval_status=Approved&wo_type=New
//
// Compatibility:
//
//	?q=WO-2026 (alias for search)
type WorkOrderPaginationInput struct {
	PaginationInput
	Status         string `json:"status"`
	ApprovalStatus string `json:"approval_status"`
	WOType         string `json:"wo_type"`
	WOKind         string `json:"wo_kind"`
}

func (p WorkOrderPaginationInput) Offset() int { return p.PaginationInput.Offset() }

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

// WorkOrderPagination parses WO list pagination params.
func WorkOrderPagination(c *app.Context) WorkOrderPaginationInput {
	base := Pagination(c)
	// FE sometimes uses `q` instead of `search`.
	if q := c.Query("q"); q != "" {
		base.Search = q
	}

	return WorkOrderPaginationInput{
		PaginationInput: base,
		Status:          c.Query("status"),
		ApprovalStatus:  c.Query("approval_status"),
		WOType:          c.Query("wo_type"),
		WOKind:          c.Query("wo_kind"),
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
	Period         string        `json:"period"`         // e.g. "October 2025"
	Status         string        `json:"status"`         // Draft | Pending | Approved | Rejected
	BudgetSubtype  string        `json:"budget_subtype"` // regular | adhoc
}

type ApprovalManagerPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	Type           string        `json:"type"`
	Status         string        `json:"status"`
	SubmittedBy    string        `json:"submitted_by"`
	CurrentLevel   int           `json:"current_level"`
	Scope          string        `json:"scope"`
}

func (p ApprovalManagerPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

func ApprovalManagerPagination(c *app.Context) ApprovalManagerPaginationInput {
	base := Pagination(c)
	currentLevel := 0
	if v := c.Query("current_level"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			currentLevel = n
		}
	}
	return ApprovalManagerPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		Type:           c.Query("type"),
		Status:         c.Query("status"),
		SubmittedBy:    c.Query("submitted_by"),
		CurrentLevel:   currentLevel,
		Scope:          c.Query("scope"),
	}
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
		BudgetSubtype:  c.Query("budget_subtype"),
	}
}

// POBoardPaginationInput extends PaginationInput with PO board-specific filters.
type POBoardPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	PoType         string        `json:"po_type"`     // RM | INDIRECT | SUBCON
	Period         string        `json:"period"`      // YYYY-MM
	SupplierID     int64         `json:"supplier_id"` // legacy bigint
	UniqCode       string        `json:"uniq_code"`
	Status         string        `json:"status"`
	LateOnly       bool          `json:"late_only"`
}

func (p POBoardPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// POBoardPagination parses PO-board-specific pagination params.
//
//	?po_type=RM&period=2024-01&supplier_id=12&uniq_code=LV7&status=draft&late_only=true
func POBoardPagination(c *app.Context) POBoardPaginationInput {
	base := Pagination(c)

	var supplierID int64
	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = n
		}
	}

	lateOnly := false
	if c.Query("late_only") == "true" {
		lateOnly = true
	}

	return POBoardPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		PoType:         c.Query("po_type"),
		Period:         c.Query("period"),
		SupplierID:     supplierID,
		UniqCode:       c.Query("uniq_code"),
		Status:         c.Query("status"),
		LateOnly:       lateOnly,
	}
}

// ---------------------------------------------------------------------------
// QC Tasks
// ---------------------------------------------------------------------------

// QCTaskPaginationInput extends PaginationInput with QC task filters.
type QCTaskPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	TaskType       string        `json:"task_type"`
	Status         string        `json:"status"`
}

type QCDashboardPaginationInput struct {
	Limit          int           `json:"limit"`
	Page           int           `json:"page"`
	Search         string        `json:"search"`
	Filter         []FilterInput `json:"filter"`
	OrderBy        string        `json:"order_by"`
	OrderDirection string        `json:"order_direction"`
	DateFrom       string        `json:"date_from"`
	DateTo         string        `json:"date_to"`
	UniqCode       string        `json:"uniq_code"`
	WONumber       string        `json:"wo_number"`
	SupplierID     int64         `json:"supplier_id"`
	PONumber       string        `json:"po_number"`
	DefectSource   string        `json:"defect_source"`
	ReasonCode     string        `json:"reason_code"`
	Status         string        `json:"status"`
	GroupBy        string        `json:"group_by"`
}

func (p QCTaskPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

func (p QCDashboardPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// QCTaskPagination parses QC task list query params.
//
//	?task_type=incoming_qc&status=pending&limit=20&page=1
func QCTaskPagination(c *app.Context) QCTaskPaginationInput {
	base := Pagination(c)
	return QCTaskPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		TaskType:       c.Query("task_type"),
		Status:         c.Query("status"),
	}
}

func QCDashboardPagination(c *app.Context) QCDashboardPaginationInput {
	base := Pagination(c)
	var supplierID int64
	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = n
		}
	}
	return QCDashboardPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		Search:         base.Search,
		Filter:         base.Filter,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		DateFrom:       c.Query("date_from"),
		DateTo:         c.Query("date_to"),
		UniqCode:       c.Query("uniq_code"),
		WONumber:       c.Query("wo_number"),
		SupplierID:     supplierID,
		PONumber:       c.Query("po_number"),
		DefectSource:   c.Query("defect_source"),
		ReasonCode:     c.Query("reason_code"),
		Status:         c.Query("status"),
		GroupBy:        c.Query("group_by"),
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

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// InventoryRMPaginationInput extends PaginationInput with RM database filters.
//
//	?search=LV7&rm_type=sheet_plate&status=low_on_stock&buy_not_buy=buy&limit=20&page=1
type InventoryRMPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	RMType         string // sheet_plate | wire | ssp | others
	RMSource       string // process | supplier
	Status         string // low_on_stock | normal | overstock
	BuyNotBuy      string // buy | not_buy
}

func (p InventoryRMPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// InventoryRMPagination parses RM database list query params.
func InventoryRMPagination(c *app.Context) InventoryRMPaginationInput {
	base := Pagination(c)
	return InventoryRMPaginationInput{
		Limit: base.Limit, Page: base.Page, Search: base.Search,
		Filter: base.Filter, OrderBy: base.OrderBy, OrderDirection: base.OrderDirection,
		RMType:    c.Query("rm_type"),
		RMSource:  c.Query("rm_source"),
		Status:    c.Query("status"),
		BuyNotBuy: c.Query("buy_not_buy"),
	}
}

// InventoryIndirectPaginationInput extends PaginationInput with Indirect RM filters.
//
//	?search=NBR&status=normal&buy_not_buy=not_buy&limit=20&page=1
type InventoryIndirectPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	Status         string
	BuyNotBuy      string
}

func (p InventoryIndirectPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// InventoryIndirectPagination parses Indirect RM list query params.
func InventoryIndirectPagination(c *app.Context) InventoryIndirectPaginationInput {
	base := Pagination(c)
	return InventoryIndirectPaginationInput{
		Limit: base.Limit, Page: base.Page, Search: base.Search,
		Filter: base.Filter, OrderBy: base.OrderBy, OrderDirection: base.OrderDirection,
		Status:    c.Query("status"),
		BuyNotBuy: c.Query("buy_not_buy"),
	}
}

// InventorySubconPaginationInput extends PaginationInput with Subcon inventory filters.
//
//	?search=EMA&po_number=PO-2026-001&supplier_id=5&period=2026-04&status=normal&limit=20&page=1
type InventorySubconPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	PONumber       string
	SupplierID     int64
	Period         string
	Status         string
}

func (p InventorySubconPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// InventorySubconPagination parses Subcon inventory list query params.
func InventorySubconPagination(c *app.Context) InventorySubconPaginationInput {
	base := Pagination(c)
	var supplierID int64
	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = n
		}
	}
	return InventorySubconPaginationInput{
		Limit: base.Limit, Page: base.Page, Search: base.Search,
		Filter: base.Filter, OrderBy: base.OrderBy, OrderDirection: base.OrderDirection,
		PONumber:   c.Query("po_number"),
		SupplierID: supplierID,
		Period:     c.Query("period"),
		Status:     c.Query("status"),
	}
}

// InventoryIncomingPaginationInput extends PaginationInput with incoming scan filters.
//
//	?search=EMA&dn_type=RM&po_number=PO-2026-001&supplier_id=5&status=pending&limit=20&page=1
type InventoryIncomingPaginationInput struct {
	Limit          int
	Page           int
	Search         string
	Filter         []FilterInput
	OrderBy        string
	OrderDirection string
	DNType         string // RM | INDIRECT | SUBCON (maps to delivery_notes.type)
	PONumber       string
	Status         string // pending | in_progress | approved | rejected
	SupplierID     int64
}

func (p InventoryIncomingPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// InventoryIncomingPagination parses incoming scan list query params.
func InventoryIncomingPagination(c *app.Context) InventoryIncomingPaginationInput {
	base := Pagination(c)
	var supplierID int64
	if v := c.Query("supplier_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierID = n
		}
	}
	return InventoryIncomingPaginationInput{
		Limit: base.Limit, Page: base.Page, Search: base.Search,
		Filter: base.Filter, OrderBy: base.OrderBy, OrderDirection: base.OrderDirection,
		DNType:     c.Query("dn_type"),
		PONumber:   c.Query("po_number"),
		Status:     c.Query("status"),
		SupplierID: supplierID,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// SupplierPerformancePaginationInput holds filters for GET /suppliers/performance.
//
// Query params:
//
//	?period_type=monthly&period_value=2026-04&search=PT&status=excellent&page=1&limit=20&sort_by=otd_percentage&sort_direction=desc
type SupplierPerformancePaginationInput struct {
	Limit         int
	Page          int
	Search        string
	PeriodType    string
	PeriodValue   string
	Status        string
	SortBy        string
	SortDirection string
}

func (p SupplierPerformancePaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// SupplierPerformancePagination parses supplier performance list query params.
func SupplierPerformancePagination(c *app.Context) SupplierPerformancePaginationInput {
	base := Pagination(c)
	return SupplierPerformancePaginationInput{
		Limit:         base.Limit,
		Page:          base.Page,
		Search:        base.Search,
		PeriodType:    c.Query("period_type"),
		PeriodValue:   c.Query("period_value"),
		Status:        c.Query("status"),
		SortBy:        c.Query("sort_by"),
		SortDirection: c.Query("sort_direction"),
	}
}

func clampLimit(n int) int {
	if n < 1 {
		return 20
	}
	if n > 200 {
		return 200
	}
	return n
}

// ---------------------------------------------------------------------------
// Scrap Stock
// ---------------------------------------------------------------------------

// ScrapStockPaginationInput holds filters for GET /scrap-stocks.
//
// Query params:
//
//	?scrap_type=process_scrap&uniq=EMA-LV7&packing_number=KBN-001&wo_number=WO-001&status=Active&date_from=2026-01-01&date_to=2026-12-31&page=1&limit=20
type ScrapStockPaginationInput struct {
	Limit          int
	Page           int
	OrderBy        string
	OrderDirection string
	ScrapType      string
	UniqCode       string
	PackingNumber  string
	WONumber       string
	Status         string
	DateFrom       string
	DateTo         string
}

func (p ScrapStockPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// ScrapStockPagination parses scrap-stock-specific pagination params.
func ScrapStockPagination(c *app.Context) ScrapStockPaginationInput {
	base := Pagination(c)
	return ScrapStockPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		ScrapType:      c.Query("scrap_type"),
		UniqCode:       c.Query("uniq"),
		PackingNumber:  c.Query("packing_number"),
		WONumber:       c.Query("wo_number"),
		Status:         c.Query("status"),
		DateFrom:       c.Query("date_from"),
		DateTo:         c.Query("date_to"),
	}
}

// ScrapReleasePaginationInput holds filters for GET /inventory/scrap-releases.
//
// Query params:
//
//	?release_type=Sell&approval_status=Pending&scrap_stock_id=5&page=1&limit=20
type ScrapReleasePaginationInput struct {
	Limit          int
	Page           int
	OrderBy        string
	OrderDirection string
	ReleaseType    string
	ApprovalStatus string
	ScrapStockID   int64
}

func (p ScrapReleasePaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// ScrapReleasePagination parses scrap-release-specific pagination params.
func ScrapReleasePagination(c *app.Context) ScrapReleasePaginationInput {
	base := Pagination(c)

	var scrapStockID int64
	if v := c.Query("scrap_stock_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			scrapStockID = n
		}
	}

	return ScrapReleasePaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		ReleaseType:    c.Query("release_type"),
		ApprovalStatus: c.Query("approval_status"),
		ScrapStockID:   scrapStockID,
	}
}

// ---------------------------------------------------------------------------
// Finished Goods
// ---------------------------------------------------------------------------

// FinishedGoodsPaginationInput holds filters for GET /finished-goods.
//
// Query params:
//
//	?search=EMA&model=CB150&status=low_on_stock&warehouse_location=WH-A&page=1&limit=20
type FinishedGoodsPaginationInput struct {
	Limit             int
	Page              int
	OrderBy           string
	OrderDirection    string
	Search            string
	Model             string
	Status            string
	WarehouseLocation string
}

func (p FinishedGoodsPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// FinishedGoodsPagination parses finished-goods-specific pagination params.
func FinishedGoodsPagination(c *app.Context) FinishedGoodsPaginationInput {
	base := Pagination(c)
	return FinishedGoodsPaginationInput{
		Limit:             base.Limit,
		Page:              base.Page,
		OrderBy:           base.OrderBy,
		OrderDirection:    base.OrderDirection,
		Search:            base.Search,
		Model:             c.Query("model"),
		Status:            c.Query("status"),
		WarehouseLocation: c.Query("warehouse_location"),
	}
}

// StatusMonitoringPaginationInput holds filters for GET /finished-goods/status-monitoring.
//
// Query params:
//
//	?alert_type=low_on_stock&page=1&limit=20
type StatusMonitoringPaginationInput struct {
	Limit     int
	Page      int
	AlertType string
}

func (p StatusMonitoringPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// StatusMonitoringPagination parses status-monitoring-specific pagination params.
func StatusMonitoringPagination(c *app.Context) StatusMonitoringPaginationInput {
	base := Pagination(c)
	return StatusMonitoringPaginationInput{
		Limit:     base.Limit,
		Page:      base.Page,
		AlertType: c.Query("alert_type"),
	}
}

// ---------------------------------------------------------------------------
// Outgoing Raw Material
// ---------------------------------------------------------------------------

// OutgoingRMPaginationInput holds filters for GET /inventory/raw-materials/outgoing.
//
// Query params:
//
//	?search=OUT-RM&date_from=2024-01-01&date_to=2024-12-31&reason=Production+Use&uniq=RM-001&transaction_id=OUT-RM-00001&work_order_no=WO-2024-001&page=1&limit=20
type OutgoingRMPaginationInput struct {
	Limit          int
	Page           int
	OrderBy        string
	OrderDirection string
	Search         string
	DateFrom       string
	DateTo         string
	Reason         string
	Uniq           string
	TransactionID  string
	WorkOrderNo    string
}

func (p OutgoingRMPaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// OutgoingRMPagination parses outgoing RM list query params.
func OutgoingRMPagination(c *app.Context) OutgoingRMPaginationInput {
	base := Pagination(c)
	return OutgoingRMPaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		Search:         base.Search,
		DateFrom:       c.Query("date_from"),
		DateTo:         c.Query("date_to"),
		Reason:         c.Query("reason"),
		Uniq:           c.Query("uniq"),
		TransactionID:  c.Query("transaction_id"),
		WorkOrderNo:    c.Query("work_order_no"),
	}
}

// ---------------------------------------------------------------------------
// Stock Opname
// ---------------------------------------------------------------------------

// StockOpnamePaginationInput holds filters for GET /stock-opname-sessions.
type StockOpnamePaginationInput struct {
	Limit          int
	Page           int
	OrderBy        string
	OrderDirection string
	Type           string
	Status         string
	Period         string
}

func (p StockOpnamePaginationInput) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// StockOpnamePagination parses stock-opname-specific pagination params.
func StockOpnamePagination(c *app.Context) StockOpnamePaginationInput {
	base := Pagination(c)
	return StockOpnamePaginationInput{
		Limit:          base.Limit,
		Page:           base.Page,
		OrderBy:        base.OrderBy,
		OrderDirection: base.OrderDirection,
		Type:           c.Query("type"),
		Status:         c.Query("status"),
		Period:         c.Query("period"),
	}
}
