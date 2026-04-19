-- =============================================================================
-- Migration: 0050_add_outbound_customer_dn_fields_up.sql
-- Feature  : Customer delivery scheduling + customer delivery notes
-- Notes    : Separate customer outbound flow from procurement delivery_notes.
-- Flow     : customer_order_documents -> delivery_schedules_customer
--            -> approval -> delivery_notes_customer -> shipment scan
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- delivery_schedules_customer (header)
-- Source of truth for frontend Delivery Schedule tab.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_schedules_customer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    schedule_number VARCHAR(64) NOT NULL UNIQUE,
    customer_order_document_id BIGINT REFERENCES customer_order_documents(id) ON DELETE SET NULL,
    customer_order_reference VARCHAR(128),
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    customer_name_snapshot VARCHAR(255),
    customer_contact_person VARCHAR(255),
    customer_phone_number VARCHAR(64),
    delivery_address TEXT,
    schedule_date DATE NOT NULL,
    priority VARCHAR(16) NOT NULL DEFAULT 'normal',
    transport_company VARCHAR(255),
    vehicle_number VARCHAR(64),
    driver_name VARCHAR(255),
    driver_contact VARCHAR(64),
    departure_at TIMESTAMPTZ,
    arrival_at TIMESTAMPTZ,
    cycle VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'scheduled',
    approval_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    delivery_instructions TEXT,
    remarks TEXT,
    created_by VARCHAR(255),
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT delivery_schedules_customer_priority_ck
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT delivery_schedules_customer_status_ck
        CHECK (status IN ('scheduled', 'partially_approved', 'approved', 'dn_created', 'cancelled')),
    CONSTRAINT delivery_schedules_customer_approval_status_ck
        CHECK (approval_status IN ('pending', 'approved', 'rejected', 'partial'))
);

CREATE INDEX IF NOT EXISTS idx_delivery_schedules_customer_date
    ON delivery_schedules_customer (schedule_date);
CREATE INDEX IF NOT EXISTS idx_delivery_schedules_customer_customer_id
    ON delivery_schedules_customer (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_schedules_customer_status
    ON delivery_schedules_customer (status);
CREATE INDEX IF NOT EXISTS idx_delivery_schedules_customer_approval_status
    ON delivery_schedules_customer (approval_status);
CREATE INDEX IF NOT EXISTS idx_delivery_schedules_customer_deleted_at
    ON delivery_schedules_customer (deleted_at);

-- -----------------------------------------------------------------------------
-- delivery_schedule_items_customer (lines)
-- Stores reviewed order items and requested delivery quantity before DN creation.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_schedule_items_customer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    schedule_id BIGINT NOT NULL REFERENCES delivery_schedules_customer(id) ON DELETE CASCADE,
    customer_order_document_item_id BIGINT REFERENCES customer_order_document_items(id) ON DELETE SET NULL,
    line_no INTEGER NOT NULL,
    item_uniq_code VARCHAR(100) NOT NULL,
    model VARCHAR(255),
    part_name VARCHAR(255) NOT NULL,
    part_number VARCHAR(128) NOT NULL,
    total_order_qty NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (total_order_qty >= 0),
    total_delivery_qty NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (total_delivery_qty >= 0),
    uom VARCHAR(32) NOT NULL,
    cycle VARCHAR(64),
    dn_number VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'scheduled',
    fg_readiness_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT delivery_schedule_items_customer_schedule_line_uq
        UNIQUE (schedule_id, line_no),
    CONSTRAINT delivery_schedule_items_customer_status_ck
        CHECK (status IN ('scheduled', 'approved', 'partial', 'dn_created', 'cancelled')),
    CONSTRAINT delivery_schedule_items_customer_fg_readiness_status_ck
        CHECK (fg_readiness_status IN ('unknown', 'ready', 'partial_ready', 'shortage')),
    CONSTRAINT delivery_schedule_items_customer_qty_ck
        CHECK (total_delivery_qty <= total_order_qty)
);

CREATE INDEX IF NOT EXISTS idx_delivery_schedule_items_customer_schedule_id
    ON delivery_schedule_items_customer (schedule_id);
CREATE INDEX IF NOT EXISTS idx_delivery_schedule_items_customer_item_uniq_code
    ON delivery_schedule_items_customer (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_delivery_schedule_items_customer_dn_number
    ON delivery_schedule_items_customer (dn_number);
CREATE INDEX IF NOT EXISTS idx_delivery_schedule_items_customer_status
    ON delivery_schedule_items_customer (status);

-- -----------------------------------------------------------------------------
-- delivery_notes_customer (header)
-- Created from approved delivery schedules, not mixed with procurement DNs.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_notes_customer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    dn_number VARCHAR(64) NOT NULL UNIQUE,
    schedule_id BIGINT REFERENCES delivery_schedules_customer(id) ON DELETE SET NULL,
    customer_order_document_id BIGINT REFERENCES customer_order_documents(id) ON DELETE SET NULL,
    customer_order_reference VARCHAR(128),
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    customer_name_snapshot VARCHAR(255),
    customer_contact_person VARCHAR(255),
    customer_phone_number VARCHAR(64),
    delivery_address TEXT,
    delivery_date DATE NOT NULL,
    priority VARCHAR(16) NOT NULL DEFAULT 'normal',
    transport_company VARCHAR(255),
    vehicle_number VARCHAR(64),
    driver_name VARCHAR(255),
    driver_contact VARCHAR(64),
    departure_at TIMESTAMPTZ,
    arrival_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    approval_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    delivery_instructions TEXT,
    remarks TEXT,
    printed_count INTEGER NOT NULL DEFAULT 0 CHECK (printed_count >= 0),
    created_by VARCHAR(255),
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT delivery_notes_customer_priority_ck
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT delivery_notes_customer_status_ck
        CHECK (status IN ('created', 'printed', 'scanned', 'in_transit', 'shipped', 'confirmed', 'cancelled')),
    CONSTRAINT delivery_notes_customer_approval_status_ck
        CHECK (approval_status IN ('pending', 'approved', 'rejected', 'partial'))
);

CREATE INDEX IF NOT EXISTS idx_delivery_notes_customer_schedule_id
    ON delivery_notes_customer (schedule_id);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_customer_customer_id
    ON delivery_notes_customer (customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_customer_delivery_date
    ON delivery_notes_customer (delivery_date);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_customer_status
    ON delivery_notes_customer (status);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_customer_approval_status
    ON delivery_notes_customer (approval_status);

-- -----------------------------------------------------------------------------
-- delivery_note_items_customer (lines)
-- QR must be stored as base64 PNG in qr column when DN is created.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_note_items_customer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    dn_id BIGINT NOT NULL REFERENCES delivery_notes_customer(id) ON DELETE CASCADE,
    schedule_item_id BIGINT REFERENCES delivery_schedule_items_customer(id) ON DELETE SET NULL,
    line_no INTEGER NOT NULL,
    item_uniq_code VARCHAR(100) NOT NULL,
    model VARCHAR(255),
    part_name VARCHAR(255) NOT NULL,
    part_number VARCHAR(128) NOT NULL,
    fg_location VARCHAR(64),
    quantity NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    qty_shipped NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (qty_shipped >= 0),
    uom VARCHAR(32) NOT NULL,
    packing_number VARCHAR(100) NOT NULL UNIQUE,
    qr TEXT,
    shipment_status VARCHAR(32) NOT NULL DEFAULT 'created',
    printed_count INTEGER NOT NULL DEFAULT 0 CHECK (printed_count >= 0),
    shipped_at TIMESTAMPTZ,
    shipped_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT delivery_note_items_customer_dn_line_uq
        UNIQUE (dn_id, line_no),
    CONSTRAINT delivery_note_items_customer_shipment_status_ck
        CHECK (shipment_status IN ('created', 'printed', 'scanned', 'partial', 'shipped', 'cancelled')),
    CONSTRAINT delivery_note_items_customer_qty_shipped_ck
        CHECK (qty_shipped <= quantity)
);

CREATE INDEX IF NOT EXISTS idx_delivery_note_items_customer_dn_id
    ON delivery_note_items_customer (dn_id);
CREATE INDEX IF NOT EXISTS idx_delivery_note_items_customer_schedule_item_id
    ON delivery_note_items_customer (schedule_item_id);
CREATE INDEX IF NOT EXISTS idx_delivery_note_items_customer_item_uniq_code
    ON delivery_note_items_customer (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_delivery_note_items_customer_shipment_status
    ON delivery_note_items_customer (shipment_status);

-- -----------------------------------------------------------------------------
-- delivery_note_logs_customer
-- Shipment scan audit log with idempotency support.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS delivery_note_logs_customer (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dn_id BIGINT NOT NULL REFERENCES delivery_notes_customer(id) ON DELETE CASCADE,
    dn_item_id BIGINT NOT NULL REFERENCES delivery_note_items_customer(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(128) UNIQUE,
    scan_ref TEXT,
    item_uniq_code VARCHAR(100) NOT NULL,
    packing_number VARCHAR(100),
    scan_type VARCHAR(20) NOT NULL,
    qty NUMERIC(15,2) NOT NULL CHECK (qty >= 0),
    from_location VARCHAR(50),
    to_location VARCHAR(50),
    scanned_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT delivery_note_logs_customer_scan_type_ck
        CHECK (scan_type IN ('printed', 'scan_out', 'scan_confirm', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_delivery_note_logs_customer_dn_id
    ON delivery_note_logs_customer (dn_id);
CREATE INDEX IF NOT EXISTS idx_delivery_note_logs_customer_dn_item_id
    ON delivery_note_logs_customer (dn_item_id);
CREATE INDEX IF NOT EXISTS idx_delivery_note_logs_customer_packing_number
    ON delivery_note_logs_customer (packing_number);
CREATE INDEX IF NOT EXISTS idx_delivery_note_logs_customer_scan_type_created_at
    ON delivery_note_logs_customer (scan_type, created_at DESC);
