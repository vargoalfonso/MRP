package repository

import (
	"context"
	"fmt"
	"math"

	"github.com/ganasa18/go-template/internal/main_dashboard/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	GetDeliveryKPI(ctx context.Context, current, previous models.DateRange) (*models.DeliveryKPI, error)
	GetCurrentProductionKPI(ctx context.Context, current, previous models.DateRange) (*models.CurrentProductionKPI, error)
	GetTotalProductionKPI(ctx context.Context, current, previous models.DateRange) (*models.TotalProductionKPI, error)
	GetPORawMaterialKPI(ctx context.Context, current, previous models.DateRange) (*models.PORawMaterialKPI, error)
	GetDeliveryPerformance(ctx context.Context) (*models.DeliveryPerformance, error)
	GetProductionPerformance(ctx context.Context, current models.DateRange) (*models.ProductionPerformance, error)
	GetTopCustomers(ctx context.Context, current models.DateRange, limit int) ([]models.TopCustomer, error)
	GetUniqProgress(ctx context.Context, limit int) ([]models.UniqProgress, error)
	GetPOSummary(ctx context.Context, current models.DateRange) (*models.POSummary, error)
	GetCategoryDistribution(ctx context.Context) ([]models.CategoryDistribution, error)
	GetTopSuppliers(ctx context.Context, limit int) ([]models.TopSupplier, error)
	ListTables(ctx context.Context, schema string, limit int) (*models.ListTablesResponse, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) GetDeliveryKPI(ctx context.Context, current, previous models.DateRange) (*models.DeliveryKPI, error) {
	type row struct {
		TotalNow  int64 `gorm:"column:total_now"`
		OnTimeNow int64 `gorm:"column:on_time_now"`
		TotalPrev int64 `gorm:"column:total_prev"`
	}
	var out row

	err := r.db.WithContext(ctx).Raw(`
		WITH cur AS (
			SELECT
				COUNT(*) AS total_now,
				COALESCE(SUM(CASE WHEN EXISTS (
					SELECT 1
					FROM delivery_note_items_customer dni
					WHERE dni.dn_id = dn.id
					  AND dni.shipped_at IS NOT NULL
					  AND dni.shipped_at::date <= dn.delivery_date
				) THEN 1 ELSE 0 END), 0) AS on_time_now
			FROM delivery_notes_customer dn
			WHERE dn.delivery_date BETWEEN ?::date AND ?::date
			  AND dn.status <> 'cancelled'
		), prev AS (
			SELECT COUNT(*) AS total_prev
			FROM delivery_notes_customer dn
			WHERE dn.delivery_date BETWEEN ?::date AND ?::date
			  AND dn.status <> 'cancelled'
		)
		SELECT cur.total_now, cur.on_time_now, prev.total_prev
		FROM cur, prev
	`, current.StartDate, current.EndDate, previous.StartDate, previous.EndDate).Scan(&out).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard delivery kpi: " + err.Error())
	}

	delta := percentDelta(float64(out.TotalNow), float64(out.TotalPrev))
	return &models.DeliveryKPI{
		Value:        out.TotalNow,
		Subtitle:     fmt.Sprintf("%d on time", out.OnTimeNow),
		DeltaPercent: round2(delta),
		DeltaLabel:   "vs last period",
		Trend:        trendByDelta(delta),
	}, nil
}

func (r *repository) GetCurrentProductionKPI(ctx context.Context, current, previous models.DateRange) (*models.CurrentProductionKPI, error) {
	dateExpr, err := r.productionScanDateExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard current production kpi schema: " + err.Error())
	}

	type row struct {
		ActiveWO    int64 `gorm:"column:active_wo"`
		ScannedNow  int64 `gorm:"column:scanned_now"`
		ScannedPrev int64 `gorm:"column:scanned_prev"`
	}
	var out row

	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH curr_scan AS (
			SELECT COUNT(DISTINCT psl.wo_id) AS scanned_now
			FROM production_scan_logs psl
			WHERE %s BETWEEN ?::date AND ?::date
		), prev_scan AS (
			SELECT COUNT(DISTINCT psl.wo_id) AS scanned_prev
			FROM production_scan_logs psl
			WHERE %s BETWEEN ?::date AND ?::date
		)
		SELECT
			(SELECT COUNT(*) FROM work_orders wo WHERE wo.status = 'In Progress') AS active_wo,
			curr_scan.scanned_now,
			prev_scan.scanned_prev
		FROM curr_scan, prev_scan
	`, dateExpr, dateExpr), current.StartDate, current.EndDate, previous.StartDate, previous.EndDate).Scan(&out).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard current production kpi: " + err.Error())
	}

	delta := percentDelta(float64(out.ScannedNow), float64(out.ScannedPrev))
	return &models.CurrentProductionKPI{
		Value:        out.ActiveWO,
		Subtitle:     fmt.Sprintf("%d active WOs", out.ScannedNow),
		DeltaPercent: round2(delta),
		DeltaLabel:   "vs last period",
		Trend:        trendByDelta(delta),
	}, nil
}

func (r *repository) GetTotalProductionKPI(ctx context.Context, current, previous models.DateRange) (*models.TotalProductionKPI, error) {
	outputExpr, err := r.productionScanOutputExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard total production kpi schema: " + err.Error())
	}
	dateExpr, err := r.productionScanDateExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard total production kpi schema: " + err.Error())
	}

	type row struct {
		ProducedNow    float64 `gorm:"column:produced_now"`
		ProducedPrev   float64 `gorm:"column:produced_prev"`
		CompletedToday int64   `gorm:"column:completed_today"`
	}
	var out row

	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH curr AS (
			SELECT COALESCE(SUM(%s), 0) AS produced_now
			FROM production_scan_logs psl
			WHERE psl.scan_type = 'scan_out'
			  AND %s BETWEEN ?::date AND ?::date
		), prev AS (
			SELECT COALESCE(SUM(%s), 0) AS produced_prev
			FROM production_scan_logs psl
			WHERE psl.scan_type = 'scan_out'
			  AND %s BETWEEN ?::date AND ?::date
		)
		SELECT
			curr.produced_now,
			prev.produced_prev,
			(SELECT COUNT(*) FROM work_orders wo WHERE wo.close_date = CURRENT_DATE) AS completed_today
		FROM curr, prev
	`, outputExpr, dateExpr, outputExpr, dateExpr), current.StartDate, current.EndDate, previous.StartDate, previous.EndDate).Scan(&out).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard total production kpi: " + err.Error())
	}

	delta := percentDelta(out.ProducedNow, out.ProducedPrev)
	return &models.TotalProductionKPI{
		Value:        round2(out.ProducedNow),
		Subtitle:     fmt.Sprintf("%d completed today", out.CompletedToday),
		DeltaPercent: round2(delta),
		DeltaLabel:   "vs last period",
		Trend:        trendByDelta(delta),
	}, nil
}

func (r *repository) GetPORawMaterialKPI(ctx context.Context, current, previous models.DateRange) (*models.PORawMaterialKPI, error) {
	type row struct {
		TotalNow  int64 `gorm:"column:total_now"`
		TotalPrev int64 `gorm:"column:total_prev"`
		BuyRecs   int64 `gorm:"column:buy_recs"`
	}
	var out row

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM purchase_orders po
			 WHERE po.created_at::date BETWEEN ?::date AND ?::date
			   AND po.status <> 'cancelled') AS total_now,
			(SELECT COUNT(*) FROM purchase_orders po
			 WHERE po.created_at::date BETWEEN ?::date AND ?::date
			   AND po.status <> 'cancelled') AS total_prev,
			(SELECT COUNT(*) FROM raw_materials rm
			 WHERE rm.deleted_at IS NULL AND rm.buy_not_buy = 'buy') AS buy_recs
	`, current.StartDate, current.EndDate, previous.StartDate, previous.EndDate).Scan(&out).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard po raw material kpi: " + err.Error())
	}

	delta := out.TotalNow - out.TotalPrev
	return &models.PORawMaterialKPI{
		Value:      out.TotalNow,
		Subtitle:   fmt.Sprintf("%d buy recommendations", out.BuyRecs),
		DeltaValue: delta,
		DeltaLabel: "vs last period",
		Trend:      trendByDelta(float64(delta)),
	}, nil
}

func (r *repository) GetDeliveryPerformance(ctx context.Context) (*models.DeliveryPerformance, error) {
	type summaryRow struct {
		Total  int64 `gorm:"column:total"`
		OnTime int64 `gorm:"column:on_time"`
	}
	var summary summaryRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN EXISTS (
				SELECT 1
				FROM delivery_note_items_customer dni
				WHERE dni.dn_id = dn.id
				  AND dni.shipped_at IS NOT NULL
				  AND dni.shipped_at::date <= dn.delivery_date
			) THEN 1 ELSE 0 END), 0) AS on_time
		FROM delivery_notes_customer dn
		WHERE dn.delivery_date >= (date_trunc('month', CURRENT_DATE) - INTERVAL '5 months')::date
		  AND dn.status <> 'cancelled'
	`).Scan(&summary).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard delivery performance summary: " + err.Error())
	}

	type trendRow struct {
		Label  string `gorm:"column:label"`
		Actual int64  `gorm:"column:actual"`
		Target int64  `gorm:"column:target"`
	}
	var trendRows []trendRow
	err = r.db.WithContext(ctx).Raw(`
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', CURRENT_DATE) - INTERVAL '5 months',
				date_trunc('month', CURRENT_DATE),
				INTERVAL '1 month'
			) AS month_start
		), actual AS (
			SELECT date_trunc('month', dn.delivery_date) AS month_start, COUNT(*) AS cnt
			FROM delivery_notes_customer dn
			WHERE dn.status <> 'cancelled'
			GROUP BY 1
		), target AS (
			SELECT date_trunc('month', sc.schedule_date) AS month_start, COUNT(*) AS cnt
			FROM delivery_schedules_customer sc
			WHERE sc.deleted_at IS NULL
			GROUP BY 1
		)
		SELECT
			to_char(m.month_start, 'Mon') AS label,
			COALESCE(a.cnt, 0) AS actual,
			COALESCE(t.cnt, 0) AS target
		FROM months m
		LEFT JOIN actual a ON a.month_start = m.month_start
		LEFT JOIN target t ON t.month_start = m.month_start
		ORDER BY m.month_start
	`).Scan(&trendRows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard delivery performance trend: " + err.Error())
	}

	trend := make([]models.DeliveryTrendPoint, 0, len(trendRows))
	for _, row := range trendRows {
		trend = append(trend, models.DeliveryTrendPoint{Label: row.Label, Actual: row.Actual, Target: row.Target})
	}

	onTimeRate := 0.0
	if summary.Total > 0 {
		onTimeRate = (float64(summary.OnTime) * 100.0) / float64(summary.Total)
	}

	return &models.DeliveryPerformance{
		TotalDeliveries:   summary.Total,
		TotalValue:        0,
		OnTimeRatePercent: round2(onTimeRate),
		OnTimeCount:       summary.OnTime,
		Trend:             trend,
	}, nil
}

func (r *repository) GetProductionPerformance(ctx context.Context, current models.DateRange) (*models.ProductionPerformance, error) {
	outputExpr, err := r.productionScanOutputExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard production performance schema: " + err.Error())
	}
	dateExpr, err := r.productionScanDateExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard production performance schema: " + err.Error())
	}
	checkedExpr, err := r.productionScanCheckedExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard production performance schema: " + err.Error())
	}

	type summaryRow struct {
		CurrentProduction int64   `gorm:"column:current_production"`
		ActiveMachines    int64   `gorm:"column:active_machines"`
		TotalMachines     int64   `gorm:"column:total_machines"`
		TotalProduced     float64 `gorm:"column:total_produced"`
		GoodQty           float64 `gorm:"column:good_qty"`
		CheckedQty        float64 `gorm:"column:checked_qty"`
	}
	var summary summaryRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT
			(SELECT COUNT(DISTINCT psl.wo_id)
			 FROM production_scan_logs psl
			 WHERE %s = CURRENT_DATE) AS current_production,
			(SELECT COUNT(DISTINCT psl.machine_id)
			 FROM production_scan_logs psl
			 WHERE %s = CURRENT_DATE AND psl.machine_id IS NOT NULL) AS active_machines,
			(SELECT COUNT(*) FROM master_machines mm WHERE mm.status = 'Active') AS total_machines,
			(SELECT COALESCE(SUM(%s), 0)
			 FROM production_scan_logs psl
			 WHERE psl.scan_type = 'scan_out'
			   AND %s >= (CURRENT_DATE - INTERVAL '6 days')) AS total_produced,
			(SELECT COALESCE(SUM(%s), 0)
			 FROM production_scan_logs psl
			 WHERE psl.scan_type = 'scan_out'
			   AND %s BETWEEN ?::date AND ?::date) AS good_qty,
			(SELECT COALESCE(SUM(%s), 0)
			 FROM production_scan_logs psl
			 WHERE psl.scan_type = 'scan_out'
			   AND %s BETWEEN ?::date AND ?::date) AS checked_qty
	`, dateExpr, dateExpr, outputExpr, dateExpr, outputExpr, dateExpr, checkedExpr, dateExpr), current.StartDate, current.EndDate, current.StartDate, current.EndDate).Scan(&summary).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard production performance summary: " + err.Error())
	}

	type trendRow struct {
		Label    string  `gorm:"column:label"`
		Produced float64 `gorm:"column:produced"`
		Target   float64 `gorm:"column:target"`
	}
	var trendRows []trendRow
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH days AS (
			SELECT generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day')::date AS day_date
		), produced AS (
			SELECT %s AS day_date, COALESCE(SUM(%s), 0) AS qty
			FROM production_scan_logs psl
			WHERE psl.scan_type = 'scan_out'
			  AND %s >= CURRENT_DATE - INTERVAL '6 days'
			GROUP BY %s
		), target AS (
			SELECT wo.target_date AS day_date, COALESCE(SUM(woi.quantity), 0) AS qty
			FROM work_orders wo
			JOIN work_order_items woi ON woi.wo_id = wo.id
			WHERE wo.target_date >= CURRENT_DATE - INTERVAL '6 days'
			  AND wo.target_date <= CURRENT_DATE
			GROUP BY wo.target_date
		)
		SELECT
			to_char(d.day_date, 'Dy') AS label,
			COALESCE(p.qty, 0) AS produced,
			COALESCE(t.qty, 0) AS target
		FROM days d
		LEFT JOIN produced p ON p.day_date = d.day_date
		LEFT JOIN target t ON t.day_date = d.day_date
		ORDER BY d.day_date
	`, dateExpr, outputExpr, dateExpr, dateExpr)).Scan(&trendRows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard production performance trend: " + err.Error())
	}

	trend := make([]models.ProductionTrendPoint, 0, len(trendRows))
	for _, row := range trendRows {
		trend = append(trend, models.ProductionTrendPoint{
			Label:    row.Label,
			Produced: round2(row.Produced),
			Target:   round2(row.Target),
		})
	}

	capacity := 0.0
	if summary.TotalMachines > 0 {
		capacity = float64(summary.ActiveMachines) * 100.0 / float64(summary.TotalMachines)
	}
	quality := 0.0
	if summary.CheckedQty > 0 {
		quality = summary.GoodQty * 100.0 / summary.CheckedQty
	}

	return &models.ProductionPerformance{
		CurrentProduction: summary.CurrentProduction,
		CapacityPercent:   round2(capacity),
		TotalProduction:   round2(summary.TotalProduced),
		QualityPercent:    round2(quality),
		Trend:             trend,
	}, nil
}

func (r *repository) GetTopCustomers(ctx context.Context, current models.DateRange, limit int) ([]models.TopCustomer, error) {
	type row struct {
		CustomerID    int64   `gorm:"column:customer_id"`
		CustomerName  string  `gorm:"column:customer_name"`
		DeliveryCount int64   `gorm:"column:delivery_count"`
		Qty           float64 `gorm:"column:qty"`
		OnTimeCount   int64   `gorm:"column:on_time_count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		WITH base AS (
			SELECT
				dn.id AS dn_id,
				dn.customer_id,
				COALESCE(c.customer_name, dn.customer_name_snapshot, '-') AS customer_name,
				COALESCE(SUM(dni.qty_shipped), 0) AS qty,
				MAX(CASE
					WHEN dni.shipped_at IS NOT NULL AND dni.shipped_at::date <= dn.delivery_date THEN 1
					ELSE 0
				END) AS is_on_time
			FROM delivery_notes_customer dn
			LEFT JOIN delivery_note_items_customer dni ON dni.dn_id = dn.id
			LEFT JOIN customers c ON c.id = dn.customer_id
			WHERE dn.delivery_date BETWEEN ?::date AND ?::date
			  AND dn.status <> 'cancelled'
			GROUP BY dn.id, dn.customer_id, COALESCE(c.customer_name, dn.customer_name_snapshot, '-')
		), agg AS (
			SELECT
				customer_id,
				customer_name,
				COUNT(dn_id) AS delivery_count,
				COALESCE(SUM(qty), 0) AS qty,
				COALESCE(SUM(is_on_time), 0) AS on_time_count
			FROM base
			GROUP BY customer_id, customer_name
		), total AS (
			SELECT COALESCE(SUM(qty), 0) AS total_qty FROM agg
		)
		SELECT
			a.customer_id,
			a.customer_name,
			a.delivery_count,
			a.qty,
			a.on_time_count
		FROM agg a, total
		ORDER BY a.qty DESC, a.customer_name ASC
		LIMIT ?
	`, current.StartDate, current.EndDate, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard top customers: " + err.Error())
	}

	totalQty := 0.0
	for _, row := range rows {
		totalQty += row.Qty
	}

	result := make([]models.TopCustomer, 0, len(rows))
	for _, row := range rows {
		share := 0.0
		if totalQty > 0 {
			share = row.Qty * 100.0 / totalQty
		}
		otd := 0.0
		if row.DeliveryCount > 0 {
			otd = float64(row.OnTimeCount) * 100.0 / float64(row.DeliveryCount)
		}
		status := "Needs Attention"
		if otd > 95 {
			status = "Excellent"
		} else if otd >= 85 {
			status = "Good"
		}
		result = append(result, models.TopCustomer{
			CustomerID:    row.CustomerID,
			CustomerName:  row.CustomerName,
			DeliveryCount: row.DeliveryCount,
			SharePercent:  round2(share),
			Status:        status,
		})
	}
	return result, nil
}

func (r *repository) GetUniqProgress(ctx context.Context, limit int) ([]models.UniqProgress, error) {
	outputExpr, err := r.productionScanOutputExpr(ctx)
	if err != nil {
		return nil, apperror.Internal("main dashboard uniq progress schema: " + err.Error())
	}

	type row struct {
		WONumber    string  `gorm:"column:wo_number"`
		UniqCode    string  `gorm:"column:uniq_code"`
		TargetQty   float64 `gorm:"column:target_qty"`
		ProducedQty float64 `gorm:"column:produced_qty"`
		Status      string  `gorm:"column:status"`
	}
	var rows []row
	err = r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		WITH target AS (
			SELECT woi.wo_id, MIN(woi.item_uniq_code) AS uniq_code, COALESCE(SUM(woi.quantity), 0) AS target_qty
			FROM work_order_items woi
			GROUP BY woi.wo_id
		), produced AS (
			SELECT psl.wo_id, COALESCE(SUM(%s), 0) AS produced_qty
			FROM production_scan_logs psl
			WHERE psl.scan_type = 'scan_out'
			GROUP BY psl.wo_id
		)
		SELECT
			wo.wo_number,
			COALESCE(t.uniq_code, '') AS uniq_code,
			COALESCE(t.target_qty, 0) AS target_qty,
			COALESCE(p.produced_qty, 0) AS produced_qty,
			COALESCE(wo.status, '') AS status
		FROM work_orders wo
		LEFT JOIN target t ON t.wo_id = wo.id
		LEFT JOIN produced p ON p.wo_id = wo.id
		WHERE wo.status IN ('In Progress', 'Completed')
		ORDER BY wo.updated_at DESC
		LIMIT ?
	`, outputExpr), limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard uniq progress: " + err.Error())
	}

	result := make([]models.UniqProgress, 0, len(rows))
	for _, row := range rows {
		progress := 0.0
		if row.TargetQty > 0 {
			progress = math.Min((row.ProducedQty/row.TargetQty)*100.0, 100.0)
		}
		result = append(result, models.UniqProgress{
			WONumber:        row.WONumber,
			UniqCode:        row.UniqCode,
			ProducedQty:     round2(row.ProducedQty),
			TargetQty:       round2(row.TargetQty),
			ProgressPercent: round2(progress),
			Status:          row.Status,
		})
	}
	return result, nil
}

func (r *repository) GetPOSummary(ctx context.Context, current models.DateRange) (*models.POSummary, error) {
	type summaryRow struct {
		TotalPOs       int64   `gorm:"column:total_pos"`
		TotalValue     float64 `gorm:"column:total_value"`
		LowStockAlerts int64   `gorm:"column:low_stock_alerts"`
		CriticalAlerts int64   `gorm:"column:critical_alerts"`
	}
	var summary summaryRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM purchase_orders po
			 WHERE po.created_at::date BETWEEN ?::date AND ?::date
			   AND po.status <> 'cancelled') AS total_pos,
			(SELECT COALESCE(SUM(po.total_amount), 0) FROM purchase_orders po
			 WHERE po.created_at::date BETWEEN ?::date AND ?::date
			   AND po.status <> 'cancelled') AS total_value,
			(SELECT COUNT(*) FROM raw_materials rm
			 WHERE rm.deleted_at IS NULL AND rm.status = 'low_on_stock') +
			(SELECT COUNT(*) FROM indirect_raw_materials irm
			 WHERE irm.deleted_at IS NULL AND irm.status = 'low_on_stock') AS low_stock_alerts,
			(SELECT COUNT(*) FROM raw_materials rm
			 WHERE rm.deleted_at IS NULL
			   AND rm.status = 'low_on_stock'
			   AND rm.safety_stock_qty IS NOT NULL
			   AND rm.stock_qty < (rm.safety_stock_qty / 2.0)) AS critical_alerts
	`, current.StartDate, current.EndDate, current.StartDate, current.EndDate).Scan(&summary).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard po summary: " + err.Error())
	}

	var trend []models.POMonthlyTrendPoint
	err = r.db.WithContext(ctx).Raw(`
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', CURRENT_DATE) - INTERVAL '5 months',
				date_trunc('month', CURRENT_DATE),
				INTERVAL '1 month'
			) AS month_start
		), ordered_po AS (
			SELECT date_trunc('month', po.created_at) AS month_start,
			       COALESCE(SUM(po.total_amount), 0) AS ordered
			FROM purchase_orders po
			WHERE po.created_at >= date_trunc('month', CURRENT_DATE) - INTERVAL '5 months'
			GROUP BY 1
		), received_po AS (
			SELECT date_trunc('month', po.created_at) AS month_start,
			       COALESCE(SUM(po.total_amount), 0) AS received
			FROM purchase_orders po
			WHERE po.created_at >= date_trunc('month', CURRENT_DATE) - INTERVAL '5 months'
			  AND po.status = 'received'
			GROUP BY 1
		)
		SELECT
			to_char(m.month_start, 'Mon') AS label,
			COALESCE(o.ordered, 0) AS ordered,
			COALESCE(rp.received, 0) AS received
		FROM months m
		LEFT JOIN ordered_po o ON o.month_start = m.month_start
		LEFT JOIN received_po rp ON rp.month_start = m.month_start
		ORDER BY m.month_start
	`).Scan(&trend).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard po monthly trend: " + err.Error())
	}

	return &models.POSummary{
		TotalPOs:       summary.TotalPOs,
		TotalValue:     round2(summary.TotalValue),
		LowStockAlerts: summary.LowStockAlerts,
		CriticalAlerts: summary.CriticalAlerts,
		MonthlyTrend:   trend,
	}, nil
}

func (r *repository) GetCategoryDistribution(ctx context.Context) ([]models.CategoryDistribution, error) {
	type row struct {
		Category string  `gorm:"column:category"`
		Qty      float64 `gorm:"column:qty"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			CASE
				WHEN lower(trim(COALESCE(rm.raw_material_type, ''))) IN (
					'sheet_plate', 'sheet plate', 'steel', 'steel_plate', 'wire', 'ssp', 'metal', 'steel & metals'
				) THEN 'Steel & Metals'
				WHEN trim(COALESCE(rm.raw_material_type, '')) = '' THEN 'Others'
				ELSE initcap(replace(replace(trim(rm.raw_material_type), '_', ' '), '-', ' '))
			END AS category,
			COALESCE(SUM(rm.stock_qty), 0) AS qty
		FROM raw_materials rm
		WHERE rm.deleted_at IS NULL
		GROUP BY 1
	`).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard category distribution: " + err.Error())
	}

	total := 0.0
	for _, row := range rows {
		total += row.Qty
	}

	result := make([]models.CategoryDistribution, 0, len(rows))
	for _, row := range rows {
		share := 0.0
		if total > 0 {
			share = row.Qty * 100.0 / total
		}
		result = append(result, models.CategoryDistribution{
			Category:     row.Category,
			SharePercent: round2(share),
		})
	}
	return result, nil
}

func (r *repository) GetTopSuppliers(ctx context.Context, limit int) ([]models.TopSupplier, error) {
	var rows []models.TopSupplier
	err := r.db.WithContext(ctx).Raw(`
		WITH latest AS (
			SELECT DISTINCT ON (supplier_uuid)
				supplier_uuid,
				supplier_code,
				supplier_name,
				otd_percentage,
				quality_percentage,
				final_grade,
				computed_score,
				evaluation_date,
				computed_at
			FROM supplier_performance_snapshots
			WHERE deleted_at IS NULL
			ORDER BY supplier_uuid, evaluation_date DESC, computed_at DESC
		)
		SELECT
			latest.supplier_uuid::text AS supplier_uuid,
			latest.supplier_code,
			latest.supplier_name,
			latest.otd_percentage AS on_time_percent,
			latest.quality_percentage AS quality_percent,
			latest.final_grade AS grade
		FROM latest
		ORDER BY latest.computed_score DESC
		LIMIT ?
	`, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard top suppliers: " + err.Error())
	}
	return rows, nil
}

func (r *repository) ListTables(ctx context.Context, schema string, limit int) (*models.ListTablesResponse, error) {
	if schema == "" {
		schema = "public"
	}
	if limit <= 0 {
		limit = 200
	}
	type row struct {
		Name        string  `gorm:"column:name"`
		RowEstimate float64 `gorm:"column:row_estimate"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.table_name AS name,
			COALESCE(c.reltuples, 0) AS row_estimate
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE t.table_schema = ?
		  AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name ASC
		LIMIT ?
	`, schema, limit).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("main dashboard list tables: " + err.Error())
	}

	tables := make([]models.TableInfo, 0, len(rows))
	for _, row := range rows {
		tables = append(tables, models.TableInfo{Name: row.Name, RowEstimate: round2(row.RowEstimate)})
	}

	return &models.ListTablesResponse{
		Schema: schema,
		Count:  int64(len(tables)),
		Tables: tables,
	}, nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func percentDelta(current, previous float64) float64 {
	if previous <= 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return ((current - previous) * 100.0) / previous
}

func trendByDelta(delta float64) string {
	if delta > 0 {
		return "up"
	}
	if delta < 0 {
		return "down"
	}
	return "flat"
}

func mapCategory(rawType string) string {
	switch rawType {
	case "sheet_plate", "wire", "ssp":
		return "Steel & Metals"
	default:
		return "Others"
	}
}

type tableColumn struct {
	ColumnName string `gorm:"column:column_name"`
}

func (r *repository) tableColumns(ctx context.Context, tableName string) (map[string]bool, error) {
	var rows []tableColumn
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

type columnExpr struct {
	column string
	expr   string
}

func firstExistingColumnExpr(cols map[string]bool, candidates []columnExpr) string {
	for _, candidate := range candidates {
		if cols[candidate.column] {
			return candidate.expr
		}
	}
	return ""
}

func (r *repository) productionScanOutputExpr(ctx context.Context) (string, error) {
	cols, err := r.tableColumns(ctx, "production_scan_logs")
	if err != nil {
		return "", err
	}

	expr := firstExistingColumnExpr(cols, []columnExpr{
		{column: "good_quantity", expr: "COALESCE(psl.good_quantity, 0)"},
		{column: "qty_output", expr: "COALESCE(psl.qty_output, 0)"},
		{column: "quantity", expr: "COALESCE(psl.quantity, 0)"},
	})
	if expr == "" {
		expr = "0"
	}

	return expr, nil
}

func (r *repository) productionScanCheckedExpr(ctx context.Context) (string, error) {
	cols, err := r.tableColumns(ctx, "production_scan_logs")
	if err != nil {
		return "", err
	}

	expr := firstExistingColumnExpr(cols, []columnExpr{
		{column: "quantity", expr: "COALESCE(psl.quantity, 0)"},
		{column: "qty_input", expr: "COALESCE(psl.qty_input, 0)"},
		{column: "good_quantity", expr: "COALESCE(psl.good_quantity, 0)"},
		{column: "qty_output", expr: "COALESCE(psl.qty_output, 0)"},
	})
	if expr == "" {
		expr = "0"
	}

	return expr, nil
}

func (r *repository) productionScanDateExpr(ctx context.Context) (string, error) {
	cols, err := r.tableColumns(ctx, "production_scan_logs")
	if err != nil {
		return "", err
	}

	expr := firstExistingColumnExpr(cols, []columnExpr{
		{column: "report_date", expr: "psl.report_date"},
		{column: "scanned_at", expr: "psl.scanned_at::date"},
		{column: "created_at", expr: "psl.created_at::date"},
	})
	if expr == "" {
		expr = "CURRENT_DATE"
	}

	return expr, nil
}
