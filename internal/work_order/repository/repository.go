package repository

import (
	"context"
	"strings"

	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	FindLastWONumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)
	CreateWorkOrder(ctx context.Context, tx *gorm.DB, wo *woModels.WorkOrder) error
	CreateWorkOrderItems(ctx context.Context, tx *gorm.DB, items []woModels.WorkOrderItem) error
	GetWorkOrderByUUID(ctx context.Context, woUUID string) (*woModels.WorkOrder, error)
	GetWorkOrderByUUIDAndKind(ctx context.Context, woUUID string, woKind string) (*woModels.WorkOrder, error)
	GetWorkOrderItemByUUID(ctx context.Context, itemUUID string) (*woModels.WorkOrderItem, error)
	GetWorkOrderItemsByWOID(ctx context.Context, woID int64) ([]woModels.WorkOrderItem, error)
	UpdateWorkOrderApprovalStatus(ctx context.Context, tx *gorm.DB, woID int64, status string) error
	UpdateWorkOrderQR(ctx context.Context, tx *gorm.DB, woID int64, base64 string) error
	UpdateWorkOrderItemQR(ctx context.Context, tx *gorm.DB, itemID int64, base64 string) error
	FindWorkOrdersByWONumbers(ctx context.Context, woNumbers []string) (map[string]*woModels.WorkOrder, error)
	FindWorkOrdersByWONumbersAndKind(ctx context.Context, woNumbers []string, woKind string) (map[string]*woModels.WorkOrder, error)
	GetSummary(ctx context.Context) (*SummaryRow, error)
	GetSummaryByKind(ctx context.Context, woKind string) (*SummaryRow, error)
	GetItemsByWOIDs(ctx context.Context, woIDs []int64) ([]woModels.WorkOrderItem, error)
	ListBulkSourceDocuments(ctx context.Context, documentType, q, targetDate string, limit int) ([]BulkSourceDocumentRow, error)
	GetBulkSourceDocument(ctx context.Context, documentUUID string) (*BulkSourceDocumentHeaderRow, error)
	ListBulkSourceDocumentItems(ctx context.Context, documentUUID string) ([]BulkSourceDocumentItemRow, error)

	ListWorkOrders(ctx context.Context, f ListFilter) ([]WorkOrderRow, int64, error)
}

type repository struct {
	db *gorm.DB
}

type BulkSourceDocumentRow struct {
	DocumentUUID   string `gorm:"column:document_uuid"`
	DocumentNumber string `gorm:"column:document_number"`
	DocumentType   string `gorm:"column:document_type"`
	DocumentDate   string `gorm:"column:document_date"`
	CustomerName   string `gorm:"column:customer_name"`
	ItemCount      int    `gorm:"column:item_count"`
}

type BulkSourceDocumentHeaderRow struct {
	DocumentUUID   string `gorm:"column:document_uuid"`
	DocumentNumber string `gorm:"column:document_number"`
	DocumentType   string `gorm:"column:document_type"`
	DocumentDate   string `gorm:"column:document_date"`
	CustomerName   string `gorm:"column:customer_name"`
}

type BulkSourceDocumentItemRow struct {
	SourceLineUUID string  `gorm:"column:source_line_uuid"`
	ItemUniqCode   string  `gorm:"column:item_uniq_code"`
	PartName       string  `gorm:"column:part_name"`
	PartNumber     string  `gorm:"column:part_number"`
	UOM            string  `gorm:"column:uom"`
	Quantity       float64 `gorm:"column:quantity"`
	TargetDate     *string `gorm:"column:target_date"`
}

func New(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) q(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *repository) FindLastWONumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error) {
	var wo woModels.WorkOrder
	err := r.q(ctx, tx).
		Select("wo_number").
		Where("wo_number LIKE ?", prefix+"-%").
		Order("wo_number DESC").
		Limit(1).
		Take(&wo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", apperror.InternalWrap("failed to query last work order number", err)
	}
	return wo.WoNumber, nil
}

func (r *repository) CreateWorkOrder(ctx context.Context, tx *gorm.DB, wo *woModels.WorkOrder) error {
	if err := r.q(ctx, tx).Create(wo).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "column \"wo_kind\"") {
			return apperror.BadRequest("missing DB column work_orders.wo_kind; run migration scripts/migrations/0041_add_work_orders_kind_up.sql")
		}
		if strings.Contains(msg, "column \"source_material_uniq\"") {
			return apperror.BadRequest("missing RM columns on work_orders; run migration scripts/migrations/0044_add_rm_processing_fields_to_work_orders_up.sql")
		}
		// unique constraint violations will bubble up; keep message generic
		return apperror.InternalWrap("failed to create work order", err)
	}
	return nil
}

func (r *repository) CreateWorkOrderItems(ctx context.Context, tx *gorm.DB, items []woModels.WorkOrderItem) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.q(ctx, tx).Create(&items).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "column \"process_flow_json\"") || strings.Contains(msg, "column \"current_step_seq\"") {
			return apperror.BadRequest("missing poka-yoke columns on work_order_items; run migration scripts/migrations/0043_create_work_order_poka_yoke_up.sql")
		}
		return apperror.InternalWrap("failed to create work order items", err)
	}
	return nil
}

func (r *repository) GetWorkOrderByUUID(ctx context.Context, woUUID string) (*woModels.WorkOrder, error) {
	var wo woModels.WorkOrder
	err := r.db.WithContext(ctx).Where("uuid = ?", woUUID).Take(&wo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("work order not found")
		}
		return nil, apperror.InternalWrap("failed to load work order", err)
	}
	return &wo, nil
}

func (r *repository) GetWorkOrderByUUIDAndKind(ctx context.Context, woUUID string, woKind string) (*woModels.WorkOrder, error) {
	var wo woModels.WorkOrder
	err := r.db.WithContext(ctx).
		Where("uuid = ? AND wo_kind = ?", woUUID, woKind).
		Take(&wo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("work order not found")
		}
		return nil, apperror.InternalWrap("failed to load work order", err)
	}
	return &wo, nil
}

func (r *repository) GetWorkOrderItemByUUID(ctx context.Context, itemUUID string) (*woModels.WorkOrderItem, error) {
	var it woModels.WorkOrderItem
	err := r.db.WithContext(ctx).Where("uuid = ?", itemUUID).Take(&it).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("work order item not found")
		}
		return nil, apperror.InternalWrap("failed to load work order item", err)
	}
	return &it, nil
}

func (r *repository) GetWorkOrderItemsByWOID(ctx context.Context, woID int64) ([]woModels.WorkOrderItem, error) {
	var items []woModels.WorkOrderItem
	if err := r.db.WithContext(ctx).
		Where("wo_id = ?", woID).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, apperror.InternalWrap("failed to load work order items", err)
	}
	return items, nil
}

func (r *repository) UpdateWorkOrderApprovalStatus(ctx context.Context, tx *gorm.DB, woID int64, approvalStatus string) error {
	cols := map[string]interface{}{
		"approval_status": approvalStatus,
		"updated_at":      gorm.Expr("NOW()"),
	}
	// When approved, advance status from Draft → Pending (ready to start).
	if approvalStatus == "Approved" {
		cols["status"] = "Pending"
	}
	res := r.q(ctx, tx).
		Table("work_orders").
		Where("id = ?", woID).
		Updates(cols)
	if res.Error != nil {
		return apperror.InternalWrap("failed to update approval status", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("work order not found")
	}
	return nil
}

func (r *repository) UpdateWorkOrderQR(ctx context.Context, tx *gorm.DB, woID int64, base64 string) error {
	res := r.q(ctx, tx).
		Table("work_orders").
		Where("id = ?", woID).
		Updates(map[string]interface{}{
			"qr_image_base64": base64,
			"updated_at":      gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return apperror.InternalWrap("failed to update work order qr", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("work order not found")
	}
	return nil
}

func (r *repository) UpdateWorkOrderItemQR(ctx context.Context, tx *gorm.DB, itemID int64, base64 string) error {
	res := r.q(ctx, tx).
		Table("work_order_items").
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"qr_image_base64": base64,
			"updated_at":      gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return apperror.InternalWrap("failed to update work order item qr", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("work order item not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Bulk helpers
// ---------------------------------------------------------------------------

func (r *repository) FindWorkOrdersByWONumbers(ctx context.Context, woNumbers []string) (map[string]*woModels.WorkOrder, error) {
	if len(woNumbers) == 0 {
		return map[string]*woModels.WorkOrder{}, nil
	}
	var list []woModels.WorkOrder
	if err := r.db.WithContext(ctx).
		Where("wo_number IN ?", woNumbers).
		Find(&list).Error; err != nil {
		return nil, apperror.InternalWrap("failed to find work orders by wo_numbers", err)
	}
	out := make(map[string]*woModels.WorkOrder, len(list))
	for i := range list {
		out[list[i].WoNumber] = &list[i]
	}
	return out, nil
}

func (r *repository) FindWorkOrdersByWONumbersAndKind(ctx context.Context, woNumbers []string, woKind string) (map[string]*woModels.WorkOrder, error) {
	if len(woNumbers) == 0 {
		return map[string]*woModels.WorkOrder{}, nil
	}
	var list []woModels.WorkOrder
	if err := r.db.WithContext(ctx).
		Where("wo_number IN ? AND wo_kind = ?", woNumbers, woKind).
		Find(&list).Error; err != nil {
		return nil, apperror.InternalWrap("failed to find work orders by wo_numbers", err)
	}
	out := make(map[string]*woModels.WorkOrder, len(list))
	for i := range list {
		out[list[i].WoNumber] = &list[i]
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

type SummaryRow struct {
	ActiveWOs  int `gorm:"column:active_wos"`
	Completed  int `gorm:"column:completed"`
	PendingWOs int `gorm:"column:pending_wos"`
	TotalUniqs int `gorm:"column:total_uniqs"`
}

func (r *repository) GetSummary(ctx context.Context) (*SummaryRow, error) {
	return r.GetSummaryByKind(ctx, "")
}

func (r *repository) GetSummaryByKind(ctx context.Context, woKind string) (*SummaryRow, error) {
	var row SummaryRow
	query := `
		SELECT
			COUNT(*) FILTER (WHERE LOWER(wo.status) = 'in progress') AS active_wos,
			COUNT(*) FILTER (WHERE LOWER(wo.status) = 'completed')   AS completed,
			COUNT(*) FILTER (WHERE LOWER(wo.status) = 'pending')     AS pending_wos,
			COALESCE((
				SELECT COUNT(DISTINCT wi.item_uniq_code)
				FROM work_order_items wi
				INNER JOIN work_orders woi ON woi.id = wi.wo_id
				WHERE LOWER(woi.status) NOT IN ('closed', 'cancelled')
				%s
			), 0) AS total_uniqs
		FROM work_orders wo
		%s
	`
	innerKindFilter := ""
	outerKindFilter := ""
	args := make([]interface{}, 0, 2)
	if strings.TrimSpace(woKind) != "" {
		innerKindFilter = " AND woi.wo_kind = ?"
		outerKindFilter = " WHERE wo.wo_kind = ?"
		args = append(args, woKind, woKind)
	}
	err := r.db.WithContext(ctx).Raw(
		strings.Replace(strings.Replace(query, "%s", innerKindFilter, 1), "%s", outerKindFilter, 1),
		args...,
	).Scan(&row).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to get work order summary", err)
	}
	return &row, nil
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

type ListFilter struct {
	Search         string
	Status         string
	ApprovalStatus string
	WOType         string
	WOKind         string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type WorkOrderRow struct {
	ID                 int64    `gorm:"column:id"`
	UUID               string   `gorm:"column:uuid"`
	WoNumber           string   `gorm:"column:wo_number"`
	WoType             string   `gorm:"column:wo_type"`
	WOKind             string   `gorm:"column:wo_kind"`
	ReferenceWO        *string  `gorm:"column:reference_wo"`
	Status             string   `gorm:"column:status"`
	ApprovalStatus     string   `gorm:"column:approval_status"`
	CreatedDate        string   `gorm:"column:created_date"`
	TargetDate         *string  `gorm:"column:target_date"`
	CreatedByName      *string  `gorm:"column:created_by_name"`
	UniqCount          int      `gorm:"column:uniq_count"`
	ItemCount          int      `gorm:"column:item_count"`
	ClosedCount        int      `gorm:"column:closed_count"`
	AgingDays          int      `gorm:"column:aging_days"`
	SourceMaterialUniq *string  `gorm:"column:source_material_uniq"`
	TargetMaterialUniq *string  `gorm:"column:target_material_uniq"`
	Model              *string  `gorm:"column:model"`
	GradeSize          *string  `gorm:"column:grade_size"`
	InputQty           *float64 `gorm:"column:input_qty"`
	InputUOM           *string  `gorm:"column:input_uom"`
	OutputQty          *float64 `gorm:"column:output_qty"`
	OutputUOM          *string  `gorm:"column:output_uom"`
	DateIssued         *string  `gorm:"column:date_issued"`
	DateCompleted      *string  `gorm:"column:date_completed"`
	CycleTimeDays      *int     `gorm:"column:cycle_time_days"`
	Remarks            *string  `gorm:"column:remarks"`
}

func (r *repository) ListWorkOrders(ctx context.Context, f ListFilter) ([]WorkOrderRow, int64, error) {
	base := r.db.WithContext(ctx).Table("work_orders")

	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		base = base.Where("(wo_number ILIKE ? OR reference_wo ILIKE ?)", like, like)
	}
	if f.Status != "" {
		base = base.Where("status = ?", f.Status)
	}
	if f.ApprovalStatus != "" {
		base = base.Where("approval_status = ?", f.ApprovalStatus)
	}
	if f.WOType != "" {
		base = base.Where("wo_type = ?", f.WOType)
	}
	if f.WOKind != "" {
		base = base.Where("wo_kind = ?", f.WOKind)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("failed to count work orders", err)
	}

	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	orderDir := strings.ToLower(f.OrderDirection)
	if orderDir != "asc" && orderDir != "desc" {
		orderDir = "desc"
	}

	// Rebuild with alias so joins work cleanly.
	dq := r.db.WithContext(ctx).Table("work_orders wo")
	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		dq = dq.Where("(wo.wo_number ILIKE ? OR wo.reference_wo ILIKE ?)", like, like)
	}
	if f.Status != "" {
		dq = dq.Where("wo.status = ?", f.Status)
	}
	if f.ApprovalStatus != "" {
		dq = dq.Where("wo.approval_status = ?", f.ApprovalStatus)
	}
	if f.WOType != "" {
		dq = dq.Where("wo.wo_type = ?", f.WOType)
	}
	if f.WOKind != "" {
		dq = dq.Where("wo.wo_kind = ?", f.WOKind)
	}

	var rows []WorkOrderRow
	err := dq.
		Select(strings.Join([]string{
			"wo.id",
			"wo.uuid",
			"wo.wo_number",
			"wo.wo_type",
			"wo.wo_kind",
			"wo.reference_wo",
			"wo.status",
			"wo.approval_status",
			"wo.created_by_name",
			"wo.source_material_uniq",
			"wo.target_material_uniq",
			"wo.model",
			"wo.grade_size",
			"wo.input_qty",
			"wo.input_uom",
			"wo.output_qty",
			"wo.output_uom",
			"CASE WHEN wo.date_issued IS NULL THEN NULL ELSE to_char(wo.date_issued, 'YYYY-MM-DD') END AS date_issued",
			"CASE WHEN wo.date_completed IS NULL THEN NULL ELSE to_char(wo.date_completed, 'YYYY-MM-DD') END AS date_completed",
			"wo.cycle_time_days",
			"wo.remarks",
			"to_char(wo.created_date, 'YYYY-MM-DD') AS created_date",
			"CASE WHEN wo.target_date IS NULL THEN NULL ELSE to_char(wo.target_date, 'YYYY-MM-DD') END AS target_date",
			"COALESCE(s.uniq_count, 0) AS uniq_count",
			"COALESCE(s.item_count, 0) AS item_count",
			"COALESCE(s.closed_count, 0) AS closed_count",
			"COALESCE(EXTRACT(DAY FROM NOW() - wo.created_date)::int, 0) AS aging_days",
		}, ", ")).
		Joins(`LEFT JOIN (
			SELECT wo_id,
				COUNT(DISTINCT item_uniq_code) AS uniq_count,
				COUNT(*) AS item_count,
				COUNT(*) FILTER (WHERE LOWER(status) = 'closed') AS closed_count
			FROM work_order_items
			GROUP BY wo_id
		) s ON s.wo_id = wo.id`).
		Order("wo." + orderBy + " " + orderDir).
		Limit(f.Limit).
		Offset(f.Offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("failed to list work orders", err)
	}
	return rows, total, nil
}

func (r *repository) GetItemsByWOIDs(ctx context.Context, woIDs []int64) ([]woModels.WorkOrderItem, error) {
	if len(woIDs) == 0 {
		return nil, nil
	}
	var items []woModels.WorkOrderItem
	if err := r.db.WithContext(ctx).
		Where("wo_id IN ?", woIDs).
		Order("wo_id ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, apperror.InternalWrap("failed to load work order items", err)
	}
	return items, nil
}

func (r *repository) ListBulkSourceDocuments(ctx context.Context, documentType, q, targetDate string, limit int) ([]BulkSourceDocumentRow, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows []BulkSourceDocumentRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			cod.uuid AS document_uuid,
			cod.document_number,
			cod.document_type,
			to_char(cod.document_date, 'YYYY-MM-DD') AS document_date,
			cod.customer_name_snapshot AS customer_name,
			COUNT(codi.id) AS item_count
		FROM customer_order_documents cod
		JOIN customer_order_document_items codi ON codi.document_id = cod.id
		WHERE cod.deleted_at IS NULL
		  AND (? = '' OR cod.document_type = ?)
		  AND (cod.document_type = 'DN' AND cod.status = 'active'
		       OR cod.document_type IN ('SO','PO') AND cod.status <> 'cancelled')
		  AND (? = '' OR cod.document_number ILIKE '%' || ? || '%' OR cod.customer_name_snapshot ILIKE '%' || ? || '%')
		  AND (? = '' OR EXISTS (
			SELECT 1
			FROM customer_order_document_items codi2
			WHERE codi2.document_id = cod.id
			  AND to_char(COALESCE(codi2.delivery_date, cod.document_date), 'YYYY-MM-DD') = ?
		  ))
		GROUP BY cod.id, cod.uuid, cod.document_number, cod.document_type, cod.document_date, cod.customer_name_snapshot
		ORDER BY cod.document_date DESC, cod.document_number DESC
		LIMIT ?
	`, documentType, documentType, q, q, q, targetDate, targetDate, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to list bulk source documents", err)
	}
	return rows, nil
}

func (r *repository) GetBulkSourceDocument(ctx context.Context, documentUUID string) (*BulkSourceDocumentHeaderRow, error) {
	var row BulkSourceDocumentHeaderRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			cod.uuid AS document_uuid,
			cod.document_number,
			cod.document_type,
			to_char(cod.document_date, 'YYYY-MM-DD') AS document_date,
			cod.customer_name_snapshot AS customer_name
		FROM customer_order_documents cod
		WHERE cod.deleted_at IS NULL
		  AND cod.uuid = ?
		  AND cod.document_type IN ('SO','PO','DN')
		  AND (cod.document_type = 'DN' AND cod.status = 'active'
		       OR cod.document_type IN ('SO','PO') AND cod.status <> 'cancelled')
		LIMIT 1
	`, documentUUID).Scan(&row).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to load bulk source document", err)
	}
	if strings.TrimSpace(row.DocumentUUID) == "" {
		return nil, apperror.NotFound("source document not found")
	}
	return &row, nil
}

func (r *repository) ListBulkSourceDocumentItems(ctx context.Context, documentUUID string) ([]BulkSourceDocumentItemRow, error) {
	var rows []BulkSourceDocumentItemRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			codi.uuid AS source_line_uuid,
			codi.item_uniq_code,
			codi.part_name,
			codi.part_number,
			codi.uom,
			codi.quantity,
			to_char(codi.delivery_date, 'YYYY-MM-DD') AS target_date
		FROM customer_order_document_items codi
		JOIN customer_order_documents cod ON cod.id = codi.document_id
		WHERE cod.deleted_at IS NULL
		  AND cod.uuid = ?
		  AND cod.document_type IN ('SO','PO','DN')
		  AND (cod.document_type = 'DN' AND cod.status = 'active'
		       OR cod.document_type IN ('SO','PO') AND cod.status <> 'cancelled')
		ORDER BY codi.line_no ASC, codi.id ASC
	`, documentUUID).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("failed to list bulk source document items", err)
	}
	return rows, nil
}
