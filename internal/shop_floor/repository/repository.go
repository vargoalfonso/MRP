package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/shop_floor/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type Filter struct {
	Limit        int
	WindowHours  int
	StaleMinutes int
}

type IRepository interface {
	GetLiveProductionSummary(ctx context.Context, filter Filter) (*models.LiveProductionSummary, error)
	GetDeliveryReadinessSummary(ctx context.Context, filter Filter) (*models.DeliveryReadinessSummary, error)
	GetProductionIssuesSummary(ctx context.Context, filter Filter) (*models.ProductionIssuesSummary, error)
	GetScanEventsSummary(ctx context.Context, filter Filter) (*models.ScanEventsSummary, error)
}

type repository struct {
	db *gorm.DB
}

type scanLogSchema struct {
	eventAtExpr        string
	scanTypeExpr       string
	inputQtyExpr       string
	outputQtyExpr      string
	machineDisplayExpr string
	productionLineExpr string
	machineKeyExpr     string
	joinMachine        string
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) GetLiveProductionSummary(ctx context.Context, filter Filter) (*models.LiveProductionSummary, error) {
	startToday, endToday := dayBounds(time.Now())
	activeSince := time.Now().Add(-time.Duration(filter.StaleMinutes) * time.Minute)
	scanSchema, err := r.getScanLogSchema(ctx)
	if err != nil {
		return nil, apperror.Internal("shop floor live production schema: " + err.Error())
	}

	type totalsRow struct {
		AsOf            *time.Time `gorm:"column:as_of"`
		ThroughputToday float64    `gorm:"column:throughput_today"`
		ActiveMachines  int64      `gorm:"column:active_machines"`
		RunningMachines int64      `gorm:"column:running_machines"`
		IdleMachines    int64      `gorm:"column:idle_machines"`
	}

	var totals totalsRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (%s)
				%s AS machine_number,
				%s AS scan_type,
				%s AS event_at
			FROM production_scan_logs psl
			%s
			ORDER BY %s, %s DESC, psl.id DESC
		)
		SELECT
			(SELECT MAX(%s) FROM production_scan_logs psl %s) AS as_of,
			(SELECT COALESCE(SUM(%s), 0) FROM production_scan_logs psl %s WHERE %s >= ? AND %s < ?) AS throughput_today,
			COUNT(*) FILTER (WHERE latest.event_at >= ?) AS active_machines,
			COUNT(*) FILTER (WHERE latest.event_at >= ? AND UPPER(latest.scan_type) IN ('IN', 'SCAN_IN', 'QC', 'SCAN_QC')) AS running_machines,
			COUNT(*) FILTER (WHERE latest.event_at < ? OR UPPER(latest.scan_type) IN ('OUT', 'SCAN_OUT')) AS idle_machines
		FROM latest
	`, scanSchema.machineKeyExpr, scanSchema.machineDisplayExpr, scanSchema.scanTypeExpr, scanSchema.eventAtExpr, scanSchema.joinMachine, scanSchema.machineKeyExpr, scanSchema.eventAtExpr, scanSchema.eventAtExpr, scanSchema.joinMachine, scanSchema.outputQtyExpr, scanSchema.joinMachine, scanSchema.eventAtExpr, scanSchema.eventAtExpr), startToday, endToday, activeSince, activeSince, activeSince).Scan(&totals).Error
	if err != nil {
		return nil, apperror.Internal("shop floor live production summary: " + err.Error())
	}

	type itemRow struct {
		MachineNumber   string     `gorm:"column:machine_number"`
		ProductionLine  string     `gorm:"column:production_line"`
		WONumber        string     `gorm:"column:wo_number"`
		CurrentUniq     string     `gorm:"column:current_uniq"`
		WOStatus        string     `gorm:"column:wo_status"`
		ProcessName     string     `gorm:"column:process_name"`
		LastScanType    string     `gorm:"column:last_scan_type"`
		LastScanAt      *time.Time `gorm:"column:last_scan_at"`
		TargetQty       float64    `gorm:"column:target_qty"`
		ThroughputToday float64    `gorm:"column:throughput_today"`
		OutputQty       float64    `gorm:"column:output_qty"`
	}

	var rows []itemRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (%s)
				%s AS machine_number,
				%s AS production_line,
				psl.wo_id,
				psl.wo_item_id,
				psl.process_name,
				%s AS scan_type,
				%s AS event_at
			FROM production_scan_logs psl
			%s
			ORDER BY %s, %s DESC, psl.id DESC
		), progress AS (
			SELECT psl.wo_item_id, COALESCE(SUM(%s), 0) AS output_qty
			FROM production_scan_logs psl
			WHERE psl.wo_item_id IS NOT NULL
			GROUP BY psl.wo_item_id
		), throughput_today AS (
			SELECT psl.wo_item_id, COALESCE(SUM(%s), 0) AS throughput_today
			FROM production_scan_logs psl
			WHERE psl.wo_item_id IS NOT NULL AND %s >= ? AND %s < ?
			GROUP BY psl.wo_item_id
		)
		SELECT
			latest.machine_number,
			latest.production_line,
			COALESCE(wo.wo_number, '') AS wo_number,
			COALESCE(woi.item_uniq_code, '') AS current_uniq,
			COALESCE(wo.status, '') AS wo_status,
			COALESCE(latest.process_name, '') AS process_name,
			COALESCE(latest.scan_type, '') AS last_scan_type,
			latest.event_at AS last_scan_at,
			COALESCE(woi.quantity, 0) AS target_qty,
			COALESCE(throughput_today.throughput_today, 0) AS throughput_today,
			COALESCE(progress.output_qty, 0) AS output_qty
		FROM latest
		LEFT JOIN work_orders wo ON wo.id = latest.wo_id
		LEFT JOIN work_order_items woi ON woi.id = latest.wo_item_id
		LEFT JOIN progress ON progress.wo_item_id = latest.wo_item_id
		LEFT JOIN throughput_today ON throughput_today.wo_item_id = latest.wo_item_id
		ORDER BY latest.event_at DESC, latest.machine_number ASC
		LIMIT ?
	`, scanSchema.machineKeyExpr, scanSchema.machineDisplayExpr, scanSchema.productionLineExpr, scanSchema.scanTypeExpr, scanSchema.eventAtExpr, scanSchema.joinMachine, scanSchema.machineKeyExpr, scanSchema.eventAtExpr, scanSchema.outputQtyExpr, scanSchema.outputQtyExpr, scanSchema.eventAtExpr, scanSchema.eventAtExpr), startToday, endToday, filter.Limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("shop floor live production items: " + err.Error())
	}

	items := make([]models.LiveProduction, 0, len(rows))
	for _, row := range rows {
		progressPercent := 0.0
		if row.TargetQty > 0 {
			progressPercent = math.Min((row.OutputQty/row.TargetQty)*100, 100)
		}

		runtimeStatus := "idle"
		if row.LastScanAt != nil && row.LastScanAt.After(activeSince) && isRunningScanType(row.LastScanType) {
			runtimeStatus = "running"
		}

		items = append(items, models.LiveProduction{
			Machine: models.LiveProductionMachine{
				Number:         row.MachineNumber,
				ProductionLine: row.ProductionLine,
				RuntimeStatus:  runtimeStatus,
			},
			Production: models.LiveProductionCurrent{
				WONumber:     row.WONumber,
				CurrentUniq:  row.CurrentUniq,
				WOStatus:     row.WOStatus,
				ProcessName:  row.ProcessName,
				LastScanType: row.LastScanType,
				LastScanAt:   row.LastScanAt,
			},
			Progress: models.LiveProductionProgress{
				TargetQty:       row.TargetQty,
				ThroughputToday: row.ThroughputToday,
				OutputQty:       row.OutputQty,
				ProgressPercent: round2(progressPercent),
			},
		})
	}

	return &models.LiveProductionSummary{
		AsOf:               totals.AsOf,
		StaleWindowMinutes: filter.StaleMinutes,
		ThroughputToday:    totals.ThroughputToday,
		ActiveMachines:     totals.ActiveMachines,
		RunningMachines:    totals.RunningMachines,
		IdleMachines:       totals.IdleMachines,
		Items:              items,
	}, nil
}

func (r *repository) GetDeliveryReadinessSummary(ctx context.Context, filter Filter) (*models.DeliveryReadinessSummary, error) {
	deliveryCols, err := r.tableColumns(ctx, "delivery_schedules_customer")
	if err != nil {
		return nil, apperror.Internal("shop floor delivery readiness schedule schema: " + err.Error())
	}

	dueTimeExpr := "NULL::text"
	dueAtExpr := "NULL::timestamptz"
	if deliveryCols["departure_at"] {
		dueTimeExpr = "CASE WHEN sc.departure_at IS NOT NULL THEN TO_CHAR(sc.departure_at, 'HH24:MI') END"
		dueAtExpr = "sc.departure_at"
	}

	type totalRow struct {
		TotalScheduled    int64   `gorm:"column:total_scheduled"`
		ReadyItems        int64   `gorm:"column:ready_items"`
		AtRiskItems       int64   `gorm:"column:at_risk_items"`
		CriticalItems     int64   `gorm:"column:critical_items"`
		TotalRequiredQty  float64 `gorm:"column:total_required_qty"`
		TotalAvailableQty float64 `gorm:"column:total_available_qty"`
		TotalShortfallQty float64 `gorm:"column:total_shortfall_qty"`
	}

	var total totalRow
	err = r.db.WithContext(ctx).Raw(`
		WITH fg AS (
			SELECT uniq_code, COALESCE(SUM(stock_qty), 0) AS fg_qty
			FROM finished_goods
			WHERE deleted_at IS NULL
			GROUP BY uniq_code
		), wip AS (
			SELECT uniq, COALESCE(SUM(stock), 0) AS wip_qty
			FROM wip_items
			GROUP BY uniq
		), base AS (
			SELECT
				sci.id,
				COALESCE(sci.total_delivery_qty, 0) AS required_qty,
				COALESCE(fg.fg_qty, 0) + COALESCE(wip.wip_qty, 0) AS available_qty
			FROM delivery_schedule_items_customer sci
			JOIN delivery_schedules_customer sc ON sc.id = sci.schedule_id
			LEFT JOIN fg ON fg.uniq_code = sci.item_uniq_code
			LEFT JOIN wip ON wip.uniq = sci.item_uniq_code
			WHERE sc.deleted_at IS NULL AND sc.status <> 'cancelled'
		)
		SELECT
			COUNT(*) AS total_scheduled,
			COUNT(*) FILTER (WHERE available_qty >= required_qty) AS ready_items,
			COUNT(*) FILTER (WHERE available_qty < required_qty AND available_qty > 0) AS at_risk_items,
			COUNT(*) FILTER (WHERE available_qty <= 0) AS critical_items,
			COALESCE(SUM(required_qty), 0) AS total_required_qty,
			COALESCE(SUM(available_qty), 0) AS total_available_qty,
			COALESCE(SUM(GREATEST(required_qty - available_qty, 0)), 0) AS total_shortfall_qty
		FROM base
	`).Scan(&total).Error
	if err != nil {
		return nil, apperror.Internal("shop floor delivery readiness summary: " + err.Error())
	}

	type itemRow struct {
		ScheduleNumber   string     `gorm:"column:schedule_number"`
		ScheduleDate     time.Time  `gorm:"column:schedule_date"`
		DueTime          *string    `gorm:"column:due_time"`
		DueAt            *time.Time `gorm:"column:due_at"`
		CustomerName     string     `gorm:"column:customer_name"`
		ItemUniqCode     string     `gorm:"column:item_uniq_code"`
		PartNumber       string     `gorm:"column:part_number"`
		PartName         string     `gorm:"column:part_name"`
		RequiredQty      float64    `gorm:"column:required_qty"`
		FGQty            float64    `gorm:"column:fg_qty"`
		WIPQty           float64    `gorm:"column:wip_qty"`
		FGReadinessState string     `gorm:"column:fg_readiness_state"`
	}

	var rows []itemRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH fg AS (
			SELECT uniq_code, COALESCE(SUM(stock_qty), 0) AS fg_qty
			FROM finished_goods
			WHERE deleted_at IS NULL
			GROUP BY uniq_code
		), wip AS (
			SELECT uniq, COALESCE(SUM(stock), 0) AS wip_qty
			FROM wip_items
			GROUP BY uniq
		)
		SELECT
			sc.schedule_number,
			sc.schedule_date,
			%s AS due_time,
			%s AS due_at,
			COALESCE(sc.customer_name_snapshot, '') AS customer_name,
			sci.item_uniq_code,
			sci.part_number,
			sci.part_name,
			COALESCE(sci.total_delivery_qty, 0) AS required_qty,
			COALESCE(fg.fg_qty, 0) AS fg_qty,
			COALESCE(wip.wip_qty, 0) AS wip_qty,
			COALESCE(sci.fg_readiness_status, 'unknown') AS fg_readiness_state
		FROM delivery_schedule_items_customer sci
		JOIN delivery_schedules_customer sc ON sc.id = sci.schedule_id
		LEFT JOIN fg ON fg.uniq_code = sci.item_uniq_code
		LEFT JOIN wip ON wip.uniq = sci.item_uniq_code
		WHERE sc.deleted_at IS NULL AND sc.status <> 'cancelled'
		ORDER BY sc.schedule_date ASC, sc.schedule_number ASC, sci.line_no ASC
		LIMIT ?
	`, dueTimeExpr, dueAtExpr), filter.Limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("shop floor delivery readiness items: " + err.Error())
	}

	items := make([]models.DeliveryReadinessItem, 0, len(rows))
	for _, row := range rows {
		availableQty := row.FGQty + row.WIPQty
		shortfallQty := math.Max(row.RequiredQty-availableQty, 0)
		coveragePercent := 0.0
		if row.RequiredQty > 0 {
			coveragePercent = math.Min((availableQty/row.RequiredQty)*100, 100)
		}

		readinessStatus := "critical"
		switch {
		case availableQty >= row.RequiredQty:
			readinessStatus = "ready"
		case availableQty > 0:
			readinessStatus = "at_risk"
		}

		dueDate := row.ScheduleDate.Format("2006-01-02")
		var hoursUntilDue *int64
		if row.DueAt != nil {
			hours := int64(math.Ceil(row.DueAt.Sub(time.Now()).Hours()))
			hoursUntilDue = &hours
		}

		items = append(items, models.DeliveryReadinessItem{
			Identity: models.DeliveryReadinessIdentity{
				ScheduleNumber: row.ScheduleNumber,
				CustomerName:   row.CustomerName,
				ItemUniqCode:   row.ItemUniqCode,
				PartNumber:     row.PartNumber,
				PartName:       row.PartName,
			},
			Delivery: models.DeliveryReadinessDelivery{
				ScheduleDate:  row.ScheduleDate,
				DueDate:       dueDate,
				DueTime:       row.DueTime,
				HoursUntilDue: hoursUntilDue,
				RequiredQty:   row.RequiredQty,
			},
			Inventory: models.DeliveryReadinessInventory{
				FGQty:            row.FGQty,
				WIPQty:           row.WIPQty,
				AvailableQty:     round2(availableQty),
				FGReadinessState: row.FGReadinessState,
			},
			Readiness: models.DeliveryReadinessAnalysis{
				ShortfallQty:    round2(shortfallQty),
				CoveragePercent: round2(coveragePercent),
				ReadinessStatus: readinessStatus,
			},
		})
	}

	return &models.DeliveryReadinessSummary{
		AsOf:              time.Now(),
		TotalScheduled:    total.TotalScheduled,
		ReadyItems:        total.ReadyItems,
		AtRiskItems:       total.AtRiskItems,
		CriticalItems:     total.CriticalItems,
		TotalRequiredQty:  round2(total.TotalRequiredQty),
		TotalAvailableQty: round2(total.TotalAvailableQty),
		TotalShortfallQty: round2(total.TotalShortfallQty),
		Items:             items,
	}, nil
}

func (r *repository) GetProductionIssuesSummary(ctx context.Context, filter Filter) (*models.ProductionIssuesSummary, error) {
	res := &models.ProductionIssuesSummary{
		AsOf:            time.Now(),
		SourceAvailable: false,
		WindowHours:     filter.WindowHours,
		Items:           []models.ProductionIssue{},
	}

	exists, err := r.tableExists(ctx, "production_issues")
	if err != nil {
		return nil, apperror.Internal("shop floor production issues schema: " + err.Error())
	}
	if !exists {
		return res, nil
	}

	cols, err := r.tableColumns(ctx, "production_issues")
	if err != nil {
		return nil, apperror.Internal("shop floor production issues columns: " + err.Error())
	}
	issueTypeExists, err := r.tableExists(ctx, "production_issue_types")
	if err != nil {
		return nil, apperror.Internal("shop floor issue types schema: " + err.Error())
	}
	issueTypeCols := map[string]bool{}
	if issueTypeExists {
		issueTypeCols, err = r.tableColumns(ctx, "production_issue_types")
		if err != nil {
			return nil, apperror.Internal("shop floor issue types columns: " + err.Error())
		}
	}

	createdExpr := firstExistingExpr(cols, map[string]string{
		"created_at":  "pi.created_at",
		"reported_at": "pi.reported_at",
		"issue_time":  "pi.issue_time",
		"occurred_at": "pi.occurred_at",
	})
	if createdExpr == "" {
		createdExpr = "NOW()"
	}
	updatedExpr := firstExistingExpr(cols, map[string]string{
		"updated_at":  "pi.updated_at",
		"resolved_at": "pi.resolved_at",
	})
	if updatedExpr == "" {
		updatedExpr = createdExpr
	}
	titleExpr := firstExistingExpr(cols, map[string]string{
		"title":       "pi.title",
		"issue_title": "pi.issue_title",
		"name":        "pi.name",
		"description": "pi.description",
	})
	if titleExpr == "" {
		titleExpr = "''"
	}
	statusExpr := firstExistingExpr(cols, map[string]string{
		"status":       "pi.status",
		"issue_status": "pi.issue_status",
	})
	if statusExpr == "" {
		statusExpr = "'open'"
	}
	severityExpr := firstExistingExpr(cols, map[string]string{
		"severity": "pi.severity",
	})
	if severityExpr == "" {
		severityExpr = "''"
	}
	priorityExpr := firstExistingExpr(cols, map[string]string{
		"priority": "pi.priority",
	})
	if priorityExpr == "" {
		priorityExpr = severityExpr
	}
	reportedByExpr := firstExistingExpr(cols, map[string]string{
		"reported_by": "pi.reported_by",
		"created_by":  "pi.created_by",
		"scanned_by":  "pi.scanned_by",
	})
	if reportedByExpr == "" {
		reportedByExpr = "''"
	}
	machineExpr := firstExistingExpr(cols, map[string]string{
		"machine":         "pi.machine",
		"machine_number":  "pi.machine_number",
		"production_line": "pi.production_line",
	})
	joinMachine := ""
	if machineExpr == "" && cols["machine_id"] {
		joinMachine = " LEFT JOIN master_machines mm ON mm.id = pi.machine_id "
		machineExpr = "COALESCE(mm.machine_number, mm.machine_name, '')"
	}
	if machineExpr == "" {
		machineExpr = "''"
	}
	issueTypeExpr := "''"
	joinIssueType := ""
	if cols["issue_type_id"] && issueTypeExists {
		joinIssueType = " LEFT JOIN production_issue_types pit ON pit.id = pi.issue_type_id "
		issueTypeExpr = firstExistingExpr(issueTypeCols, map[string]string{
			"name":        "pit.name",
			"type_name":   "pit.type_name",
			"title":       "pit.title",
			"description": "pit.description",
		})
		if issueTypeExpr == "" {
			issueTypeExpr = "''"
		}
	}

	since := time.Now().Add(-time.Duration(filter.WindowHours) * time.Hour)
	openExpr := fmt.Sprintf("CASE WHEN LOWER(COALESCE(%s::text, 'open')) IN ('resolved', 'closed', 'done', 'cancelled') THEN 0 ELSE 1 END", statusExpr)
	criticalExpr := fmt.Sprintf("CASE WHEN LOWER(COALESCE(%s::text, '')) = 'critical' THEN 1 ELSE 0 END", severityExpr)
	highExpr := fmt.Sprintf("CASE WHEN LOWER(COALESCE(%s::text, '')) IN ('high', 'critical') THEN 1 ELSE 0 END", priorityExpr)

	type totalRow struct {
		TotalIssues    int64 `gorm:"column:total_issues"`
		OpenIssues     int64 `gorm:"column:open_issues"`
		CriticalIssues int64 `gorm:"column:critical_issues"`
		HighPriority   int64 `gorm:"column:high_priority"`
	}
	var total totalRow
	countSQL := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_issues,
			COALESCE(SUM(%s), 0) AS open_issues,
			COALESCE(SUM(%s), 0) AS critical_issues,
			COALESCE(SUM(%s), 0) AS high_priority
		FROM production_issues pi
		%s
		%s
		WHERE %s >= ?
	`, openExpr, criticalExpr, highExpr, joinMachine, joinIssueType, createdExpr)
	if err := r.db.WithContext(ctx).Raw(countSQL, since).Scan(&total).Error; err != nil {
		return nil, apperror.Internal("shop floor production issues summary: " + err.Error())
	}

	type itemRow struct {
		ID         int64      `gorm:"column:id"`
		Title      string     `gorm:"column:title"`
		IssueType  string     `gorm:"column:issue_type"`
		Machine    string     `gorm:"column:machine"`
		Status     string     `gorm:"column:status"`
		Severity   string     `gorm:"column:severity"`
		Priority   string     `gorm:"column:priority"`
		ReportedBy string     `gorm:"column:reported_by"`
		OccurredAt *time.Time `gorm:"column:occurred_at"`
		UpdatedAt  *time.Time `gorm:"column:updated_at"`
	}

	var rows []itemRow
	itemSQL := fmt.Sprintf(`
		SELECT
			pi.id,
			COALESCE(%s::text, '') AS title,
			COALESCE(%s::text, '') AS issue_type,
			COALESCE(%s::text, '') AS machine,
			COALESCE(%s::text, '') AS status,
			COALESCE(%s::text, '') AS severity,
			COALESCE(%s::text, '') AS priority,
			COALESCE(%s::text, '') AS reported_by,
			%s AS occurred_at,
			%s AS updated_at
		FROM production_issues pi
		%s
		%s
		WHERE %s >= ?
		ORDER BY %s DESC, pi.id DESC
		LIMIT ?
	`, titleExpr, issueTypeExpr, machineExpr, statusExpr, severityExpr, priorityExpr, reportedByExpr, createdExpr, updatedExpr, joinMachine, joinIssueType, createdExpr, createdExpr)
	if err := r.db.WithContext(ctx).Raw(itemSQL, since, filter.Limit).Scan(&rows).Error; err != nil {
		return nil, apperror.Internal("shop floor production issues items: " + err.Error())
	}

	items := make([]models.ProductionIssue, 0, len(rows))
	for _, row := range rows {
		items = append(items, models.ProductionIssue(row))
	}

	res.SourceAvailable = true
	res.TotalIssues = total.TotalIssues
	res.OpenIssues = total.OpenIssues
	res.CriticalIssues = total.CriticalIssues
	res.HighPriority = total.HighPriority
	res.Items = items
	return res, nil
}

func (r *repository) GetScanEventsSummary(ctx context.Context, filter Filter) (*models.ScanEventsSummary, error) {
	since := time.Now().Add(-time.Duration(filter.WindowHours) * time.Hour)
	scanSchema, err := r.getScanLogSchema(ctx)
	if err != nil {
		return nil, apperror.Internal("shop floor scan events schema: " + err.Error())
	}

	type totalRow struct {
		AsOf         *time.Time `gorm:"column:as_of"`
		TotalEvents  int64      `gorm:"column:total_events"`
		ScanInCount  int64      `gorm:"column:scan_in_count"`
		ScanOutCount int64      `gorm:"column:scan_out_count"`
		QCCount      int64      `gorm:"column:qc_count"`
	}
	var total totalRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT
			MAX(%s) AS as_of,
			COUNT(*) AS total_events,
			COUNT(*) FILTER (WHERE UPPER(%s) IN ('IN', 'SCAN_IN')) AS scan_in_count,
			COUNT(*) FILTER (WHERE UPPER(%s) IN ('OUT', 'SCAN_OUT')) AS scan_out_count,
			COUNT(*) FILTER (WHERE UPPER(%s) IN ('QC', 'SCAN_QC')) AS qc_count
		FROM production_scan_logs psl
		%s
		WHERE %s >= ?
	`, scanSchema.eventAtExpr, scanSchema.scanTypeExpr, scanSchema.scanTypeExpr, scanSchema.scanTypeExpr, scanSchema.joinMachine, scanSchema.eventAtExpr), since).Scan(&total).Error
	if err != nil {
		return nil, apperror.Internal("shop floor scan events summary: " + err.Error())
	}

	type itemRow struct {
		ScannedAt      time.Time `gorm:"column:scanned_at"`
		ScanType       string    `gorm:"column:scan_type"`
		MachineNumber  string    `gorm:"column:machine_number"`
		ProductionLine string    `gorm:"column:production_line"`
		WONumber       string    `gorm:"column:wo_number"`
		CurrentUniq    string    `gorm:"column:current_uniq"`
		ProcessName    string    `gorm:"column:process_name"`
		Qty            float64   `gorm:"column:qty"`
		ScannedBy      string    `gorm:"column:scanned_by"`
	}

	var rows []itemRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT
			%s AS scanned_at,
			%s AS scan_type,
			%s AS machine_number,
			%s AS production_line,
			COALESCE(wo.wo_number, '') AS wo_number,
			COALESCE(woi.item_uniq_code, '') AS current_uniq,
			COALESCE(psl.process_name, '') AS process_name,
			CASE
				WHEN UPPER(%s) IN ('OUT', 'SCAN_OUT') THEN %s
				WHEN UPPER(%s) IN ('QC', 'SCAN_QC') THEN GREATEST(%s, %s)
				ELSE %s
			END AS qty,
			COALESCE(psl.scanned_by, '') AS scanned_by
		FROM production_scan_logs psl
		%s
		LEFT JOIN work_orders wo ON wo.id = psl.wo_id
		LEFT JOIN work_order_items woi ON woi.id = psl.wo_item_id
		WHERE %s >= ?
		ORDER BY %s DESC, psl.id DESC
		LIMIT ?
	`, scanSchema.eventAtExpr, scanSchema.scanTypeExpr, scanSchema.machineDisplayExpr, scanSchema.productionLineExpr, scanSchema.scanTypeExpr, scanSchema.outputQtyExpr, scanSchema.scanTypeExpr, scanSchema.outputQtyExpr, scanSchema.inputQtyExpr, scanSchema.inputQtyExpr, scanSchema.joinMachine, scanSchema.eventAtExpr, scanSchema.eventAtExpr), since, filter.Limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("shop floor scan events items: " + err.Error())
	}

	items := make([]models.ScanEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, models.ScanEvent(row))
	}

	return &models.ScanEventsSummary{
		AsOf:         total.AsOf,
		WindowHours:  filter.WindowHours,
		TotalEvents:  total.TotalEvents,
		ScanInCount:  total.ScanInCount,
		ScanOutCount: total.ScanOutCount,
		QCCount:      total.QCCount,
		Items:        items,
	}, nil
}

func (r *repository) tableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?
		)
	`, tableName).Scan(&exists).Error
	return exists, err
}

func (r *repository) tableColumns(ctx context.Context, tableName string) (map[string]bool, error) {
	type row struct {
		ColumnName string `gorm:"column:column_name"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?
	`, tableName).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	columns := make(map[string]bool, len(rows))
	for _, row := range rows {
		columns[row.ColumnName] = true
	}
	return columns, nil
}

func dayBounds(now time.Time) (time.Time, time.Time) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return start, start.Add(24 * time.Hour)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func isRunningScanType(scanType string) bool {
	switch strings.ToUpper(scanType) {
	case "IN", "SCAN_IN", "QC", "SCAN_QC":
		return true
	default:
		return false
	}
}

func firstExistingExpr(cols map[string]bool, candidates map[string]string) string {
	for key, expr := range candidates {
		if cols[key] {
			return expr
		}
	}
	return ""
}

func (r *repository) getScanLogSchema(ctx context.Context) (*scanLogSchema, error) {
	cols, err := r.tableColumns(ctx, "production_scan_logs")
	if err != nil {
		return nil, err
	}

	machineCols := map[string]bool{}
	machineTableExists, err := r.tableExists(ctx, "master_machines")
	if err != nil {
		return nil, err
	}
	joinMachine := ""
	if machineTableExists && cols["machine_id"] {
		machineCols, err = r.tableColumns(ctx, "master_machines")
		if err != nil {
			return nil, err
		}
		joinMachine = "LEFT JOIN master_machines mm ON CAST(mm.id AS text) = CAST(psl.machine_id AS text)"
	}

	eventAtExpr := firstExistingExpr(cols, map[string]string{
		"scanned_at": "psl.scanned_at",
		"created_at": "psl.created_at",
	})
	if eventAtExpr == "" {
		eventAtExpr = "NOW()"
	}

	scanTypeExpr := firstExistingExpr(cols, map[string]string{
		"scan_type": "COALESCE(psl.scan_type, '')",
	})
	if scanTypeExpr == "" {
		scanTypeExpr = "''"
	}

	inputQtyExpr := firstExistingExpr(cols, map[string]string{
		"qty_input": "COALESCE(psl.qty_input, 0)",
		"quantity":  "COALESCE(psl.quantity, 0)",
	})
	if inputQtyExpr == "" {
		inputQtyExpr = "0"
	}

	outputQtyExpr := firstExistingExpr(cols, map[string]string{
		"qty_output":    "COALESCE(psl.qty_output, 0)",
		"good_quantity": "COALESCE(psl.good_quantity, 0)",
		"quantity":      "COALESCE(psl.quantity, 0)",
	})
	if outputQtyExpr == "" {
		outputQtyExpr = inputQtyExpr
	}

	machineDisplayExpr := firstExistingExpr(cols, map[string]string{
		"machine_number": "COALESCE(NULLIF(psl.machine_number, ''), 'UNASSIGNED')",
	})
	if machineDisplayExpr == "" {
		if joinMachine != "" {
			machineDisplayExpr = firstExistingExpr(machineCols, map[string]string{
				"machine_number": "COALESCE(NULLIF(mm.machine_number, ''), NULLIF(mm.machine_name, ''), CAST(psl.machine_id AS text), 'UNASSIGNED')",
				"machine_name":   "COALESCE(NULLIF(mm.machine_name, ''), CAST(psl.machine_id AS text), 'UNASSIGNED')",
			})
			if machineDisplayExpr == "" {
				machineDisplayExpr = "COALESCE(CAST(psl.machine_id AS text), 'UNASSIGNED')"
			}
		} else if cols["machine_id"] {
			machineDisplayExpr = "COALESCE(CAST(psl.machine_id AS text), 'UNASSIGNED')"
		} else {
			machineDisplayExpr = "'UNASSIGNED'"
		}
	}

	productionLineExpr := firstExistingExpr(cols, map[string]string{
		"production_line": "COALESCE(NULLIF(psl.production_line, ''), '-')",
	})
	if productionLineExpr == "" && joinMachine != "" {
		productionLineExpr = firstExistingExpr(machineCols, map[string]string{
			"production_line": "COALESCE(NULLIF(mm.production_line, ''), '-')",
		})
	}
	if productionLineExpr == "" {
		productionLineExpr = "'-'"
	}

	machineKeyExpr := fmt.Sprintf("COALESCE(NULLIF(%s::text, ''), CONCAT('line:', COALESCE(NULLIF(%s::text, ''), 'UNASSIGNED'))) ", machineDisplayExpr, productionLineExpr)

	return &scanLogSchema{
		eventAtExpr:        eventAtExpr,
		scanTypeExpr:       scanTypeExpr,
		inputQtyExpr:       inputQtyExpr,
		outputQtyExpr:      outputQtyExpr,
		machineDisplayExpr: machineDisplayExpr,
		productionLineExpr: productionLineExpr,
		machineKeyExpr:     machineKeyExpr,
		joinMachine:        joinMachine,
	}, nil
}
