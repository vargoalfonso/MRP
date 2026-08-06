package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/production_dashboard/models"
	"gorm.io/gorm"
)

// IRepository exposes the aggregation queries backing each dashboard view.
// All data is sourced from production_scan_logs (shop-floor scan events) joined
// with work_orders, work_order_items and master_machines.
type IRepository interface {
	GetSummaryCards(ctx context.Context) (models.SummaryCards, error)
	ListFinishedGoods(ctx context.Context) ([]models.FinishedGoodsRow, error)
	ListWip(ctx context.Context) ([]models.WipRow, error)
	ListOutputPerMachine(ctx context.Context) ([]models.OutputMachineRow, error)
	ListSummaryStroke(ctx context.Context) ([]models.SummaryStrokeRow, error)
	ListRuntime(ctx context.Context) ([]models.RuntimeRow, error)
}

type repository struct{ db *gorm.DB }

// New builds the Production Dashboard repository.
func New(db *gorm.DB) IRepository { return &repository{db: db} }

// rowLimit caps every table view so the dashboard never returns unbounded data.
const rowLimit = 500

// scanSchema holds SQL fragments resolved against the ACTUAL columns present in
// production_scan_logs. The schema has drifted across environments (e.g.
// good_quantity vs qty_output, report_date vs scanned_at, machine_id UUID vs
// machine_number), so every fragment is picked from the columns that truly
// exist. This mirrors the resolver used by the shop_floor / main_dashboard
// modules and prevents "column does not exist" 500 errors.
type scanSchema struct {
	reportDate  string
	eventAt     string
	shift       string
	output      string
	input       string
	ngSetting   string
	ngProcess   string
	scrap       string
	rework      string
	scanType    string
	machine     string
	line        string
	joinMachine string
}

type colExpr struct {
	col  string
	expr string
}

func firstExpr(cols map[string]bool, candidates []colExpr, fallback string) string {
	for _, c := range candidates {
		if cols[c.col] {
			return c.expr
		}
	}
	return fallback
}

func (r *repository) tableColumns(ctx context.Context, table string) (map[string]bool, error) {
	type row struct {
		ColumnName string `gorm:"column:column_name"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?
	`, table).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	cols := make(map[string]bool, len(rows))
	for _, row := range rows {
		cols[row.ColumnName] = true
	}
	return cols, nil
}

func (r *repository) resolveSchema(ctx context.Context) (*scanSchema, error) {
	cols, err := r.tableColumns(ctx, "production_scan_logs")
	if err != nil {
		return nil, err
	}
	machineCols, err := r.tableColumns(ctx, "master_machines")
	if err != nil {
		return nil, err
	}

	joinMachine := ""
	if len(machineCols) > 0 && cols["machine_id"] {
		joinMachine = "LEFT JOIN master_machines mm ON CAST(mm.id AS text) = CAST(psl.machine_id AS text)"
	}

	s := &scanSchema{joinMachine: joinMachine}

	s.reportDate = firstExpr(cols, []colExpr{
		{"report_date", "psl.report_date"},
		{"scanned_at", "psl.scanned_at::date"},
		{"created_at", "psl.created_at::date"},
	}, "CURRENT_DATE")

	s.eventAt = firstExpr(cols, []colExpr{
		{"scanned_at", "psl.scanned_at"},
		{"created_at", "psl.created_at"},
	}, "NOW()")

	s.shift = firstExpr(cols, []colExpr{
		{"shift", "COALESCE(psl.shift, '')"},
	}, "''")

	s.output = firstExpr(cols, []colExpr{
		{"good_quantity", "COALESCE(psl.good_quantity, 0)"},
		{"qty_output", "COALESCE(psl.qty_output, 0)"},
		{"quantity", "COALESCE(psl.quantity, 0)"},
	}, "0")

	s.input = firstExpr(cols, []colExpr{
		{"quantity", "COALESCE(psl.quantity, 0)"},
		{"qty_input", "COALESCE(psl.qty_input, 0)"},
		{"good_quantity", "COALESCE(psl.good_quantity, 0)"},
	}, "0")

	s.ngSetting = firstExpr(cols, []colExpr{
		{"ng_setting_machine", "COALESCE(psl.ng_setting_machine, 0)"},
		{"ng_machine", "COALESCE(psl.ng_machine, 0)"},
		{"ng_setting", "COALESCE(psl.ng_setting, 0)"},
	}, "0")

	s.ngProcess = firstExpr(cols, []colExpr{
		{"ng_process", "COALESCE(psl.ng_process, 0)"},
	}, "0")

	s.scrap = firstExpr(cols, []colExpr{
		{"scrap_quantity", "COALESCE(psl.scrap_quantity, 0)"},
		{"qty_scrap", "COALESCE(psl.qty_scrap, 0)"},
	}, "0")

	s.rework = firstExpr(cols, []colExpr{
		{"qty_rework", "COALESCE(psl.qty_rework, 0)"},
		{"rework_quantity", "COALESCE(psl.rework_quantity, 0)"},
	}, "0")

	s.scanType = firstExpr(cols, []colExpr{
		{"scan_type", "COALESCE(psl.scan_type, '')"},
	}, "''")

	// Machine display name: prefer a column on the scan log, else the joined
	// master_machines row, else the raw machine_id, else UNASSIGNED.
	s.machine = firstExpr(cols, []colExpr{
		{"machine_number", "COALESCE(NULLIF(psl.machine_number, ''), 'UNASSIGNED')"},
	}, "")
	if s.machine == "" {
		if joinMachine != "" {
			s.machine = firstExpr(machineCols, []colExpr{
				{"machine_number", "COALESCE(NULLIF(mm.machine_number, ''), NULLIF(mm.machine_name, ''), CAST(psl.machine_id AS text), 'UNASSIGNED')"},
				{"machine_name", "COALESCE(NULLIF(mm.machine_name, ''), CAST(psl.machine_id AS text), 'UNASSIGNED')"},
			}, "COALESCE(CAST(psl.machine_id AS text), 'UNASSIGNED')")
		} else if cols["machine_id"] {
			s.machine = "COALESCE(CAST(psl.machine_id AS text), 'UNASSIGNED')"
		} else {
			s.machine = "'UNASSIGNED'"
		}
	}

	s.line = firstExpr(cols, []colExpr{
		{"production_line", "COALESCE(NULLIF(psl.production_line, ''), '-')"},
	}, "")
	if s.line == "" && joinMachine != "" {
		s.line = firstExpr(machineCols, []colExpr{
			{"production_line", "COALESCE(NULLIF(mm.production_line, ''), '-')"},
		}, "")
	}
	if s.line == "" {
		s.line = "'-'"
	}

	return s, nil
}

// scanFilterOut / scanFilterIn tolerate both the ('scan_in','scan_out') and
// legacy ('IN','OUT') scan_type conventions.
func (s *scanSchema) whereOut() string {
	return "UPPER(" + s.scanType + ") IN ('OUT', 'SCAN_OUT')"
}

func (s *scanSchema) whereIn() string {
	return "UPPER(" + s.scanType + ") IN ('IN', 'SCAN_IN')"
}

func (r *repository) run(ctx context.Context, tmpl string, repl *strings.Replacer, dest interface{}) error {
	query := repl.Replace(tmpl)
	return r.db.WithContext(ctx).Raw(query, rowLimit).Scan(dest).Error
}

func (s *scanSchema) baseReplacer() *strings.Replacer {
	return strings.NewReplacer(
		"{REPORT_DATE}", s.reportDate,
		"{EVENT_AT}", s.eventAt,
		"{SHIFT}", s.shift,
		"{OUTPUT}", s.output,
		"{INPUT}", s.input,
		"{NG_SETTING}", s.ngSetting,
		"{NG_PROCESS}", s.ngProcess,
		"{SCRAP}", s.scrap,
		"{REWORK}", s.rework,
		"{MACHINE}", s.machine,
		"{LINE}", s.line,
		"{JOIN_MACHINE}", s.joinMachine,
		"{WHERE_OUT}", s.whereOut(),
		"{WHERE_IN}", s.whereIn(),
	)
}

func (r *repository) GetSummaryCards(ctx context.Context) (models.SummaryCards, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return models.SummaryCards{}, err
	}
	var card models.SummaryCards
	const tmpl = `
SELECT
	COALESCE(SUM({OUTPUT}) FILTER (WHERE {WHERE_OUT}), 0)                    AS fg_output,
	COALESCE(SUM({INPUT})  FILTER (WHERE {WHERE_IN}), 0)                     AS wip_output,
	COALESCE(SUM({NG_SETTING} + {NG_PROCESS}), 0)                           AS total_ng,
	COALESCE(SUM({REWORK}), 0)                                              AS total_rework
FROM production_scan_logs psl`
	// LIMIT ? is unused here; use a fixed query without the shared runner.
	query := s.baseReplacer().Replace(tmpl)
	if err := r.db.WithContext(ctx).Raw(query).Scan(&card).Error; err != nil {
		return models.SummaryCards{}, err
	}
	return card, nil
}

func (r *repository) ListFinishedGoods(ctx context.Context) ([]models.FinishedGoodsRow, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]models.FinishedGoodsRow, 0)
	const tmpl = `
SELECT
	CAST(MIN(psl.id) AS TEXT)               AS id,
	{REPORT_DATE}                           AS report_date,
	COALESCE(woi.item_uniq_code, '')        AS uniq,
	COALESCE(woi.part_name, '')             AS product_name,
	COALESCE(wo.wo_number, '')              AS wo_number,
	{SHIFT}                                 AS shift,
	COALESCE(SUM({OUTPUT}), 0)              AS fg_output,
	COALESCE(SUM({NG_SETTING}), 0)          AS ng_setting,
	COALESCE(SUM({NG_PROCESS}), 0)          AS ng_process,
	COALESCE(SUM({REWORK}), 0)             AS rework,
	COALESCE(SUM({SCRAP}), 0)              AS scrap
FROM production_scan_logs psl
LEFT JOIN work_orders wo        ON wo.id = psl.wo_id
LEFT JOIN work_order_items woi  ON woi.id = psl.wo_item_id
WHERE {WHERE_OUT}
GROUP BY {REPORT_DATE}, woi.item_uniq_code, woi.part_name, wo.wo_number, {SHIFT}
ORDER BY {REPORT_DATE} DESC
LIMIT ?`
	if err := r.run(ctx, tmpl, s.baseReplacer(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) ListWip(ctx context.Context) ([]models.WipRow, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]models.WipRow, 0)
	const tmpl = `
SELECT
	CAST(MIN(psl.id) AS TEXT)               AS id,
	{REPORT_DATE}                           AS report_date,
	COALESCE(woi.item_uniq_code, '')        AS uniq,
	COALESCE(woi.part_name, '')             AS product_name,
	COALESCE(psl.process_name, '')          AS process_name,
	COALESCE(wo.wo_number, '')              AS wo_number,
	{SHIFT}                                 AS shift,
	COALESCE(SUM({INPUT}), 0)              AS wip_output,
	COALESCE(SUM({NG_SETTING}), 0)          AS ng_setting,
	COALESCE(SUM({NG_PROCESS}), 0)          AS ng_process,
	COALESCE(SUM({REWORK}), 0)             AS rework,
	COALESCE(SUM({SCRAP}), 0)              AS scrap
FROM production_scan_logs psl
LEFT JOIN work_orders wo        ON wo.id = psl.wo_id
LEFT JOIN work_order_items woi  ON woi.id = psl.wo_item_id
WHERE {WHERE_IN}
GROUP BY {REPORT_DATE}, woi.item_uniq_code, woi.part_name, psl.process_name, wo.wo_number, {SHIFT}
ORDER BY {REPORT_DATE} DESC
LIMIT ?`
	if err := r.run(ctx, tmpl, s.baseReplacer(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) ListOutputPerMachine(ctx context.Context) ([]models.OutputMachineRow, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]models.OutputMachineRow, 0)
	const tmpl = `
SELECT
	CAST(MIN(psl.id) AS TEXT)               AS id,
	{REPORT_DATE}                           AS report_date,
	{LINE}                                  AS line_process,
	{MACHINE}                               AS machine_name,
	COALESCE(woi.item_uniq_code, '')        AS uniq,
	{SHIFT}                                 AS shift,
	COALESCE(wo.wo_number, '')              AS wo_number,
	COALESCE(SUM({OUTPUT}), 0)              AS product_output,
	COALESCE(SUM({NG_SETTING}), 0)          AS ng_setting,
	COALESCE(SUM({NG_PROCESS}), 0)          AS ng_process,
	COALESCE(SUM({REWORK}), 0)             AS rework,
	COALESCE(SUM({SCRAP}), 0)              AS scrap
FROM production_scan_logs psl
{JOIN_MACHINE}
LEFT JOIN work_orders wo        ON wo.id = psl.wo_id
LEFT JOIN work_order_items woi  ON woi.id = psl.wo_item_id
WHERE {WHERE_OUT}
GROUP BY {REPORT_DATE}, {LINE}, {MACHINE}, woi.item_uniq_code, {SHIFT}, wo.wo_number
ORDER BY {REPORT_DATE} DESC
LIMIT ?`
	if err := r.run(ctx, tmpl, s.baseReplacer(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) ListSummaryStroke(ctx context.Context) ([]models.SummaryStrokeRow, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]models.SummaryStrokeRow, 0)
	// stroke / dandori / set-qc time are not captured by scan logs today, so they
	// are reported as 0. machine_time_min is derived from the span between the
	// first and last scan of the line on that day.
	const tmpl = `
SELECT
	CAST(MIN(psl.id) AS TEXT)                                                                  AS id,
	{REPORT_DATE}                                                                              AS report_date,
	{LINE}                                                                                     AS production_line,
	0                                                                                          AS stroke,
	COALESCE(SUM({OUTPUT}), 0)                                                                  AS production_output,
	COALESCE(ROUND(EXTRACT(EPOCH FROM (MAX({EVENT_AT}) - MIN({EVENT_AT}))) / 60.0), 0)          AS machine_time_min,
	0                                                                                          AS dandori_time_min,
	0                                                                                          AS set_qc_time_min
FROM production_scan_logs psl
{JOIN_MACHINE}
WHERE {WHERE_OUT}
GROUP BY {REPORT_DATE}, {LINE}
ORDER BY {REPORT_DATE} DESC
LIMIT ?`
	if err := r.run(ctx, tmpl, s.baseReplacer(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) ListRuntime(ctx context.Context) ([]models.RuntimeRow, error) {
	s, err := r.resolveSchema(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]models.RuntimeRow, 0)
	// total_machine_time_min is derived from the span between the first and last
	// scan for the WO/machine on that day. dandori / set-qc time are not tracked
	// by scan logs yet and are reported as 0.
	const tmpl = `
SELECT
	CAST(MIN(psl.id) AS TEXT)                                                                  AS id,
	{REPORT_DATE}                                                                              AS report_date,
	COALESCE(wo.wo_number, '')                                                                 AS wo_number,
	{LINE}                                                                                     AS production_line,
	{MACHINE}                                                                                  AS machine_number,
	COALESCE(ROUND(EXTRACT(EPOCH FROM (MAX({EVENT_AT}) - MIN({EVENT_AT}))) / 60.0), 0)          AS total_machine_time_min,
	0                                                                                          AS dandori_time_min,
	0                                                                                          AS set_qc_time_min
FROM production_scan_logs psl
{JOIN_MACHINE}
LEFT JOIN work_orders wo ON wo.id = psl.wo_id
WHERE {WHERE_OUT} OR {WHERE_IN}
GROUP BY {REPORT_DATE}, wo.wo_number, {LINE}, {MACHINE}
ORDER BY {REPORT_DATE} DESC
LIMIT ?`
	if err := r.run(ctx, tmpl, s.baseReplacer(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
