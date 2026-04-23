package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"gorm.io/gorm"
)

type IIncomingRepository interface {
	InsertScan(ctx context.Context, scan *models.IncomingReceivingScan) error
	IsIdempotentExist(ctx context.Context, key string) (bool, error)
	GetDNItem(ctx context.Context, id int64) (*models.DeliveryNoteItem, error)
	CreateQCTask(ctx context.Context, qc *models.QCTask) error
	AttachQCToScan(ctx context.Context, scanID int64, qcID int64) error
}

type incomingRepository struct {
	db *gorm.DB
}

func NewIncomingRepository(db *gorm.DB) IIncomingRepository {
	return &incomingRepository{db: db}
}

func (r *incomingRepository) InsertScan(ctx context.Context, scan *models.IncomingReceivingScan) error {
	return r.db.WithContext(ctx).Create(scan).Error
}

func (r *incomingRepository) IsIdempotentExist(ctx context.Context, key string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.IncomingReceivingScan{}).
		Where("idempotency_key = ?", key).
		Count(&count).Error

	return count > 0, err
}

func (r *incomingRepository) GetDNItem(ctx context.Context, id int64) (*models.DeliveryNoteItem, error) {
	var item models.DeliveryNoteItem
	err := r.db.WithContext(ctx).First(&item, id).Error
	return &item, err
}

func (r *incomingRepository) CreateQCTask(ctx context.Context, qc *models.QCTask) error {
	return r.db.WithContext(ctx).Create(qc).Error
}

func (r *incomingRepository) AttachQCToScan(ctx context.Context, scanID int64, qcID int64) error {
	return r.db.WithContext(ctx).
		Model(&models.IncomingReceivingScan{}).
		Where("id = ?", scanID).
		Update("qc_task_id", qcID).Error
}
