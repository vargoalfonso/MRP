-- Migration 0075: Create raw_material_logs table
-- Purpose: menyimpan pemakaian raw material yang di-log saat Production Scan In.

CREATE TABLE IF NOT EXISTS raw_material_logs (
    id BIGSERIAL PRIMARY KEY,
    uuid VARCHAR(64) NOT NULL,

    wo_id BIGINT NOT NULL,
    wo_item_id BIGINT NOT NULL,
    uniq_code VARCHAR(64),
    rm_uuid VARCHAR(64),

    part_number VARCHAR(128),
    part_name VARCHAR(255),
    uom VARCHAR(32),
    qty_used NUMERIC(15, 4) NOT NULL DEFAULT 0,

    scanned_by VARCHAR(255),
    scanned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_raw_material_logs_wo_id
    ON raw_material_logs (wo_id);

CREATE INDEX IF NOT EXISTS idx_raw_material_logs_wo_item_id
    ON raw_material_logs (wo_item_id);

CREATE INDEX IF NOT EXISTS idx_raw_material_logs_rm_uuid
    ON raw_material_logs (rm_uuid);