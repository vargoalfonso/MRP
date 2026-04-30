package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ForecastingTrainingRun mirrors the forecasting_training_runs table.
type ForecastingTrainingRun struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID            string          `gorm:"uniqueIndex;not null" json:"id"`
	RequestID       string          `gorm:"column:request_id;not null" json:"request_id"`
	TrainingRunID   string          `gorm:"column:training_run_id" json:"training_run_id"`
	Domain          string          `gorm:"not null" json:"domain"`
	Scope           string          `gorm:"not null" json:"scope"`
	Tenant          string          `gorm:"column:tenant" json:"tenant,omitempty"`
	Uniq            string          `gorm:"column:uniq" json:"uniq,omitempty"`
	DatasetID       string          `gorm:"column:dataset_id" json:"dataset_id,omitempty"`
	DatasetName     string          `gorm:"column:dataset_name" json:"dataset_name,omitempty"`
	DatasetVersion  string          `gorm:"column:dataset_version" json:"dataset_version,omitempty"`
	SourceMode      string          `gorm:"column:source_mode" json:"source_mode,omitempty"`
	GCSURI          string          `gorm:"column:gcs_uri" json:"gcs_uri,omitempty"`
	RowCount        int64           `gorm:"column:row_count" json:"row_count,omitempty"`
	ItemCount       int64           `gorm:"column:item_count" json:"item_count,omitempty"`
	FineTune        bool            `gorm:"not null;default:false" json:"fine_tune"`
	TimeLimit       int             `gorm:"column:time_limit" json:"time_limit,omitempty"`
	Presets         string          `gorm:"column:presets" json:"presets,omitempty"`
	JobName         string          `gorm:"column:job_name" json:"job_name,omitempty"`
	Region          string          `gorm:"column:region" json:"region,omitempty"`
	OperationName   string          `gorm:"column:operation_name" json:"operation_name,omitempty"`
	Status          string          `gorm:"not null;default:'PENDING'" json:"status"`
	ModelVersionID  string          `gorm:"column:model_version_id" json:"model_version_id,omitempty"`
	LastResponse    json.RawMessage `gorm:"column:last_response;type:jsonb" json:"last_response,omitempty"`
	ErrorMessage    string          `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedBy       string          `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"column:deleted_at;index" json:"-"`
}

// TableName returns the table name for GORM.
func (ForecastingTrainingRun) TableName() string {
	return "forecasting_training_runs"
}

// ForecastingInferenceResult mirrors the forecasting_inference_results table.
type ForecastingInferenceResult struct {
	ID               int64           `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID             string         `gorm:"uniqueIndex;not null" json:"id"`
	RequestID        string         `gorm:"column:request_id;not null" json:"request_id"`
	Domain           string         `gorm:"not null" json:"domain"`
	Tenant           string         `gorm:"column:tenant" json:"tenant,omitempty"`
	ItemID           string         `gorm:"column:item_id" json:"item_id,omitempty"`
	ModelVersionID   string         `gorm:"column:model_version_id;not null" json:"model_version_id"`
	Horizon          int            `gorm:"not null" json:"horizon"`
	LookbackPoints   int            `gorm:"column:lookback_points" json:"lookback_points,omitempty"`
	Mode             string         `gorm:"not null" json:"mode"`
	RequestPayload   json.RawMessage `gorm:"column:request_payload;type:jsonb;not null" json:"request_payload"`
	ResponsePayload  json.RawMessage `gorm:"column:response_payload;type:jsonb;not null" json:"response_payload"`
	Status           string         `gorm:"not null" json:"status"`
	ErrorMessage     string         `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedBy        string         `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt        time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName returns the table name for GORM.
func (ForecastingInferenceResult) TableName() string {
	return "forecasting_inference_results"
}
