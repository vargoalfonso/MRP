package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	qcModels "github.com/ganasa18/go-template/internal/qc/models"
	"github.com/ganasa18/go-template/internal/qc_dashboard/models"
	scrapModels "github.com/ganasa18/go-template/internal/scrap_stock/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Filter struct {
	Limit        int
	Page         int
	Offset       int
	Search       string
	DateFrom     string
	DateTo       string
	Status       string
	WONumber     string
	UniqCode     string
	SupplierID   int64
	PONumber     string
	DefectSource string
	ReasonCode   string
	GroupBy      string
	WindowHours  int
}

type IRepository interface {
	GetOverviewCards(ctx context.Context, filter Filter) (models.OverviewCards, time.Time, error)
	GetOverviewBySource(ctx context.Context, filter Filter) ([]models.SourceSummary, error)
	GetTopIssues(ctx context.Context, filter Filter, limit int) ([]models.IssueSummary, error)
	CountPendingRework(ctx context.Context) (int64, error)
	ListProductionQC(ctx context.Context, filter Filter) (*models.ProductionQCListResponse, error)
	ListIncomingQC(ctx context.Context, filter Filter) (*models.IncomingQCListResponse, error)
	ListDefects(ctx context.Context, filter Filter) (*models.DefectListResponse, error)
	CreateManualQCReport(ctx context.Context, req models.CreateManualQCReportRequest, performedBy string) error
	CreateReworkTask(ctx context.Context, defectID int64, performedBy string) error
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) GetOverviewCards(ctx context.Context, filter Filter) (models.OverviewCards, time.Time, error) {
	where, args := buildRangeWhere(filter, "checked_at")
	query := `
		SELECT
			COUNT(*) AS total_reports,
			COALESCE(SUM(qty_defect), 0) AS total_defects,
			COALESCE(SUM(qty_scrap), 0) AS total_scrap,
			COALESCE(MAX(checked_at), NOW()) AS as_of
		FROM qc_logs` + where
	var cards struct {
		TotalReports int64     `gorm:"column:total_reports"`
		TotalDefects float64   `gorm:"column:total_defects"`
		TotalScrap   float64   `gorm:"column:total_scrap"`
		AsOf         time.Time `gorm:"column:as_of"`
	}
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&cards).Error; err != nil {
		return models.OverviewCards{}, time.Time{}, apperror.Internal("qc dashboard overview: " + err.Error())
	}
	return models.OverviewCards{
		TotalReports: cards.TotalReports,
		TotalDefects: cards.TotalDefects,
		TotalScrap:   cards.TotalScrap,
	}, cards.AsOf, nil
}

func (r *repository) CountPendingRework(ctx context.Context) (int64, error) {
	var pendingRework int64
	if err := r.db.WithContext(ctx).Table("qc_tasks").Where("task_type = ? AND status IN ?", "rework_qc", []string{"pending", "in_progress"}).Count(&pendingRework).Error; err != nil {
		return 0, apperror.Internal("qc dashboard rework count: " + err.Error())
	}
	return pendingRework, nil
}

func (r *repository) GetOverviewBySource(ctx context.Context, filter Filter) ([]models.SourceSummary, error) {
	where, args := buildRangeWhere(filter, "checked_at")
	var bySource []models.SourceSummary
	if err := r.db.WithContext(ctx).Raw(`
		SELECT defect_source, COALESCE(SUM(qty_defect), 0) AS qty_defect, COALESCE(SUM(qty_scrap), 0) AS qty_scrap
		FROM qc_logs `+where+`
		GROUP BY defect_source
		ORDER BY defect_source ASC
	`, args...).Scan(&bySource).Error; err != nil {
		return nil, apperror.Internal("qc dashboard by source: " + err.Error())
	}
	return bySource, nil
}

func (r *repository) GetTopIssues(ctx context.Context, filter Filter, limit int) ([]models.IssueSummary, error) {
	defectWhere, defectArgs := buildRangeWhere(filter, "reported_at")
	var topIssues []models.IssueSummary
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(NULLIF(defect_reason_code, ''), 'OTHER') AS reason_code,
			COALESCE(NULLIF(defect_reason_text, ''), 'Other') AS reason_text,
			COALESCE(SUM(qty_defect), 0) AS qty_defect
		FROM qc_defect_items `+defectWhere+`
		GROUP BY COALESCE(NULLIF(defect_reason_code, ''), 'OTHER'), COALESCE(NULLIF(defect_reason_text, ''), 'Other')
		ORDER BY qty_defect DESC, reason_code ASC
		LIMIT ?
	`, append(defectArgs, limit)...).Scan(&topIssues).Error; err != nil {
		return nil, apperror.Internal("qc dashboard top issues: " + err.Error())
	}
	return topIssues, nil
}

func (r *repository) ListProductionQC(ctx context.Context, filter Filter) (*models.ProductionQCListResponse, error) {
	where, args := buildProductionWhere(filter)
	countQuery := `
		SELECT COUNT(*)
		FROM qc_logs ql
		LEFT JOIN work_orders wo ON wo.id = ql.wo_id
		LEFT JOIN work_order_items woi ON woi.id = ql.wo_item_id` + where
	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal("count production qc: " + err.Error())
	}

	query := `
		SELECT
			ql.id AS qc_log_id,
			TO_CHAR(ql.checked_at::date, 'YYYY-MM-DD') AS report_date,
			COALESCE(wo.wo_number, '') AS wo_number,
			ql.uniq_code,
			COALESCE(woi.kanban_number, '') AS kanban_number,
			ql.qty_checked AS items_checked,
			(
				SELECT NULLIF(qdi.defect_reason_text, '')
				FROM qc_defect_items qdi
				WHERE qdi.qc_log_id = ql.id
				ORDER BY qdi.qty_defect DESC, qdi.id ASC
				LIMIT 1
			) AS issue_label,
			ql.qty_defect,
			ql.qty_scrap,
			CASE WHEN ql.qty_checked > 0 THEN ROUND((ql.qty_pass / ql.qty_checked) * 100, 2) ELSE 0 END AS quality_rate_percent,
			CASE WHEN UPPER(ql.status) IN ('PASSED', 'APPROVED') THEN 'passed' ELSE 'not_passed' END AS status
		FROM qc_logs ql
		LEFT JOIN work_orders wo ON wo.id = ql.wo_id
		LEFT JOIN work_order_items woi ON woi.id = ql.wo_item_id` + where + `
		ORDER BY ql.checked_at DESC, ql.id DESC
		LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)
	var items []models.ProductionQCItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, apperror.Internal("list production qc: " + err.Error())
	}
	return &models.ProductionQCListResponse{Items: items, Pagination: buildPagination(total, filter.Page, filter.Limit)}, nil
}

func (r *repository) ListIncomingQC(ctx context.Context, filter Filter) (*models.IncomingQCListResponse, error) {
	where, args := buildIncomingWhere(filter)
	countQuery := `
		SELECT COUNT(*)
		FROM qc_logs ql
		LEFT JOIN qc_tasks qt ON qt.id = ql.qc_task_id
		LEFT JOIN delivery_note_items dni ON dni.id = ql.dn_item_id
		LEFT JOIN delivery_notes dn ON dn.id = dni.dn_id
		LEFT JOIN suppliers s ON s.id = dn.supplier_id` + where
	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal("count incoming qc: " + err.Error())
	}

	query := `
		SELECT
			ql.id AS qc_log_id,
			COALESCE(qt.id, 0) AS qc_task_id,
			TO_CHAR(ql.checked_at::date, 'YYYY-MM-DD') AS report_date,
			COALESCE(dn.dn_number, '') AS dn_number,
			COALESCE(dni.packing_number, '') AS kanban_pl_scan,
			COALESCE(dn.po_number, '') AS po_number,
			dn.supplier_id,
			COALESCE(s.supplier_name, '') AS supplier_name,
			ql.uniq_code,
			ql.qty_checked AS items_checked,
			(
				SELECT NULLIF(qdi.defect_reason_text, '')
				FROM qc_defect_items qdi
				WHERE qdi.qc_log_id = ql.id
				ORDER BY qdi.qty_defect DESC, qdi.id ASC
				LIMIT 1
			) AS issue_label,
			ql.qty_defect,
			ql.qty_scrap,
			CASE WHEN ql.qty_checked > 0 THEN ROUND((ql.qty_pass / ql.qty_checked) * 100, 2) ELSE 0 END AS quality_rate_percent,
			CASE WHEN UPPER(ql.status) IN ('PASSED', 'APPROVED') THEN 'passed' ELSE 'not_passed' END AS status
		FROM qc_logs ql
		LEFT JOIN qc_tasks qt ON qt.id = ql.qc_task_id
		LEFT JOIN delivery_note_items dni ON dni.id = ql.dn_item_id
		LEFT JOIN delivery_notes dn ON dn.id = dni.dn_id
		LEFT JOIN suppliers s ON s.id = dn.supplier_id` + where + `
		ORDER BY ql.checked_at DESC, ql.id DESC
		LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)
	var items []models.IncomingQCItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, apperror.Internal("list incoming qc: " + err.Error())
	}
	return &models.IncomingQCListResponse{Items: items, Pagination: buildPagination(total, filter.Page, filter.Limit)}, nil
}

func (r *repository) ListDefects(ctx context.Context, filter Filter) (*models.DefectListResponse, error) {
	where, args := buildDefectWhere(filter)
	countQuery := `
		SELECT COUNT(*)
		FROM qc_defect_items qdi
		JOIN qc_logs ql ON ql.id = qdi.qc_log_id
		LEFT JOIN work_order_items woi ON woi.id = qdi.wo_item_id` + where
	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, apperror.Internal("count defects: " + err.Error())
	}

	query := `
		SELECT
			qdi.id AS defect_id,
			qdi.qc_log_id,
			TO_CHAR(ql.checked_at::date, 'YYYY-MM-DD') AS report_date,
			qdi.defect_source,
			COALESCE(woi.kanban_number, dni.packing_number, '') AS kanban_pl,
			qdi.uniq_code,
			COALESCE(woi.part_name, qdi.uniq_code) AS product_name,
			COALESCE(qdi.defect_reason_code, '') AS reason_code,
			COALESCE(qdi.defect_reason_text, '') AS reason_text,
			qdi.qty_defect,
			qdi.qty_scrap,
			qdi.is_repairable,
			COALESCE(rt.status, 'none') AS wo_rework_status,
			rt.id AS rework_qc_task_id
		FROM qc_defect_items qdi
		JOIN qc_logs ql ON ql.id = qdi.qc_log_id
		LEFT JOIN work_order_items woi ON woi.id = qdi.wo_item_id
		LEFT JOIN delivery_note_items dni ON dni.id = qdi.dn_item_id
		LEFT JOIN LATERAL (
			SELECT qt.id, qt.status
			FROM qc_tasks qt
			WHERE qt.task_type = 'rework_qc'
			  AND qt.round_results::text ILIKE '%' || '"source_defect_id":' || qdi.id || '%'
			ORDER BY qt.id DESC
			LIMIT 1
		) rt ON TRUE` + where + `
		ORDER BY ql.checked_at DESC, qdi.id DESC
		LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)
	var items []models.DefectItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, apperror.Internal("list defects: " + err.Error())
	}
	return &models.DefectListResponse{
		Items:              items,
		Pagination:         buildPagination(total, filter.Page, filter.Limit),
		ImplementationNote: "Product return QC belum diimplementasikan; data yang muncul saat ini production dan incoming QC.",
	}, nil
}

func (r *repository) CreateManualQCReport(ctx context.Context, req models.CreateManualQCReportRequest, performedBy string) error {
	qcType := strings.ToLower(strings.TrimSpace(req.QCType))
	if qcType == "" {
		return apperror.BadRequest("qc_type is required")
	}
	if qcType == "product_return" {
		return apperror.New(501, apperror.CodeBadRequest, "product return QC belum diimplementasikan")
	}
	if qcType != "production" && qcType != "incoming" {
		return apperror.BadRequest("qc_type must be production or incoming")
	}
	if strings.TrimSpace(req.ReportDate) == "" {
		return apperror.BadRequest("report_date is required")
	}
	checkedAt, err := time.Parse("2006-01-02", req.ReportDate)
	if err != nil {
		return apperror.BadRequest("report_date must be YYYY-MM-DD")
	}
	if strings.TrimSpace(req.UniqCode) == "" {
		return apperror.BadRequest("uniq_code is required")
	}
	if req.NumberOfItemCheck <= 0 {
		return apperror.BadRequest("number_of_item_check must be greater than 0")
	}
	if req.NumberOfDefect < 0 || req.NumberOfScrap < 0 {
		return apperror.BadRequest("number_of_defect and number_of_scrap must be >= 0")
	}
	if req.NumberOfScrap > req.NumberOfDefect {
		return apperror.BadRequest("number_of_scrap cannot exceed number_of_defect")
	}
	status, err := normalizeManualQCStatus(qcType, req.Status)
	if err != nil {
		return err
	}
	performedBy = strings.TrimSpace(performedBy)
	if performedBy == "" {
		performedBy = "system"
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var (
			woID     *int64
			woItemID *int64
			dnItemID *int64
			kanban   string
			uom      string
			woNumber string
		)

		if qcType == "production" {
			ctxRow, err := loadProductionManualContext(tx, req.ReferenceNumber, req.UniqCode)
			if err != nil {
				return err
			}
			woID = &ctxRow.WOID
			woItemID = &ctxRow.WOItemID
			kanban = ctxRow.KanbanNumber
			uom = ctxRow.UOM
			woNumber = ctxRow.WONumber
		} else {
			ctxRow, err := loadIncomingManualContext(tx, req.ReferenceNumber, req.UniqCode)
			if err != nil {
				return err
			}
			dnItemID = &ctxRow.DNItemID
			kanban = ctxRow.PackingNumber
			uom = ctxRow.UOM
		}

		qtyPass := math.Max(req.NumberOfItemCheck-req.NumberOfDefect, 0)
		defectSource := "incoming_material"
		if qcType == "production" {
			defectSource = "process"
		}

		qcLog := map[string]interface{}{
			"uuid":          uuid.New().String(),
			"wo_id":         woID,
			"wo_item_id":    woItemID,
			"dn_item_id":    dnItemID,
			"uniq_code":     strings.TrimSpace(req.UniqCode),
			"qc_round":      1,
			"qty_checked":   req.NumberOfItemCheck,
			"qty_pass":      qtyPass,
			"qty_defect":    req.NumberOfDefect,
			"qty_scrap":     req.NumberOfScrap,
			"status":        status,
			"defect_source": defectSource,
			"checked_by":    performedBy,
			"checked_at":    checkedAt,
			"created_at":    time.Now(),
		}
		if err := tx.Table("qc_logs").Create(&qcLog).Error; err != nil {
			return apperror.Internal("create manual qc log: " + err.Error())
		}
		qcLogID := asInt64(qcLog["id"])
		if qcLogID == 0 {
			var row struct {
				ID int64 `gorm:"column:id"`
			}
			if err := tx.Raw("SELECT currval(pg_get_serial_sequence('qc_logs','id')) AS id").Scan(&row).Error; err != nil {
				return apperror.Internal("create manual qc log: missing inserted id")
			}
			qcLogID = row.ID
		}

		issueText := strings.TrimSpace(req.IssueReasonText)
		if issueText == "" {
			issueText = strings.TrimSpace(req.IssueReasonCode)
		}
		if req.NumberOfDefect > 0 || req.NumberOfScrap > 0 || issueText != "" {
			defectItem := map[string]interface{}{
				"qc_log_id":          qcLogID,
				"wo_id":              woID,
				"wo_item_id":         woItemID,
				"dn_item_id":         dnItemID,
				"uniq_code":          strings.TrimSpace(req.UniqCode),
				"defect_source":      defectSource,
				"defect_reason_code": strings.TrimSpace(req.IssueReasonCode),
				"defect_reason_text": issueText,
				"qty_defect":         req.NumberOfDefect,
				"qty_scrap":          req.NumberOfScrap,
				"is_repairable":      false,
				"process_name":       nil,
				"reported_by":        performedBy,
				"reported_at":        checkedAt,
			}
			if err := tx.Table("qc_defect_items").Create(&defectItem).Error; err != nil {
				return apperror.Internal("create manual qc defect item: " + err.Error())
			}
		}

		if qcType == "production" && req.NumberOfScrap > 0 {
			scrapStock := scrapModels.ScrapStock{
				UUID:          uuid.New(),
				UniqCode:      strings.TrimSpace(req.UniqCode),
				PackingNumber: stringPtrOrNil(strings.TrimSpace(kanban)),
				WONumber:      stringPtrOrNil(strings.TrimSpace(woNumber)),
				ScrapType:     scrapModels.ScrapTypeProcess,
				Quantity:      req.NumberOfScrap,
				UOM:           stringPtrOrNil(strings.TrimSpace(uom)),
				DateReceived:  &checkedAt,
				Validator:     stringPtrOrNil(performedBy),
				Status:        scrapModels.StockStatusActive,
				CreatedBy:     stringPtrOrNil(performedBy),
				UpdatedBy:     stringPtrOrNil(performedBy),
				SourceQCLogID: &qcLogID,
			}
			if err := tx.Create(&scrapStock).Error; err != nil {
				return apperror.Internal("create manual qc scrap stock: " + err.Error())
			}
		}

		return nil
	})
}

func (r *repository) CreateReworkTask(ctx context.Context, defectID int64, performedBy string) error {
	performedBy = strings.TrimSpace(performedBy)
	if performedBy == "" {
		performedBy = "system"
	}
	type defectRow struct {
		ID         int64   `gorm:"column:id"`
		QCLogID    int64   `gorm:"column:qc_log_id"`
		WOID       *int64  `gorm:"column:wo_id"`
		WOItemID   *int64  `gorm:"column:wo_item_id"`
		UniqCode   string  `gorm:"column:uniq_code"`
		ReasonCode string  `gorm:"column:defect_reason_code"`
		ReasonText string  `gorm:"column:defect_reason_text"`
		QtyDefect  float64 `gorm:"column:qty_defect"`
		QtyScrap   float64 `gorm:"column:qty_scrap"`
	}
	var defect defectRow
	if err := r.db.WithContext(ctx).Table("qc_defect_items").Where("id = ?", defectID).Scan(&defect).Error; err != nil {
		return apperror.Internal("load defect: " + err.Error())
	}
	if defect.ID == 0 {
		return apperror.NotFound("defect not found")
	}

	var existing int64
	if err := r.db.WithContext(ctx).Table("qc_tasks").Where("task_type = 'rework_qc' AND round_results::text ILIKE ?", fmt.Sprintf("%%\"source_defect_id\":%d%%", defectID)).Count(&existing).Error; err != nil {
		return apperror.Internal("check rework task: " + err.Error())
	}
	if existing > 0 {
		return apperror.Conflict("rework task already exists for this defect")
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"event":            "rework_qc_created",
		"source_qc_log_id": defect.QCLogID,
		"source_defect_id": defect.ID,
		"qty_to_rework":    math.Max(defect.QtyDefect-defect.QtyScrap, 0),
		"reason_code":      defect.ReasonCode,
		"reason_text":      defect.ReasonText,
		"uniq_code":        defect.UniqCode,
		"performed_by":     performedBy,
		"occurred_at":      time.Now().UTC().Format(time.RFC3339),
	})
	task := qcModels.QCTask{
		TaskType:     "rework_qc",
		Status:       "pending",
		WOID:         defect.WOID,
		WOItemID:     defect.WOItemID,
		Round:        1,
		RoundResults: datatypes.JSON(raw),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&task).Error; err != nil {
		return apperror.Internal("create rework task: " + err.Error())
	}
	return nil
}

func buildPagination(total int64, page, limit int) models.Pagination {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return models.Pagination{Total: total, Page: page, Limit: limit, TotalPages: int(math.Ceil(float64(total) / float64(limit)))}
}

func buildRangeWhere(filter Filter, column string) (string, []interface{}) {
	parts := []string{"1=1"}
	args := make([]interface{}, 0, 4)
	if filter.DateFrom != "" {
		parts = append(parts, fmt.Sprintf("%s >= ?", column))
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != "" {
		parts = append(parts, fmt.Sprintf("%s < (?::date + INTERVAL '1 day')", column))
		args = append(args, filter.DateTo)
	} else if filter.DateFrom == "" && filter.WindowHours > 0 {
		parts = append(parts, fmt.Sprintf("%s >= ?", column))
		args = append(args, time.Now().Add(-time.Duration(filter.WindowHours)*time.Hour))
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func buildProductionWhere(filter Filter) (string, []interface{}) {
	parts := []string{"(ql.defect_source IN ('process','setting_machine') OR ql.wo_id IS NOT NULL)"}
	args := make([]interface{}, 0, 8)
	parts, args = appendDateParts(parts, args, filter, "ql.checked_at")
	if filter.Status != "" {
		if strings.EqualFold(filter.Status, "passed") {
			parts = append(parts, "UPPER(ql.status) IN ('PASSED','APPROVED')")
		} else if strings.EqualFold(filter.Status, "not_passed") {
			parts = append(parts, "UPPER(ql.status) NOT IN ('PASSED','APPROVED')")
		}
	}
	if filter.WONumber != "" {
		parts = append(parts, "wo.wo_number ILIKE ?")
		args = append(args, like(filter.WONumber))
	}
	if filter.UniqCode != "" {
		parts = append(parts, "ql.uniq_code ILIKE ?")
		args = append(args, like(filter.UniqCode))
	}
	if filter.Search != "" {
		parts = append(parts, "(wo.wo_number ILIKE ? OR ql.uniq_code ILIKE ? OR woi.kanban_number ILIKE ?)")
		search := like(filter.Search)
		args = append(args, search, search, search)
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func buildIncomingWhere(filter Filter) (string, []interface{}) {
	parts := []string{"ql.defect_source = 'incoming_material'"}
	args := make([]interface{}, 0, 8)
	parts, args = appendDateParts(parts, args, filter, "ql.checked_at")
	if filter.Status != "" {
		if strings.EqualFold(filter.Status, "passed") {
			parts = append(parts, "UPPER(ql.status) IN ('PASSED','APPROVED')")
		} else if strings.EqualFold(filter.Status, "not_passed") {
			parts = append(parts, "UPPER(ql.status) NOT IN ('PASSED','APPROVED')")
		}
	}
	if filter.SupplierID > 0 {
		parts = append(parts, "dn.supplier_id = ?")
		args = append(args, filter.SupplierID)
	}
	if filter.PONumber != "" {
		parts = append(parts, "dn.po_number ILIKE ?")
		args = append(args, like(filter.PONumber))
	}
	if filter.Search != "" {
		parts = append(parts, "(dn.po_number ILIKE ? OR ql.uniq_code ILIKE ? OR COALESCE(s.supplier_name,'') ILIKE ? OR COALESCE(dn.dn_number,'') ILIKE ?)")
		search := like(filter.Search)
		args = append(args, search, search, search, search)
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func buildDefectWhere(filter Filter) (string, []interface{}) {
	parts := []string{"1=1"}
	args := make([]interface{}, 0, 8)
	parts, args = appendDateParts(parts, args, filter, "ql.checked_at")
	if filter.DefectSource != "" {
		parts = append(parts, "qdi.defect_source = ?")
		args = append(args, filter.DefectSource)
	}
	if filter.ReasonCode != "" {
		parts = append(parts, "qdi.defect_reason_code = ?")
		args = append(args, filter.ReasonCode)
	}
	if filter.UniqCode != "" {
		parts = append(parts, "qdi.uniq_code ILIKE ?")
		args = append(args, like(filter.UniqCode))
	}
	if filter.Search != "" {
		parts = append(parts, "(qdi.uniq_code ILIKE ? OR COALESCE(woi.part_name,'') ILIKE ? OR COALESCE(qdi.defect_reason_text,'') ILIKE ?)")
		search := like(filter.Search)
		args = append(args, search, search, search)
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func appendDateParts(parts []string, args []interface{}, filter Filter, column string) ([]string, []interface{}) {
	if filter.DateFrom != "" {
		parts = append(parts, fmt.Sprintf("%s >= ?", column))
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != "" {
		parts = append(parts, fmt.Sprintf("%s < (?::date + INTERVAL '1 day')", column))
		args = append(args, filter.DateTo)
	}
	return parts, args
}

func like(v string) string {
	return "%" + strings.TrimSpace(v) + "%"
}

type productionManualContext struct {
	WOID         int64  `gorm:"column:wo_id"`
	WOItemID     int64  `gorm:"column:wo_item_id"`
	WONumber     string `gorm:"column:wo_number"`
	KanbanNumber string `gorm:"column:kanban_number"`
	UOM          string `gorm:"column:uom"`
}

type incomingManualContext struct {
	DNItemID      int64  `gorm:"column:dn_item_id"`
	PackingNumber string `gorm:"column:packing_number"`
	UOM           string `gorm:"column:uom"`
}

func loadProductionManualContext(tx *gorm.DB, referenceNumber, uniqCode string) (*productionManualContext, error) {
	var row productionManualContext
	q := tx.Table("work_order_items woi").
		Select("wo.id AS wo_id, woi.id AS wo_item_id, COALESCE(wo.wo_number, '') AS wo_number, COALESCE(woi.kanban_number, '') AS kanban_number, COALESCE(woi.uom, '') AS uom").
		Joins("JOIN work_orders wo ON wo.id = woi.wo_id").
		Where("woi.item_uniq_code = ?", strings.TrimSpace(uniqCode))
	if strings.TrimSpace(referenceNumber) != "" {
		q = q.Where("wo.wo_number = ?", strings.TrimSpace(referenceNumber))
	}
	if err := q.Order("woi.id DESC").Limit(1).Scan(&row).Error; err != nil {
		return nil, apperror.Internal("load production context: " + err.Error())
	}
	if row.WOItemID == 0 {
		return nil, apperror.NotFound("production WO/uniq context not found")
	}
	return &row, nil
}

func loadIncomingManualContext(tx *gorm.DB, referenceNumber, uniqCode string) (*incomingManualContext, error) {
	var row incomingManualContext
	q := tx.Table("delivery_note_items dni").
		Select("dni.id AS dn_item_id, COALESCE(dni.packing_number, '') AS packing_number, COALESCE(dni.uom, '') AS uom").
		Joins("JOIN delivery_notes dn ON dn.id = dni.dn_id").
		Where("dni.item_uniq_code = ?", strings.TrimSpace(uniqCode))
	if ref := strings.TrimSpace(referenceNumber); ref != "" {
		q = q.Where("(dn.dn_number = ? OR dn.po_number = ?)", ref, ref)
	}
	if err := q.Order("dni.id DESC").Limit(1).Scan(&row).Error; err != nil {
		return nil, apperror.Internal("load incoming context: " + err.Error())
	}
	if row.DNItemID == 0 {
		return nil, apperror.NotFound("incoming DN/PO/uniq context not found")
	}
	return &row, nil
}

func normalizeManualQCStatus(qcType, raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	switch qcType {
	case "production":
		switch status {
		case "passed", "pass":
			return "PASSED", nil
		case "failed", "fail", "not_passed", "not passed", "rejected", "reject":
			return "FAILED", nil
		}
	case "incoming":
		switch status {
		case "passed", "pass", "approved", "approve":
			return "APPROVED", nil
		case "failed", "fail", "not_passed", "not passed", "rejected", "reject":
			return "REJECTED", nil
		}
	}
	return "", apperror.BadRequest("invalid status for qc_type")
}

func asInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case uint:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func stringPtrOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
