package models

// ---------------------------------------------------------------------------
// Upload Dataset
// ---------------------------------------------------------------------------

// UploadDatasetRequest is the handler input for multipart upload.
// Fields come from form-data; the file comes separately.
type UploadDatasetRequest struct {
	RequestID  string `form:"request_id" validate:"required"`
	Domain     string `form:"domain"`     // dn | prl (default dn)
	SourceMode string `form:"source_mode"` // v4_excel | dn_excel_horizontal | prl_excel_horizontal | parquet_tar
	Name       string `form:"name"`
	Version    string `form:"version"`
	Freq       string `form:"freq"`    // D | M (auto if empty)
	Scope      string `form:"scope"`   // global | custom
	Tenant     string `form:"tenant"`  // required if scope=custom
	Uniq       string `form:"uniq"`   // required if scope=custom
}

// ---------------------------------------------------------------------------
// Train
// ---------------------------------------------------------------------------

// TrainGlobalRequest is the handler input for triggering a global training.
type TrainGlobalRequest struct {
	RequestID string `json:"request_id" validate:"required"`
	Domain    string `json:"domain" validate:"required"`
	DatasetID string `json:"dataset_id" validate:"required"`
	FineTune  bool   `json:"fine_tune"`
	TimeLimit int    `json:"time_limit"`
	Presets   string `json:"presets,omitempty"`
}

// TrainCustomRequest is the handler input for triggering a custom (per-tenant/per-uniq) training.
type TrainCustomRequest struct {
	RequestID string `json:"request_id" validate:"required"`
	Domain    string `json:"domain" validate:"required"`
	DatasetID string `json:"dataset_id" validate:"required"`
	Tenant    string `json:"tenant" validate:"required"`
	Uniq      string `json:"uniq" validate:"required"`
	FineTune  bool   `json:"fine_tune"`
	TimeLimit int    `json:"time_limit"`
	Presets   string `json:"presets,omitempty"`
}

// ---------------------------------------------------------------------------
// Promote
// ---------------------------------------------------------------------------

// PromoteRequest is the handler input for promoting a model version.
type PromoteRequest struct {
	RequestID      string `json:"request_id" validate:"required"`
	Domain         string `json:"domain" validate:"required"`
	ModelVersionID string `json:"model_version_id" validate:"required"`
	Stage          string `json:"stage" validate:"required"` // prod | staging
	Scope          string `json:"scope"`   // global | custom
	Tenant         string `json:"tenant"`
	Uniq           string `json:"uniq"`
}

// ---------------------------------------------------------------------------
// Predict
// ---------------------------------------------------------------------------

// PredictRequest is the handler input for forecast prediction.
// Supports both auto-observation mode and manual payload mode.
type PredictRequest struct {
	RequestID         string                 `json:"request_id" validate:"required"`
	Domain           string                 `json:"domain" validate:"required"`
	Tenant          string                 `json:"tenant,omitempty"`
	AutoObservations bool                   `json:"auto_observations,omitempty"`
	ItemID          string                 `json:"item_id,omitempty"`
	Horizon         int                    `json:"horizon" validate:"required,min=1"`
	LookbackPoints  int                    `json:"lookback_points,omitempty"`
	Observations    []ObservationItemDTO    `json:"observations,omitempty"`
	FutureCovariates []FutureCovariateDTO  `json:"future_covariates,omitempty"`
}

type ObservationItemDTO struct {
	ItemID     string                 `json:"item_id"`
	Timestamp  string                 `json:"timestamp"`
	Target     float64                `json:"target"`
	Covariates map[string]interface{} `json:"covariates,omitempty"`
}

type FutureCovariateDTO struct {
	CovariateName string    `json:"covariate_name"`
	Values        []float64 `json:"values"`
}
