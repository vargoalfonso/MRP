package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/departement/models"
	"gorm.io/gorm"
)

type IDepartementRepository interface {
	FindAll(ctx context.Context) ([]models.Department, error)
	FindByID(ctx context.Context, id int64) (*models.Department, error)
	Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error)
	Update(ctx context.Context, id int64, req models.UpdateDepartmentRequest) (*models.Department, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IDepartementRepository {
	return &repository{db: db}
}

func (r *repository) FindAll(ctx context.Context) ([]models.Department, error) {
	var departments []models.Department

	// ambil semua department
	err := r.db.WithContext(ctx).Find(&departments).Error
	if err != nil {
		return nil, err
	}

	// bikin map id → department
	deptMap := make(map[int64]models.Department)
	for _, d := range departments {
		deptMap[d.ID] = d
	}

	// assign parent
	for i := range departments {
		if departments[i].ParentDepartmentID != nil {
			parent, ok := deptMap[*departments[i].ParentDepartmentID]
			if ok {
				// ambil field yang perlu aja (optional)
				departments[i].Parent = &models.Department{
					ID:             parent.ID,
					DepartmentCode: parent.DepartmentCode,
					DepartmentName: parent.DepartmentName,
					Status:         parent.Status,
				}
			}
		}
	}

	return departments, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.Department, error) {
	var department models.Department

	// ambil department utama
	err := r.db.WithContext(ctx).First(&department, id).Error
	if err != nil {
		return nil, err
	}

	if department.ParentDepartmentID != nil {
		var parent models.Department

		err := r.db.WithContext(ctx).
			Select("id", "department_name").
			First(&parent, *department.ParentDepartmentID).Error
		if err != nil {
			return nil, err
		}

		department.Parent = &parent
	}

	return &department, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error) {
	department := models.Department{
		DepartmentCode:     req.DepartmentCode,
		DepartmentName:     req.DepartmentName,
		Description:        req.Description,
		ManagerID:          req.ManagerID,
		ParentDepartmentID: req.ParentDepartmentID,
		Status:             req.Status,
	}

	err := r.db.WithContext(ctx).Create(&department).Error
	if err != nil {
		return nil, err
	}

	return &department, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateDepartmentRequest) (*models.Department, error) {
	var department models.Department

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&department).Error
	if err != nil {
		return nil, err
	}

	department.DepartmentCode = req.DepartmentCode
	department.DepartmentName = req.DepartmentName
	department.Description = req.Description
	department.ManagerID = req.ManagerID
	department.ParentDepartmentID = req.ParentDepartmentID
	department.Status = req.Status

	err = r.db.WithContext(ctx).Save(&department).Error
	if err != nil {
		return nil, err
	}

	return &department, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Department{}).Error
}
