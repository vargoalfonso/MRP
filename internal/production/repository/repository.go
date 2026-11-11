package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/production/models"
	"gorm.io/gorm"
)

type IProductionRepository interface {
	Create(ctx context.Context, data *models.ProductionScanLog) error
	BulkCreate(ctx context.Context, data []models.ProductionScanLog) error

	GetByWO(ctx context.Context, woID int64) ([]models.ProductionScanLog, error)
	GetLastByKanban(ctx context.Context, kanban string) (*models.ProductionScanLog, error)

	GetWOItemDetail(ctx context.Context, kanban string) (*models.WOItemDetail, error)
}

type productionRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IProductionRepository {
	return &productionRepository{db}
}

func (r *productionRepository) Create(ctx context.Context, data *models.ProductionScanLog) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *productionRepository) BulkCreate(ctx context.Context, data []models.ProductionScanLog) error {
	return r.db.WithContext(ctx).Create(&data).Error
}

func (r *productionRepository) GetByWO(ctx context.Context, woID int64) ([]models.ProductionScanLog, error) {
	var result []models.ProductionScanLog

	err := r.db.WithContext(ctx).
		Where("wo_id = ?", woID).
		Order("scanned_at asc").
		Find(&result).Error

	return result, err
}

func (r *productionRepository) GetLastByKanban(ctx context.Context, kanban string) (*models.ProductionScanLog, error) {
	var result models.ProductionScanLog

	err := r.db.WithContext(ctx).
		Where("kanban_number = ?", kanban).
		Order("scanned_at desc").
		First(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *productionRepository) GetWOItemDetail(ctx context.Context, kanban string) (*models.WOItemDetail, error) {

	var result models.WOItemDetail

	err := r.db.WithContext(ctx).
		Table("work_order_items wi").
		Select(`
			wo.id as wo_id,
			wo.wo_number as wo_number,
			wi.process_name,
			m.production_line as production_line,
			m.machine_number as machine_number,
		 wi.kanban_number as packing_number,
			wi.kanban_param_number as kanban_number,
			wi.part_name as product_name,
			wi.quantity AS qty_plan,
			wi.uom as unit
		`).
		Joins("JOIN work_orders wo ON wo.id = wi.wo_id").
		Joins("LEFT JOIN master_machines m ON m.id = wi.machine_id").
		Where("wi.kanban_number = ?", kanban).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}
