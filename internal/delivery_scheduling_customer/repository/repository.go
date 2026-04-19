package repository

import (
	"context"
	"math"

	"github.com/ganasa18/go-template/internal/delivery_scheduling_customer/models"
	"gorm.io/gorm"
)

// ─── Interface ────────────────────────────────────────────────────────────────

type IRepository interface {
	// Schedule number sequence
	FindLastScheduleNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)
	FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error)

	// Schedule CRUD
	CreateSchedule(ctx context.Context, tx *gorm.DB, s *models.ScheduleCustomer) error
	CreateScheduleItems(ctx context.Context, tx *gorm.DB, items []models.ScheduleItemCustomer) error
	GetScheduleByUUID(ctx context.Context, uuid string) (*models.ScheduleCustomer, error)
	GetScheduleByNumber(ctx context.Context, scheduleNumber string) (*models.ScheduleCustomer, error)
	GetScheduleByID(ctx context.Context, id int64) (*models.ScheduleCustomer, error)
	GetScheduleItemsByScheduleID(ctx context.Context, scheduleID int64) ([]models.ScheduleItemCustomer, error)
	UpdateScheduleStatus(ctx context.Context, tx *gorm.DB, scheduleID int64, status, approvalStatus string) error
	UpdateScheduleItemDNNumber(ctx context.Context, tx *gorm.DB, itemID int64, dnNumber, status string) error
	GetSchedulesSummary(ctx context.Context, deliveryDate string) (map[string]int, error)
	GetSchedulesList(ctx context.Context, f models.ScheduleListFilter) ([]models.ScheduleCustomer, int64, error)
	GetSchedulesByDateAndCustomer(ctx context.Context, deliveryDate string, customerID int64) ([]models.ScheduleCustomer, error)
	GetSchedulesByUUIDs(ctx context.Context, uuids []string) ([]models.ScheduleCustomer, error)

	// DN CRUD
	CreateDN(ctx context.Context, tx *gorm.DB, dn *models.DNCustomer) error
	CreateDNItems(ctx context.Context, tx *gorm.DB, items []models.DNItemCustomer) error
	UpdateDNItemQR(ctx context.Context, tx *gorm.DB, itemID int64, qr string) error
	GetDNByUUID(ctx context.Context, uuid string) (*models.DNCustomer, error)
	GetDNList(ctx context.Context, f models.DNListFilter) ([]models.DNCustomer, int64, error)
	GetDNItemsByDNID(ctx context.Context, dnID int64) ([]models.DNItemCustomer, error)
	GetDNItemByDNNumberAndUniqCode(ctx context.Context, dnNumber, uniqCode string) (*models.DNItemCustomer, error)
	UpdateDNStatus(ctx context.Context, tx *gorm.DB, dnID int64, status string) error
	UpdateDNItemShipment(ctx context.Context, tx *gorm.DB, itemID int64, qtyShipped float64, status string) error

	// FG
	GetFGStockQty(ctx context.Context, uniqCode string) (float64, error)
	DeductFGStock(ctx context.Context, tx *gorm.DB, uniqCode string, qty float64, dnNumber, deductedBy string) error

	// Idempotency
	IdempotencyKeyExists(ctx context.Context, key string) (bool, error)
	CreateDNLog(ctx context.Context, tx *gorm.DB, log *models.DNLogCustomer) error

	// DN aggregate status
	GetDNItemStatuses(ctx context.Context, dnID int64) ([]string, error)
}

// ─── Implementation ───────────────────────────────────────────────────────────

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) FindLastScheduleNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error) {
	db := r.db
	if tx != nil {
		db = tx
	}
	var last string
	err := db.WithContext(ctx).
		Model(&models.ScheduleCustomer{}).
		Where("schedule_number LIKE ?", prefix+"%").
		Order("schedule_number DESC").
		Limit(1).
		Pluck("schedule_number", &last).Error
	return last, err
}

func (r *repository) FindLastDNNumber(ctx context.Context, tx *gorm.DB, prefix string) (string, error) {
	db := r.db
	if tx != nil {
		db = tx
	}
	var last string
	err := db.WithContext(ctx).
		Model(&models.DNCustomer{}).
		Where("dn_number LIKE ?", prefix+"%").
		Order("dn_number DESC").
		Limit(1).
		Pluck("dn_number", &last).Error
	return last, err
}

func (r *repository) CreateSchedule(ctx context.Context, tx *gorm.DB, s *models.ScheduleCustomer) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(s).Error
}

func (r *repository) CreateScheduleItems(ctx context.Context, tx *gorm.DB, items []models.ScheduleItemCustomer) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(&items).Error
}

func (r *repository) GetScheduleByUUID(ctx context.Context, uuid string) (*models.ScheduleCustomer, error) {
	var s models.ScheduleCustomer
	err := r.db.WithContext(ctx).
		Where("uuid = ? AND deleted_at IS NULL", uuid).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) GetScheduleByNumber(ctx context.Context, scheduleNumber string) (*models.ScheduleCustomer, error) {
	var s models.ScheduleCustomer
	err := r.db.WithContext(ctx).
		Where("schedule_number = ? AND deleted_at IS NULL", scheduleNumber).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) GetScheduleByID(ctx context.Context, id int64) (*models.ScheduleCustomer, error) {
	var s models.ScheduleCustomer
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) GetScheduleItemsByScheduleID(ctx context.Context, scheduleID int64) ([]models.ScheduleItemCustomer, error) {
	var items []models.ScheduleItemCustomer
	err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Order("line_no ASC").
		Find(&items).Error
	return items, err
}

func (r *repository) UpdateScheduleStatus(ctx context.Context, tx *gorm.DB, scheduleID int64, status, approvalStatus string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Model(&models.ScheduleCustomer{}).
		Where("id = ?", scheduleID).
		Updates(map[string]interface{}{
			"status":          status,
			"approval_status": approvalStatus,
		}).Error
}

func (r *repository) UpdateScheduleItemDNNumber(ctx context.Context, tx *gorm.DB, itemID int64, dnNumber, status string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Model(&models.ScheduleItemCustomer{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"dn_number": dnNumber,
			"status":    status,
		}).Error
}

func (r *repository) GetSchedulesSummary(ctx context.Context, deliveryDate string) (map[string]int, error) {
	type row struct {
		Status string
		Count  int
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Model(&models.ScheduleCustomer{}).
		Select("status, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("status")

	if deliveryDate != "" {
		q = q.Where("schedule_date::date = ?", deliveryDate)
	}

	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := map[string]int{}
	for _, r := range rows {
		result[r.Status] = r.Count
	}
	return result, nil
}

func (r *repository) GetSchedulesList(ctx context.Context, f models.ScheduleListFilter) ([]models.ScheduleCustomer, int64, error) {
	var schedules []models.ScheduleCustomer
	var total int64

	q := r.db.WithContext(ctx).Model(&models.ScheduleCustomer{}).Where("deleted_at IS NULL")

	if f.DeliveryDate != "" {
		q = q.Where("schedule_date::date = ?", f.DeliveryDate)
	}
	if f.CustomerID > 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.Limit
	err := q.Preload("Items").
		Order("schedule_date ASC, schedule_number ASC").
		Offset(offset).Limit(f.Limit).
		Find(&schedules).Error

	return schedules, total, err
}

func (r *repository) GetSchedulesByDateAndCustomer(ctx context.Context, deliveryDate string, customerID int64) ([]models.ScheduleCustomer, error) {
	var schedules []models.ScheduleCustomer
	q := r.db.WithContext(ctx).
		Model(&models.ScheduleCustomer{}).
		Where("deleted_at IS NULL AND schedule_date::date = ?", deliveryDate).
		Where("status NOT IN ('approved','dn_created','cancelled')")

	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}

	err := q.Preload("Items").Find(&schedules).Error
	return schedules, err
}

func (r *repository) GetSchedulesByUUIDs(ctx context.Context, scheduleNumbers []string) ([]models.ScheduleCustomer, error) {
	var schedules []models.ScheduleCustomer
	err := r.db.WithContext(ctx).
		Where("schedule_number IN ? AND deleted_at IS NULL", scheduleNumbers).
		Where("status NOT IN ('approved','dn_created','cancelled')").
		Preload("Items").
		Find(&schedules).Error
	return schedules, err
}

func (r *repository) CreateDN(ctx context.Context, tx *gorm.DB, dn *models.DNCustomer) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(dn).Error
}

func (r *repository) CreateDNItems(ctx context.Context, tx *gorm.DB, items []models.DNItemCustomer) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(&items).Error
}

func (r *repository) UpdateDNItemQR(ctx context.Context, tx *gorm.DB, itemID int64, qr string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Model(&models.DNItemCustomer{}).
		Where("id = ?", itemID).
		Update("qr", qr).Error
}

func (r *repository) GetDNByUUID(ctx context.Context, uuid string) (*models.DNCustomer, error) {
	var dn models.DNCustomer
	err := r.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		First(&dn).Error
	if err != nil {
		return nil, err
	}
	return &dn, nil
}

func (r *repository) GetDNList(ctx context.Context, f models.DNListFilter) ([]models.DNCustomer, int64, error) {
	var dns []models.DNCustomer
	var total int64

	q := r.db.WithContext(ctx).Model(&models.DNCustomer{})

	if f.DeliveryDate != "" {
		q = q.Where("delivery_date::date = ?", f.DeliveryDate)
	}
	if f.CustomerID > 0 {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.Limit
	err := q.Preload("Items").
		Order("delivery_date ASC, dn_number ASC").
		Offset(offset).Limit(f.Limit).
		Find(&dns).Error

	return dns, total, err
}

func (r *repository) GetDNItemsByDNID(ctx context.Context, dnID int64) ([]models.DNItemCustomer, error) {
	var items []models.DNItemCustomer
	err := r.db.WithContext(ctx).
		Where("dn_id = ?", dnID).
		Order("line_no ASC").
		Find(&items).Error
	return items, err
}

func (r *repository) GetDNItemByDNNumberAndUniqCode(ctx context.Context, dnNumber, uniqCode string) (*models.DNItemCustomer, error) {
	var item models.DNItemCustomer
	err := r.db.WithContext(ctx).
		Joins("JOIN delivery_notes_customer dn ON dn.id = delivery_note_items_customer.dn_id").
		Where("dn.dn_number = ? AND delivery_note_items_customer.item_uniq_code = ?", dnNumber, uniqCode).
		Where("delivery_note_items_customer.shipment_status NOT IN ('shipped','cancelled')").
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) UpdateDNStatus(ctx context.Context, tx *gorm.DB, dnID int64, status string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Model(&models.DNCustomer{}).
		Where("id = ?", dnID).
		Update("status", status).Error
}

func (r *repository) UpdateDNItemShipment(ctx context.Context, tx *gorm.DB, itemID int64, qtyShipped float64, status string) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Model(&models.DNItemCustomer{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"qty_shipped":      qtyShipped,
			"shipment_status":  status,
		}).Error
}

func (r *repository) GetFGStockQty(ctx context.Context, uniqCode string) (float64, error) {
	var qty float64
	err := r.db.WithContext(ctx).
		Table("finished_goods").
		Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).
		Pluck("stock_qty", &qty).Error
	return qty, err
}

func (r *repository) DeductFGStock(ctx context.Context, tx *gorm.DB, uniqCode string, qty float64, dnNumber, deductedBy string) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	// Deduct stock on finished_goods
	if err := db.WithContext(ctx).
		Exec(`UPDATE finished_goods SET stock_qty = stock_qty - ?, updated_at = NOW()
			  WHERE uniq_code = ? AND deleted_at IS NULL AND stock_qty >= ?`,
			qty, uniqCode, qty).Error; err != nil {
		return err
	}

	// Append movement log
	return db.WithContext(ctx).
		Exec(`INSERT INTO fg_movement_logs
			  (fg_id, uniq_code, movement_type, qty_change, qty_before, qty_after, source_flag, dn_number, notes, logged_by, logged_at)
			  SELECT id, uniq_code, 'delivery_scan', ?, stock_qty + ?, stock_qty, 'delivery', ?, 'outbound delivery scan', ?, NOW()
			  FROM finished_goods
			  WHERE uniq_code = ? AND deleted_at IS NULL`,
			-qty, qty, dnNumber, deductedBy, uniqCode).Error
}

func (r *repository) IdempotencyKeyExists(ctx context.Context, key string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DNLogCustomer{}).
		Where("idempotency_key = ?", key).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) CreateDNLog(ctx context.Context, tx *gorm.DB, log *models.DNLogCustomer) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(log).Error
}

func (r *repository) GetDNItemStatuses(ctx context.Context, dnID int64) ([]string, error) {
	var statuses []string
	err := r.db.WithContext(ctx).
		Model(&models.DNItemCustomer{}).
		Where("dn_id = ?", dnID).
		Pluck("shipment_status", &statuses).Error
	return statuses, err
}

// ─── Pagination helper ────────────────────────────────────────────────────────

func BuildPagination(total int64, page, limit int) models.Pagination {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return models.Pagination{
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}
