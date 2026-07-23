package repository

import (
	"context"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/wip/models"
	"gorm.io/gorm"
)

type IWIPRepository interface {
	// TRANSACTION
	BeginTx(ctx context.Context) IWIPRepository
	Commit() error
	Rollback() error

	// WIP
	FindAllWIPPaginated(ctx context.Context, page, limit int) ([]models.WIPListResponse, int64, error)
	FindWIPByID(ctx context.Context, id int64) (*models.WIPDetailResponse, error)
	CreateWIP(ctx context.Context, req models.CreateWIPRequest) (*models.WIP, error)
	UpdateWIP(ctx context.Context, id int64, req models.UpdateWIPRequest) (*models.WIP, error)
	DeleteWIP(ctx context.Context, id int64) error
	GetWOIDByWIPID(ctx context.Context, wipID int64) (*models.WIP, error)
	GetWO(ctx context.Context, id int64) (string, error)

	// WIP ITEMS
	FindItemsByWIP(ctx context.Context, wipID int64) ([]models.WIPItem, error)
	FindItemByID(ctx context.Context, id int64) (*models.WIPItem, error)
	InsertWIPItem(ctx context.Context, item models.WIPItem) (*models.WIPItem, error)

	// SCAN
	UpdateItemScan(ctx context.Context, id int64, data models.UpdateWIPItemScan) error

	// LOG
	CreateLog(ctx context.Context, req models.CreateWIPLogRequest) (*models.WIPLog, error)

	// INSERT FINISH GOOD
	InsertFinishedGoods(ctx context.Context, fg models.FinishedGoods) error

	// GET TABLE
	GetItemInfo(ctx context.Context, uniq string) (*ItemInfo, error)
	IsParentItem(ctx context.Context, itemID int) (bool, error)

	// WIP HISTORY (MOVEMENT LOG)
	ListWIPMovementLogs(ctx context.Context, uniqCode string, page, limit int) ([]models.WIPMovementLogItem, int64, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IWIPRepository {
	return &repository{db: db}
}

func (r *repository) BeginTx(ctx context.Context) IWIPRepository {
	return &repository{
		db: r.db.WithContext(ctx).Begin(),
	}
}

func (r *repository) Commit() error {
	return r.db.Commit().Error
}

func (r *repository) Rollback() error {
	return r.db.Rollback().Error
}

func (r *repository) FindAllWIPPaginated(ctx context.Context, page, limit int) ([]models.WIPListResponse, int64, error) {
	var data []models.WIPListResponse
	var total int64

	db := r.db.WithContext(ctx)

	// count total (pakai join juga biar konsisten)
	err := db.
		Table("wips").
		Joins("JOIN wip_items ON wips.id = wip_items.wip_id").
		Joins("JOIN work_orders ON wips.wo_id = work_orders.id").
		Joins("JOIN items ON wip_items.uniq = items.uniq_code").
		Joins("LEFT JOIN kanban_parameters ON wip_items.uniq = kanban_parameters.item_uniq_code").
		Where("wip_items.status <> ?", "done").
		Count(&total).Error

	if err != nil {
		return nil, 0, err
	}

	// ambil data
	err = db.
		Table("wips").
		Select(`
			wips.id AS id,
			wip_items.process_name AS process,
			wip_items.uniq AS uniq,
			items.part_number AS part_number,
			items.part_name AS part_info,
			work_orders.wo_number AS wo_number,
			wip_items.stock AS stock,
			wip_items.packing_number AS kanban_number,
			wip_items.wip_type AS type,
			kanban_parameters.kanban_qty AS stock_to_complete_kanban,
			kanban_parameters.kanban_qty AS kanban
		`).
		Joins("JOIN wip_items ON wips.id = wip_items.wip_id").
		Joins("JOIN work_orders ON wips.wo_id = work_orders.id").
		Joins("JOIN items ON wip_items.uniq = items.uniq_code").
		Joins("LEFT JOIN kanban_parameters ON wip_items.uniq = kanban_parameters.item_uniq_code").
		Where("wip_items.status <> ?", "done").
		Order("wips.created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&data).Error

	if err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

func (r *repository) FindWIPByID(ctx context.Context, id int64) (*models.WIPDetailResponse, error) {
	var rows []struct {
		ID         int64
		WONumber   string
		Uniq       string
		PartNumber string
		PartName   string
		Process    string
		Stock      int
	}

	err := r.db.WithContext(ctx).
		Table("wips").
		Select(`
			wips.id,
			work_orders.wo_number,
			wip_items.uniq,
			items.part_number,
			items.part_name,
			wip_items.process_name AS process,
			wip_items.stock
		`).
		Joins("JOIN wip_items ON wips.id = wip_items.wip_id").
		Joins("JOIN work_orders ON wips.wo_id = work_orders.id").
		Joins("JOIN items ON wip_items.uniq = items.uniq_code").
		Where("wips.id = ?", id).
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// build response
	res := &models.WIPDetailResponse{
		ID:         rows[0].ID,
		WONumber:   rows[0].WONumber,
		Uniq:       rows[0].Uniq,
		PartNumber: rows[0].PartNumber,
		PartName:   rows[0].PartName,
		Processes:  []models.WIPProcess{},
	}

	for _, row := range rows {
		res.Processes = append(res.Processes, models.WIPProcess{
			Process: row.Process,
			Stock:   row.Stock,
		})
	}

	return res, nil
}

func (r *repository) CreateWIP(ctx context.Context, req models.CreateWIPRequest) (*models.WIP, error) {
	data := models.WIP{
		WoID:   req.WoID,
		Status: "active",
	}

	err := r.db.WithContext(ctx).Create(&data).Error
	return &data, err
}

func (r *repository) UpdateWIP(ctx context.Context, id int64, req models.UpdateWIPRequest) (*models.WIP, error) {
	var data models.WIP

	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		return nil, err
	}

	if req.Status != "" {
		data.Status = req.Status
	}

	err := r.db.WithContext(ctx).Save(&data).Error
	return &data, err
}

func (r *repository) DeleteWIP(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Delete(&models.WIP{}, id).Error
}

func (r *repository) GetWOIDByWIPID(ctx context.Context, wipID int64) (*models.WIP, error) {
	var data models.WIP

	err := r.db.WithContext(ctx).
		First(&data, wipID).Error

	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *repository) GetWO(ctx context.Context, id int64) (string, error) {

	var item struct {
		WONumber string `gorm:"column:wo_number"`
	}

	err := r.db.WithContext(ctx).
		Table("work_orders").
		Select("wo_number").
		Where("id = ?", id).
		Limit(1).
		Scan(&item).Error

	if err != nil {
		return "", err
	}

	return item.WONumber, nil
}

func (r *repository) FindItemsByWIP(ctx context.Context, wipID int64) ([]models.WIPItem, error) {
	var data []models.WIPItem

	err := r.db.WithContext(ctx).
		Where("wip_id = ?", wipID).
		Order("op_seq ASC").
		Find(&data).Error

	return data, err
}

func (r *repository) FindItemByID(ctx context.Context, id int64) (*models.WIPItem, error) {
	var data models.WIPItem

	err := r.db.WithContext(ctx).
		First(&data, id).Error

	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) InsertWIPItem(ctx context.Context, item models.WIPItem) (*models.WIPItem, error) {
	err := r.db.WithContext(ctx).Create(&item).Error
	return &item, err
}

func (r *repository) UpdateItemScan(ctx context.Context, id int64, data models.UpdateWIPItemScan) error {
	return r.db.WithContext(ctx).
		Model(&models.WIPItem{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"qty_in":        data.QtyIn,
			"qty_out":       data.QtyOut,
			"qty_remaining": data.QtyRemaining,
			"status":        data.Status,
		}).Error
}

func (r *repository) CreateLog(ctx context.Context, req models.CreateWIPLogRequest) (*models.WIPLog, error) {
	data := models.WIPLog{
		WipItemID: req.WipItemID,
		Action:    req.Action,
		Qty:       req.Qty,
	}

	err := r.db.WithContext(ctx).Create(&data).Error
	return &data, err
}

func (r *repository) InsertFinishedGoods(ctx context.Context, fg models.FinishedGoods) error {
	return r.db.WithContext(ctx).
		Create(&fg).Error
}

type ItemInfo struct {
	ID         int    `gorm:"column:id"`
	PartNumber string `gorm:"column:part_number"`
	PartName   string `gorm:"column:part_name"`
	Model      string `gorm:"column:model"`
}

func (r *repository) GetItemInfo(ctx context.Context, uniq string) (*ItemInfo, error) {

	var item ItemInfo

	err := r.db.WithContext(ctx).
		Table("items").
		Select("id", "part_number", "part_name", "model").
		Where("uniq_code = ?", uniq).
		Limit(1).
		Scan(&item).Error

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *repository) IsParentItem(ctx context.Context, itemID int) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("bom_lines").
		Where("parent_item_id = ?", itemID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *repository) ListWIPMovementLogs(ctx context.Context, uniqCode string, page, limit int) ([]models.WIPMovementLogItem, int64, error) {
	var rows []struct {
		ID           int64
		UniqCode     string
		MovementType string
		QtyChange    float64
		SourceFlag   string
		DNNumber     string
		ReferenceID  string
		Notes        string
		LoggedBy     string
		LoggedAt     time.Time
	}

	var total int64

	countDB := r.db.WithContext(ctx).
		Table("inventory_movement_logs").
		Where("movement_category = ?", "wip")
	if uniqCode != "" {
		countDB = countDB.Where("uniq_code = ?", uniqCode)
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := r.db.WithContext(ctx).
		Table("inventory_movement_logs").
		Select(`
			id,
			uniq_code,
			movement_type,
			qty_change,
			source_flag,
			dn_number,
			reference_id,
			notes,
			logged_by,
			logged_at
		`).
		Where("movement_category = ?", "wip")
	if uniqCode != "" {
		q = q.Where("uniq_code = ?", uniqCode)
	}
	q = q.Order("logged_at DESC, id DESC")
	if limit > 0 {
		q = q.Offset((page - 1) * limit).Limit(limit)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]models.WIPMovementLogItem, 0, len(rows))
	for _, row := range rows {
		qty := row.QtyChange
		if strings.EqualFold(row.MovementType, "outgoing") && qty > 0 {
			qty = -qty
		}
		items = append(items, models.WIPMovementLogItem{
			ID:           row.ID,
			UniqCode:     row.UniqCode,
			MovementType: row.MovementType,
			Reason:       row.SourceFlag,
			QtyChange:    qty,
			WONumber:     row.ReferenceID,
			DNNumber:     row.DNNumber,
			ReferenceID:  row.ReferenceID,
			Notes:        row.Notes,
			LoggedBy:     row.LoggedBy,
			LoggedAt:     row.LoggedAt,
		})
	}

	return items, total, nil
}
