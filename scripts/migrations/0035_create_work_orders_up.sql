-- =============================================================================
-- Migration: 0035_create_work_orders_up.sql
-- Feature  : Work Order (WO) + WO Items + Production Scan Logs (shop floor)
-- Notes    : 1 WO memiliki banyak items (1 item = 1 kanban/batch)
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- work_orders (header)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS work_orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4 () UNIQUE,
    wo_number VARCHAR(64) NOT NULL UNIQUE,
    wo_type VARCHAR(32) NOT NULL DEFAULT 'New', -- New | Assembly | Rework | Addendum
    reference_wo VARCHAR(64), -- reference WO number (optional)
    status VARCHAR(32) NOT NULL DEFAULT 'Draft',
    approval_status VARCHAR(32) NOT NULL DEFAULT 'Pending',
    created_date DATE NOT NULL DEFAULT CURRENT_DATE,
    target_date DATE,
    scan_start_date DATE,
    close_date DATE,
    operator_name VARCHAR(255),
    notes TEXT,
    qr_image_base64 TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders (status);

CREATE INDEX IF NOT EXISTS idx_work_orders_approval ON work_orders (approval_status);

CREATE INDEX IF NOT EXISTS idx_work_orders_target_date ON work_orders (target_date);

CREATE INDEX IF NOT EXISTS idx_work_orders_type ON work_orders (wo_type);

-- -----------------------------------------------------------------------------
-- work_order_items (lines)
-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS work_order_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4 () UNIQUE,
    wo_id BIGINT NOT NULL REFERENCES work_orders (id) ON DELETE CASCADE,
    item_uniq_code VARCHAR(64) NOT NULL,
    part_name VARCHAR(255),
    part_number VARCHAR(128),
    uom VARCHAR(32),
    qr_image_base64 TEXT,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 0,
    process_name VARCHAR(64),
    kanban_number VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_work_order_items_wo_id ON work_order_items (wo_id);

CREATE INDEX IF NOT EXISTS idx_work_order_items_uniq_code ON work_order_items (item_uniq_code);

CREATE INDEX IF NOT EXISTS idx_work_order_items_status ON work_order_items (status);

-- -----------------------------------------------------------------------------
-- production_scan_logs (append-only shopfloor scan events)
-- -----------------------------------------------------------------------------


CREATE TABLE IF NOT EXISTS production_scan_logs (
    id                 BIGINT       GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid               UUID         NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    wo_id              BIGINT REFERENCES work_orders(id) ON DELETE SET NULL,
    wo_item_id         BIGINT REFERENCES work_order_items(id) ON DELETE SET NULL,
    scan_type          VARCHAR(32) NOT NULL,
    process_name       VARCHAR(64) NOT NULL,
    machine_id         UUID,
raw_material_uuid  UUID, -- reserved: can reference raw_materials.uuid when needed
    quantity           NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    quantity_rm_used   NUMERIC(15,4),
    good_quantity      NUMERIC(15,4),
    ng_setting_machine NUMERIC(15,4),
    ng_process         NUMERIC(15,4),
    scrap_quantity     NUMERIC(15,4),
    report_date        DATE DEFAULT CURRENT_DATE,
    kanban_number      VARCHAR(64),
    scanned_by         VARCHAR(255),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT production_scan_logs_scan_type_ck
        CHECK (scan_type IN ('scan_in', 'scan_out', 'rm_return'))
);

CREATE INDEX IF NOT EXISTS idx_prod_scan_logs_wo_id_created_at ON production_scan_logs (wo_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prod_scan_logs_wo_item_id_created_at ON production_scan_logs (wo_item_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prod_scan_logs_machine_id_created_at ON production_scan_logs (machine_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prod_scan_logs_report_date_process ON production_scan_logs (report_date, process_name);

CREATE INDEX IF NOT EXISTS idx_prod_scan_logs_kanban_number ON production_scan_logs (kanban_number);

-- -----------------------------------------------------------------------------
-- Extend qc_tasks to support production QC (non-breaking for incoming QC)
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS qc_tasks
ADD COLUMN IF NOT EXISTS wo_id BIGINT,
ADD COLUMN IF NOT EXISTS wo_item_id BIGINT,
ADD COLUMN IF NOT EXISTS source_scan_id BIGINT;

-- NOTE: PostgreSQL does not support "ADD CONSTRAINT IF NOT EXISTS".
-- Use a guarded DO block instead so migration is re-runnable.
DO $$
BEGIN
    IF to_regclass('public.qc_tasks') IS NOT NULL THEN

        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'qc_tasks_wo_id_fk') THEN
            ALTER TABLE qc_tasks
            ADD CONSTRAINT qc_tasks_wo_id_fk
            FOREIGN KEY (wo_id) REFERENCES work_orders (id) ON DELETE SET NULL;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'qc_tasks_wo_item_id_fk') THEN
            ALTER TABLE qc_tasks
            ADD CONSTRAINT qc_tasks_wo_item_id_fk
            FOREIGN KEY (wo_item_id) REFERENCES work_order_items (id) ON DELETE SET NULL;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'qc_tasks_source_scan_id_fk') THEN
            ALTER TABLE qc_tasks
            ADD CONSTRAINT qc_tasks_source_scan_id_fk
            FOREIGN KEY (source_scan_id) REFERENCES production_scan_logs (id) ON DELETE SET NULL;
        END IF;

    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_qc_tasks_wo_id ON qc_tasks (wo_id);

CREATE INDEX IF NOT EXISTS idx_qc_tasks_wo_item_id ON qc_tasks (wo_item_id);

CREATE INDEX IF NOT EXISTS idx_qc_tasks_source_scan ON qc_tasks (source_scan_id);
