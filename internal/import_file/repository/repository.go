package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/import_file/models"
	"gorm.io/gorm"
)

type ImportRepository interface {
	GetLatestCustomerByName(ctx context.Context, name string) (*models.Customer, error)
	CountPRL(ctx context.Context) (int64, error)
	InsertPRL(ctx context.Context, prl *models.PRL) error
	InsertPRLBulk(ctx context.Context, prls []models.PRL) error
	GetItemByUniqCode(ctx context.Context, uniqCode string) (*models.Item, error)
	GetMaxPRLNumber(ctx context.Context, year string) (int64, error)
	GetAllSuppliers(ctx context.Context) ([]map[string]interface{}, error)
	GetAllCustomers(ctx context.Context) ([]map[string]interface{}, error)
	GetAllSupplierItems(ctx context.Context) ([]map[string]interface{}, error)
	GetAllKanban(ctx context.Context) ([]models.KanbanParameter, error)
	IsKanbanExist(ctx context.Context, uniqCode string) (bool, error)
	CreateKanban(ctx context.Context, data *models.KanbanParameter) error
	CountKanban(ctx context.Context) (int64, error)
	GetExistingItemUniqCodes(ctx context.Context, codes []string) (map[string]bool, error)
	GetExistingKanbanItemCodes(ctx context.Context, codes []string) (map[string]bool, error)
	CreateKanbanBatch(ctx context.Context, data []models.KanbanParameter) error
}

type importRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) ImportRepository {
	return &importRepository{db: db}
}

func (r *importRepository) GetLatestCustomerByName(ctx context.Context, name string) (*models.Customer, error) {
	var customer models.Customer

	err := r.db.WithContext(ctx).
		Where("customer_name = ? AND deleted_at IS NULL", name).
		Order("created_at DESC").
		Limit(1).
		Find(&customer).Error

	if err != nil {
		return nil, err
	}

	// kalau tidak ditemukan
	if customer.ID == 0 {
		return nil, nil
	}

	return &customer, nil
}

func (r *importRepository) CountPRL(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.PRL{}).
		Where("deleted_at IS NULL").
		Count(&count).Error

	return count, err
}

func (r *importRepository) InsertPRL(ctx context.Context, prl *models.PRL) error {
	return r.db.WithContext(ctx).
		Create(prl).
		Error
}

func (r *importRepository) InsertPRLBulk(ctx context.Context, prls []models.PRL) error {
	if len(prls) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		CreateInBatches(prls, 500).
		Error
}

func (r *importRepository) GetItemByUniqCode(ctx context.Context, uniqCode string) (*models.Item, error) {
	var item models.Item

	err := r.db.WithContext(ctx).
		Where("uniq_code = ? AND deleted_at IS NULL", uniqCode).
		Order("created_at DESC").
		First(&item).Error

	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *importRepository) GetMaxPRLNumber(ctx context.Context, year string) (int64, error) {
	var max int64

	err := r.db.WithContext(ctx).
		Raw(`
			SELECT COALESCE(MAX(
				CAST(SPLIT_PART(prl_id, '-', 4) AS BIGINT)
			), 0)
			FROM prls
			WHERE prl_id LIKE ?
			AND SPLIT_PART(prl_id, '-', 4) ~ '^[0-9]+$'
		`, "PRL-"+year+"-%").
		Scan(&max).Error

	return max, err
}

func (r *importRepository) GetAllSuppliers(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	err := r.db.WithContext(ctx).
		Raw(`
			SELECT ROW_NUMBER() OVER (ORDER BY id) as no, supplier_name
			FROM suppliers
			WHERE status = 'Active' AND deleted_at IS NULL
			ORDER BY supplier_name
		`).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *importRepository) GetAllCustomers(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	err := r.db.WithContext(ctx).
		Raw(`
            SELECT ROW_NUMBER() OVER (ORDER BY id) as no, customer_name
            FROM customers
			WHERE deleted_at IS NULL
            ORDER BY customer_name
        `).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *importRepository) GetAllSupplierItems(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	err := r.db.WithContext(ctx).
		Raw(`
			SELECT ROW_NUMBER() OVER (ORDER BY id) as no, uniq_code as product_name
			FROM supplier_item
			WHERE status = 'active' AND deleted_at IS NULL
			ORDER BY uniq_code
		`).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *importRepository) GetAllKanban(ctx context.Context) ([]models.KanbanParameter, error) {
	var data []models.KanbanParameter

	err := r.db.WithContext(ctx).
		Order("kanban_number asc").
		Find(&data).Error

	return data, err
}

func (r *importRepository) IsKanbanExist(ctx context.Context, uniqCode string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.KanbanParameter{}).
		Where("item_uniq_code = ?", uniqCode).
		Count(&count).Error

	return count > 0, err
}

func (r *importRepository) CreateKanban(ctx context.Context, data *models.KanbanParameter) error {
	return r.db.WithContext(ctx).
		Create(data).
		Error
}

func (r *importRepository) CountKanban(ctx context.Context) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&models.KanbanParameter{}).
		Count(&total).Error

	return total, err
}

// GetExistingItemUniqCodes returns a set of item uniq codes (from the given
// list) that exist in the item master. Uses a single IN query instead of one
// query per row, so bulk imports do not run thousands of sequential queries.
func (r *importRepository) GetExistingItemUniqCodes(ctx context.Context, codes []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(codes) == 0 {
		return result, nil
	}

	var found []string
	if err := r.db.WithContext(ctx).
		Model(&models.Item{}).
		Where("uniq_code IN ? AND deleted_at IS NULL", codes).
		Distinct().
		Pluck("uniq_code", &found).Error; err != nil {
		return nil, err
	}

	for _, c := range found {
		result[c] = true
	}
	return result, nil
}

// GetExistingKanbanItemCodes returns a set of item uniq codes (from the given
// list) that already have a kanban parameter, using a single IN query.
func (r *importRepository) GetExistingKanbanItemCodes(ctx context.Context, codes []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(codes) == 0 {
		return result, nil
	}

	var found []string
	if err := r.db.WithContext(ctx).
		Model(&models.KanbanParameter{}).
		Where("item_uniq_code IN ?", codes).
		Distinct().
		Pluck("item_uniq_code", &found).Error; err != nil {
		return nil, err
	}

	for _, c := range found {
		result[c] = true
	}
	return result, nil
}

// CreateKanbanBatch inserts kanban parameters in batches (single round trip per
// batch) instead of one INSERT per row.
func (r *importRepository) CreateKanbanBatch(ctx context.Context, data []models.KanbanParameter) error {
	if len(data) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(data, 200).Error
}
