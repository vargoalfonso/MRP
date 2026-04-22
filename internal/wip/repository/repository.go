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
	FindAllWIPPaginated(ctx context.Context, page, limit int) ([]models.WIP, int64, error)
	FindWIPByID(ctx context.Context, id int64) (*models.WIP, error)
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

func (r *repository) FindAllWIPPaginated(ctx context.Context, page, limit int) ([]models.WIP, int64, error) {
	var data []models.WIP
	var total int64

	db := r.db.WithContext(ctx)

	db.Model(&models.WIP{}).Count(&total)

	err := db.
		Offset((page - 1) * limit).
		Limit(limit).
		Order("created_at DESC").
		Find(&data).Error

	return data, total, err
}

func (r *repository) FindWIPByID(ctx context.Context, id int64) (*models.WIP, error) {
	var data models.WIP

	err := r.db.WithContext(ctx).
		First(&data, id).Error

	if err != nil {
		return nil, err
	}

	return &data, nil
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
