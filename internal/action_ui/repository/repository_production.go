package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IProductionRepository interface {
	FindWOItemByUuid(ctx context.Context, uuid string) (models.WorkOrderItem, error)
	FindWOByID(ctx context.Context, id int64) (models.WorkOrder, error)
	FindMachineByID(ctx context.Context, id int) (models.MasterMachine, error)
	FindWOByNumber(ctx context.Context, woNumber string) (models.WorkOrder, error)
	FindWOByKanbanNumber(ctx context.Context, woNumber string) (models.WorkOrderItem, error)
	FindWOItemsByWOID(ctx context.Context, woid int64) ([]models.WorkOrderItem, error)
	FindWOItemByUniq(ctx context.Context, uniq string) (models.WorkOrderItem, error)
	FindWOItemByUniqAndWO(ctx context.Context, uniq string, woID int64) (models.WorkOrderItem, error)

	UpdateWOItem(ctx context.Context, item models.WorkOrderItem) error

	InsertScanLog(ctx context.Context, log models.ProductionScanLog) error
	InsertRawMaterial(ctx context.Context, rm models.RawMaterialLog) error
	InsertQCLog(ctx context.Context, qc models.QCLog) error
	InsertFinishedGoods(ctx context.Context, fg models.FinishedGoods) error

	IsQCPendingExist(ctx context.Context, woItemID int64, process string) (bool, error)
	CreateQC(ctx context.Context, qc *models.QCTask) error
	InsertProductIssue(ctx context.Context, data models.ProductionIssue) error
}

type productionRepo struct {
	db *gorm.DB
}

func NewProductionRepository(db *gorm.DB) IProductionRepository {
	return &productionRepo{db: db}
}

//
// ==============================
// 🔍 FIND DATA
// ==============================
//

func (r *productionRepo) InsertProductIssue(ctx context.Context, data models.ProductionIssue) error {
	return r.db.WithContext(ctx).Create(&data).Error
}

func (r *productionRepo) FindWOItemByUniqAndWO(ctx context.Context, uniq string, woID int64) (models.WorkOrderItem, error) {
	var item models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ? AND wo_id = ?", uniq, woID).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, apperror.NotFound("uniq not found in wo")
		}
		return item, err
	}

	return item, nil
}

func (r *productionRepo) FindWOItemByUuid(ctx context.Context, uuid string) (models.WorkOrderItem, error) {
	var item models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, apperror.NotFound("uniq not found")
		}
		return item, err
	}

	return item, nil
}

func (r *productionRepo) FindWOItemByUniq(ctx context.Context, uniq string) (models.WorkOrderItem, error) {
	var item models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("item_uniq_code = ?", uniq).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, apperror.NotFound("uniq not found")
		}
		return item, err
	}

	return item, nil
}

func (r *productionRepo) FindWOByID(ctx context.Context, id int64) (models.WorkOrder, error) {
	var wo models.WorkOrder

	err := r.db.WithContext(ctx).
		First(&wo, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wo, apperror.NotFound("wo not found")
		}
		return wo, err
	}

	return wo, nil
}

func (r *productionRepo) FindWOByNumber(ctx context.Context, woNumber string) (models.WorkOrder, error) {
	var wo models.WorkOrder

	err := r.db.WithContext(ctx).
		Where("wo_number = ?", woNumber).
		First(&wo).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wo, apperror.NotFound("wo not found")
		}
		return wo, err
	}

	return wo, nil
}

func (r *productionRepo) FindWOByKanbanNumber(ctx context.Context, woNumber string) (models.WorkOrderItem, error) {
	var wo models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("kanban_number LIKE ?", "%"+woNumber+"%").
		First(&wo).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wo, apperror.NotFound("wo not found")
		}
		return wo, err
	}

	return wo, nil
}

func (r *productionRepo) FindWOItemsByWOID(ctx context.Context, woid int64) ([]models.WorkOrderItem, error) {
	var items []models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("wo_id = ?", woid).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, apperror.NotFound("wo items not found")
	}

	return items, nil
}

func (r *productionRepo) FindMachineByID(ctx context.Context, id int) (models.MasterMachine, error) {
	var machine models.MasterMachine

	err := r.db.WithContext(ctx).
		First(&machine, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return machine, apperror.NotFound("machine not found")
		}
		return machine, err
	}

	return machine, nil
}

//
// ==============================
// 📝 UPDATE
// ==============================
//

func (r *productionRepo) UpdateWOItem(ctx context.Context, item models.WorkOrderItem) error {
	return r.db.WithContext(ctx).
		Save(&item).Error
}

//
// ==============================
// ➕ INSERT
// ==============================
//

func (r *productionRepo) InsertScanLog(ctx context.Context, log models.ProductionScanLog) error {
	return r.db.WithContext(ctx).
		Create(&log).Error
}

func (r *productionRepo) InsertRawMaterial(ctx context.Context, rm models.RawMaterialLog) error {
	return r.db.WithContext(ctx).
		Create(&rm).Error
}

func (r *productionRepo) InsertQCLog(ctx context.Context, qc models.QCLog) error {
	return r.db.WithContext(ctx).
		Create(&qc).Error
}

func (r *productionRepo) InsertFinishedGoods(ctx context.Context, fg models.FinishedGoods) error {
	return r.db.WithContext(ctx).
		Create(&fg).Error
}

func (r *productionRepo) IsQCPendingExist(ctx context.Context, woItemID int64, process string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("qc_tasks").
		Where("task_type = ?", "production_qc").
		Where("status = ?", "pending").
		Where("round_results->>'process_name' = ?", process).
		Where("round_results->>'wo_id' = ?", fmt.Sprintf("%d", woItemID)).
		Count(&count).Error

	return count > 0, err
}

func (r *productionRepo) CreateQC(ctx context.Context, qc *models.QCTask) error {
	return r.db.WithContext(ctx).Create(qc).Error
}
