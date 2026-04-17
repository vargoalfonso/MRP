-- Migration 0045: Create scrap_stocks and scrap_releases tables
-- scrap_stocks  : balance per scrap record with traceability to WO
-- scrap_releases: release event (Sell/Dump) with approval gate + release number

-- ---------------------------------------------------------------------------
-- 1. scrap_stocks
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scrap_stocks (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4 (),
    uniq_code VARCHAR(64) NOT NULL,
    part_number VARCHAR(128),
    part_name VARCHAR(255),
    model VARCHAR(128),
    packing_number VARCHAR(128),
    wo_number VARCHAR(128),
    scrap_type VARCHAR(64) NOT NULL,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 0,
    uom VARCHAR(32),
    weight_kg NUMERIC(15, 4),
    date_received DATE,
    validator VARCHAR(255), -- person who validated the entry (from JWT)
    remarks TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'Active',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scrap_stocks_uuid ON scrap_stocks (uuid);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_uniq_code ON scrap_stocks (uniq_code);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_scrap_type ON scrap_stocks (scrap_type);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_packing_number ON scrap_stocks (packing_number);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_wo_number ON scrap_stocks (wo_number);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_status ON scrap_stocks (status);

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_deleted_at ON scrap_stocks (deleted_at);

-- ---------------------------------------------------------------------------
-- 2. scrap_releases
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scrap_releases (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4 (),
    release_number VARCHAR(64) NOT NULL,
    scrap_stock_id BIGINT NOT NULL REFERENCES scrap_stocks (id),
    release_date DATE,
    release_type VARCHAR(32) NOT NULL,
    release_qty NUMERIC(15, 4) NOT NULL,
    weight_released NUMERIC(15, 4),
    customer_name VARCHAR(255),
    price_per_unit NUMERIC(15, 4),
    total_value NUMERIC(15, 4),
    disposal_reason TEXT,
    approval_status VARCHAR(32) NOT NULL DEFAULT 'Pending',
    validator VARCHAR(255), -- person who created/submitted the release
    approver VARCHAR(255), -- intended approver role/name
    approved_by VARCHAR(255), -- actual approver (from JWT at approval time)
    approved_at TIMESTAMPTZ,
    remarks TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scrap_releases_uuid ON scrap_releases (uuid);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scrap_releases_release_number ON scrap_releases (release_number);

CREATE INDEX IF NOT EXISTS idx_scrap_releases_scrap_stock_id ON scrap_releases (scrap_stock_id);

CREATE INDEX IF NOT EXISTS idx_scrap_releases_release_type ON scrap_releases (release_type);

CREATE INDEX IF NOT EXISTS idx_scrap_releases_approval_status ON scrap_releases (approval_status);

CREATE INDEX IF NOT EXISTS idx_scrap_releases_deleted_at ON scrap_releases (deleted_at);