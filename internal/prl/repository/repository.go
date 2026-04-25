package repository

import (
	"context"
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
	err := r.db.WithContext(ctx).Where("uniq_code = ?", uniqCode).First(&item).Error
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
		query = query.Where("uniq_code = ?", *filters.UniqCode)
	}
	return query
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
