package repository

import (
	"context"
	"database/sql"

	//"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	customerModels "github.com/ganasa18/go-template/internal/customer/models"
	"github.com/ganasa18/go-template/internal/prl/models"
	"github.com/ganasa18/go-template/pkg/apperror"

	//"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// func wrapPRLPersistError(msg string, err error) error {
// 	var pgErr *pgconn.PgError
// 	if errors.As(err, &pgErr) {
// 		lowerMsg := strings.ToLower(pgErr.Message)
// 		// Check violation (e.g., old constraint prls_forecast_period_check).
// 		if pgErr.Code == "23514" && (strings.Contains(pgErr.ConstraintName, "prls_forecast_period_check") || strings.Contains(lowerMsg, "prls_forecast_period_check")) {
// 			return apperror.BadRequest(
// 				"forecast_period is now free-text, but DB still enforces quarter format; run migration scripts/migrations/0042_prls_forecast_period_freetext_up.sql",
// 			)
// 		}
// 		// String truncation (e.g., forecast_period is still VARCHAR(7)).
// 		if pgErr.Code == "22001" {
// 			// On old schema: message is typically "value too long for type character varying(7)" (no column name).
// 			if strings.Contains(lowerMsg, "character varying(7)") || strings.Contains(lowerMsg, "varchar(7)") || strings.Contains(lowerMsg, "forecast_period") {
// 				return apperror.BadRequest(
// 					"forecast_period is longer than the DB column allows; run migration scripts/migrations/0042_prls_forecast_period_freetext_up.sql",
// 				)
// 			}
// 			return apperror.BadRequest("a field is longer than the DB column allows")
// 		}
// 	}
// 	return apperror.InternalWrap(msg, err)
// }

type IRepository interface {
	CreateUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error
	FindUniqBOMByUUID(ctx context.Context, uuid string) (*models.UniqBillOfMaterial, error)
	FindUniqBOMByUniqCode(ctx context.Context, uniqCode string) (*models.UniqBillOfMaterial, error)
	ListUniqBOMs(ctx context.Context, filters models.UniqBOMListFilters) ([]models.UniqBillOfMaterial, int64, error)
	UpdateUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error
	DeleteUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error

	CreatePRLs(ctx context.Context, items []*models.PRL) error
	FindPRLByID(ctx context.Context, id int64) (*models.PRL, error)
	FindPRLByUUID(ctx context.Context, uuid string) (*models.PRL, error)
	ListPRLs(ctx context.Context, filters models.PRLListFilters) ([]models.PRL, int64, error)
	ListPRLsForExport(ctx context.Context, filters models.PRLListFilters) ([]models.PRL, error)
	ListPRLHistoryVsDelivery(ctx context.Context, filters models.PRLHistoryFilters) ([]models.PRLHistoryListItem, int64, error)
	ListPRLMachinePatterns(ctx context.Context, filters models.PRLHistoryFilters) ([]models.PRLMachinePatternListItem, int64, error)
	GetPRLHistorySummary(ctx context.Context, uniqCode, forecastPeriod string) (*models.PRLHistoryDetailSummary, error)
	ListPRLHistoryTimeline(ctx context.Context, uniqCode, forecastPeriod string, limit int) ([]models.PRLHistoryLogItem, error)
	GetMachinePatternByUniqCode(ctx context.Context, uniqCode string) (string, error)
	UpdatePRL(ctx context.Context, item *models.PRL) error
	DeletePRL(ctx context.Context, item *models.PRL) error
	BulkSetStatus(ctx context.Context, ids []string, status string) (int64, error)

	ListCustomers(ctx context.Context, search string) ([]models.CustomerLookup, error)
	FindCustomerByUUID(ctx context.Context, uuid string) (*customerModels.Customer, error)
	FindCustomerByRowID(ctx context.Context, id int64) (*customerModels.Customer, error)
	FindCustomerByCode(ctx context.Context, customerCode string) (*customerModels.Customer, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) CreateUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return apperror.InternalWrap("create uniq bom failed", err)
	}
	return nil
}

func (r *repository) FindUniqBOMByUUID(ctx context.Context, uuid string) (*models.UniqBillOfMaterial, error) {
	var item models.UniqBillOfMaterial
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("uniq bom not found")
		}
		return nil, apperror.InternalWrap("find uniq bom failed", err)
	}
	return &item, nil
}

func (r *repository) FindUniqBOMByUniqCode(ctx context.Context, uniqCode string) (*models.UniqBillOfMaterial, error) {
	var item models.UniqBillOfMaterial
	err := r.db.WithContext(ctx).Where("uniq_code ILIKE ?", uniqCode).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("uniq bom not found")
		}
		return nil, apperror.InternalWrap("find uniq bom failed", err)
	}
	return &item, nil
}

func (r *repository) ListUniqBOMs(ctx context.Context, filters models.UniqBOMListFilters) ([]models.UniqBillOfMaterial, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.UniqBillOfMaterial{})
	if filters.Search != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where("uniq_code ILIKE ? OR product_model ILIKE ? OR part_name ILIKE ? OR part_number ILIKE ?", search, search, search, search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count uniq boms failed", err)
	}

	var items []models.UniqBillOfMaterial
	err := query.Order("uniq_code ASC").Limit(filters.Limit).Offset(filters.Offset).Find(&items).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list uniq boms failed", err)
	}

	return items, total, nil
}

func (r *repository) UpdateUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return apperror.InternalWrap("update uniq bom failed", err)
	}
	return nil
}

func (r *repository) DeleteUniqBOM(ctx context.Context, item *models.UniqBillOfMaterial) error {
	if err := r.db.WithContext(ctx).Delete(item).Error; err != nil {
		return apperror.InternalWrap("delete uniq bom failed", err)
	}
	return nil
}

func (r *repository) CreatePRLs(ctx context.Context, items []*models.PRL) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE public.prls IN EXCLUSIVE MODE").Error; err != nil {
			return apperror.InternalWrap("lock prls table failed", err)
		}

		nextSequence, err := nextPRLSequence(tx)
		if err != nil {
			return err
		}

		year := time.Now().Year()
		for _, item := range items {
			item.PRLID = fmt.Sprintf("PRL-%d-%03d", year, nextSequence)
			nextSequence++
			if err := tx.Create(item).Error; err != nil {
				//return wrapPRLPersistError("create prl failed", err)
			}
		}

		return nil
	})
}

func (r *repository) FindPRLByUUID(ctx context.Context, uuid string) (*models.PRL, error) {
	var item models.PRL
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("prl not found")
		}
		return nil, apperror.InternalWrap("find prl failed", err)
	}
	return &item, nil
}

func (r *repository) FindPRLByID(ctx context.Context, id int64) (*models.PRL, error) {
	var item models.PRL
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("prl not found")
		}
		return nil, apperror.InternalWrap("find prl failed", err)
	}
	return &item, nil
}

func (r *repository) ListPRLs(ctx context.Context, filters models.PRLListFilters) ([]models.PRL, int64, error) {
	query := r.applyPRLFilters(r.db.WithContext(ctx).Model(&models.PRL{}), filters)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count prls failed", err)
	}

	var items []models.PRL
	err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&items).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list prls failed", err)
	}

	return items, total, nil
}

func (r *repository) ListPRLsForExport(ctx context.Context, filters models.PRLListFilters) ([]models.PRL, error) {
	query := r.applyPRLFilters(r.db.WithContext(ctx).Model(&models.PRL{}), filters)
	var items []models.PRL
	err := query.Order("created_at DESC").Find(&items).Error
	if err != nil {
		return nil, apperror.InternalWrap("list prls for export failed", err)
	}
	return items, nil
}

func (r *repository) ListPRLHistoryVsDelivery(ctx context.Context, filters models.PRLHistoryFilters) ([]models.PRLHistoryListItem, int64, error) {
	total, err := r.countPRLHistoryGroups(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]models.PRLHistoryListItem, 0)
	query := `
		WITH grouped_prl AS (
			SELECT
				p.forecast_period,
				p.uniq_code,
				SUM(p.quantity)::numeric(18,4) AS prl_quantity,
				MAX(p.updated_at) AS prl_last_updated
			FROM prls p
			WHERE p.deleted_at IS NULL
			  AND (? = '' OR p.uniq_code ILIKE '%' || ? || '%')
			  AND (? = '' OR p.forecast_period ILIKE '%' || ? || '%')
			  AND (? = '' OR (p.uniq_code ILIKE '%' || ? || '%' OR p.forecast_period ILIKE '%' || ? || '%'))
			GROUP BY p.forecast_period, p.uniq_code
		)
		SELECT
			gp.forecast_period,
			gp.uniq_code,
			COALESCE(gp.prl_quantity, 0) AS prl_quantity,
			COALESCE((
				SELECT SUM(dsi.total_delivery_qty)::numeric(18,4)
				FROM delivery_schedule_items_customer dsi
				JOIN delivery_schedules_customer dsc ON dsc.id = dsi.schedule_id AND dsc.deleted_at IS NULL
				WHERE lower(trim(dsi.item_uniq_code)) = lower(trim(gp.uniq_code))
				  AND (
					to_char(dsc.schedule_date, 'YYYY-"Q"Q') = gp.forecast_period
					OR to_char(dsc.schedule_date, 'Mon YYYY') = gp.forecast_period
					OR trim(to_char(dsc.schedule_date, 'Month YYYY')) = gp.forecast_period
					OR to_char(dsc.schedule_date, 'YYYY-MM') = gp.forecast_period
					OR (
						gp.forecast_period ~ '^[A-Za-z]{3} [0-9]{4}$'
						AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Mon YYYY'))
					)
					OR (
						gp.forecast_period ~ '^[A-Za-z]+ [0-9]{4}$'
						AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Month YYYY'))
					)
					OR (
						gp.forecast_period ~ '^[0-9]{4}/[0-9]{2}$'
						AND to_char(dsc.schedule_date, 'YYYY/MM') = gp.forecast_period
					)
				  )
			), 0) AS delivery_qty,
			GREATEST(
				gp.prl_last_updated,
				COALESCE((
					SELECT MAX(dsi.updated_at)
					FROM delivery_schedule_items_customer dsi
					JOIN delivery_schedules_customer dsc ON dsc.id = dsi.schedule_id AND dsc.deleted_at IS NULL
					WHERE lower(trim(dsi.item_uniq_code)) = lower(trim(gp.uniq_code))
					  AND (
						to_char(dsc.schedule_date, 'YYYY-"Q"Q') = gp.forecast_period
						OR to_char(dsc.schedule_date, 'Mon YYYY') = gp.forecast_period
						OR trim(to_char(dsc.schedule_date, 'Month YYYY')) = gp.forecast_period
						OR to_char(dsc.schedule_date, 'YYYY-MM') = gp.forecast_period
						OR (
							gp.forecast_period ~ '^[A-Za-z]{3} [0-9]{4}$'
							AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Mon YYYY'))
						)
						OR (
							gp.forecast_period ~ '^[A-Za-z]+ [0-9]{4}$'
							AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Month YYYY'))
						)
						OR (
							gp.forecast_period ~ '^[0-9]{4}/[0-9]{2}$'
							AND to_char(dsc.schedule_date, 'YYYY/MM') = gp.forecast_period
						)
					  )
				), gp.prl_last_updated)
			) AS last_updated
		FROM grouped_prl gp
		ORDER BY last_updated DESC, gp.forecast_period DESC, gp.uniq_code ASC
		LIMIT ? OFFSET ?;
	`

	if err := r.db.WithContext(ctx).Raw(
		query,
		filters.UniqCode, filters.UniqCode,
		filters.ForecastPeriod, filters.ForecastPeriod,
		filters.Search, filters.Search, filters.Search,
		filters.Limit, filters.Offset,
	).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list prl history vs delivery failed", err)
	}

	return rows, total, nil
}

func (r *repository) ListPRLMachinePatterns(ctx context.Context, filters models.PRLHistoryFilters) ([]models.PRLMachinePatternListItem, int64, error) {
	total, err := r.countPRLHistoryGroups(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]models.PRLMachinePatternListItem, 0)
	query := `
		WITH grouped_prl AS (
			SELECT
				p.forecast_period,
				p.uniq_code,
				SUM(p.quantity)::numeric(18,4) AS prl_quantity
			FROM prls p
			WHERE p.deleted_at IS NULL
			  AND (? = '' OR p.uniq_code ILIKE '%' || ? || '%')
			  AND (? = '' OR p.forecast_period ILIKE '%' || ? || '%')
			  AND (? = '' OR (p.uniq_code ILIKE '%' || ? || '%' OR p.forecast_period ILIKE '%' || ? || '%'))
			GROUP BY p.forecast_period, p.uniq_code
		),
		latest_headers AS (
			SELECT DISTINCT ON (rh.item_id)
				rh.item_id,
				rh.id
			FROM routing_headers rh
			ORDER BY rh.item_id, rh.version DESC, rh.id DESC
		),
		machine_patterns AS (
			SELECT
				i.uniq_code,
				string_agg(pp.process_name, ' > ' ORDER BY COALESCE(pp.sequence, 0), ro.op_seq, ro.id) AS machine_pattern
			FROM items i
			JOIN latest_headers lh ON lh.item_id = i.id
			JOIN routing_operations ro ON ro.routing_header_id = lh.id
			JOIN process_parameters pp ON pp.id = ro.process_id
			WHERE i.deleted_at IS NULL
			GROUP BY i.uniq_code
		)
		SELECT
			gp.forecast_period,
			gp.uniq_code,
			COALESCE(mp.machine_pattern, '-') AS machine_pattern,
			COALESCE((
				SELECT SUM(dsi.total_delivery_qty)::numeric(18,4)
				FROM delivery_schedule_items_customer dsi
				JOIN delivery_schedules_customer dsc ON dsc.id = dsi.schedule_id AND dsc.deleted_at IS NULL
				WHERE lower(trim(dsi.item_uniq_code)) = lower(trim(gp.uniq_code))
				  AND (
					to_char(dsc.schedule_date, 'YYYY-"Q"Q') = gp.forecast_period
					OR to_char(dsc.schedule_date, 'Mon YYYY') = gp.forecast_period
					OR trim(to_char(dsc.schedule_date, 'Month YYYY')) = gp.forecast_period
					OR to_char(dsc.schedule_date, 'YYYY-MM') = gp.forecast_period
					OR (
						gp.forecast_period ~ '^[A-Za-z]{3} [0-9]{4}$'
						AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Mon YYYY'))
					)
					OR (
						gp.forecast_period ~ '^[A-Za-z]+ [0-9]{4}$'
						AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(gp.forecast_period, 'Month YYYY'))
					)
					OR (
						gp.forecast_period ~ '^[0-9]{4}/[0-9]{2}$'
						AND to_char(dsc.schedule_date, 'YYYY/MM') = gp.forecast_period
					)
				  )
			), 0) AS production_output
		FROM grouped_prl gp
		LEFT JOIN machine_patterns mp ON mp.uniq_code = gp.uniq_code
		ORDER BY gp.forecast_period DESC, gp.uniq_code ASC
		LIMIT ? OFFSET ?;
	`

	if err := r.db.WithContext(ctx).Raw(
		query,
		filters.UniqCode, filters.UniqCode,
		filters.ForecastPeriod, filters.ForecastPeriod,
		filters.Search, filters.Search, filters.Search,
		filters.Limit, filters.Offset,
	).Scan(&rows).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list prl machine patterns failed", err)
	}

	return rows, total, nil
}

func (r *repository) GetPRLHistorySummary(ctx context.Context, uniqCode, forecastPeriod string) (*models.PRLHistoryDetailSummary, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	forecastPeriod = strings.TrimSpace(forecastPeriod)

	var prlAgg struct {
		Count       int64        `gorm:"column:count"`
		PRLQuantity float64      `gorm:"column:prl_quantity"`
		LastUpdated sql.NullTime `gorm:"column:last_updated"`
	}

	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS count,
			COALESCE(SUM(quantity), 0)::float8 AS prl_quantity,
			MAX(updated_at) AS last_updated
		FROM prls
		WHERE deleted_at IS NULL
		  AND uniq_code ILIKE ?
		  AND forecast_period = ?
	`, uniqCode, forecastPeriod).Scan(&prlAgg).Error; err != nil {
		return nil, apperror.InternalWrap("get prl history summary failed", err)
	}

	if prlAgg.Count == 0 {
		return nil, apperror.NotFound("prl history not found")
	}

	var deliveryAgg struct {
		DeliveryQty float64      `gorm:"column:delivery_qty"`
		LastUpdated sql.NullTime `gorm:"column:last_updated"`
		TotalLogs   int64        `gorm:"column:total_logs"`
	}

	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(dsi.total_delivery_qty), 0)::float8 AS delivery_qty,
			MAX(dsi.updated_at) AS last_updated,
			COUNT(*) AS total_logs
		FROM delivery_schedule_items_customer dsi
		JOIN delivery_schedules_customer dsc ON dsc.id = dsi.schedule_id AND dsc.deleted_at IS NULL
		WHERE lower(trim(dsi.item_uniq_code)) = lower(trim(?))
		  AND (
			to_char(dsc.schedule_date, 'YYYY-"Q"Q') = ?
			OR to_char(dsc.schedule_date, 'Mon YYYY') = ?
			OR trim(to_char(dsc.schedule_date, 'Month YYYY')) = ?
			OR to_char(dsc.schedule_date, 'YYYY-MM') = ?
			OR (
				? ~ '^[A-Za-z]{3} [0-9]{4}$'
				AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(?, 'Mon YYYY'))
			)
			OR (
				? ~ '^[A-Za-z]+ [0-9]{4}$'
				AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(?, 'Month YYYY'))
			)
			OR (
				? ~ '^[0-9]{4}/[0-9]{2}$'
				AND to_char(dsc.schedule_date, 'YYYY/MM') = ?
			)
		  )
	`, uniqCode,
		forecastPeriod, forecastPeriod, forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
	).Scan(&deliveryAgg).Error; err != nil {
		return nil, apperror.InternalWrap("get delivery aggregate failed", err)
	}

	totalLogs := prlAgg.Count + deliveryAgg.TotalLogs
	if totalLogs < 0 {
		totalLogs = 0
	}

	lastUpdated := time.Now().UTC()
	if prlAgg.LastUpdated.Valid {
		lastUpdated = prlAgg.LastUpdated.Time
	}
	if deliveryAgg.LastUpdated.Valid && deliveryAgg.LastUpdated.Time.After(lastUpdated) {
		lastUpdated = deliveryAgg.LastUpdated.Time
	}

	return &models.PRLHistoryDetailSummary{
		UniqCode:       uniqCode,
		ForecastPeriod: forecastPeriod,
		TotalLogs:      totalLogs,
		PRLQuantity:    prlAgg.PRLQuantity,
		DeliveryQty:    deliveryAgg.DeliveryQty,
		LastUpdated:    lastUpdated,
	}, nil
}

func (r *repository) ListPRLHistoryTimeline(ctx context.Context, uniqCode, forecastPeriod string, limit int) ([]models.PRLHistoryLogItem, error) {
	if limit <= 0 {
		limit = 50
	}

	rows := make([]models.PRLHistoryLogItem, 0)
	query := `
		SELECT
			activity,
			description,
			actor,
			event_time,
			source,
			old_value,
			new_value
		FROM (
			SELECT
				CASE WHEN p.updated_at > p.created_at THEN 'Updated PRL' ELSE 'Created PRL Entry' END AS activity,
				CASE
					WHEN p.updated_at > p.created_at THEN 'Updated PRL quantity to ' || p.quantity::text
					ELSE 'Created new PRL entry for ' || p.forecast_period
				END AS description,
				COALESCE(ai.submitted_by, '') AS actor,
				p.updated_at AS event_time,
				'prl' AS source,
				NULL::float8 AS old_value,
				p.quantity::float8 AS new_value
			FROM prls p
			LEFT JOIN approval_instances ai
				ON ai.action_name = 'prl'
				AND ai.reference_table = 'prls'
				AND ai.reference_id = p.id
			WHERE p.deleted_at IS NULL
			  AND p.uniq_code ILIKE ?
			  AND p.forecast_period = ?

			UNION ALL

			SELECT
				CASE WHEN dsi.updated_at > dsi.created_at THEN 'Updated Delivery' ELSE 'Created Delivery Schedule' END AS activity,
				CASE
					WHEN dsi.updated_at > dsi.created_at THEN 'Updated delivery quantity to ' || dsi.total_delivery_qty::text
					ELSE 'Created delivery schedule for period ' || ?
				END AS description,
				COALESCE(dsc.created_by, '') AS actor,
				dsi.updated_at AS event_time,
				'delivery' AS source,
				NULL::float8 AS old_value,
				dsi.total_delivery_qty::float8 AS new_value
			FROM delivery_schedule_items_customer dsi
			JOIN delivery_schedules_customer dsc ON dsc.id = dsi.schedule_id AND dsc.deleted_at IS NULL
			WHERE lower(trim(dsi.item_uniq_code)) = lower(trim(?))
			  AND (
				to_char(dsc.schedule_date, 'YYYY-"Q"Q') = ?
				OR to_char(dsc.schedule_date, 'Mon YYYY') = ?
				OR trim(to_char(dsc.schedule_date, 'Month YYYY')) = ?
				OR to_char(dsc.schedule_date, 'YYYY-MM') = ?
				OR (
					? ~ '^[A-Za-z]{3} [0-9]{4}$'
					AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(?, 'Mon YYYY'))
				)
				OR (
					? ~ '^[A-Za-z]+ [0-9]{4}$'
					AND date_trunc('month', dsc.schedule_date) = date_trunc('month', to_date(?, 'Month YYYY'))
				)
				OR (
					? ~ '^[0-9]{4}/[0-9]{2}$'
					AND to_char(dsc.schedule_date, 'YYYY/MM') = ?
				)
			  )
		) logs
		ORDER BY event_time DESC
		LIMIT ?;
	`

	if err := r.db.WithContext(ctx).Raw(
		query,
		uniqCode, forecastPeriod,
		forecastPeriod,
		uniqCode,
		forecastPeriod, forecastPeriod, forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
		forecastPeriod, forecastPeriod,
		limit,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("list prl history timeline failed", err)
	}

	return rows, nil
}

func (r *repository) GetMachinePatternByUniqCode(ctx context.Context, uniqCode string) (string, error) {
	var result struct {
		MachinePattern string `gorm:"column:machine_pattern"`
	}

	err := r.db.WithContext(ctx).Raw(`
		WITH latest_headers AS (
			SELECT DISTINCT ON (rh.item_id)
				rh.item_id,
				rh.id
			FROM routing_headers rh
			ORDER BY rh.item_id, rh.version DESC, rh.id DESC
		)
		SELECT
			string_agg(pp.process_name, ' > ' ORDER BY COALESCE(pp.sequence, 0), ro.op_seq, ro.id) AS machine_pattern
		FROM items i
		JOIN latest_headers lh ON lh.item_id = i.id
		JOIN routing_operations ro ON ro.routing_header_id = lh.id
		JOIN process_parameters pp ON pp.id = ro.process_id
		WHERE i.deleted_at IS NULL
		  AND i.uniq_code ILIKE ?
		GROUP BY i.uniq_code
		LIMIT 1
	`, uniqCode).Scan(&result).Error
	if err != nil {
		return "", apperror.InternalWrap("get machine pattern failed", err)
	}

	return strings.TrimSpace(result.MachinePattern), nil
}

func (r *repository) UpdatePRL(ctx context.Context, item *models.PRL) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		//return wrapPRLPersistError("update prl failed", err)
	}
	return nil
}

func (r *repository) DeletePRL(ctx context.Context, item *models.PRL) error {
	if err := r.db.WithContext(ctx).Delete(item).Error; err != nil {
		return apperror.InternalWrap("delete prl failed", err)
	}
	return nil
}

func (r *repository) BulkSetStatus(ctx context.Context, ids []string, status string) (int64, error) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":      status,
		"updated_at":  now,
		"approved_at": nil,
		"rejected_at": nil,
	}
	if status == models.PRLStatusApproved {
		updates["approved_at"] = now
	}
	if status == models.PRLStatusRejected {
		updates["rejected_at"] = now
	}

	result := r.db.WithContext(ctx).Model(&models.PRL{}).Where("uuid IN ?", ids).Updates(updates)
	if result.Error != nil {
		return 0, apperror.InternalWrap("update prl status failed", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *repository) ListCustomers(ctx context.Context, search string) ([]models.CustomerLookup, error) {
	query := r.db.WithContext(ctx).Model(&customerModels.Customer{})
	if search != "" {
		term := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("customer_id ILIKE ? OR customer_name ILIKE ?", term, term)
	}

	var rows []struct {
		UUID         string
		CustomerID   string
		CustomerName string
	}
	err := query.Select("uuid, customer_id, customer_name").Order("customer_name ASC").Limit(100).Find(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("list customers failed", err)
	}

	items := make([]models.CustomerLookup, 0, len(rows))
	for _, row := range rows {
		items = append(items, models.CustomerLookup{ID: row.UUID, CustomerID: row.CustomerID, CustomerName: row.CustomerName})
	}
	return items, nil
}

func (r *repository) FindCustomerByUUID(ctx context.Context, uuid string) (*customerModels.Customer, error) {
	var item customerModels.Customer
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, apperror.InternalWrap("find customer failed", err)
	}
	return &item, nil
}

func (r *repository) FindCustomerByRowID(ctx context.Context, id int64) (*customerModels.Customer, error) {
	var item customerModels.Customer
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, apperror.InternalWrap("find customer failed", err)
	}
	return &item, nil
}

func (r *repository) FindCustomerByCode(ctx context.Context, customerCode string) (*customerModels.Customer, error) {
	var item customerModels.Customer
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerCode).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, apperror.InternalWrap("find customer failed", err)
	}
	return &item, nil
}

func (r *repository) applyPRLFilters(query *gorm.DB, filters models.PRLListFilters) *gorm.DB {
	if filters.Search != "" {
		term := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where("prl_id ILIKE ? OR customer_name ILIKE ? OR customer_code ILIKE ? OR uniq_code ILIKE ? OR product_model ILIKE ? OR part_name ILIKE ? OR part_number ILIKE ?", term, term, term, term, term, term, term)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.ForecastPeriod != nil {
		query = query.Where("forecast_period = ?", *filters.ForecastPeriod)
	}
	if filters.CustomerUUID != nil {
		query = query.Where("customer_uuid = ?", *filters.CustomerUUID)
	}
	if filters.UniqCode != nil {
		query = query.Where("uniq_code ILIKE ?", *filters.UniqCode)
	}
	return query
}

func (r *repository) countPRLHistoryGroups(ctx context.Context, filters models.PRLHistoryFilters) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM (
			SELECT 1
			FROM prls p
			WHERE p.deleted_at IS NULL
			  AND (? = '' OR p.uniq_code ILIKE '%' || ? || '%')
			  AND (? = '' OR p.forecast_period ILIKE '%' || ? || '%')
			  AND (? = '' OR (p.uniq_code ILIKE '%' || ? || '%' OR p.forecast_period ILIKE '%' || ? || '%'))
			GROUP BY p.forecast_period, p.uniq_code
		) grouped
	`,
		filters.UniqCode, filters.UniqCode,
		filters.ForecastPeriod, filters.ForecastPeriod,
		filters.Search, filters.Search, filters.Search,
	).Scan(&total).Error
	if err != nil {
		return 0, apperror.InternalWrap("count prl history groups failed", err)
	}
	return total, nil
}

func nextPRLSequence(tx *gorm.DB) (int, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("PRL-%d-", year)

	var latestRecord struct{ PRLID string }
	err := tx.Unscoped().Model(&models.PRL{}).
		Select("prl_id").
		Where("prl_id LIKE ?", prefix+"%").
		Order("prl_id DESC").
		Limit(1).
		Take(&latestRecord).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, apperror.InternalWrap("get latest prl id failed", err)
	}

	sequence := 1
	if err == nil && latestRecord.PRLID != "" {
		parts := strings.Split(latestRecord.PRLID, "-")
		if len(parts) == 3 {
			lastNumber, convErr := strconv.Atoi(parts[2])
			if convErr != nil {
				return 0, apperror.InternalWrap("parse latest prl id failed", convErr)
			}
			sequence = lastNumber + 1
		}
	}

	return sequence, nil
}
