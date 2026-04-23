package repository

import (
	"context"

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

	// WIP ITEMS
	FindItemsByWIP(ctx context.Context, wipID int64) ([]models.WIPItem, error)
	FindItemByID(ctx context.Context, id int64) (*models.WIPItem, error)
	InsertWIPItem(ctx context.Context, item models.WIPItem) (*models.WIPItem, error)

	// SCAN
	UpdateItemScan(ctx context.Context, id int64, data models.UpdateWIPItemScan) error

	// LOG
	CreateLog(ctx context.Context, req models.CreateWIPLogRequest) (*models.WIPLog, error)
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
		Joins("JOIN kanban_parameters ON wip_items.uniq = kanban_parameters.item_uniq_code").
		Count(&total).Error

	if err != nil {
		return nil, 0, err
	}

	// ambil data
	err = db.
		Table("wips").
		Select(`
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
		Joins("JOIN kanban_parameters ON wip_items.uniq = kanban_parameters.item_uniq_code").
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
		Status: "open",
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
