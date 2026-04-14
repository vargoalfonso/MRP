package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ganasa18/go-template/internal/approval_workflow/models"
	"gorm.io/gorm"
)

type IApprovalWorkflowRepository interface {
	FindAll(ctx context.Context) ([]models.ApprovalWorkflow, error)
	FindByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error)
	FindByActionName(ctx context.Context, actionName string) (*models.ApprovalWorkflow, error)
	Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	Delete(ctx context.Context, id int64) error
	CreateInstance(ctx context.Context, tx *gorm.DB, data *models.ApprovalInstance) error
	ResetInstancesByWorkflow(ctx context.Context, workflowID int64, progress models.ApprovalProgress, maxLevel int) error

	FindInstanceByID(ctx context.Context, id int64) (*models.ApprovalInstance, error)
	UpdateInstance(ctx context.Context, instance *models.ApprovalInstance) error
	UpdateReferenceStatus(ctx context.Context, table string, id int64, status string) error
	FindWorkflowByActionName(ctx context.Context, action string) (*models.ApprovalWorkflow, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IApprovalWorkflowRepository {
	return &repository{db: db}
}

func (r *repository) FindInstanceByID(ctx context.Context, id int64) (*models.ApprovalInstance, error) {
	var instance models.ApprovalInstance

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&instance).Error

	if err != nil {
		return nil, err
	}

	return &instance, nil
}

func (r *repository) UpdateInstance(ctx context.Context, instance *models.ApprovalInstance) error {

	return r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]interface{}{
			"approval_progress": instance.ApprovalProgress,
			"status":            instance.Status,
			"current_level":     instance.CurrentLevel,
			"updated_at":        time.Now(),
		}).Error
}

func (r *repository) UpdateReferenceStatus(ctx context.Context, table string, id int64, status string) error {

	query := fmt.Sprintf("UPDATE %s SET status = ? WHERE id = ?", table)

	return r.db.WithContext(ctx).
		Exec(query, status, id).Error
}

func (r *repository) FindWorkflowByActionName(ctx context.Context, action string) (*models.ApprovalWorkflow, error) {

	var workflow models.ApprovalWorkflow

	err := r.db.WithContext(ctx).
		Where("action_name = ?", action).
		First(&workflow).Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

func (r *repository) CreateInstance(ctx context.Context, tx *gorm.DB, data *models.ApprovalInstance) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *repository) FindByActionName(ctx context.Context, actionName string) (*models.ApprovalWorkflow, error) {
	var approvalWorkflow models.ApprovalWorkflow

	err := r.db.WithContext(ctx).
		Where("action_name = ?", actionName).
		First(&approvalWorkflow).Error

	if err != nil {
		return nil, err
	}

	return &approvalWorkflow, nil
}

func (r *repository) ResetInstancesByWorkflow(ctx context.Context, workflowID int64, progress models.ApprovalProgress, maxLevel int) error {

	return r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("approval_workflow_id = ?", workflowID).
		Updates(map[string]interface{}{
			"approval_progress": progress,
			"current_level":     1,
			"max_level":         maxLevel,
			"status":            "pending",
			"updated_at":        time.Now(),
		}).Error
}

func (r *repository) FindAll(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	var approvalWorkflows []models.ApprovalWorkflow

	err := r.db.WithContext(ctx).
		Order("id DESC").
		Find(&approvalWorkflows).Error

	if err != nil {
		return nil, err
	}

	return approvalWorkflows, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error) {
	var approvalWorkflow models.ApprovalWorkflow

	err := r.db.WithContext(ctx).
		First(&approvalWorkflow, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("approval workflow not found")
		}
		return nil, err
	}

	return &approvalWorkflow, nil
}

func (r *repository) Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {
	approvalWorkflow := models.ApprovalWorkflow{
		ActionName: req.ActionName,
		Level1Role: req.Level1Role,
		Level2Role: req.Level2Role,
		Level3Role: req.Level3Role,
		Level4Role: req.Level4Role,
		Status:     req.Status,
		CreatedBy:  req.CreatedBy,
	}

	if err := r.db.WithContext(ctx).
		Create(&approvalWorkflow).Error; err != nil {
		return nil, err
	}

	return &approvalWorkflow, nil
}

func (r *repository) Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {
	var data models.ApprovalWorkflow

	// cek data
	if err := r.db.WithContext(ctx).
		First(&data, id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("approval workflow not found")
		}
		return nil, err
	}

	// mapping update
	updateData := map[string]interface{}{
		"action_name":  req.ActionName,
		"level_1_role": req.Level1Role,
		"level_2_role": req.Level2Role,
		"level_3_role": req.Level3Role,
		"level_4_role": req.Level4Role,
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
		Delete(&models.ApprovalWorkflow{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("approval workflow not found")
	}

	return nil
}
