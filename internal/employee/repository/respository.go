package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/employee/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IRepository is the employee repository contract.
type IRepository interface {
	GetAllDataEmployee(ctx context.Context) ([]models.EmployeeResp, error)
	GetDataEmployeeByID(ctx context.Context, employeeID string) (models.EmployeeResp, error)
	InsertEmployee(ctx context.Context, req models.EmployeeReq) error
	UpdateDataEmployee(ctx context.Context, req models.EmployeeReq) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository returns an IRepository backed by the provided *gorm.DB.
func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) GetAllDataEmployee(ctx context.Context) ([]models.EmployeeResp, error) {
	var employees []models.Employee
	if err := r.db.WithContext(ctx).Find(&employees).Error; err != nil {
		return nil, apperror.InternalWrap("GetAllDataEmployee failed", err)
	}

	resp := make([]models.EmployeeResp, 0, len(employees))
	for _, e := range employees {
		resp = append(resp, toResp(e))
	}
	return resp, nil
}

func (r *repository) GetDataEmployeeByID(ctx context.Context, employeeID string) (models.EmployeeResp, error) {
	var employee models.Employee
	err := r.db.WithContext(ctx).Where("employee_id = ?", employeeID).First(&employee).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.EmployeeResp{}, apperror.NotFound("employee not found")
		}
		return models.EmployeeResp{}, apperror.InternalWrap("GetDataEmployeeByID failed", err)
	}
	return toResp(employee), nil
}

func (r *repository) InsertEmployee(ctx context.Context, req models.EmployeeReq) error {
	employee := models.Employee{
		EmployeeID:  uuid.New().String(),
		Name:        req.Name,
		CompanyName: req.CompanyName,
		Email:       req.Email,
		Address:     req.Address,
	}
	if err := r.db.WithContext(ctx).Create(&employee).Error; err != nil {
		return apperror.InternalWrap("InsertEmployee failed", err)
	}
	return nil
}

func (r *repository) UpdateDataEmployee(ctx context.Context, req models.EmployeeReq) error {
	result := r.db.WithContext(ctx).
		Model(&models.Employee{}).
		Where("employee_id = ?", req.EmployeeID).
		Updates(map[string]interface{}{
			"name":         req.Name,
			"company_name": req.CompanyName,
			"email":        req.Email,
			"address":      req.Address,
		})
	if result.Error != nil {
		return apperror.InternalWrap("UpdateDataEmployee failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("employee not found")
	}
	return nil
}

func toResp(e models.Employee) models.EmployeeResp {
	return models.EmployeeResp{
		EmployeeID:  e.EmployeeID,
		Name:        e.Name,
		CompanyName: e.CompanyName,
		Email:       e.Email,
		Address:     e.Address,
	}
}
