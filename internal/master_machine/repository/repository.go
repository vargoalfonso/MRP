package repository

import (
	"context"
	"errors"

	"github.com/ganasa18/go-template/internal/master_machine/models"
	"gorm.io/gorm"
)

type IMasterMachineRepository interface {
	FindAll(ctx context.Context) ([]models.MasterMachine, error)
	FindByID(ctx context.Context, id int64) (*models.MasterMachine, error)
	Create(ctx context.Context, req models.CreateMachineRequest) (*models.MasterMachine, error)
	Update(ctx context.Context, id int64, req models.UpdateMachineRequest) (*models.MasterMachine, error)
	UpdateQR(ctx context.Context, id int64, qrDataURL string) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IMasterMachineRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.MasterMachine, error) {
	var data []models.MasterMachine
	err := r.db.WithContext(ctx).
		Order("machine_name ASC").
		Find(&data).Error
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.MasterMachine, error) {
	var data models.MasterMachine
	err := r.db.WithContext(ctx).First(&data, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("master machine not found")
		}
		return nil, err
	}
	return &data, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateMachineRequest) (*models.MasterMachine, error) {
	pid := req.ProcessID
	cap := req.MachineCapacity

	data := models.MasterMachine{
		MachineNumber:   req.MachineNumber,
		MachineName:     req.MachineName,
		ProductionLine:  req.ProductionLine,
		ProcessID:       &pid,
		MachineCapacity: &cap,
		Status:          req.Status,
	}

	if err := r.db.WithContext(ctx).Create(&data).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateMachineRequest) (*models.MasterMachine, error) {
	var data models.MasterMachine
	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("master machine not found")
		}
		return nil, err
	}

	pid := req.ProcessID
	cap := req.MachineCapacity

	updateData := map[string]interface{}{
		"machine_number":   req.MachineNumber,
		"machine_name":     req.MachineName,
		"production_line":  req.ProductionLine,
		"process_id":       pid,
		"machine_capacity": cap,
		"status":           req.Status,
	}

	if err := r.db.WithContext(ctx).Model(&data).Updates(updateData).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) UpdateQR(ctx context.Context, id int64, qrDataURL string) error {
	return r.db.WithContext(ctx).
		Model(&models.MasterMachine{}).
		Where("id = ?", id).
		Update("qr_image_base64", qrDataURL).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&models.MasterMachine{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("master machine not found")
	}
	return nil
}
