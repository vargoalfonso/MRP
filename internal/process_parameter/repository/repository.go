package repository

import (
	"context"
	"errors"

	"github.com/ganasa18/go-template/internal/process_parameter/models"
	"gorm.io/gorm"
)

type IProcessParameterRepository interface {
	FindAll(ctx context.Context) ([]models.ProcessParameter, error)
	FindByID(ctx context.Context, id int64) (*models.ProcessParameter, error)
	Create(ctx context.Context, req models.CreateProcessRequest) (*models.ProcessParameter, error)
	Update(ctx context.Context, id int64, req models.UpdateProcessRequest) (*models.ProcessParameter, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IProcessParameterRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.ProcessParameter, error) {
	var data []models.ProcessParameter

	err := r.db.WithContext(ctx).
		Order("sequence ASC").
		Find(&data).Error

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.ProcessParameter, error) {
	var data models.ProcessParameter

	err := r.db.WithContext(ctx).
		First(&data, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("process parameter not found")
		}
		return nil, err
	}

	return &data, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateProcessRequest) (*models.ProcessParameter, error) {
	data := models.ProcessParameter{
		ProcessCode: req.ProcessCode,
		ProcessName: req.ProcessName,
		Category:    req.Category,
		Sequence:    req.Sequence,
		Status:      req.Status,
	}

	if err := r.db.WithContext(ctx).
		Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateProcessRequest) (*models.ProcessParameter, error) {
	var data models.ProcessParameter

	// cek data
	if err := r.db.WithContext(ctx).
		First(&data, id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("process parameter not found")
		}
		return nil, err
	}

	// mapping update
	updateData := map[string]interface{}{
		"process_name": req.ProcessName,
		"category":     req.Category,
		"sequence":     req.Sequence,
		"status":       req.Status,
	}

	if err := r.db.WithContext(ctx).
		Model(&data).
		Updates(updateData).Error; err != nil {
		return nil, err
	}

	// ambil data terbaru
	if err := r.db.WithContext(ctx).
		First(&data, id).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).
		Delete(&models.ProcessParameter{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("process parameter not found")
	}

	return nil
}
