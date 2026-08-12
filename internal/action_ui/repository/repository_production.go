package repository

import (
	"context"
	"errors"
	"fmt"
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
	FindQueueWIPItem(ctx context.Context, wipID int64, uniq string, processName string, seq int) (models.WIPItem, error)
	// [wip-source] cari WIP item proses saat ini (queue/process) untuk material input.
	FindIncomingWIPForItem(ctx context.Context, woID int64, uniq string, processName string) (models.WIPItem, error)
	CreateWIPItem(ctx context.Context, data *models.WIPItem) error
	UpdateWIPItem(ctx context.Context, data *models.WIPItem) error
	CreateWIPLog(ctx context.Context, data *models.WIPLog) error
	MarkWIPDoneByWO(ctx context.Context, woID int64) error

	ListWorkOrders(ctx context.Context, search string, limit int) ([]WOListAgg, error)
	FindRawMaterialByCode(ctx context.Context, code string) (models.RawMaterial, error)
	LookupItemUniqByPacking(ctx context.Context, packingNumber string) (string, error)
	FindItemByUniq(ctx context.Context, uniq string) (ItemLookup, error)
	FindRawMaterialByUUID(ctx context.Context, rmUUID string) (models.RawMaterial, error)
	DecreaseRawMaterialStock(ctx context.Context, id int64, qty float64) (before float64, after float64, err error)
	InsertInventoryMovementLog(ctx context.Context, log models.InventoryMovementLog) error

	FindMachineByNumber(ctx context.Context, number string) (models.MasterMachine, error)
	FindProcessNameByID(ctx context.Context, id int64) (string, error)
	UpdateWOItemMachine(ctx context.Context, itemID int64, machineID int, lastProcess string) error

	FindWOItemByID(ctx context.Context, id int64) (models.WorkOrderItem, error)

	FindProductionRawMaterialLogsByWOItemID(ctx context.Context, woItemID int64) ([]models.RawMaterialLog, error)
	// [proc-scope] log RM per step proses; log proses lain tidak ikut terbawa.
	FindProductionRawMaterialLogsByWOItemStep(ctx context.Context, woItemID int64, stepSeq int) ([]models.RawMaterialLog, error)
	DeleteRawMaterialLogsByWOItemStep(ctx context.Context, woItemID int64, stepSeq int) error
	FindLatestScanInLog(ctx context.Context, woItemID int64) (models.ProductionScanLog, error)

	FindRawMaterialMetaByKeys(ctx context.Context, uuids []string, uniqCodes []string) ([]RawMaterialMeta, error)

	FindBomMaterialsByRootUniq(ctx context.Context, rootUniq string) ([]BomMaterialRow, error)

	// [repacking] list packing/kanban milik satu Raw Material (uniq_code).
	ListRMPackingList(ctx context.Context, uniqCode string) ([]RMPackingRow, error)
	// [repacking] set qty_opname (qty berjalan) satu packing setelah repack.
	ApplyPackingQtyOpname(ctx context.Context, itemUniqCode, packingNumber string, finalQty float64, at time.Time) error
	// [packing-deduct] pengurangan qty packing (relatif) saat scan out.
	DeductPackingQty(ctx context.Context, itemUniqCode, packingNumber string, deductQty float64, at time.Time) error
	// [qty-tersedia-scanout] total stok berjalan (qty_opname/quantity) untuk
	// sekumpulan packing/kanban milik satu RM; found=false bila tak ada baris.
	SumLivePackingQty(ctx context.Context, itemUniqCode string, packingNumbers []string) (total float64, found bool, err error)
	// [scanin-draft-db] draft scan-in (seed) bersama lintas gadget.
	ListScanInDrafts(ctx context.Context, woID int64, currentStep int) ([]models.ProductionScaninDraft, error)
	UpsertScanInDraft(ctx context.Context, draft models.ProductionScaninDraft) error
	DeleteScanInDraft(ctx context.Context, woID, woItemID int64, currentStep int) error
	// [repack-sisa] tambah qty packing tujuan saat repacking sisa material.
	AddPackingQty(ctx context.Context, itemUniqCode, packingNumber string, addQty float64, at time.Time) error
}

// BomMaterialRow = hasil query flatten BOM tree untuk kebutuhan Action UI.
type BomMaterialRow struct {
	ItemID          int64    `gorm:"column:item_id"`
	Level           int      `gorm:"column:level"`
	LineID          *int64   `gorm:"column:line_id"`
	QtyPerUniq      *float64 `gorm:"column:qty_per_uniq"`
	LineUom         *string  `gorm:"column:line_uom"`
	UniqCode        string   `gorm:"column:uniq_code"`
	PartName        string   `gorm:"column:part_name"`
	PartNumber      *string  `gorm:"column:part_number"`
	ItemUom         string   `gorm:"column:item_uom"`
	MaterialGrade   *string  `gorm:"column:material_grade"`
	Grade           *string  `gorm:"column:grade"`
	TypeMaterial    *string  `gorm:"column:type_material"`
	Form            *string  `gorm:"column:form"`
	WidthMm         *float64 `gorm:"column:width_mm"`
	DiameterMm      *float64 `gorm:"column:diameter_mm"`
	ThicknessMm     *float64 `gorm:"column:thickness_mm"`
	LengthMm        *float64 `gorm:"column:length_mm"`
	WeightKg        *float64 `gorm:"column:weight_kg"`
	SupplierName    *string  `gorm:"column:supplier_name"`
	RMUUID          *string  `gorm:"column:rm_uuid"`
	RMUniqCode      *string  `gorm:"column:rm_uniq_code"`
	RawMaterialType *string  `gorm:"column:raw_material_type"`
	RMUom           *string  `gorm:"column:rm_uom"`
	StockQty        *float64 `gorm:"column:stock_qty"`
	StockWeightKg   *float64 `gorm:"column:stock_weight_kg"`
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

func (r *productionRepo) FindQueueWIPItem(ctx context.Context, wipID int64, uniq string, processName string, seq int) (models.WIPItem, error) {
	var row models.WIPItem

	err := r.db.WithContext(ctx).
		Where(`
			wip_id = ?
			AND uniq = ?
			AND process_name = ?
			AND status = ?
			AND seq = ?
		`, wipID, uniq, processName, "queue", seq).
		Order("id asc").
		First(&row).Error

	return row, err
}

func (r *productionRepo) FindIncomingWIPForItem(ctx context.Context, woID int64, uniq string, processName string) (models.WIPItem, error) {
	var row models.WIPItem
	var wipIDs []int64
	if err := r.db.WithContext(ctx).Table("wips").
		Where("wo_id = ?", woID).Pluck("id", &wipIDs).Error; err != nil {
		return row, err
	}
	if len(wipIDs) == 0 {
		return row, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Where("wip_id IN ? AND uniq = ? AND process_name = ? AND status IN ?",
			wipIDs, uniq, processName, []string{"queue", "process"}).
		Order("id desc").
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

func (r *productionRepo) MarkWIPDoneByWO(ctx context.Context, woID int64) error {
	now := time.Now()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) tandai semua wip_items milik WO ini jadi "done"
		if err := tx.
			Model(&models.WIPItem{}).
			Where(`
				wip_id IN (SELECT id FROM wips WHERE wo_id = ?)
				AND status <> ?
			`, woID, "done").
			Updates(map[string]interface{}{
				"status":        "done",
				"qty_out":       gorm.Expr("qty_in"),
				"qty_remaining": 0,
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		// 2) tandai header WIP jadi "done"
		return tx.
			Model(&models.WIP{}).
			Where("wo_id = ?", woID).
			Updates(map[string]interface{}{
				"status":     "done",
				"updated_at": now,
			}).Error
	})
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
	WOType         string
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
			w.wo_type                         AS wo_type,
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

func (r *productionRepo) LookupItemUniqByPacking(ctx context.Context, packingNumber string) (string, error) {
	packingNumber = strings.TrimSpace(packingNumber)
	if packingNumber == "" {
		return "", errors.New("packing number empty")
	}

	var uniqCode string
	err := r.db.WithContext(ctx).Raw(`
		SELECT item_uniq_code FROM delivery_note_items WHERE packing_number = ?
		UNION ALL
		SELECT item_uniq_code FROM work_order_items WHERE kanban_number = ?
		LIMIT 1
	`, packingNumber, packingNumber).Scan(&uniqCode).Error

	if err != nil {
		return "", err
	}
	if uniqCode == "" {
		return "", gorm.ErrRecordNotFound
	}
	return uniqCode, nil
}

type ItemLookup struct {
	ID         int64
	UniqCode   string
	PartNumber *string
	PartName   *string
	UOM        *string
	StockQty   float64
}

func (r *productionRepo) FindItemByUniq(ctx context.Context, uniq string) (ItemLookup, error) {
	var item ItemLookup
	err := r.db.WithContext(ctx).Raw(`
		SELECT i.id, i.uniq_code, i.part_number, i.part_name, i.uom,
		       COALESCE((SELECT stock_qty FROM finished_goods WHERE uniq_code = i.uniq_code LIMIT 1), 0) AS stock_qty
		FROM items i WHERE i.uniq_code = ? AND i.deleted_at IS NULL LIMIT 1
	`, uniq).Scan(&item).Error
	if err != nil {
		return item, err
	}
	if item.UniqCode == "" {
		return item, gorm.ErrRecordNotFound
	}
	return item, nil
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

const bomMaterialsQuery = `
WITH root AS (
	SELECT id
	FROM items
	WHERE uniq_code = ? AND deleted_at IS NULL
	ORDER BY id DESC
	LIMIT 1
),
cur_bom AS (
	SELECT bi.id, bi.root_item_revision_id
	FROM bom_item bi
	JOIN root ON root.id = bi.item_id
	ORDER BY bi.is_current DESC, bi.version DESC, bi.id DESC
	LIMIT 1
),
nodes AS (
	SELECT root.id AS item_id, 0 AS level, NULL::bigint AS line_id,
	       NULL::numeric AS qty_per_uniq, NULL::text AS line_uom,
	       (SELECT root_item_revision_id FROM cur_bom) AS revision_id
	FROM root
	UNION ALL
	SELECT bl.child_item_id AS item_id, bl.level::int AS level, bl.id AS line_id,
	       bl.qty_per_uniq, bl.uom AS line_uom, bl.child_item_revision_id AS revision_id
	FROM bom_lines bl
	JOIN cur_bom ON cur_bom.id = bl.bom_item_id
)
SELECT
	n.item_id, n.level, n.line_id, n.qty_per_uniq, n.line_uom,
	it.uniq_code, it.part_name, it.part_number, it.uom AS item_uom,
	ms.material_grade, ms.grade, ms.type_material, ms.form,
	ms.width_mm, ms.diameter_mm, ms.thickness_mm, ms.length_mm, ms.weight_kg, ms.supplier_name,
	CAST(rm.uuid AS TEXT) AS rm_uuid, rm.uniq_code AS rm_uniq_code,
	rm.raw_material_type, rm.uom AS rm_uom,
	rm.stock_qty, rm.stock_weight_kg
FROM nodes n
JOIN items it ON it.id = n.item_id
LEFT JOIN LATERAL (
	SELECT ir.id
	FROM item_revisions ir
	WHERE ir.item_id = n.item_id
	ORDER BY ((ir.id = n.revision_id) IS TRUE) DESC, ir.id DESC
	LIMIT 1
) rev ON TRUE
LEFT JOIN item_material_specs ms ON ms.item_revision_id = rev.id
-- [rm-source] Master Raw Material dicari lewat material code pada
-- spesifikasi BOM (mis. BR50) lebih dulu, baru fallback ke uniq item
-- (mis. M19). Sebelumnya hanya uniq item yang dipakai, sehingga Qty
-- Tersedia yang tampil adalah stok item WO, bukan stok raw material.
LEFT JOIN LATERAL (
	SELECT r.uuid, r.uniq_code, r.raw_material_type, r.uom, r.stock_qty, r.stock_weight_kg
	FROM raw_materials r
	WHERE r.deleted_at IS NULL
	  AND (
		r.uniq_code = NULLIF(TRIM(ms.material_grade), '')
		OR r.uniq_code = it.uniq_code
	  )
	ORDER BY ((r.uniq_code = NULLIF(TRIM(ms.material_grade), '')) IS TRUE) DESC, r.id DESC
	LIMIT 1
) rm ON TRUE
ORDER BY n.level ASC, n.line_id ASC
`

func (r *productionRepo) FindBomMaterialsByRootUniq(ctx context.Context, rootUniq string) ([]BomMaterialRow, error) {
	rootUniq = strings.TrimSpace(rootUniq)
	if rootUniq == "" {
		return nil, nil
	}
	var rows []BomMaterialRow
	if err := r.db.WithContext(ctx).Raw(bomMaterialsQuery, rootUniq).Scan(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("FindBomMaterialsByRootUniq", err)
	}
	return rows, nil
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

// [proc-scope] Log RM milik SATU step proses saja. Baris lama yang belum punya
// step_seq dianggap milik step pertama supaya data lama tetap tampil.
func (r *productionRepo) FindProductionRawMaterialLogsByWOItemStep(
	ctx context.Context, woItemID int64, stepSeq int,
) ([]models.RawMaterialLog, error) {
	var logs []models.RawMaterialLog
	q := r.db.WithContext(ctx).Where("wo_item_id = ?", woItemID)
	if stepSeq <= 1 {
		q = q.Where("COALESCE(step_seq, 0) <= 1")
	} else {
		q = q.Where("COALESCE(step_seq, 0) = ?", stepSeq)
	}
	err := q.Order("id ASC").Find(&logs).Error
	return logs, err
}

// [proc-scope] Hapus log RM milik satu step proses saja.
func (r *productionRepo) DeleteRawMaterialLogsByWOItemStep(
	ctx context.Context, woItemID int64, stepSeq int,
) error {
	q := r.db.WithContext(ctx).Where("wo_item_id = ?", woItemID)
	if stepSeq <= 1 {
		q = q.Where("COALESCE(step_seq, 0) <= 1")
	} else {
		q = q.Where("COALESCE(step_seq, 0) = ?", stepSeq)
	}
	return q.Delete(&models.RawMaterialLog{}).Error
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

// ---------------------------------------------------------------------------
// [repacking] Packing / Kanban list per Raw Material
// ---------------------------------------------------------------------------

// RMPackingRow = baris mentah packing/kanban untuk satu uniq_code RM.
type RMPackingRow struct {
	DNNumber      *string `gorm:"column:dn_number"`
	PackingNumber string  `gorm:"column:packing_number"`
	Quantity      float64 `gorm:"column:quantity"`
	QtyCurrent    float64 `gorm:"column:qty_current"`
	QtyMax        float64 `gorm:"column:qty_max"`
	Status        *string `gorm:"column:status"`
	WONumber      *string `gorm:"column:wo_number"`
	Source        string  `gorm:"column:source"`
}

// ListRMPackingList mengambil daftar packing/kanban untuk satu uniq_code RM.
// Sumber: delivery_note_items (DN Management) -- Raw Material masuk lewat DN,
// BUKAN work_order_items (WO khusus pembuatan finished goods). Di-join ke
// delivery_notes untuk DN number & kanban_parameters (kanban_qty = qty maksimal).
// qty_current diambil dari qty_opname (hasil repack) bila ada, jika tidak dari
// quantity pada DN. Sama dengan branch 'delivery_note' packing list RM di ERP.
// [qty-tersedia-scanout] SumLivePackingQty menjumlahkan stok berjalan
// (qty_opname bila ada, jika tidak quantity) dari delivery_note_items untuk
// packing/kanban tertentu. Dipakai Scan Out agar Qty Tersedia mengikuti stok
// yang sudah berkurang, sama seperti "Qty saat ini" di detail RM ERP.
func (r *productionRepo) SumLivePackingQty(ctx context.Context, itemUniqCode string, packingNumbers []string) (float64, bool, error) {
	cleaned := make([]string, 0, len(packingNumbers))
	for _, pn := range packingNumbers {
		if strings.TrimSpace(pn) != "" {
			cleaned = append(cleaned, strings.TrimSpace(pn))
		}
	}
	if len(cleaned) == 0 {
		return 0, false, nil
	}
	var res struct {
		Total float64 `gorm:"column:total"`
		Cnt   int64   `gorm:"column:cnt"`
	}
	q := r.db.WithContext(ctx).
		Table("delivery_note_items").
		Select("COALESCE(SUM(COALESCE(qty_opname, quantity, 0)), 0) AS total, COUNT(*) AS cnt").
		Where("packing_number IN ?", cleaned)
	if strings.TrimSpace(itemUniqCode) != "" {
		q = q.Where("item_uniq_code = ?", strings.TrimSpace(itemUniqCode))
	}
	if err := q.Scan(&res).Error; err != nil {
		return 0, false, err
	}
	return res.Total, res.Cnt > 0, nil
}

func (r *productionRepo) ListRMPackingList(ctx context.Context, uniqCode string) ([]RMPackingRow, error) {
	uniqCode = strings.TrimSpace(uniqCode)
	if uniqCode == "" {
		return []RMPackingRow{}, nil
	}

	var rows []RMPackingRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT * FROM (
			SELECT
				dn.dn_number      AS dn_number,
				woi.kanban_number AS packing_number,
				COALESCE(woi.quantity, 0) AS quantity,
				COALESCE(
					woi.qty_opname,
					dni.qty_opname,
					CASE WHEN woi.status IN ('FINISHED', 'DONE', 'COMPLETED') THEN NULLIF(woi.total_good_qty, 0) ELSE NULL END,
					dni.quantity,
					0
				) AS qty_current,
				COALESCE(
					NULLIF(kp.kanban_qty, 0),
					NULLIF(dni.pcs_per_kanban, 0),
					NULLIF(woi.quantity, 0),
					0
				) AS qty_max,
				woi.status   AS status,
				w.wo_number  AS wo_number,
				'work_order' AS source,
				woi.created_at AS sort_at
			FROM work_order_items woi
			JOIN work_orders w ON w.id = woi.wo_id
			LEFT JOIN delivery_note_items dni ON dni.packing_number = woi.kanban_number
			LEFT JOIN delivery_notes dn ON dn.id = dni.dn_id
			LEFT JOIN LATERAL (
				SELECT kanban_qty
				FROM kanban_parameters
				WHERE item_uniq_code = woi.item_uniq_code
				ORDER BY id DESC
				LIMIT 1
			) kp ON TRUE
			WHERE woi.item_uniq_code = ?

			UNION ALL

			SELECT
				dn.dn_number       AS dn_number,
				dni.packing_number AS packing_number,
				COALESCE(dni.quantity, 0) AS quantity,
				COALESCE(dni.qty_opname, dni.quantity, 0) AS qty_current,
				COALESCE(
					NULLIF(dni.pcs_per_kanban, 0),
					NULLIF(kp.kanban_qty, 0),
					NULLIF(dni.quantity, 0),
					0
				) AS qty_max,
				NULLIF(dni.check, '') AS status,
				NULL            AS wo_number,
				'delivery_note' AS source,
				dni.created_at AS sort_at
			FROM delivery_note_items dni
			JOIN delivery_notes dn ON dn.id = dni.dn_id
			LEFT JOIN LATERAL (
				SELECT kanban_qty
				FROM kanban_parameters
				WHERE item_uniq_code = dni.item_uniq_code
				ORDER BY id DESC
				LIMIT 1
			) kp ON TRUE
			WHERE dni.item_uniq_code = ?
			  AND COALESCE(dni.packing_number, '') <> ''
			  AND NOT EXISTS (
				SELECT 1 FROM work_order_items woi2
				WHERE woi2.kanban_number = dni.packing_number
			  )
		) packing
		ORDER BY sort_at DESC, packing_number ASC
	`, uniqCode, uniqCode).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal("action_ui list rm packing list: " + err.Error())
	}
	if rows == nil {
		rows = []RMPackingRow{}
	}
	return rows, nil
}

// ApplyPackingQtyOpname men-set qty berjalan (qty_opname) satu packing setelah
// repack. Dipakai fitur Repacking saat scan out untuk menyesuaikan qty per
// packing/kanban pada delivery_note_items.
func (r *productionRepo) ApplyPackingQtyOpname(ctx context.Context, itemUniqCode, packingNumber string, finalQty float64, at time.Time) error {
	packingNumber = strings.TrimSpace(packingNumber)
	if packingNumber == "" {
		return nil
	}
	err := r.db.WithContext(ctx).Exec(`
		UPDATE delivery_note_items
		SET qty_opname = ?, qty_opname_at = ?, updated_at = ?
		WHERE item_uniq_code = ? AND packing_number = ?
	`, finalQty, at, at, itemUniqCode, packingNumber).Error
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE work_order_items
		SET qty_opname = ?, updated_at = ?
		WHERE item_uniq_code = ? AND kanban_number = ?
	`, finalQty, at, itemUniqCode, packingNumber).Error
}

// DeductPackingQty mengurangi qty berjalan (qty_opname) satu packing
// sebesar deductQty. Dipakai saat scan out untuk packing yang discan di
// Step 1 tanpa lewat Repacking, supaya "Qty saat ini" dan Progress pada
// packing list Raw Material ikut berkurang seperti stok RM-nya.
func (r *productionRepo) DeductPackingQty(ctx context.Context, itemUniqCode, packingNumber string, deductQty float64, at time.Time) error {
	packingNumber = strings.TrimSpace(packingNumber)
	if packingNumber == "" || deductQty <= 0 {
		return nil
	}
	// [packing-guard] Tolak over-deduction: bila qty yang diminta melebihi stok
	// berjalan packing (qty_opname / quantity), kembalikan error alih-alih diam-diam
	// floor ke 0. Ini mencegah alokasi packing yang sama ke banyak kanban
	// (auto-apply Step 1 POKA YOKE) menghabiskan stok tanpa peringatan.
	var cur struct{ Qty float64 }
	if e := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(qty_opname, quantity, 0) AS qty
		FROM delivery_note_items
		WHERE item_uniq_code = ? AND packing_number = ?
		LIMIT 1
	`, itemUniqCode, packingNumber).Scan(&cur).Error; e == nil {
		if deductQty > cur.Qty+1e-6 {
			return fmt.Errorf("stok packing %s tidak cukup: butuh %.4f, tersedia %.4f", packingNumber, deductQty, cur.Qty)
		}
	}
	err := r.db.WithContext(ctx).Exec(`
		UPDATE delivery_note_items
		SET qty_opname = GREATEST(COALESCE(qty_opname, quantity, 0) - ?, 0),
		    qty_opname_at = ?,
		    updated_at = ?
		WHERE item_uniq_code = ? AND packing_number = ?
	`, deductQty, at, at, itemUniqCode, packingNumber).Error
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE work_order_items
		SET qty_opname = GREATEST(COALESCE(qty_opname, quantity, 0) - ?, 0),
		    updated_at = ?
		WHERE item_uniq_code = ? AND kanban_number = ?
	`, deductQty, at, itemUniqCode, packingNumber).Error
}

// [repack-sisa] AddPackingQty menambah qty berjalan (qty_opname) satu packing
// sebesar addQty. Dipakai saat Repacking: sisa material dari packing asal
// dipindahkan ke packing tujuan yang masih punya slot.
func (r *productionRepo) AddPackingQty(ctx context.Context, itemUniqCode, packingNumber string, addQty float64, at time.Time) error {
	packingNumber = strings.TrimSpace(packingNumber)
	if packingNumber == "" || addQty <= 0 {
		return nil
	}
	err := r.db.WithContext(ctx).Exec(`
		UPDATE delivery_note_items
		SET qty_opname = COALESCE(qty_opname, quantity, 0) + ?,
		    qty_opname_at = ?,
		    updated_at = ?
		WHERE item_uniq_code = ? AND packing_number = ?
	`, addQty, at, at, itemUniqCode, packingNumber).Error
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE work_order_items
		SET qty_opname = COALESCE(qty_opname, quantity, 0) + ?,
		    updated_at = ?
		WHERE item_uniq_code = ? AND kanban_number = ?
	`, addQty, at, itemUniqCode, packingNumber).Error
}

// ================================
// [scanin-draft-db] Draft Scan-In (seed) bersama lintas gadget
// ================================

// ListScanInDrafts mengambil draft scan-in satu WO. Bila currentStep > 0,
// hanya draft step tersebut yang dikembalikan; bila <= 0, semua step.
func (r *productionRepo) ListScanInDrafts(ctx context.Context, woID int64, currentStep int) ([]models.ProductionScaninDraft, error) {
	var rows []models.ProductionScaninDraft
	q := r.db.WithContext(ctx).Where("wo_id = ?", woID)
	if currentStep > 0 {
		q = q.Where("current_step = ?", currentStep)
	}
	err := q.Order("wo_item_id asc").Find(&rows).Error
	return rows, err
}

// UpsertScanInDraft menyimpan (insert/update) satu draft berdasarkan
// (wo_id, wo_item_id, current_step).
func (r *productionRepo) UpsertScanInDraft(ctx context.Context, draft models.ProductionScaninDraft) error {
	now := time.Now()
	if draft.CurrentStep <= 0 {
		draft.CurrentStep = 1
	}
	draft.UpdatedAt = now
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "wo_id"}, {Name: "wo_item_id"}, {Name: "current_step"}},
		DoUpdates: clause.AssignmentColumns([]string{"payload", "updated_by", "updated_at"}),
	}).Create(&draft).Error
}

// DeleteScanInDraft menghapus draft. Bila currentStep <= 0, semua step milik
// wo_item tersebut dihapus (dipakai saat "Mulai Produksi").
func (r *productionRepo) DeleteScanInDraft(ctx context.Context, woID, woItemID int64, currentStep int) error {
	q := r.db.WithContext(ctx).Where("wo_id = ? AND wo_item_id = ?", woID, woItemID)
	if currentStep > 0 {
		q = q.Where("current_step = ?", currentStep)
	}
	return q.Delete(&models.ProductionScaninDraft{}).Error
}
