-- +migrate Up

CREATE TABLE IF NOT EXISTS raw_materials (
    id                       BIGSERIAL PRIMARY KEY,
    uuid                     UUID        NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    uniq_code                VARCHAR(64) NOT NULL UNIQUE,

part_number VARCHAR(128),
part_name VARCHAR(255),
item_id BIGINT,

raw_material_type VARCHAR(32) NOT NULL DEFAULT 'others', -- sheet_plate | wire | ssp | others
rm_source VARCHAR(32) NOT NULL DEFAULT 'supplier', -- process | supplier

warehouse_location VARCHAR(255),
uom VARCHAR(32),

stock_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
stock_weight_kg NUMERIC(15, 4),
kanban_count INTEGER,

kanban_standard_qty INTEGER,
safety_stock_qty NUMERIC(15, 4),
daily_usage_qty NUMERIC(15, 4),

status VARCHAR(32) NOT NULL DEFAULT 'normal', -- low_on_stock | normal | overstock
stock_days INTEGER,
buy_not_buy VARCHAR(10) NOT NULL DEFAULT 'n/a', -- buy | not_buy | n/a
stock_to_complete_kanban NUMERIC(15, 4),

created_by               VARCHAR(255),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by               VARCHAR(255),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_raw_materials_item_id ON raw_materials (item_id);

CREATE INDEX IF NOT EXISTS idx_raw_materials_deleted_at ON raw_materials (deleted_at);

-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS indirect_raw_materials (
    id                       BIGSERIAL PRIMARY KEY,
    uuid                     UUID        NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    uniq_code                VARCHAR(64) NOT NULL UNIQUE,

part_number VARCHAR(128),
part_name VARCHAR(255),
item_id BIGINT,

warehouse_location VARCHAR(255),
uom VARCHAR(32),

stock_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
stock_weight_kg NUMERIC(15, 4),
kanban_count INTEGER,

kanban_standard_qty INTEGER,
safety_stock_qty NUMERIC(15, 4),
daily_usage_qty NUMERIC(15, 4),

status VARCHAR(32), 
stock_days INTEGER,
buy_not_buy VARCHAR(10) NOT NULL DEFAULT 'n/a',
stock_to_complete_kanban NUMERIC(15, 4),

created_by               VARCHAR(255),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by               VARCHAR(255),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_indirect_raw_materials_item_id ON indirect_raw_materials (item_id);

CREATE INDEX IF NOT EXISTS idx_indirect_raw_materials_deleted_at ON indirect_raw_materials (deleted_at);

-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS subcon_inventories (
    id                  BIGSERIAL PRIMARY KEY,
    uuid                UUID        NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    uniq_code           VARCHAR(64) NOT NULL UNIQUE,

part_number VARCHAR(128),
part_name VARCHAR(255),

po_number VARCHAR(128),
po_period VARCHAR(32),
subcon_vendor_id BIGINT,
subcon_vendor_name VARCHAR(255),

stock_at_vendor_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
total_po_qty NUMERIC(15, 4),
total_received_qty NUMERIC(15, 4),
delta_po NUMERIC(15, 4), -- PO qty - stock

safety_stock_qty NUMERIC(15, 4),
date_delivery TIMESTAMPTZ,

status VARCHAR(32) NOT NULL DEFAULT 'normal', -- low_on_stock | normal | overstock | above_po

created_by          VARCHAR(255),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subcon_inventories_po_number ON subcon_inventories (po_number);

CREATE INDEX IF NOT EXISTS idx_subcon_inventories_vendor_id ON subcon_inventories (subcon_vendor_id);

CREATE INDEX IF NOT EXISTS idx_subcon_inventories_deleted_at ON subcon_inventories (deleted_at);

-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS inventory_movement_logs (
    id                BIGSERIAL PRIMARY KEY,

movement_category VARCHAR(64) NOT NULL, -- raw_material | indirect_raw_material | subcon
movement_type VARCHAR(64) NOT NULL, -- incoming | outgoing | stock_opname | adjustment | received_from_vendor

uniq_code VARCHAR(64) NOT NULL,

entity_id BIGINT,

qty_change NUMERIC(15, 4) NOT NULL DEFAULT 0,
weight_change NUMERIC(15, 4),

source_flag VARCHAR(64), -- incoming_scan | qc_approve | wo_scan | manual | stock_opname
dn_number VARCHAR(128),
reference_id VARCHAR(255),
notes TEXT,

logged_by         VARCHAR(255),
    logged_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_category ON inventory_movement_logs (movement_category);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_type ON inventory_movement_logs (movement_type);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_uniq_code ON inventory_movement_logs (uniq_code);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_entity_id ON inventory_movement_logs (entity_id);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_source ON inventory_movement_logs (source_flag);

CREATE INDEX IF NOT EXISTS idx_inv_move_logs_logged_at ON inventory_movement_logs (logged_at);