-- Migration 0054: Create outgoing_raw_material transaction table
-- Purpose: store business transaction rows for outgoing RM UI while inventory_movement_logs
-- remains the centralized audit trail for stock changes.

CREATE TABLE IF NOT EXISTS outgoing_raw_material (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4 (),

    transaction_id VARCHAR(64) NOT NULL,
    transaction_date DATE NOT NULL DEFAULT CURRENT_DATE,

    raw_material_id BIGINT REFERENCES raw_materials (id),
    packing_list_rm VARCHAR(128),
    uniq VARCHAR(64) NOT NULL,
    rm_name VARCHAR(255),

    unit VARCHAR(32),
    quantity_out NUMERIC(15, 4) NOT NULL,
    stock_before NUMERIC(15, 4) NOT NULL,
    stock_after NUMERIC(15, 4) NOT NULL,

    reason VARCHAR(64) NOT NULL,
    purpose TEXT,
    work_order_no VARCHAR(128),
    destination_location VARCHAR(255),
    requested_by VARCHAR(255),
    remarks TEXT,

    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_outgoing_raw_material_qty_positive CHECK (quantity_out > 0),
    CONSTRAINT chk_outgoing_raw_material_stock_non_negative CHECK (stock_after >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_outgoing_raw_material_uuid
    ON outgoing_raw_material (uuid);

CREATE UNIQUE INDEX IF NOT EXISTS idx_outgoing_raw_material_transaction_id
    ON outgoing_raw_material (transaction_id);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_raw_material_id
    ON outgoing_raw_material (raw_material_id);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_uniq
    ON outgoing_raw_material (uniq);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_transaction_date
    ON outgoing_raw_material (transaction_date);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_reason
    ON outgoing_raw_material (reason);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_work_order_no
    ON outgoing_raw_material (work_order_no);

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_deleted_at
    ON outgoing_raw_material (deleted_at);
