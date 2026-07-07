package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	UpdateWOStatus(ctx context.Context, woID int64, status string) error

	InsertScanLog(ctx context.Context, log models.ProductionScanLog) error
	InsertRawMaterial(ctx context.Context, rm models.RawMaterialLog) error
	DeleteRawMaterialLogsByWOItemID(ctx context.Context, woItemID int64) error
	InsertQCLog(ctx context.Context, qc models.QCLog) error
	InsertFinishedGoods(ctx context.Context, fg models.FinishedGoods) error

	IsQCPendingExist(ctx context.Context, woItemID int64, process string) (bool, error)
	CreateQC(ctx context.Context, qc *models.QCTask) error
	InsertProductIssue(ctx context.Context, data models.ProductionIssue) error

	CountQCLogs(ctx context.Context, workOrderID int64) (int64, error)

	// WIP
	FindOrCreateWIP(ctx context.Context, woID int64) (models.WIP, error)
	FindQueueWIPItem(ctx context.Context, wipID int64, uniq string, processName string) (models.WIPItem, error)
	CreateWIPItem(ctx context.Context, data *models.WIPItem) error
	UpdateWIPItem(ctx context.Context, data *models.WIPItem) error
	CreateWIPLog(ctx context.Context, data *models.WIPLog) error

	ListWorkOrders(ctx context.Context, search string, limit int) ([]WOListAgg, error)
	FindRawMaterialByCode(ctx context.Context, code string) (models.RawMaterial, error)
	FindRawMaterialByUUID(ctx context.Context, rmUUID string) (models.RawMaterial, error)
	DecreaseRawMaterialStock(ctx context.Context, id int64, qty float64) (before float64, after float64, err error)
	InsertInventoryMovementLog(ctx context.Context, log models.InventoryMovementLog) error

	FindMachineByNumber(ctx context.Context, number string) (models.MasterMachine, error)
	FindProcessNameByID(ctx context.Context, id int64) (string, error)
	UpdateWOItemMachine(ctx context.Context, itemID int64, machineID int, lastProcess string) error

	FindWOItemByID(ctx context.Context, id int64) (models.WorkOrderItem, error)

	FindProductionRawMaterialLogsByWOItemID(ctx context.Context, woItemID int64) ([]models.RawMaterialLog, error)
	FindLatestScanInLog(ctx context.Context, woItemID int64) (models.ProductionScanLog, error)

	FindRawMaterialMetaByKeys(ctx context.Context, uuids []string, uniqCodes []string) ([]RawMaterialMeta, error)
}

type productionRepo struct {
	db *gorm.DB
}

func NewProductionRepository(db *gorm.DB) IProductionRepository {
	return &productionRepo{db: db}
}

func (r *productionRepo) FindOrCreateWIP(ctx context.Context, woID int64) (models.WIP, error) {
	var row models.WIP

	err := r.db.WithContext(ctx).
		Where("wo_id = ?", woID).
		First(&row).Error

	if err == nil {
		return row, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return row, err
	}

	now := time.Now()

	row = models.WIP{
		WoID:      woID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = r.db.WithContext(ctx).Create(&row).Error
	return row, err
}

func (r *productionRepo) FindQueueWIPItem(ctx context.Context, wipID int64, uniq string, processName string) (models.WIPItem, error) {
	var row models.WIPItem

	err := r.db.WithContext(ctx).
		Where(`
			wip_id = ?
			AND uniq = ?
			AND process_name = ?
			AND status = ?
		`, wipID, uniq, processName, "queue").
		Order("id asc").
		First(&row).Error

	return row, err
}

func (r *productionRepo) CreateWIPItem(ctx context.Context, data *models.WIPItem) error {
	return r.db.WithContext(ctx).
		Create(data).Error
}

func (r *productionRepo) UpdateWIPItem(ctx context.Context, data *models.WIPItem) error {
	return r.db.WithContext(ctx).
		Save(data).Error
}

func (r *productionRepo) CreateWIPLog(ctx context.Context, data *models.WIPLog) error {
	return r.db.WithContext(ctx).
		Create(data).Error
}

//
// ==============================
// 🔍 FIND DATA
// ==============================
//

func (r *productionRepo) CountQCLogs(ctx context.Context, workOrderID int64) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&models.QCLog{}).
		Where("work_order_id = ?", workOrderID).
		Count(&total).Error

	if err != nil {
		return 0, err
	}

	return total, nil
}

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
			return item, apperror.NotFound("uniq tidak ditemukan in wo")
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
			return item, apperror.NotFound("uniq tidak ditemukan")
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
			return item, apperror.NotFound("uniq tidak ditemukan")
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
			return wo, apperror.NotFound("wo tidak ditemukan")
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
			return wo, apperror.NotFound("wo tidak ditemukan")
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
			return wo, apperror.NotFound("wo tidak ditemukan")
		}
		return wo, err
	}

	return wo, nil
}

func (r *productionRepo) FindWOItemsByWOID(ctx context.Context, woid int64) ([]models.WorkOrderItem, error) {
	var items []models.WorkOrderItem

	err := r.db.WithContext(ctx).
	Where("wo_id = ?", woid).
	Order("id ASC").
	Find(&items).Error

	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, apperror.NotFound("wo items tidak ditemukan")
	}

	return items, nil
}

func (r *productionRepo) FindMachineByID(ctx context.Context, id int) (models.MasterMachine, error) {
	var machine models.MasterMachine

	err := r.db.WithContext(ctx).
		First(&machine, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return machine, apperror.NotFound("machine tidak ditemukan")
		}
		return machine, err
	}

	return machine, nil
}

func (r *productionRepo) FindMachineByNumber(ctx context.Context, number string) (models.MasterMachine, error) {
	var m models.MasterMachine
	err := r.db.WithContext(ctx).
		Where("machine_number = ?", number).
		First(&m).Error
	return m, err
}

func (r *productionRepo) FindProcessNameByID(ctx context.Context, id int64) (string, error) {
	var name string
	err := r.db.WithContext(ctx).
		Table("process_parameters").
		Select("process_name").
		Where("id = ?", id).
		Scan(&name).Error
	return name, err
}

func (r *productionRepo) UpdateWOItemMachine(ctx context.Context, itemID int64, machineID int, lastProcess string) error {
	return r.db.WithContext(ctx).
		Model(&models.WorkOrderItem{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"machine_id":           machineID,
			"last_scanned_process": lastProcess,
			"updated_at":           time.Now(),
		}).Error
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

func (r *productionRepo) UpdateWOStatus(ctx context.Context, woID int64, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.WorkOrder{}).
		Where("id = ?", woID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
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

func (r *productionRepo) DeleteRawMaterialLogsByWOItemID(ctx context.Context, woItemID int64) error {
	return r.db.WithContext(ctx).
		Where("wo_item_id = ?", woItemID).
		Delete(&models.RawMaterialLog{}).Error
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
		Where("wo_item_id = ?", woItemID).
		Count(&count).Error

	return count > 0, err
}

func (r *productionRepo) CreateQC(ctx context.Context, qc *models.QCTask) error {
	return r.db.WithContext(ctx).Create(qc).Error
}


type WOListAgg struct {
	ID             int64
	WONumber       string
	Status         string
	Model          string
	PartName       string
	ProductionLine string
	TotalQty       float64
	UniqCount      int64
}

func (r *productionRepo) ListWorkOrders(ctx context.Context, search string, limit int) ([]WOListAgg, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	term := strings.TrimSpace(search)
	like := "%" + term + "%"

	var rows []WOListAgg
	err := r.db.WithContext(ctx).
		Table("work_orders AS w").
		Select(`
			w.id                              AS id,
			w.wo_number                       AS wo_number,
			w.status                          AS status,
			w.model                           AS model,
			COALESCE(MIN(i.part_name), '')    AS part_name,
			COALESCE(MIN(m.production_line),'') AS production_line,
			COALESCE(SUM(i.quantity), 0)      AS total_qty,
			COUNT(i.id)                       AS uniq_count
		`).
		Joins("LEFT JOIN work_order_items i ON i.wo_id = w.id").
		Joins("LEFT JOIN master_machines m ON m.id = i.machine_id").
		Where("(? = '' OR w.wo_number ILIKE ? OR w.model ILIKE ?)", term, like, like).
		Group("w.id").
		Order("w.created_at DESC").
		Limit(limit).
		Scan(&rows).Error

	return rows, err
}

func (r *productionRepo) FindRawMaterialByCode(ctx context.Context, code string) (models.RawMaterial, error) {
	var rm models.RawMaterial
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("uniq_code = ? OR part_number = ?", code, code).
		First(&rm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rm, apperror.NotFound("raw material tidak ditemukan")
		}
		return rm, err
	}
	return rm, nil
}

func (r *productionRepo) FindRawMaterialByUUID(ctx context.Context, rmUUID string) (models.RawMaterial, error) {
	var rm models.RawMaterial
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND uuid = ?", rmUUID).
		First(&rm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rm, apperror.NotFound("raw material tidak ditemukan")
		}
		return rm, err
	}
	return rm, nil
}

func (r *productionRepo) DecreaseRawMaterialStock(ctx context.Context, id int64, qty float64) (float64, float64, error) {
	var before, after float64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rm models.RawMaterial
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rm, id).Error; err != nil {
			return err
		}
		before = rm.StockQty
		after = before - qty
		if after < 0 {
			after = 0
		}
		return tx.Model(&models.RawMaterial{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"stock_qty":  after,
				"updated_at": time.Now(),
			}).Error
	})
	return before, after, err
}

func (r *productionRepo) InsertInventoryMovementLog(ctx context.Context, log models.InventoryMovementLog) error {
	return r.db.WithContext(ctx).Create(&log).Error
}

func (r *productionRepo) FindWOItemByID(ctx context.Context, id int64) (models.WorkOrderItem, error) {
	var item models.WorkOrderItem

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, apperror.NotFound("wo item tidak ditemukan")
		}
		return item, err
	}

	return item, nil
}

// RawMaterialMeta = data master RM untuk enrich tampilan scan out.
type RawMaterialMeta struct {
	UUID            string  `gorm:"column:uuid"`
	UniqCode        string  `gorm:"column:uniq_code"`
	RawMaterialType string  `gorm:"column:raw_material_type"`
	UOM             string  `gorm:"column:uom"`
	StockQty        float64 `gorm:"column:stock_qty"`
	StockWeightKg   float64 `gorm:"column:stock_weight_kg"`
}

func (r *productionRepo) FindProductionRawMaterialLogsByWOItemID(
	ctx context.Context, woItemID int64,
) ([]models.RawMaterialLog, error) {
	var logs []models.RawMaterialLog
	err := r.db.WithContext(ctx).
		Where("wo_item_id = ?", woItemID).
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}

func (r *productionRepo) FindLatestScanInLog(ctx context.Context, woItemID int64) (models.ProductionScanLog, error) {
	var log models.ProductionScanLog
	err := r.db.WithContext(ctx).
		Where("wo_item_id = ? AND scan_type = ?", woItemID, "SCAN_IN").
		Order("id DESC").
		First(&log).Error
	return log, err
}

func (r *productionRepo) FindRawMaterialMetaByKeys(
	ctx context.Context, uuids []string, uniqCodes []string,
) ([]RawMaterialMeta, error) {
	var rows []RawMaterialMeta
	q := r.db.WithContext(ctx).
		Table("raw_materials").
		Select(`CAST(uuid AS TEXT) AS uuid, uniq_code, raw_material_type,
			COALESCE(uom, '') AS uom,
			COALESCE(stock_qty, 0) AS stock_qty,
			COALESCE(stock_weight_kg, 0) AS stock_weight_kg`)

	switch {
	case len(uuids) > 0 && len(uniqCodes) > 0:
		q = q.Where("CAST(uuid AS TEXT) IN ? OR uniq_code IN ?", uuids, uniqCodes)
	case len(uuids) > 0:
		q = q.Where("CAST(uuid AS TEXT) IN ?", uuids)
	case len(uniqCodes) > 0:
		q = q.Where("uniq_code IN ?", uniqCodes)
	default:
		return rows, nil
	}

	err := q.Scan(&rows).Error
	return rows, err
}