-- =============================================================================
-- Migration: 0015_dn_feature_up.sql
-- Feature  : Incoming receiving + QC over existing delivery_* tables
-- Strategy : ALTER delivery_notes + delivery_note_items, then create qc_tasks
--            and incoming_receiving_scans with bigint references.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- Extend delivery_notes for incoming flow context.
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS delivery_notes
    ADD COLUMN IF NOT EXISTS supplier_id bigint,
    ADD COLUMN IF NOT EXISTS total_po_qty int,
    ADD COLUMN IF NOT EXISTS total_dn_created int,
    ADD COLUMN IF NOT EXISTS total_dn_incoming int;

CREATE INDEX IF NOT EXISTS idx_delivery_notes_dn_number  ON delivery_notes (dn_number);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_po_number  ON delivery_notes (po_number);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_status     ON delivery_notes (status);
CREATE INDEX IF NOT EXISTS idx_delivery_notes_supplier   ON delivery_notes (supplier_id);

-- -----------------------------------------------------------------------------
-- Extend delivery_note_items for incoming scan + QC fields.
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS delivery_note_items
    ADD COLUMN IF NOT EXISTS order_qty int,
    ADD COLUMN IF NOT EXISTS date_incoming date,
    ADD COLUMN IF NOT EXISTS qty_stated int,
    ADD COLUMN IF NOT EXISTS qty_received int,
    ADD COLUMN IF NOT EXISTS weight_received numeric(15,4),
    ADD COLUMN IF NOT EXISTS quality_status varchar(32),
    ADD COLUMN IF NOT EXISTS packing_number varchar(64),
    ADD COLUMN IF NOT EXISTS pcs_per_kanban int,
    ADD COLUMN IF NOT EXISTS received_at timestamptz;

UPDATE delivery_note_items
SET order_qty = COALESCE(order_qty, quantity),
    date_incoming = COALESCE(date_incoming, CURRENT_DATE),
    qty_stated = COALESCE(qty_stated, quantity),
    qty_received = COALESCE(qty_received, 0),
    quality_status = COALESCE(NULLIF(quality_status, ''), 'Pending')
WHERE order_qty IS NULL
   OR date_incoming IS NULL
   OR qty_stated IS NULL
   OR qty_received IS NULL
   OR quality_status IS NULL
   OR quality_status = '';

ALTER TABLE IF EXISTS delivery_note_items
    ALTER COLUMN order_qty SET DEFAULT 0,
    ALTER COLUMN date_incoming SET DEFAULT CURRENT_DATE,
    ALTER COLUMN qty_stated SET DEFAULT 0,
    ALTER COLUMN qty_received SET DEFAULT 0,
    ALTER COLUMN quality_status SET DEFAULT 'Pending';

CREATE INDEX IF NOT EXISTS idx_delivery_note_items_dn_id      ON delivery_note_items (dn_id);
CREATE INDEX IF NOT EXISTS idx_delivery_note_items_uniq_code  ON delivery_note_items (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_delivery_note_items_packing    ON delivery_note_items (packing_number);

-- -----------------------------------------------------------------------------
-- qc_tasks for incoming QC process (bigint FK to delivery_note_items.id)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS qc_tasks (
    id                  bigserial PRIMARY KEY,
    task_type           varchar(32) NOT NULL DEFAULT 'incoming_qc',
    status              varchar(32) NOT NULL DEFAULT 'pending',
    incoming_dn_item_id bigint,
    good_quantity       int,
    ng_quantity         int,
    scrap_quantity      int,
    round               int NOT NULL DEFAULT 1,
    round_results       jsonb NOT NULL DEFAULT '[]',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_qc_tasks_status            ON qc_tasks (status);
CREATE INDEX IF NOT EXISTS idx_qc_tasks_task_type         ON qc_tasks (task_type);
CREATE INDEX IF NOT EXISTS idx_qc_tasks_incoming_dn_item  ON qc_tasks (incoming_dn_item_id);

-- -----------------------------------------------------------------------------
-- incoming_receiving_scans append-only scan audit table.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incoming_receiving_scans (
    id                  bigserial PRIMARY KEY,
    incoming_dn_item_id bigint NOT NULL REFERENCES delivery_note_items(id) ON DELETE CASCADE,
    idempotency_key     varchar(128) NULL UNIQUE,
    scan_ref            text NOT NULL,
    qty                 numeric(15,4) NOT NULL CHECK (qty > 0),
    weight_kg           numeric(15,4) NULL CHECK (weight_kg IS NULL OR weight_kg >= 0),
    scanned_at          timestamptz NOT NULL DEFAULT now(),
    scanned_by          varchar(255),
    CONSTRAINT incoming_receiving_scans_dn_item_scanref_uq UNIQUE (incoming_dn_item_id, scan_ref)
);

CREATE INDEX IF NOT EXISTS idx_incoming_receiving_scans_dn_item ON incoming_receiving_scans (incoming_dn_item_id);
CREATE INDEX IF NOT EXISTS idx_incoming_receiving_scans_time    ON incoming_receiving_scans (scanned_at DESC);