
CREATE TABLE IF NOT EXISTS forecasting_training_runs (
    id BIGSERIAL PRIMARY KEY,
    uuid VARCHAR(64) UNIQUE NOT NULL,
    request_id VARCHAR(150) NOT NULL,
    training_run_id VARCHAR(64) NULL,
    domain VARCHAR(20) NOT NULL,
    scope VARCHAR(20) NOT NULL,
    tenant VARCHAR(100) NULL,
    uniq VARCHAR(150) NULL,
    dataset_id VARCHAR(64) NULL,
    dataset_name VARCHAR(255) NULL,
    dataset_version VARCHAR(255) NULL,
    source_mode VARCHAR(50) NULL,
    gcs_uri TEXT NULL,
    row_count BIGINT NULL,
    item_count BIGINT NULL,
    fine_tune BOOLEAN NOT NULL DEFAULT FALSE,
    time_limit INTEGER NULL,
    presets TEXT NULL,
    job_name VARCHAR(255) NULL,
    region VARCHAR(100) NULL,
    operation_name TEXT NULL,
    status VARCHAR(30) NOT NULL,
    model_version_id VARCHAR(64) NULL,
    last_response JSONB NULL,
    error_message TEXT NULL,
    created_by VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_forecasting_training_runs_uuid ON forecasting_training_runs (uuid);
CREATE INDEX IF NOT EXISTS idx_forecasting_training_runs_training_run_id ON forecasting_training_runs (training_run_id);
CREATE INDEX IF NOT EXISTS idx_forecasting_training_runs_status ON forecasting_training_runs (status);
CREATE INDEX IF NOT EXISTS idx_forecasting_training_runs_scope_tenant_uniq ON forecasting_training_runs (scope, tenant, uniq);
CREATE INDEX IF NOT EXISTS idx_forecasting_training_runs_created_at ON forecasting_training_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_forecasting_training_runs_deleted_at ON forecasting_training_runs (deleted_at);


CREATE TABLE IF NOT EXISTS forecasting_inference_results (
    id BIGSERIAL PRIMARY KEY,
    uuid VARCHAR(64) UNIQUE NOT NULL,
    request_id VARCHAR(150) NOT NULL,
    domain VARCHAR(20) NOT NULL,
    tenant VARCHAR(100) NULL,
    item_id VARCHAR(150) NULL,
    model_version_id VARCHAR(64) NOT NULL,
    horizon INTEGER NOT NULL,
    lookback_points INTEGER NULL,
    mode VARCHAR(20) NOT NULL,
    request_payload JSONB NOT NULL,
    response_payload JSONB NOT NULL,
    status VARCHAR(30) NOT NULL,
    error_message TEXT NULL,
    created_by VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_forecasting_inference_results_uuid ON forecasting_inference_results (uuid);
CREATE INDEX IF NOT EXISTS idx_forecasting_inference_results_request_id ON forecasting_inference_results (request_id);
CREATE INDEX IF NOT EXISTS idx_forecasting_inference_results_model_version_id ON forecasting_inference_results (model_version_id);
CREATE INDEX IF NOT EXISTS idx_forecasting_inference_results_item_id ON forecasting_inference_results (item_id);
CREATE INDEX IF NOT EXISTS idx_forecasting_inference_results_created_at ON forecasting_inference_results (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_forecasting_inference_results_deleted_at ON forecasting_inference_results (deleted_at);
