package models

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// Training Run List / Detail Response
// ---------------------------------------------------------------------------

type TrainingRunListResponse struct {
	Items      []TrainingRunResponse `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type TrainingRunResponse struct {
	ID              string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	TrainingRunID  string                 `json:"training_run_id,omitempty"`
	Domain         string                 `json:"domain"`
	Scope          string                 `json:"scope"`
	Tenant         string                 `json:"tenant,omitempty"`
	Uniq           string                 `json:"uniq,omitempty"`
	DatasetID      string                 `json:"dataset_id,omitempty"`
	DatasetName    string                 `json:"dataset_name,omitempty"`
	DatasetVersion string                 `json:"dataset_version,omitempty"`
	SourceMode     string                 `json:"source_mode,omitempty"`
	GCSURI         string                 `json:"gcs_uri,omitempty"`
	RowCount       int64                  `json:"row_count,omitempty"`
	ItemCount      int64                  `json:"item_count,omitempty"`
	FineTune       bool                   `json:"fine_tune"`
	TimeLimit      int                    `json:"time_limit,omitempty"`
	Presets        string                 `json:"presets,omitempty"`
	JobName        string                 `json:"job_name,omitempty"`
	Region         string                 `json:"region,omitempty"`
	OperationName  string                 `json:"operation_name,omitempty"`
	Status         string                 `json:"status"`
	ModelVersionID string                 `json:"model_version_id,omitempty"`
	LastResponse   map[string]interface{} `json:"last_response,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Inference Result List / Detail Response
// ---------------------------------------------------------------------------

type InferenceResultListResponse struct {
	Items      []InferenceResultResponse `json:"items"`
	Pagination PaginationMeta           `json:"pagination"`
}

type InferenceResultResponse struct {
	ID               string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	Domain          string                 `json:"domain"`
	Tenant          string                 `json:"tenant,omitempty"`
	ItemID          string                 `json:"item_id,omitempty"`
	ModelVersionID  string                 `json:"model_version_id"`
	Horizon         int                    `json:"horizon"`
	LookbackPoints  int                    `json:"lookback_points,omitempty"`
	Mode            string                 `json:"mode"`
	AvgMean         float64                `json:"avg_mean"` // Calculated from forecasts mean
	RequestPayload  map[string]interface{} `json:"request_payload"`
	ResponsePayload map[string]interface{} `json:"response_payload"`
	Status          string                 `json:"status"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedBy       string                 `json:"created_by"`
	CreatedAt       time.Time              `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Predict Response (mirrors external API)
// ---------------------------------------------------------------------------

type PredictResponse struct {
	RequestID      string        `json:"request_id"`
	ModelVersionID string        `json:"model_version_id"`
	Forecasts      []ForecastDTO `json:"forecasts"`
	AvgMean        float64       `json:"avg_mean"` // Average of all forecast means
}

type ForecastDTO struct {
	ItemID    string  `json:"item_id"`
	Timestamp string  `json:"timestamp"`
	Mean      float64 `json:"mean"`
	P10       float64 `json:"0.1,omitempty"`
	P50       float64 `json:"0.5,omitempty"`
	P90       float64 `json:"0.9,omitempty"`
}

// ---------------------------------------------------------------------------
// Promote Response (mirrors external API)
// ---------------------------------------------------------------------------

type PromoteResponse struct {
	RequestID      string `json:"request_id"`
	Domain         string `json:"domain"`
	ModelVersionID string `json:"model_version_id"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	DeployedAt     string `json:"deployed_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Proxy-only responses (forwarded from external)
// ---------------------------------------------------------------------------

type ProxyListResponse struct {
	Items []map[string]interface{} `json:"items"`
}

// ---------------------------------------------------------------------------
// Pagination Meta
// ---------------------------------------------------------------------------

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginationMeta builds pagination meta from count + input.
func NewPaginationMeta(total int64, page, limit int) PaginationMeta {
	pages := 0
	if limit > 0 {
		pages = int((total + int64(limit) - 1) / int64(limit))
	}
	return PaginationMeta{
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: pages,
	}
}

// ---------------------------------------------------------------------------
// Helpers to convert DB models to response DTOs
// ---------------------------------------------------------------------------

func ToTrainingRunResponse(run *ForecastingTrainingRun) TrainingRunResponse {
	resp := TrainingRunResponse{
		ID:              run.UUID,
		RequestID:      run.RequestID,
		TrainingRunID:  run.TrainingRunID,
		Domain:         run.Domain,
		Scope:          run.Scope,
		Tenant:         run.Tenant,
		Uniq:           run.Uniq,
		DatasetID:      run.DatasetID,
		DatasetName:    run.DatasetName,
		DatasetVersion: run.DatasetVersion,
		SourceMode:     run.SourceMode,
		GCSURI:         run.GCSURI,
		RowCount:       run.RowCount,
		ItemCount:      run.ItemCount,
		FineTune:       run.FineTune,
		TimeLimit:      run.TimeLimit,
		Presets:        run.Presets,
		JobName:        run.JobName,
		Region:         run.Region,
		OperationName:  run.OperationName,
		Status:         run.Status,
		ModelVersionID: run.ModelVersionID,
		ErrorMessage:   run.ErrorMessage,
		CreatedBy:      run.CreatedBy,
		CreatedAt:      run.CreatedAt,
		UpdatedAt:      run.UpdatedAt,
	}
	if len(run.LastResponse) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(run.LastResponse, &m); err == nil {
			resp.LastResponse = m
		}
	}
	return resp
}

func ToInferenceResultResponse(result *ForecastingInferenceResult) InferenceResultResponse {
	resp := InferenceResultResponse{
		ID:              result.UUID,
		RequestID:       result.RequestID,
		Domain:          result.Domain,
		Tenant:          result.Tenant,
		ItemID:          result.ItemID,
		ModelVersionID:  result.ModelVersionID,
		Horizon:         result.Horizon,
		LookbackPoints:  result.LookbackPoints,
		Mode:            result.Mode,
		Status:          result.Status,
		ErrorMessage:    result.ErrorMessage,
		CreatedBy:       result.CreatedBy,
		CreatedAt:       result.CreatedAt,
	}
	if len(result.RequestPayload) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(result.RequestPayload, &m); err == nil {
			resp.RequestPayload = m
		}
	}
	if len(result.ResponsePayload) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(result.ResponsePayload, &m); err == nil {
			resp.ResponsePayload = m
			// Calculate avg_mean from forecasts
			if forecasts, ok := m["forecasts"].([]interface{}); ok {
				var totalMean float64
				for _, f := range forecasts {
					if forecast, ok := f.(map[string]interface{}); ok {
						if mean, ok := forecast["mean"].(float64); ok {
							totalMean += mean
						}
					}
				}
				if len(forecasts) > 0 {
					resp.AvgMean = totalMean / float64(len(forecasts))
				}
			}
		}
	}
	return resp
}

// NullableTime is a helper for nullable time.Time pointers in responses.
type NullableTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullableTime) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
	nt.Time, nt.Valid = value.(time.Time), true
	return nil
}

func (nt NullableTime) MarshalJSON() (out []byte, err error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

func (nt *NullableTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	nt.Valid = true
	return nt.Time.UnmarshalJSON(data)
}
