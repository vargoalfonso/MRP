package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/import_file/models"
	"gorm.io/gorm"
)

type ImportRepository interface {
	BulkInsert(ctx context.Context, data []models.ImportData) error
}

type importRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) ImportRepository {
	return &importRepository{db: db}
}

func (r *importRepository) BulkInsert(ctx context.Context, data []models.ImportData) error {
	if len(data) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		CreateInBatches(data, 1000).
		Error
}
