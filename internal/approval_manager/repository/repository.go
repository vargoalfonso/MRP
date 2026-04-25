package repository

import (
	"context"
	"math"
	"strings"

	approvalModels "github.com/ganasa18/go-template/internal/approval_manager/models"
	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"gorm.io/gorm"
)

type IRepository interface {
	GetSummary(ctx context.Context, filterType string) (*approvalModels.SummaryResponse, error)
	ListItems(ctx context.Context, q approvalModels.ListQuery) ([]approvalModels.Item, int64, error)
	GetDetail(ctx context.Context, instanceID int64) (*approvalModels.DetailResponse, error)
	BuildPagination(total int64, page, limit int) approvalModels.Pagination
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) GetSummary(ctx context.Context, filterType string) (*approvalModels.SummaryResponse, error) {
	base, args := approvalManagerBaseQuery(filterType, "", "", 0)
	query := `SELECT COALESCE(COUNT(*) FILTER (WHERE src.status = 'pending'), 0) AS pending, COALESCE(COUNT(*) FILTER (WHERE src.status = 'approved'), 0) AS approved, COALESCE(COUNT(*) FILTER (WHERE src.status = 'rejected'), 0) AS rejected, COALESCE(COUNT(*), 0) AS total FROM (` + base + `) src`
	var row struct {
		Pending  int64 `gorm:"column:pending"`
		Approved int64 `gorm:"column:approved"`
		Rejected int64 `gorm:"column:rejected"`
		Total    int64 `gorm:"column:total"`
	}
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&row).Error; err != nil {
		return nil, err
	}
	return &approvalModels.SummaryResponse{Type: normalizeType(filterType), Pending: row.Pending, Approved: row.Approved, Rejected: row.Rejected, Total: row.Total}, nil
}

func (r *repository) ListItems(ctx context.Context, q approvalModels.ListQuery) ([]approvalModels.Item, int64, error) {
	base, args := approvalManagerBaseQuery(q.Type, q.Status, q.Search, q.CurrentLevel)
	if submitted := strings.TrimSpace(q.SubmittedBy); submitted != "" {
		base = `SELECT * FROM (` + base + `) src WHERE src.submitted_by ILIKE ?`
		args = append(args, "%"+submitted+"%")
	}
	countQuery := `SELECT COUNT(*) FROM (` + base + `) src`
	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	listQuery := `SELECT * FROM (` + base + `) src ORDER BY src.created_at DESC, src.instance_id DESC LIMIT ? OFFSET ?`
	args = append(args, q.Limit, q.Offset)
	var rows []approvalModels.Item
	if err := r.db.WithContext(ctx).Raw(listQuery, args...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repository) GetDetail(ctx context.Context, instanceID int64) (*approvalModels.DetailResponse, error) {
	base, args := approvalManagerBaseQuery("all", "", "", 0)
	query := `SELECT * FROM (` + base + `) src WHERE src.instance_id = ? LIMIT 1`
	args = append(args, instanceID)
	var row struct {
		InstanceID       int64                     `gorm:"column:instance_id"`
		Module           string                    `gorm:"column:module"`
		ReferenceTable   string                    `gorm:"column:reference_table"`
		ReferenceID      int64                     `gorm:"column:reference_id"`
		WorkflowID       int64                     `gorm:"column:workflow_id"`
		WorkflowAction   string                    `gorm:"column:workflow_action"`
		Level1Role       string                    `gorm:"column:level_1_role"`
		Level2Role       string                    `gorm:"column:level_2_role"`
		Level3Role       string                    `gorm:"column:level_3_role"`
		Level4Role       string                    `gorm:"column:level_4_role"`
		CurrentLevel     int                       `gorm:"column:current_level"`
		MaxLevel         int                       `gorm:"column:max_level"`
		Status           string                    `gorm:"column:status"`
		SubmittedBy      string                    `gorm:"column:submitted_by"`
		ApprovalProgress awmodels.ApprovalProgress `gorm:"column:approval_progress;type:jsonb;serializer:json"`
		DocumentID       string                    `gorm:"column:document_id"`
		DocumentUUID     string                    `gorm:"column:document_uuid"`
		ItemName         string                    `gorm:"column:item_name"`
		DetailURL        string                    `gorm:"column:detail_url"`
	}
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.InstanceID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &approvalModels.DetailResponse{InstanceID: row.InstanceID, Module: row.Module, ReferenceTable: row.ReferenceTable, ReferenceID: row.ReferenceID, Workflow: awmodels.ApprovalWorkflow{ID: row.WorkflowID, ActionName: row.WorkflowAction, Level1Role: row.Level1Role, Level2Role: row.Level2Role, Level3Role: row.Level3Role, Level4Role: row.Level4Role}, CurrentLevel: row.CurrentLevel, MaxLevel: row.MaxLevel, Status: row.Status, SubmittedBy: row.SubmittedBy, ApprovalProgress: row.ApprovalProgress, Document: awmodels.ApprovalDocument{ID: row.ReferenceID, UUID: row.DocumentUUID, Code: row.DocumentID, Name: row.ItemName, DetailURL: row.DetailURL}}, nil
}

func (r *repository) BuildPagination(total int64, page, limit int) approvalModels.Pagination {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}
	return approvalModels.Pagination{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}

func normalizeType(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	switch v {
	case "bom", "prl", "po_budget", "stock_opname":
		return v
	default:
		return "all"
	}
}

func approvalManagerBaseQuery(filterType, status, search string, currentLevel int) (string, []interface{}) {
	query := `
	SELECT
		ai.id AS instance_id,
		ai.action_name AS module,
		CASE ai.action_name
			WHEN 'bom' THEN 'Bill of Material'
			WHEN 'prl' THEN 'PRL Management'
			WHEN 'po_budget' THEN 'PO Budget'
			WHEN 'stock_opname' THEN 'Stock Opname'
			ELSE ai.action_name
		END AS module_label,
		ai.reference_table,
		ai.reference_id,
		ai.approval_workflow_id AS workflow_id,
		aw.action_name AS workflow_action,
		aw.level_1_role,
		aw.level_2_role,
		aw.level_3_role,
		aw.level_4_role,
		ai.current_level,
		ai.max_level,
		ai.status,
		ai.submitted_by,
		ai.approval_progress,
		ai.created_at,
		CASE ai.current_level WHEN 1 THEN aw.level_1_role WHEN 2 THEN aw.level_2_role WHEN 3 THEN aw.level_3_role WHEN 4 THEN aw.level_4_role ELSE '' END AS current_level_role,
		CASE WHEN ai.current_level >= ai.max_level THEN TRUE ELSE FALSE END AS is_final_level,
		COALESCE(src.document_id, ai.reference_id::text) AS document_id,
		COALESCE(src.document_uuid, '') AS document_uuid,
		COALESCE(src.item_name, '') AS item_name,
		COALESCE(src.item_code, '') AS item_code,
		COALESCE(src.detail_url, '') AS detail_url,
		COALESCE(src.approval_url, '') AS approval_url,
		COALESCE(src.submitted_by_name, ai.submitted_by) AS submitted_by_name,
		COALESCE(src.submitted_at, ai.created_at::text) AS submitted_at
	FROM approval_instances ai
	JOIN approval_workflows aw ON aw.id = ai.approval_workflow_id
	LEFT JOIN (
		SELECT 'bom' AS action_name, bi.id AS reference_id, bi.id::text AS document_id, ''::text AS document_uuid,
			COALESCE(i.part_name, 'BOM') AS item_name, COALESCE(i.uniq_code, bi.id::text) AS item_code,
			('/api/v1/products/bom/' || bi.id)::text AS detail_url,
			('/api/v1/products/bom/' || bi.id || '/approval')::text AS approval_url,
			COALESCE(ai2.submitted_by, '') AS submitted_by_name,
			ai2.created_at::text AS submitted_at
		FROM bom_item bi
		LEFT JOIN items i ON i.id = bi.item_id
		LEFT JOIN approval_instances ai2 ON ai2.action_name = 'bom' AND ai2.reference_table = 'bom_item' AND ai2.reference_id = bi.id
		UNION ALL
		SELECT 'prl' AS action_name, p.id AS reference_id, COALESCE(p.prl_id, p.id::text) AS document_id, COALESCE(p.uuid::text, '') AS document_uuid,
			COALESCE(p.part_name, 'PRL') AS item_name, COALESCE(p.uniq_code, p.part_number, p.prl_id, p.id::text) AS item_code,
			('/api/v1/prls/' || p.id || '/detail')::text AS detail_url,
			('/api/v1/prls/actions/approve')::text AS approval_url,
			''::text AS submitted_by_name,
			p.created_at::text AS submitted_at
		FROM prls p
		UNION ALL
		SELECT 'stock_opname' AS action_name, s.id AS reference_id, COALESCE(s.session_number, s.id::text) AS document_id, COALESCE(s.uuid::text, '') AS document_uuid,
			COALESCE('Stock Opname - ' || COALESCE(s.warehouse_location, '') || ' ' || LPAD(s.period_month::text, 2, '0') || '/' || s.period_year::text, 'Stock Opname') AS item_name,
			COALESCE(s.session_number, s.id::text) AS item_code,
			('/api/v1/stock-opname-sessions/' || s.id)::text AS detail_url,
			('/api/v1/stock-opname-sessions/' || s.id || '/approve')::text AS approval_url,
			COALESCE(s.submitted_by, '') AS submitted_by_name,
			COALESCE(s.submitted_at::text, s.created_at::text) AS submitted_at
		FROM stock_opname_sessions s
		UNION ALL
		SELECT 'po_budget' AS action_name, pbe.id AS reference_id, COALESCE(pbe.po_budget_ref, pbe.id::text) AS document_id, ''::text AS document_uuid,
			COALESCE(pbe.part_name, 'PO Budget') AS item_name, COALESCE(pbe.uniq_code, pbe.id::text) AS item_code,
			('/api/v1/po-budget/' || REPLACE(pbe.budget_type, '_', '-') || '/budget/' || pbe.id)::text AS detail_url,
			('')::text AS approval_url,
			COALESCE(pbe.created_by, '') AS submitted_by_name,
			pbe.created_at::text AS submitted_at
		FROM po_budget_entries pbe
	) src ON src.action_name = ai.action_name AND src.reference_id = ai.reference_id
	WHERE 1=1`
	args := make([]interface{}, 0)
	if t := normalizeType(filterType); t != "all" {
		query += ` AND ai.action_name = ?`
		args = append(args, t)
	}
	if s := strings.TrimSpace(status); s != "" {
		query += ` AND ai.status = ?`
		args = append(args, s)
	}
	if currentLevel > 0 {
		query += ` AND ai.current_level = ?`
		args = append(args, currentLevel)
	}
	if s := strings.TrimSpace(search); s != "" {
		query += ` AND (COALESCE(src.document_id, '') ILIKE ? OR COALESCE(src.item_name, '') ILIKE ? OR COALESCE(src.item_code, '') ILIKE ?)`
		like := "%" + s + "%"
		args = append(args, like, like, like)
	}
	return query, args
}
