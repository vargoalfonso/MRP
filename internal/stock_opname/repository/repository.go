package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionFilter struct {
	Type           string
	Status         string
	Period         string
	Page           int
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

type HistoryLogRow struct {
	UniqCode   string    `gorm:"column:uniq_code"`
	Packing    *string   `gorm:"column:packing"`
	QtyChange  float64   `gorm:"column:qty_change"`
	Reason     *string   `gorm:"column:reason"`
	Qty        *float64  `gorm:"column:qty"`
	LastUpdate time.Time `gorm:"column:last_update"`
}

type SessionListRow struct {
	ID                int64      `gorm:"column:id"`
	UUID              uuid.UUID  `gorm:"column:uuid"`
	SessionNumber     string     `gorm:"column:session_number"`
	InventoryType     string     `gorm:"column:inventory_type"`
	Method            string     `gorm:"column:method"`
	PeriodMonth       int        `gorm:"column:period_month"`
	PeriodYear        int        `gorm:"column:period_year"`
	WarehouseLocation *string    `gorm:"column:warehouse_location"`
	ScheduleDate      *time.Time `gorm:"column:schedule_date"`
	CountedDate       *time.Time `gorm:"column:counted_date"`
	Remarks           *string    `gorm:"column:remarks"`
	TotalEntries      int        `gorm:"column:total_entries"`
	TotalVarianceQty  float64    `gorm:"column:total_variance_qty"`
	SystemQtyTotal    float64    `gorm:"column:system_qty_total"`
	PhysicalQtyTotal  float64    `gorm:"column:physical_qty_total"`
	VarianceQtyTotal  float64    `gorm:"column:variance_qty_total"`
	VariancePctTotal  *float64   `gorm:"column:variance_pct_total"`
	CostImpact        float64    `gorm:"column:cost_impact"`
	Status            string     `gorm:"column:status"`
	SubmittedBy       *string    `gorm:"column:submitted_by"`
	SubmittedAt       *time.Time `gorm:"column:submitted_at"`
	Approver          *string    `gorm:"column:approver"`
	ApprovedBy        *string    `gorm:"column:approved_by"`
	ApprovedAt        *time.Time `gorm:"column:approved_at"`
	ApprovalRemarks   *string    `gorm:"column:approval_remarks"`
	CreatedBy         *string    `gorm:"column:created_by"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

type IRepository interface {
	GetStats(ctx context.Context, inventoryType string) (*stockModels.StockOpnameStats, error)
	ListSessions(ctx context.Context, f SessionFilter) ([]SessionListRow, int64, error)
	GetSessionByID(ctx context.Context, id int64) (*stockModels.StockOpnameSession, error)
	GetSessionWithEntries(ctx context.Context, id int64) (*stockModels.StockOpnameSession, error)
	CreateSession(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession) error
	GenerateSessionNumber(ctx context.Context, tx *gorm.DB, inventoryType string, month, year int) (string, error)
	GetSessionByIDTx(ctx context.Context, tx *gorm.DB, id int64, lock bool) (*stockModels.StockOpnameSession, error)
	UpdateSession(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession) error
	SoftDeleteSession(ctx context.Context, tx *gorm.DB, id int64, actor string) error
	CreateEntry(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry) error
	GetEntryByIDTx(ctx context.Context, tx *gorm.DB, id int64, lock bool) (*stockModels.StockOpnameEntry, error)
	UpdateEntry(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry) error
	DeleteEntry(ctx context.Context, tx *gorm.DB, id int64) error
	ListEntriesBySessionTx(ctx context.Context, tx *gorm.DB, sessionID int64) ([]stockModels.StockOpnameEntry, error)
	RecalculateSessionTotals(ctx context.Context, tx *gorm.DB, sessionID int64) error
	DeriveSessionStatus(ctx context.Context, tx *gorm.DB, sessionID int64) (string, error)
	ListHistoryLogs(ctx context.Context, inventoryType, uniqCode, from, to string, limit, offset int) ([]HistoryLogRow, int64, error)
	CreateAuditLog(ctx context.Context, tx *gorm.DB, log *stockModels.StockOpnameAuditLog) error
	ListAuditLogs(ctx context.Context, sessionID int64, limit, offset int) ([]stockModels.StockOpnameAuditLog, int64, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) GetStats(ctx context.Context, inventoryType string) (*stockModels.StockOpnameStats, error) {
	type row struct {
		TotalRecords int64 `gorm:"column:total_records"`
		Completed    int64 `gorm:"column:completed"`
		WithVariance int64 `gorm:"column:with_variance"`
	}
	var res row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS total_records,
			COUNT(*) FILTER (WHERE status IN ('approved', 'rejected', 'partially_approved')) AS completed,
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1 FROM stock_opname_entries e
					WHERE e.session_id = s.id AND e.variance_qty <> 0
				)
			) AS with_variance
		FROM stock_opname_sessions s
		WHERE deleted_at IS NULL AND inventory_type = ?
	`, inventoryType).Scan(&res).Error
	if err != nil {
		return nil, apperror.Internal("get stock opname stats: " + err.Error())
	}
	return &stockModels.StockOpnameStats{TotalRecords: res.TotalRecords, Completed: res.Completed, WithVariance: res.WithVariance, CostImpact: 0}, nil
}

func (r *repository) ListSessions(ctx context.Context, f SessionFilter) ([]SessionListRow, int64, error) {
	q := r.db.WithContext(ctx).Table("stock_opname_sessions s").Where("s.deleted_at IS NULL")
	if f.Type != "" {
		q = q.Where("s.inventory_type = ?", f.Type)
	}
	if f.Status != "" {
		q = q.Where("s.status = ?", f.Status)
	}
	if f.Period != "" {
		parts := strings.SplitN(f.Period, "-", 2)
		if len(parts) == 2 {
			q = q.Where("s.period_year = ? AND s.period_month = ?", parts[0], parts[1])
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count stock opname sessions: " + err.Error())
	}
	var rows []SessionListRow
	order := safeOrder("created_at", f.OrderBy, f.OrderDirection, []string{"created_at", "updated_at", "period_year", "period_month", "session_number", "status"})
	selectSQL := `
		s.id,
		s.uuid,
		s.session_number,
		s.inventory_type,
		s.method,
		s.period_month,
		s.period_year,
		s.warehouse_location,
		s.schedule_date,
		s.counted_date,
		s.remarks,
		s.total_entries,
		s.total_variance_qty,
		COALESCE((SELECT SUM(e.system_qty_snapshot) FROM stock_opname_entries e WHERE e.session_id = s.id), 0) AS system_qty_total,
		COALESCE((SELECT SUM(e.counted_qty) FROM stock_opname_entries e WHERE e.session_id = s.id), 0) AS physical_qty_total,
		COALESCE((SELECT SUM(e.counted_qty - e.system_qty_snapshot) FROM stock_opname_entries e WHERE e.session_id = s.id), 0) AS variance_qty_total,
		CASE
			WHEN COALESCE((SELECT SUM(e.system_qty_snapshot) FROM stock_opname_entries e WHERE e.session_id = s.id), 0) = 0 THEN NULL
			ELSE (
				COALESCE((SELECT SUM(e.counted_qty - e.system_qty_snapshot) FROM stock_opname_entries e WHERE e.session_id = s.id), 0)
				/
				NULLIF((SELECT SUM(e.system_qty_snapshot) FROM stock_opname_entries e WHERE e.session_id = s.id), 0)
			) * 100
		END AS variance_pct_total,
		0::numeric AS cost_impact,
		s.status,
		s.submitted_by,
		s.submitted_at,
		s.approver,
		s.approved_by,
		s.approved_at,
		s.approval_remarks,
		s.created_by,
		s.created_at,
		s.updated_at
	`
	if err := q.Select(selectSQL).Order("s." + order).Limit(f.Limit).Offset(f.Offset).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("list stock opname sessions: " + err.Error())
	}
	return rows, total, nil
}

func (r *repository) GetSessionByID(ctx context.Context, id int64) (*stockModels.StockOpnameSession, error) {
	var row stockModels.StockOpnameSession
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("stock opname session id %d not found", id))
	}
	if err != nil {
		return nil, apperror.Internal("get stock opname session: " + err.Error())
	}
	return &row, nil
}

func (r *repository) GetSessionWithEntries(ctx context.Context, id int64) (*stockModels.StockOpnameSession, error) {
	var row stockModels.StockOpnameSession
	err := r.db.WithContext(ctx).Preload("Entries", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).Where("id = ? AND deleted_at IS NULL", id).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("stock opname session id %d not found", id))
	}
	if err != nil {
		return nil, apperror.Internal("get stock opname detail: " + err.Error())
	}
	return &row, nil
}

func (r *repository) CreateSession(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession) error {
	session.UUID = uuid.New()
	if err := tx.WithContext(ctx).Create(session).Error; err != nil {
		return apperror.Internal("create stock opname session: " + err.Error())
	}
	return nil
}

func (r *repository) GenerateSessionNumber(ctx context.Context, tx *gorm.DB, inventoryType string, month, year int) (string, error) {
	prefix := fmt.Sprintf("SO-%s-%02d%d-%%", inventoryType, month, year)
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT COUNT(*) FROM stock_opname_sessions WHERE session_number LIKE ?`, prefix).Scan(&count).Error; err != nil {
		return "", apperror.Internal("generate stock opname session number: " + err.Error())
	}
	return fmt.Sprintf("SO-%s-%02d%d-%03d", inventoryType, month, year, count+1), nil
}

func (r *repository) GetSessionByIDTx(ctx context.Context, tx *gorm.DB, id int64, lock bool) (*stockModels.StockOpnameSession, error) {
	q := tx.WithContext(ctx).Model(&stockModels.StockOpnameSession{})
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row stockModels.StockOpnameSession
	err := q.Where("id = ? AND deleted_at IS NULL", id).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("stock opname session id %d not found", id))
	}
	if err != nil {
		return nil, apperror.Internal("get stock opname session tx: " + err.Error())
	}
	return &row, nil
}

func (r *repository) UpdateSession(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession) error {
	if err := tx.WithContext(ctx).Save(session).Error; err != nil {
		return apperror.Internal("update stock opname session: " + err.Error())
	}
	return nil
}

func (r *repository) SoftDeleteSession(ctx context.Context, tx *gorm.DB, id int64, actor string) error {
	now := time.Now()
	res := tx.WithContext(ctx).Model(&stockModels.StockOpnameSession{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]interface{}{"deleted_at": now, "updated_at": now, "updated_by": actor})
	if res.Error != nil {
		return apperror.Internal("delete stock opname session: " + res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound(fmt.Sprintf("stock opname session id %d not found", id))
	}
	return nil
}

func (r *repository) CreateEntry(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry) error {
	entry.UUID = uuid.New()
	if err := tx.WithContext(ctx).Create(entry).Error; err != nil {
		return apperror.Internal("create stock opname entry: " + err.Error())
	}
	return nil
}

func (r *repository) GetEntryByIDTx(ctx context.Context, tx *gorm.DB, id int64, lock bool) (*stockModels.StockOpnameEntry, error) {
	q := tx.WithContext(ctx).Model(&stockModels.StockOpnameEntry{})
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row stockModels.StockOpnameEntry
	err := q.Where("id = ?", id).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound(fmt.Sprintf("stock opname entry id %d not found", id))
	}
	if err != nil {
		return nil, apperror.Internal("get stock opname entry: " + err.Error())
	}
	return &row, nil
}

func (r *repository) UpdateEntry(ctx context.Context, tx *gorm.DB, entry *stockModels.StockOpnameEntry) error {
	if err := tx.WithContext(ctx).Save(entry).Error; err != nil {
		return apperror.Internal("update stock opname entry: " + err.Error())
	}
	return nil
}

func (r *repository) DeleteEntry(ctx context.Context, tx *gorm.DB, id int64) error {
	if err := tx.WithContext(ctx).Delete(&stockModels.StockOpnameEntry{}, id).Error; err != nil {
		return apperror.Internal("delete stock opname entry: " + err.Error())
	}
	return nil
}

func (r *repository) ListEntriesBySessionTx(ctx context.Context, tx *gorm.DB, sessionID int64) ([]stockModels.StockOpnameEntry, error) {
	var rows []stockModels.StockOpnameEntry
	if err := tx.WithContext(ctx).Where("session_id = ?", sessionID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, apperror.Internal("list stock opname entries: " + err.Error())
	}
	return rows, nil
}

func (r *repository) RecalculateSessionTotals(ctx context.Context, tx *gorm.DB, sessionID int64) error {
	type totalRow struct {
		TotalEntries     int     `gorm:"column:total_entries"`
		TotalVarianceQty float64 `gorm:"column:total_variance_qty"`
	}
	var row totalRow
	if err := tx.WithContext(ctx).Raw(`SELECT COUNT(*) AS total_entries, COALESCE(SUM(ABS(variance_qty)), 0) AS total_variance_qty FROM stock_opname_entries WHERE session_id = ?`, sessionID).Scan(&row).Error; err != nil {
		return apperror.Internal("recalculate stock opname totals: " + err.Error())
	}
	if err := tx.WithContext(ctx).Model(&stockModels.StockOpnameSession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{"total_entries": row.TotalEntries, "total_variance_qty": row.TotalVarianceQty}).Error; err != nil {
		return apperror.Internal("update stock opname totals: " + err.Error())
	}
	return nil
}

func (r *repository) DeriveSessionStatus(ctx context.Context, tx *gorm.DB, sessionID int64) (string, error) {
	type row struct{ Pending, Approved, Rejected int64 }
	var res row
	err := tx.WithContext(ctx).Raw(`SELECT COUNT(*) FILTER (WHERE status = 'pending') AS pending, COUNT(*) FILTER (WHERE status = 'approved') AS approved, COUNT(*) FILTER (WHERE status = 'rejected') AS rejected FROM stock_opname_entries WHERE session_id = ?`, sessionID).Scan(&res).Error
	if err != nil {
		return "", apperror.Internal("derive stock opname status: " + err.Error())
	}
	if res.Pending > 0 {
		return stockModels.SessionStatusPendingApproval, nil
	}
	if res.Approved > 0 && res.Rejected > 0 {
		return stockModels.SessionStatusPartiallyApproved, nil
	}
	if res.Approved > 0 {
		return stockModels.SessionStatusApproved, nil
	}
	if res.Rejected > 0 {
		return stockModels.SessionStatusRejected, nil
	}
	return stockModels.SessionStatusPendingApproval, nil
}

func (r *repository) ListHistoryLogs(ctx context.Context, inventoryType, uniqCode, from, to string, limit, offset int) ([]HistoryLogRow, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	countSQL, dataSQL, args := buildHistoryQuery(inventoryType, uniqCode, from, to)
	var total int64
	if err := r.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count stock opname history logs: " + err.Error())
	}
	var rows []HistoryLogRow
	args = append(args, limit, offset)
	if err := r.db.WithContext(ctx).Raw(dataSQL, args...).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("list stock opname history logs: " + err.Error())
	}
	return rows, total, nil
}

func (r *repository) CreateAuditLog(ctx context.Context, tx *gorm.DB, log *stockModels.StockOpnameAuditLog) error {
	log.UUID = uuid.New()
	if len(log.Metadata) == 0 {
		log.Metadata = []byte("{}")
	}
	if err := tx.WithContext(ctx).Create(log).Error; err != nil {
		return apperror.Internal("create stock opname audit log: " + err.Error())
	}
	return nil
}

func (r *repository) ListAuditLogs(ctx context.Context, sessionID int64, limit, offset int) ([]stockModels.StockOpnameAuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&stockModels.StockOpnameAuditLog{}).Where("session_id = ?", sessionID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal("count stock opname audit logs: " + err.Error())
	}
	var rows []stockModels.StockOpnameAuditLog
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, apperror.Internal("list stock opname audit logs: " + err.Error())
	}
	return rows, total, nil
}

func ToJSONMap(v map[string]interface{}) []byte {
	if len(v) == 0 {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func buildHistoryQuery(inventoryType, uniqCode, from, to string) (string, string, []interface{}) {
	args := []interface{}{}
	where := []string{}
	switch inventoryType {
	case stockModels.InventoryTypeFG:
		base := ` FROM fg_movement_logs WHERE movement_type = 'stock_opname' `
		if uniqCode != "" {
			where = append(where, "uniq_code = ?")
			args = append(args, uniqCode)
		}
		if from != "" {
			where = append(where, "logged_at >= ?")
			args = append(args, from)
		}
		if to != "" {
			where = append(where, "logged_at <= ?")
			args = append(args, to+" 23:59:59")
		}
		w := joinWhere(where)
		return `SELECT COUNT(*)` + base + w, `SELECT uniq_code, COALESCE(reference_id, NULL)::varchar AS packing, qty_change, movement_type AS reason, qty_after AS qty, logged_at AS last_update` + base + w + ` ORDER BY logged_at DESC LIMIT ? OFFSET ?`, args
	case stockModels.InventoryTypeRM:
		base := ` FROM inventory_movement_logs iml LEFT JOIN raw_materials rm ON rm.id = iml.entity_id WHERE iml.movement_category = 'raw_material' AND iml.movement_type = 'stock_opname' `
		if uniqCode != "" {
			where = append(where, "iml.uniq_code = ?")
			args = append(args, uniqCode)
		}
		if from != "" {
			where = append(where, "iml.logged_at >= ?")
			args = append(args, from)
		}
		if to != "" {
			where = append(where, "iml.logged_at <= ?")
			args = append(args, to+" 23:59:59")
		}
		w := joinWhere(where)
		return `SELECT COUNT(*)` + base + w, `SELECT iml.uniq_code, COALESCE(iml.reference_id, NULL)::varchar AS packing, iml.qty_change, iml.movement_type AS reason, rm.stock_qty AS qty, iml.logged_at AS last_update` + base + w + ` ORDER BY iml.logged_at DESC LIMIT ? OFFSET ?`, args
	case stockModels.InventoryTypeIDR:
		base := ` FROM inventory_movement_logs iml LEFT JOIN indirect_raw_materials irm ON irm.id = iml.entity_id WHERE iml.movement_category = 'indirect_raw_material' AND iml.movement_type = 'stock_opname' `
		if uniqCode != "" {
			where = append(where, "iml.uniq_code = ?")
			args = append(args, uniqCode)
		}
		if from != "" {
			where = append(where, "iml.logged_at >= ?")
			args = append(args, from)
		}
		if to != "" {
			where = append(where, "iml.logged_at <= ?")
			args = append(args, to+" 23:59:59")
		}
		w := joinWhere(where)
		return `SELECT COUNT(*)` + base + w, `SELECT iml.uniq_code, COALESCE(iml.reference_id, NULL)::varchar AS packing, iml.qty_change, iml.movement_type AS reason, irm.stock_qty AS qty, iml.logged_at AS last_update` + base + w + ` ORDER BY iml.logged_at DESC LIMIT ? OFFSET ?`, args
	default:
		base := ` FROM wip_logs wl JOIN wip_items wi ON wi.id = wl.wip_item_id WHERE wl.action = 'stock_opname' `
		if uniqCode != "" {
			where = append(where, "wi.uniq = ?")
			args = append(args, uniqCode)
		}
		if from != "" {
			where = append(where, "wl.created_at >= ?")
			args = append(args, from)
		}
		if to != "" {
			where = append(where, "wl.created_at <= ?")
			args = append(args, to+" 23:59:59")
		}
		w := joinWhere(where)
		return `SELECT COUNT(*)` + base + w, `SELECT wi.uniq AS uniq_code, wi.packing_number AS packing, wl.qty::numeric AS qty_change, wl.action AS reason, wi.stock::numeric AS qty, wl.created_at AS last_update` + base + w + ` ORDER BY wl.created_at DESC LIMIT ? OFFSET ?`, args
	}
}

func joinWhere(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return " AND " + strings.Join(parts, " AND ")
}

func safeOrder(def, col, dir string, allowed []string) string {
	allowedMap := map[string]bool{}
	for _, item := range allowed {
		allowedMap[item] = true
	}
	if !allowedMap[col] {
		col = def
	}
	if strings.ToLower(dir) == "asc" {
		return col + " ASC"
	}
	return col + " DESC"
}
