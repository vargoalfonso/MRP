package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ganasa18/go-template/internal/employee/models"
	"gorm.io/gorm"
)

type IEmployeeRepository interface {
	FindAll(ctx context.Context) ([]models.Employee, error)
	FindByID(ctx context.Context, id int64) (*models.Employee, error)

	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error)
	Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error)
	Delete(ctx context.Context, id int64) error

	IsEmailExist(ctx context.Context, email string) (bool, error)
	SaveActivationToken(ctx context.Context, token *models.UserActivation) error

	Tx(ctx context.Context, fn func(repo IEmployeeRepository) error) error
	GetTx() *gorm.DB
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IEmployeeRepository {
	return &repository{db: db}
}

func (r *repository) Tx(ctx context.Context, fn func(repo IEmployeeRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := &repository{db: tx}
		return fn(repoTx)
	})
}

func (r *repository) GetTx() *gorm.DB {
	return r.db
}

func (r *repository) FindAll(ctx context.Context) ([]models.Employee, error) {
	var employees []models.Employee

	err := r.db.WithContext(ctx).
		Order("id DESC").
		Find(&employees).Error

	if err != nil {
		return nil, err
	}

	return employees, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.Employee, error) {
	var employee models.Employee

	err := r.db.WithContext(ctx).
		First(&employee, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	var employee models.Employee

	// cek data
	if err := r.db.WithContext(ctx).
		First(&employee, id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("employee not found")
		}
		return nil, err
	}

	// mapping update
	updateData := map[string]interface{}{
		"full_name":     req.FullName,
		"email":         req.Email,
		"phone_number":  req.PhoneNumber,
		"job_title":     req.JobTitle,
		"unit_cost":     req.UnitCost,
		"join_date":     req.JoinDate,
		"role_id":       req.RoleID,
		"department_id": req.DepartmentID,
		"reports_to_id": req.ReportsToID,
		"status":        req.Status,
		"notes":         req.Notes,
	}

	if err := r.db.WithContext(ctx).
		Model(&employee).
		Updates(updateData).Error; err != nil {
		return nil, err
	}

	// ambil data terbaru
	if err := r.db.WithContext(ctx).
		First(&employee, id).Error; err != nil {
		return nil, err
	}

	return &employee, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).
		Delete(&models.Employee{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("employee not found")
	}

	return nil
}

func (r *repository) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {

	emp := models.Employee{
		FullName:     req.FullName,
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		JobTitle:     req.JobTitle,
		UnitCost:     req.UnitCost,
		JoinDate:     req.JoinDate,
		RoleID:       req.RoleID,
		DepartmentID: req.DepartmentID,
		ReportsToID:  req.ReportsToID,
		Status:       req.Status,
		Notes:        req.Notes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&emp).Error; err != nil {
		return nil, err
	}

	return &emp, nil
}

func (r *repository) IsEmailExist(ctx context.Context, email string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.Employee{}).
		Where("email = ?", email).
		Count(&count).Error

	return count > 0, err
}

func (r *repository) SaveActivationToken(ctx context.Context, token *models.UserActivation) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *repository) CreateUserTx(ctx context.Context, tx *gorm.DB, user *models.User) error {
	return tx.WithContext(ctx).Create(user).Error
}
