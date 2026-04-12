// Package repository provides data-access for the Bill of Material module.
package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRepository interface {
	// Core writes
	CreateItem(ctx context.Context, item *models.Item) error
	UpdateItem(ctx context.Context, item *models.Item) error
	CreateRevision(ctx context.Context, rev *models.ItemRevision) error
	CreateAsset(ctx context.Context, asset *models.ItemAsset) error
	UpdateAsset(ctx context.Context, assetID int64, fileURL, assetType string) error
	GetAssetByID(ctx context.Context, assetID int64) (*models.ItemAsset, error)
	UpsertMaterialSpec(ctx context.Context, spec *models.ItemMaterialSpec) error
	CreateRoutingHeader(ctx context.Context, h *models.RoutingHeader) error
	CreateOperation(ctx context.Context, op *models.RoutingOperation) error
	CreateTooling(ctx context.Context, t *models.RoutingOperationTooling) error
	CreateBomItem(ctx context.Context, b *models.BomItem) error
	UpdateBomItem(ctx context.Context, bom *models.BomItem) error
	CreateBomLine(ctx context.Context, line *models.BomLine) error
	DeleteBomItem(ctx context.Context, bomID int64) error
	DeleteBomLinesByIDs(ctx context.Context, bomID int64, lineIDs []int64) (int64, error)
	DeleteRoutingByItemID(ctx context.Context, itemID int64) error
	UpsertItemAssetURL(ctx context.Context, itemID int64, assetType, url string) error
	UpdateBomLine(ctx context.Context, line *models.BomLine) error
	GetBomLineByID(ctx context.Context, bomID, lineID int64) (*models.BomLine, error)

	// Reads
	GetItemByID(ctx context.Context, id int64) (*models.Item, error)
	GetItemByUniq(ctx context.Context, uniqCode string) (*models.Item, error)
	GetLatestRevision(ctx context.Context, itemID int64) (*models.ItemRevision, error)
	GetFirstAsset(ctx context.Context, itemID int64) (*models.ItemAsset, error)
	GetMaterialSpec(ctx context.Context, revisionID int64) (*models.ItemMaterialSpec, error)
	GetRoutingWithOps(ctx context.Context, itemID int64) (*models.RoutingHeader, []models.RoutingOperation, []models.RoutingOperationTooling, error)
	GetSupplierName(ctx context.Context, id uuid.UUID) string
	GetProcessName(ctx context.Context, id int64) string
	GetMachineName(ctx context.Context, id int64) string
	GetItemsByIDs(ctx context.Context, ids []int64) ([]models.Item, error)
	GetLatestRevisionsByItemIDs(ctx context.Context, itemIDs []int64) ([]models.ItemRevision, error)
	GetFirstAssetsByItemIDs(ctx context.Context, itemIDs []int64) ([]models.ItemAsset, error)
	GetMaterialSpecsByRevisionIDs(ctx context.Context, revisionIDs []int64) ([]models.ItemMaterialSpec, error)
	GetBomLinesByBomIDs(ctx context.Context, bomItemIDs []int64) ([]models.BomLine, error)
	GetLatestRoutingHeadersByItemIDs(ctx context.Context, itemIDs []int64) ([]models.RoutingHeader, error)
	GetRoutingOperationsByHeaderIDs(ctx context.Context, headerIDs []int64) ([]models.RoutingOperation, error)
	GetToolingsByOperationIDs(ctx context.Context, operationIDs []int64) ([]models.RoutingOperationTooling, error)
	GetSupplierNamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
	GetProcessNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
	GetProcessSequencesByIDs(ctx context.Context, ids []int64) (map[int64]int, error)
	GetMachineNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)

	// BOM list — returns parent bom_items with their item info
	ListBomItems(ctx context.Context, filter ListFilter) ([]models.BomItem, int64, error)
	GetBomByID(ctx context.Context, bomID int64) (*models.BomItem, error)
	// BOM lines flat for a given bom_item_id, ordered by level then line creation
	GetBomLines(ctx context.Context, bomItemID int64) ([]models.BomLine, error)
}

type ListFilter struct {
	UniqCode       string
	Status         string
	Search         string // ILIKE on uniq_code OR part_name
	Page           int
	Limit          int
	OrderBy        string
	OrderDirection string
}

// ---------------------------------------------------------------------------

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

// ---------------------------------------------------------------------------
// Writes
// ---------------------------------------------------------------------------

func (r *repository) CreateItem(ctx context.Context, item *models.Item) error {
	item.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return apperror.InternalWrap("Create Item Error", err)
	}
	return nil
}

func (r *repository) UpdateItem(ctx context.Context, item *models.Item) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return apperror.InternalWrap("Update Item Error", err)
	}
	return nil
}

func (r *repository) CreateRevision(ctx context.Context, rev *models.ItemRevision) error {
	rev.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(rev).Error; err != nil {
		return apperror.InternalWrap("Create Revision Error", err)
	}
	return nil
}

func (r *repository) CreateAsset(ctx context.Context, asset *models.ItemAsset) error {
	asset.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(asset).Error; err != nil {
		return apperror.InternalWrap("CreateAsset", err)
	}
	return nil
}

func (r *repository) UpdateAsset(ctx context.Context, assetID int64, fileURL, assetType string) error {
	if err := r.db.WithContext(ctx).
		Model(&models.ItemAsset{}).
		Where("id = ?", assetID).
		Updates(map[string]interface{}{"file_url": fileURL, "asset_type": assetType}).Error; err != nil {
		return apperror.InternalWrap("UpdateAsset", err)
	}
	return nil
}

func (r *repository) GetAssetByID(ctx context.Context, assetID int64) (*models.ItemAsset, error) {
	var a models.ItemAsset
	if err := r.db.WithContext(ctx).First(&a, "id = ?", assetID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("asset not found")
		}
		return nil, apperror.InternalWrap("GetAssetByID", err)
	}
	return &a, nil
}

func (r *repository) UpsertMaterialSpec(ctx context.Context, spec *models.ItemMaterialSpec) error {
	if err := r.db.WithContext(ctx).
		Where(models.ItemMaterialSpec{ItemRevisionID: spec.ItemRevisionID}).
		Assign(*spec).
		FirstOrCreate(spec).Error; err != nil {
		return apperror.InternalWrap("UpsertMaterialSpec", err)
	}
	return nil
}

func (r *repository) CreateRoutingHeader(ctx context.Context, h *models.RoutingHeader) error {
	h.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(h).Error; err != nil {
		return apperror.InternalWrap("CreateRoutingHeader", err)
	}
	return nil
}

func (r *repository) CreateOperation(ctx context.Context, op *models.RoutingOperation) error {
	if err := r.db.WithContext(ctx).Create(op).Error; err != nil {
		return apperror.InternalWrap("CreateOperation", err)
	}
	return nil
}

func (r *repository) CreateTooling(ctx context.Context, t *models.RoutingOperationTooling) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return apperror.InternalWrap("CreateTooling", err)
	}
	return nil
}

func (r *repository) CreateBomItem(ctx context.Context, b *models.BomItem) error {
	b.UUID = uuid.New()
	if err := r.db.WithContext(ctx).Create(b).Error; err != nil {
		return apperror.InternalWrap("CreateBomItem", err)
	}
	return nil
}

func (r *repository) CreateBomLine(ctx context.Context, line *models.BomLine) error {
	if err := r.db.WithContext(ctx).Create(line).Error; err != nil {
		return apperror.InternalWrap("CreateBomLine", err)
	}
	return nil
}

func (r *repository) DeleteBomItem(ctx context.Context, bomID int64) error {
	res := r.db.WithContext(ctx).Where("id = ?", bomID).Delete(&models.BomItem{})
	if res.Error != nil {
		return apperror.InternalWrap("DeleteBomItem", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("bom item not found")
	}
	return nil
}

func (r *repository) DeleteBomLinesByIDs(ctx context.Context, bomID int64, lineIDs []int64) (int64, error) {
	if len(lineIDs) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).
		Where("bom_item_id = ? AND id IN ?", bomID, lineIDs).
		Delete(&models.BomLine{})
	if res.Error != nil {
		return 0, apperror.InternalWrap("DeleteBomLinesByIDs", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *repository) UpdateBomItem(ctx context.Context, bom *models.BomItem) error {
	if err := r.db.WithContext(ctx).Save(bom).Error; err != nil {
		return apperror.InternalWrap("UpdateBomItem", err)
	}
	return nil
}

// DeleteRoutingByItemID removes all routing headers, operations, and toolings for an item.
func (r *repository) DeleteRoutingByItemID(ctx context.Context, itemID int64) error {
	var headers []models.RoutingHeader
	if err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Find(&headers).Error; err != nil {
		return apperror.InternalWrap("DeleteRoutingByItemID find headers", err)
	}
	if len(headers) == 0 {
		return nil
	}

	headerIDs := make([]int64, len(headers))
	for i, h := range headers {
		headerIDs[i] = h.ID
	}

	var ops []models.RoutingOperation
	if err := r.db.WithContext(ctx).Where("routing_header_id IN ?", headerIDs).Find(&ops).Error; err != nil {
		return apperror.InternalWrap("DeleteRoutingByItemID find ops", err)
	}
	if len(ops) > 0 {
		opIDs := make([]int64, len(ops))
		for i, op := range ops {
			opIDs[i] = op.ID
		}
		if err := r.db.WithContext(ctx).Where("routing_operation_id IN ?", opIDs).Delete(&models.RoutingOperationTooling{}).Error; err != nil {
			return apperror.InternalWrap("DeleteRoutingByItemID delete toolings", err)
		}
		if err := r.db.WithContext(ctx).Where("id IN ?", opIDs).Delete(&models.RoutingOperation{}).Error; err != nil {
			return apperror.InternalWrap("DeleteRoutingByItemID delete ops", err)
		}
	}
	if err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Delete(&models.RoutingHeader{}).Error; err != nil {
		return apperror.InternalWrap("DeleteRoutingByItemID delete headers", err)
	}
	return nil
}

// UpsertItemAssetURL updates the first active asset for the item or creates one if none exists.
func (r *repository) UpsertItemAssetURL(ctx context.Context, itemID int64, assetType, url string) error {
	res := r.db.WithContext(ctx).
		Model(&models.ItemAsset{}).
		Where("item_id = ? AND status = 'Active'", itemID).
		Updates(map[string]interface{}{"file_url": url, "asset_type": assetType})
	if res.Error != nil {
		return apperror.InternalWrap("UpsertItemAssetURL", res.Error)
	}
	if res.RowsAffected == 0 {
		return r.CreateAsset(ctx, &models.ItemAsset{
			ItemID:    itemID,
			AssetType: assetType,
			FileURL:   url,
			Status:    "Active",
		})
	}
	return nil
}

func (r *repository) UpdateBomLine(ctx context.Context, line *models.BomLine) error {
	if err := r.db.WithContext(ctx).Save(line).Error; err != nil {
		return apperror.InternalWrap("UpdateBomLine", err)
	}
	return nil
}

func (r *repository) GetBomLineByID(ctx context.Context, bomID, lineID int64) (*models.BomLine, error) {
	var line models.BomLine
	err := r.db.WithContext(ctx).
		Where("id = ? AND bom_item_id = ?", lineID, bomID).
		First(&line).Error
	return one(&line, err, "bom line")
}

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

func (r *repository) GetItemByID(ctx context.Context, id int64) (*models.Item, error) {
	var item models.Item
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	return one(&item, err, "item")
}

func (r *repository) GetItemByUniq(ctx context.Context, uniqCode string) (*models.Item, error) {
	var item models.Item
	err := r.db.WithContext(ctx).Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).First(&item).Error
	return one(&item, err, "item")
}

func (r *repository) GetLatestRevision(ctx context.Context, itemID int64) (*models.ItemRevision, error) {
	var rev models.ItemRevision
	err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Order("id DESC").First(&rev).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // no revision is not an error
	}
	return one(&rev, err, "revision")
}

func (r *repository) GetFirstAsset(ctx context.Context, itemID int64) (*models.ItemAsset, error) {
	var a models.ItemAsset
	err := r.db.WithContext(ctx).Where("item_id = ? AND status = 'Active'", itemID).First(&a).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return one(&a, err, "asset")
}

func (r *repository) GetMaterialSpec(ctx context.Context, revisionID int64) (*models.ItemMaterialSpec, error) {
	var spec models.ItemMaterialSpec
	err := r.db.WithContext(ctx).Where("item_revision_id = ?", revisionID).First(&spec).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return one(&spec, err, "material spec")
}

func (r *repository) GetRoutingWithOps(ctx context.Context, itemID int64) (*models.RoutingHeader, []models.RoutingOperation, []models.RoutingOperationTooling, error) {
	var header models.RoutingHeader
	err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Order("version DESC").First(&header).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, apperror.InternalWrap("GetRoutingHeader", err)
	}

	var ops []models.RoutingOperation
	if err := r.db.WithContext(ctx).
		Joins("JOIN process_parameters pp ON pp.id = routing_operations.process_id").
		Where("routing_header_id = ?", header.ID).
		Order("pp.sequence ASC, routing_operations.op_seq ASC").
		Find(&ops).Error; err != nil {
		return nil, nil, nil, apperror.InternalWrap("GetRoutingOps", err)
	}

	var opIDs []int64
	for _, o := range ops {
		opIDs = append(opIDs, o.ID)
	}
	var toolings []models.RoutingOperationTooling
	if len(opIDs) > 0 {
		if err := r.db.WithContext(ctx).Where("routing_operation_id IN ?", opIDs).Find(&toolings).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), `relation "routing_operation_toolings" does not exist`) {
				return &header, ops, nil, nil
			}
			return nil, nil, nil, apperror.InternalWrap("GetToolings", err)
		}
	}
	return &header, ops, toolings, nil
}

func (r *repository) GetSupplierName(ctx context.Context, id uuid.UUID) string {
	var s models.Supplier
	if err := r.db.WithContext(ctx).Select("supplier_name").Where("id = ?", id).First(&s).Error; err != nil {
		return ""
	}
	return s.SupplierName
}

func (r *repository) GetProcessName(ctx context.Context, id int64) string {
	var p models.ProcessParameter
	if err := r.db.WithContext(ctx).Select("process_name").Where("id = ?", id).First(&p).Error; err != nil {
		return ""
	}
	return p.ProcessName
}

func (r *repository) GetMachineName(ctx context.Context, id int64) string {
	var m models.MasterMachine
	if err := r.db.WithContext(ctx).Select("machine_name").Where("id = ?", id).First(&m).Error; err != nil {
		return ""
	}
	return m.MachineName
}

func (r *repository) GetItemsByIDs(ctx context.Context, ids []int64) ([]models.Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []models.Item
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&items).Error; err != nil {
		return nil, apperror.InternalWrap("GetItemsByIDs", err)
	}
	return items, nil
}

func (r *repository) GetLatestRevisionsByItemIDs(ctx context.Context, itemIDs []int64) ([]models.ItemRevision, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	sub := r.db.WithContext(ctx).
		Model(&models.ItemRevision{}).
		Select("MAX(id) AS id").
		Where("item_id IN ?", itemIDs).
		Group("item_id")

	var revisions []models.ItemRevision
	if err := r.db.WithContext(ctx).
		Where("id IN (?)", sub).
		Find(&revisions).Error; err != nil {
		return nil, apperror.InternalWrap("GetLatestRevisionsByItemIDs", err)
	}
	return revisions, nil
}

func (r *repository) GetFirstAssetsByItemIDs(ctx context.Context, itemIDs []int64) ([]models.ItemAsset, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	sub := r.db.WithContext(ctx).
		Model(&models.ItemAsset{}).
		Select("MIN(id) AS id").
		Where("item_id IN ? AND status = 'Active'", itemIDs).
		Group("item_id")

	var assets []models.ItemAsset
	if err := r.db.WithContext(ctx).
		Where("id IN (?)", sub).
		Find(&assets).Error; err != nil {
		return nil, apperror.InternalWrap("GetFirstAssetsByItemIDs", err)
	}
	return assets, nil
}

func (r *repository) GetMaterialSpecsByRevisionIDs(ctx context.Context, revisionIDs []int64) ([]models.ItemMaterialSpec, error) {
	if len(revisionIDs) == 0 {
		return nil, nil
	}
	var specs []models.ItemMaterialSpec
	if err := r.db.WithContext(ctx).
		Where("item_revision_id IN ?", revisionIDs).
		Find(&specs).Error; err != nil {
		return nil, apperror.InternalWrap("GetMaterialSpecsByRevisionIDs", err)
	}
	return specs, nil
}

func (r *repository) GetBomLinesByBomIDs(ctx context.Context, bomItemIDs []int64) ([]models.BomLine, error) {
	if len(bomItemIDs) == 0 {
		return nil, nil
	}
	var lines []models.BomLine
	if err := r.db.WithContext(ctx).
		Where("bom_item_id IN ?", bomItemIDs).
		Order("bom_item_id ASC, level ASC, id ASC").
		Find(&lines).Error; err != nil {
		return nil, apperror.InternalWrap("GetBomLinesByBomIDs", err)
	}
	return lines, nil
}

func (r *repository) GetLatestRoutingHeadersByItemIDs(ctx context.Context, itemIDs []int64) ([]models.RoutingHeader, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	sub := r.db.WithContext(ctx).
		Model(&models.RoutingHeader{}).
		Select("MAX(id) AS id").
		Where("item_id IN ?", itemIDs).
		Group("item_id")

	var headers []models.RoutingHeader
	if err := r.db.WithContext(ctx).
		Where("id IN (?)", sub).
		Find(&headers).Error; err != nil {
		return nil, apperror.InternalWrap("GetLatestRoutingHeadersByItemIDs", err)
	}
	return headers, nil
}

func (r *repository) GetRoutingOperationsByHeaderIDs(ctx context.Context, headerIDs []int64) ([]models.RoutingOperation, error) {
	if len(headerIDs) == 0 {
		return nil, nil
	}
	var ops []models.RoutingOperation
	if err := r.db.WithContext(ctx).
		Joins("JOIN process_parameters pp ON pp.id = routing_operations.process_id").
		Where("routing_header_id IN ?", headerIDs).
		Order("routing_header_id ASC, pp.sequence ASC, routing_operations.op_seq ASC").
		Find(&ops).Error; err != nil {
		return nil, apperror.InternalWrap("GetRoutingOperationsByHeaderIDs", err)
	}
	return ops, nil
}

func (r *repository) GetToolingsByOperationIDs(ctx context.Context, operationIDs []int64) ([]models.RoutingOperationTooling, error) {
	if len(operationIDs) == 0 {
		return nil, nil
	}
	var toolings []models.RoutingOperationTooling
	if err := r.db.WithContext(ctx).
		Where("routing_operation_id IN ?", operationIDs).
		Find(&toolings).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), `relation "routing_operation_toolings" does not exist`) {
			return nil, nil
		}
		return nil, apperror.InternalWrap("GetToolingsByOperationIDs", err)
	}
	return toolings, nil
}

func (r *repository) GetSupplierNamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	if len(ids) == 0 {
		return result, nil
	}
	var suppliers []models.Supplier
	if err := r.db.WithContext(ctx).
		Select("id", "supplier_name").
		Where("id IN ?", ids).
		Find(&suppliers).Error; err != nil {
		return nil, apperror.InternalWrap("GetSupplierNamesByIDs", err)
	}
	for _, supplier := range suppliers {
		result[supplier.ID] = supplier.SupplierName
	}
	return result, nil
}

func (r *repository) GetProcessNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	if len(ids) == 0 {
		return result, nil
	}
	var processes []models.ProcessParameter
	if err := r.db.WithContext(ctx).
		Select("id", "process_name").
		Where("id IN ?", ids).
		Find(&processes).Error; err != nil {
		return nil, apperror.InternalWrap("GetProcessNamesByIDs", err)
	}
	for _, process := range processes {
		result[process.ID] = process.ProcessName
	}
	return result, nil
}

func (r *repository) GetProcessSequencesByIDs(ctx context.Context, ids []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(ids) == 0 {
		return result, nil
	}
	var processes []models.ProcessParameter
	if err := r.db.WithContext(ctx).
		Select("id", "sequence").
		Where("id IN ?", ids).
		Find(&processes).Error; err != nil {
		return nil, apperror.InternalWrap("GetProcessSequencesByIDs", err)
	}
	for _, p := range processes {
		result[p.ID] = p.Sequence
	}
	return result, nil
}

func (r *repository) GetMachineNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string)
	if len(ids) == 0 {
		return result, nil
	}
	var machines []models.MasterMachine
	if err := r.db.WithContext(ctx).
		Select("id", "machine_name").
		Where("id IN ?", ids).
		Find(&machines).Error; err != nil {
		return nil, apperror.InternalWrap("GetMachineNamesByIDs", err)
	}
	for _, machine := range machines {
		result[machine.ID] = machine.MachineName
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// BOM list & detail
// ---------------------------------------------------------------------------

func (r *repository) ListBomItems(ctx context.Context, f ListFilter) ([]models.BomItem, int64, error) {
	needJoin := f.UniqCode != "" || f.Search != ""

	q := r.db.WithContext(ctx).Model(&models.BomItem{})
	if needJoin {
		q = q.Joins("JOIN items ON items.id = bom_item.item_id").
			Where("items.deleted_at IS NULL")
	}
	if f.Status != "" {
		q = q.Where("bom_item.status = ?", f.Status)
	}
	if f.UniqCode != "" {
		q = q.Where("items.uniq_code ILIKE ?", "%"+f.UniqCode+"%")
	}
	if f.Search != "" {
		q = q.Where("items.uniq_code ILIKE ? OR items.part_name ILIKE ?", "%"+f.Search+"%", "%"+f.Search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("ListBomItems count", err)
	}

	limit, offset := limitOffset(f.Limit, f.Page)
	orderClause := safeOrder(f.OrderBy, f.OrderDirection)

	var items []models.BomItem
	if err := q.Order(orderClause).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("ListBomItems", err)
	}
	return items, total, nil
}

func (r *repository) GetBomByID(ctx context.Context, bomID int64) (*models.BomItem, error) {
	var b models.BomItem
	err := r.db.WithContext(ctx).First(&b, bomID).Error
	return one(&b, err, "bom item")
}

func (r *repository) GetBomLines(ctx context.Context, bomItemID int64) ([]models.BomLine, error) {
	var lines []models.BomLine
	if err := r.db.WithContext(ctx).
		Where("bom_item_id = ?", bomItemID).
		Order("level ASC, id ASC").
		Find(&lines).Error; err != nil {
		return nil, apperror.InternalWrap("GetBomLines", err)
	}
	return lines, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// limitOffset returns SQL LIMIT and OFFSET from pagination input.
func limitOffset(limit, page int) (int, int) {
	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	return limit, (page - 1) * limit
}

// safeOrder builds an ORDER BY clause, whitelisting direction to prevent injection.
func safeOrder(col, dir string) string {
	if col == "" {
		col = "bom_item.id"
	}
	if dir != "asc" && dir != "desc" {
		dir = "asc"
	}
	return col + " " + dir
}

func one[T any](v *T, err error, entity string) (*T, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound(entity + " not found")
		}
		return nil, apperror.InternalWrap("get "+entity, err)
	}
	return v, nil
}
