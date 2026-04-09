package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/access_control/models"
	"gorm.io/gorm"
)

type IACMRepository interface {
	FindAll(ctx context.Context) ([]models.AccessControlMatrix, error)
	FindByID(ctx context.Context, id int64) (*models.AccessControlMatrix, error)
	Create(ctx context.Context, req models.CreateACMRequest) (*models.AccessControlMatrix, error)
	Update(ctx context.Context, id int64, req models.UpdateACMRequest) (*models.AccessControlMatrix, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IACMRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.AccessControlMatrix, error) {
	var data []models.AccessControlMatrix

	err := r.db.WithContext(ctx).Find(&data).Error
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.AccessControlMatrix, error) {
	var data models.AccessControlMatrix

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&data).Error

	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateACMRequest) (*models.AccessControlMatrix, error) {
	data := models.AccessControlMatrix{
		FullName:     req.FullName,
		EmployeeID:   req.EmployeeID,
		RoleID:       req.RoleID,
		DepartmentID: req.DepartmentID,
		Status:       req.Status,
	}

	err := r.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateACMRequest) (*models.AccessControlMatrix, error) {
	var data models.AccessControlMatrix

	err := r.db.WithContext(ctx).First(&data, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	data.FullName = req.FullName
	data.EmployeeID = req.EmployeeID
	data.RoleID = req.RoleID
	data.DepartmentID = req.DepartmentID
	data.Status = req.Status

	err = r.db.WithContext(ctx).Save(&data).Error
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.AccessControlMatrix{}).Error
}
