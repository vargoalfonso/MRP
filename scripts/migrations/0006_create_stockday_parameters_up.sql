-- =============================================================================
-- Migration: 0004_create_upload_sessions_up.sql
-- Module   : Chunked Upload (resumable file upload for 2D/3D CAD assets)
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS stockdays_parameters (
    id BIGSERIAL PRIMARY KEY,
    inventory_type VARCHAR(50) NOT NULL,
    item_uniq_code VARCHAR(100) NOT NULL,
    calculation_type VARCHAR(50) NOT NULL,
    constanta DOUBLE PRECISION DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- 🔥 prevent duplicate per item
    CONSTRAINT unique_stockdays UNIQUE (inventory_type, item_uniq_code)
);
