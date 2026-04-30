package forecastingclient

// ---------------------------------------------------------------------------
// Datasets - Upload
// ---------------------------------------------------------------------------

type UploadDatasetRequest struct {
	RequestID  string `json:"request_id"`
	Domain     string `json:"domain,omitempty"`
	SourceMode string `json:"source_mode,omitempty"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	Freq       string `json:"freq,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Tenant     string `json:"tenant,omitempty"`
	Uniq       string `json:"uniq,omitempty"`
}

type UploadDatasetResponse struct {
	RequestID      string  `json:"request_id"`
	Domain         string  `json:"domain"`
	Status         string  `json:"status"`
	DatasetID      string  `json:"dataset_id"`
	Name           string  `json:"name"`
	Version        string  `json:"version"`
	SourceMode     string  `json:"source_mode"`
	GCSURI         string  `json:"gcs_uri"`
	SHA256         string  `json:"sha256"`
	RowCount       int64   `json:"row_count"`
	ItemCount      int64   `json:"item_count"`
	Scope          string  `json:"scope"`
	Tenant         string  `json:"tenant"`
	Uniq           string  `json:"uniq"`
	TrainingRunID  *string `json:"training_run_id"`
	OperationName  *string `json:"operation_name"`
}

// ---------------------------------------------------------------------------
// Train - Global / Custom
// ---------------------------------------------------------------------------

type TrainGlobalRequest struct {
	RequestID  string `json:"request_id"`
	Domain     string `json:"domain"`
	DatasetID  string `json:"dataset_id"`
	FineTune   bool   `json:"fine_tune"`
	TimeLimit  int    `json:"time_limit"`
	Presets    string `json:"presets,omitempty"`
}

type TrainCustomRequest struct {
	RequestID  string `json:"request_id"`
	Domain     string `json:"domain"`
	DatasetID  string `json:"dataset_id"`
	Tenant     string `json:"tenant"`
	Uniq       string `json:"uniq"`
	FineTune   bool   `json:"fine_tune"`
	TimeLimit  int    `json:"time_limit"`
	Presets    string `json:"presets,omitempty"`
}

type TrainResponse struct {
	RequestID     string  `json:"request_id"`
	Domain        string  `json:"domain"`
	Status        string  `json:"status"`
	TrainingRunID string  `json:"training_run_id"`
	JobName       string  `json:"job_name"`
	Region        string  `json:"region"`
	Scope         string  `json:"scope"`
	Tenant        string  `json:"tenant"`
	Uniq          string  `json:"uniq"`
	OperationName string  `json:"operation_name"`
}

// ---------------------------------------------------------------------------
// Monitor - Training Run
// ---------------------------------------------------------------------------

type TrainingRunStatus string

const (
	TrainingStatusPending   TrainingRunStatus = "PENDING"
	TrainingStatusRunning   TrainingRunStatus = "RUNNING"
	TrainingStatusSucceeded TrainingRunStatus = "SUCCEEDED"
	TrainingStatusFailed    TrainingRunStatus = "FAILED"
	TrainingStatusCancelled TrainingRunStatus = "CANCELLED"
)

type TrainingRunDetail struct {
	RequestID       string  `json:"request_id"`
	Domain          string  `json:"domain"`
	Status          string  `json:"status"`
	TrainingRunID   string  `json:"training_run_id"`
	JobName         string  `json:"job_name"`
	Region          string  `json:"region"`
	Scope           string  `json:"scope"`
	Tenant          string  `json:"tenant"`
	Uniq            string  `json:"uniq"`
	DatasetID       string  `json:"dataset_id"`
	DatasetName     string  `json:"dataset_name"`
	DatasetVersion  string  `json:"dataset_version"`
	SourceMode      string  `json:"source_mode"`
	GCSURI          string  `json:"gcs_uri"`
	RowCount        int64   `json:"row_count"`
	ItemCount       int64   `json:"item_count"`
	ModelVersionID  *string `json:"model_version_id,omitempty"`
	OperationName   string  `json:"operation_name"`
	ErrorMessage    string  `json:"error_message,omitempty"`
}

// ---------------------------------------------------------------------------
// Monitor - List / Lookup
// ---------------------------------------------------------------------------

type TrainingRunListItem struct {
	RequestID       string  `json:"request_id"`
	Domain          string  `json:"domain"`
	Status          string  `json:"status"`
	TrainingRunID   string  `json:"training_run_id"`
	Scope           string  `json:"scope"`
	Tenant          string  `json:"tenant"`
	Uniq            string  `json:"uniq"`
	DatasetID       string  `json:"dataset_id"`
	DatasetName     string  `json:"dataset_name"`
	ModelVersionID  *string `json:"model_version_id,omitempty"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

type DatasetListItem struct {
	DatasetID   string `json:"dataset_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Domain      string `json:"domain"`
	Scope       string `json:"scope"`
	RowCount    int64  `json:"row_count"`
	ItemCount   int64  `json:"item_count"`
	SourceMode  string `json:"source_mode"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type ModelVersionItem struct {
	ModelVersionID string  `json:"model_version_id"`
	Domain         string  `json:"domain"`
	Scope          string  `json:"scope"`
	Tenant         string  `json:"tenant"`
	Uniq           string  `json:"uniq"`
	Status         string  `json:"status"`
	Accuracy       float64 `json:"accuracy,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
}

type DeploymentItem struct {
	DeploymentID  string  `json:"deployment_id"`
	ModelVersionID string `json:"model_version_id"`
	Domain       string  `json:"domain"`
	Scope        string  `json:"scope"`
	Tenant       string  `json:"tenant"`
	Uniq         string  `json:"uniq"`
	Stage        string  `json:"stage"`
	CreatedAt    string  `json:"created_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Inference Results
// ---------------------------------------------------------------------------

type InferenceResultItem struct {
	RequestID       string `json:"request_id"`
	Domain          string `json:"domain"`
	Tenant          string `json:"tenant"`
	ItemID          string `json:"item_id"`
	ModelVersionID  string `json:"model_version_id"`
	Horizon         int    `json:"horizon"`
	Mode            string `json:"mode"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

type InferenceResultDetail struct {
	RequestID       string       `json:"request_id"`
	Domain          string       `json:"domain"`
	Tenant          string       `json:"tenant"`
	ItemID          string       `json:"item_id"`
	ModelVersionID  string       `json:"model_version_id"`
	Horizon         int          `json:"horizon"`
	LookbackPoints  int          `json:"lookback_points,omitempty"`
	Mode            string       `json:"mode"`
	RequestPayload  interface{}  `json:"request_payload"`
	ResponsePayload interface{}  `json:"response_payload"`
	Status          string       `json:"status"`
	ErrorMessage    string       `json:"error_message,omitempty"`
	CreatedAt       string       `json:"created_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Promote
// ---------------------------------------------------------------------------

type PromoteRequest struct {
	RequestID      string `json:"request_id"`
	Domain         string `json:"domain"`
	ModelVersionID string `json:"model_version_id"`
	Stage          string `json:"stage"`
	Scope          string `json:"scope"`
	Tenant         string `json:"tenant"`
	Uniq           string `json:"uniq"`
}

type PromoteResponse struct {
	RequestID      string `json:"request_id"`
	Domain         string `json:"domain"`
	ModelVersionID string `json:"model_version_id"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	DeployedAt     string `json:"deployed_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Predict
// ---------------------------------------------------------------------------

type PredictRequest struct {
	RequestID          string                      `json:"request_id"`
	Domain             string                      `json:"domain"`
	Tenant             string                      `json:"tenant,omitempty"`
	AutoObservations  bool                        `json:"auto_observations,omitempty"`
	ItemID             string                      `json:"item_id,omitempty"`
	Horizon            int                         `json:"horizon"`
	LookbackPoints     int                         `json:"lookback_points,omitempty"`
	Observations       []ObservationItem           `json:"observations,omitempty"`
	FutureCovariates   []FutureCovariateItem      `json:"future_covariates,omitempty"`
}

type ObservationItem struct {
	ItemID     string                 `json:"item_id"`
	Timestamp  string                 `json:"timestamp"`
	Target     float64                `json:"target"`
	Covariates map[string]interface{} `json:"covariates,omitempty"`
}

type FutureCovariateItem struct {
	CovariateName string      `json:"covariate_name"`
	Values        []float64   `json:"values"`
}

type PredictResponse struct {
	RequestID       string             `json:"request_id"`
	ModelVersionID  string             `json:"model_version_id"`
	Forecasts       []ForecastItem     `json:"forecasts"`
}

type ForecastItem struct {
	ItemID    string  `json:"item_id"`
	Timestamp string  `json:"timestamp"`
	Mean      float64 `json:"mean"`
	P10       float64 `json:"0.1,omitempty"`
	P50       float64 `json:"0.5,omitempty"`
	P90       float64 `json:"0.9,omitempty"`
}
