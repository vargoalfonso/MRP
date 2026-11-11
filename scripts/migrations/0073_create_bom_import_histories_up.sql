-- Migration: 0073_create_bom_import_histories_up.sql
-- Create bom_import_histories table to persist BOM Excel bulk-import results

CREATE TABLE IF NOT EXISTS bom_import_histories (
    id              BIGSERIAL PRIMARY KEY,
    file_name       VARCHAR(255) NOT NULL,
    file_size_kb    INTEGER NOT NULL DEFAULT 0,
    row_count       INTEGER NOT NULL DEFAULT 0,
    uploaded_by     VARCHAR(255),
    status          VARCHAR(20) NOT NULL DEFAULT 'success',
    summary         TEXT,
    imported_count  INTEGER NOT NULL DEFAULT 0,
    failed_count    INTEGER NOT NULL DEFAULT 0,
    request_id      VARCHAR(64),
    error_file      BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_bom_import_histories_created_at ON bom_import_histories(created_at DESC);