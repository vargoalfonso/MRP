package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/pkg/apperror"

	"gorm.io/gorm"
)

type IRepository interface {
	// Training Run CRUD
	CreateTrainingRun(ctx context.Context, run *models.ForecastingTrainingRun) error
	FindTrainingRunByUUID(ctx context.Context, uuid string) (*models.ForecastingTrainingRun, error)
	FindTrainingRunByExternalID(ctx context.Context, trainingRunID string) (*models.ForecastingTrainingRun, error)
	FindTrainingRunByRequestID(ctx context.Context, requestID string) (*models.ForecastingTrainingRun, error)
	UpdateTrainingRun(ctx context.Context, run *models.ForecastingTrainingRun) error
	UpsertTrainingRunFromExternal(ctx context.Context, external *ExternalTrainingRunDetail) error
	ListTrainingRuns(ctx context.Context, filters TrainingRunFilters, pagination PaginationInput) ([]models.ForecastingTrainingRun, int64, error)

	// Inference Result CRUD
	CreateInferenceResult(ctx context.Context, result *models.ForecastingInferenceResult) error
	FindInferenceResultByUUID(ctx context.Context, uuid string) (*models.ForecastingInferenceResult, error)
	FindInferenceResultByRequestID(ctx context.Context, requestID string) (*models.ForecastingInferenceResult, error)
	ListInferenceResults(ctx context.Context, filters InferenceResultFilters, pagination PaginationInput) ([]models.ForecastingInferenceResult, int64, error)
}

// ExternalTrainingRunDetail is the shape returned by GET /admin/training-runs/{id}.
type ExternalTrainingRunDetail struct {
	RequestID      string  `json:"request_id"`
	Domain         string  `json:"domain"`
	Status         string  `json:"status"`
	TrainingRunID  string  `json:"training_run_id"`
	JobName        string  `json:"job_name"`
	Region         string  `json:"region"`
	Scope          string  `json:"scope"`
	Tenant         string  `json:"tenant"`
	Uniq           string  `json:"uniq"`
	DatasetID      string  `json:"dataset_id"`
	DatasetName    string  `json:"dataset_name"`
	DatasetVersion string  `json:"dataset_version"`
	SourceMode     string  `json:"source_mode"`
	GCSURI         string  `json:"gcs_uri"`
	RowCount       int64   `json:"row_count"`
	ItemCount      int64   `json:"item_count"`
	ModelVersionID *string `json:"model_version_id,omitempty"`
	OperationName  string  `json:"operation_name"`
	ErrorMessage   string  `json:"error_message,omitempty"`
}

type TrainingRunFilters struct {
	Scope   string
	Tenant  string
	Uniq    string
	Domain  string
	Status  string
}

type InferenceResultFilters struct {
	Domain         string
	Tenant         string
	ItemID         string
	Status         string
	ModelVersionID string
}

type PaginationInput struct {
	Limit  int
	Page   int
	Offset int
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

// ---------------------------------------------------------------------------
// Training Run
// ---------------------------------------------------------------------------

func (r *repository) CreateTrainingRun(ctx context.Context, run *models.ForecastingTrainingRun) error {
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return apperror.InternalWrap("create training run failed", err)
	}
	return nil
}

func (r *repository) FindTrainingRunByUUID(ctx context.Context, uuid string) (*models.ForecastingTrainingRun, error) {
	var run models.ForecastingTrainingRun
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("training run not found")
		}
		return nil, apperror.InternalWrap("find training run failed", err)
	}
	return &run, nil
}

func (r *repository) FindTrainingRunByExternalID(ctx context.Context, trainingRunID string) (*models.ForecastingTrainingRun, error) {
	var run models.ForecastingTrainingRun
	err := r.db.WithContext(ctx).Where("training_run_id = ?", trainingRunID).First(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("training run not found")
		}
		return nil, apperror.InternalWrap("find training run by external id failed", err)
	}
	return &run, nil
}

func (r *repository) FindTrainingRunByRequestID(ctx context.Context, requestID string) (*models.ForecastingTrainingRun, error) {
	var run models.ForecastingTrainingRun
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("training run not found")
		}
		return nil, apperror.InternalWrap("find training run by request id failed", err)
	}
	return &run, nil
}

func (r *repository) UpdateTrainingRun(ctx context.Context, run *models.ForecastingTrainingRun) error {
	if err := r.db.WithContext(ctx).Save(run).Error; err != nil {
		return apperror.InternalWrap("update training run failed", err)
	}
	return nil
}

func (r *repository) UpsertTrainingRunFromExternal(ctx context.Context, ext *ExternalTrainingRunDetail) error {
	var run models.ForecastingTrainingRun
	err := r.db.WithContext(ctx).Where("training_run_id = ?", ext.TrainingRunID).First(&run).Error
	if err == gorm.ErrRecordNotFound {
		// No local record — skip upsert (train trigger should have created it first)
		return nil
	}
	if err != nil {
		return apperror.InternalWrap("upsert training run failed", err)
	}

	run.Status = ext.Status
	run.JobName = ext.JobName
	run.Region = ext.Region
	run.OperationName = ext.OperationName
	run.ErrorMessage = ext.ErrorMessage
	if ext.ModelVersionID != nil {
		run.ModelVersionID = *ext.ModelVersionID
	}

	lastResp, _ := json.Marshal(ext)
	run.LastResponse = lastResp
	run.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Save(&run).Error
}

func (r *repository) ListTrainingRuns(ctx context.Context, filters TrainingRunFilters, pagination PaginationInput) ([]models.ForecastingTrainingRun, int64, error) {
	var runs []models.ForecastingTrainingRun
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ForecastingTrainingRun{})

	if filters.Scope != "" {
		query = query.Where("scope = ?", filters.Scope)
	}
	if filters.Tenant != "" {
		query = query.Where("tenant = ?", filters.Tenant)
	}
	if filters.Uniq != "" {
		query = query.Where("uniq = ?", filters.Uniq)
	}
	if filters.Domain != "" {
		query = query.Where("domain = ?", filters.Domain)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count training runs failed", err)
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if pagination.Offset > 0 {
		query = query.Offset(pagination.Offset)
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&runs).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list training runs failed", err)
	}

	return runs, total, nil
}

// ---------------------------------------------------------------------------
// Inference Result
// ---------------------------------------------------------------------------

func (r *repository) CreateInferenceResult(ctx context.Context, result *models.ForecastingInferenceResult) error {
	if err := r.db.WithContext(ctx).Create(result).Error; err != nil {
		return apperror.InternalWrap("create inference result failed", err)
	}
	return nil
}

func (r *repository) FindInferenceResultByUUID(ctx context.Context, uuid string) (*models.ForecastingInferenceResult, error) {
	var result models.ForecastingInferenceResult
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("inference result not found")
		}
		return nil, apperror.InternalWrap("find inference result failed", err)
	}
	return &result, nil
}

func (r *repository) FindInferenceResultByRequestID(ctx context.Context, requestID string) (*models.ForecastingInferenceResult, error) {
	var result models.ForecastingInferenceResult
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("inference result not found")
		}
		return nil, apperror.InternalWrap("find inference result by request id failed", err)
	}
	return &result, nil
}

func (r *repository) ListInferenceResults(ctx context.Context, filters InferenceResultFilters, pagination PaginationInput) ([]models.ForecastingInferenceResult, int64, error) {
	var results []models.ForecastingInferenceResult
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ForecastingInferenceResult{})

	if filters.Domain != "" {
		query = query.Where("domain = ?", filters.Domain)
	}
	if filters.Tenant != "" {
		query = query.Where("tenant = ?", filters.Tenant)
	}
	if filters.ItemID != "" {
		query = query.Where("item_id = ?", filters.ItemID)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.ModelVersionID != "" {
		query = query.Where("model_version_id = ?", filters.ModelVersionID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count inference results failed", err)
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if pagination.Offset > 0 {
		query = query.Offset(pagination.Offset)
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&results).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list inference results failed", err)
	}

	return results, total, nil
}
