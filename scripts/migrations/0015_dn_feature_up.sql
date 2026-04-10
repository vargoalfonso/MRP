-- =============================================================================
-- Migration: 0015_dn_feature_up.sql
-- Feature  : Delivery Notes (DN) + Receiving (QR) + Incoming QC link
-- Strategy : Reuse existing legacy tables `incoming_dns`, `incoming_dn_items`, `qc_tasks`.
--            Add packing fields + append-only scan log.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- Improve DN header (legacy): incoming_dns
-- -----------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_incoming_dns_dn_number ON incoming_dns (dn_number);
CREATE INDEX IF NOT EXISTS idx_incoming_dns_po_number ON incoming_dns (po_number);
CREATE INDEX IF NOT EXISTS idx_incoming_dns_status    ON incoming_dns (status);
CREATE INDEX IF NOT EXISTS idx_incoming_dns_supplier  ON incoming_dns (supplier_id);

-- -----------------------------------------------------------------------------
-- Improve DN items (legacy): incoming_dn_items
-- Add packing fields needed by UI "Print Packing Number" and scan lookup.
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS incoming_dn_items
    ADD COLUMN IF NOT EXISTS packing_number varchar(64),
    ADD COLUMN IF NOT EXISTS pcs_per_kanban int,
    ADD COLUMN IF NOT EXISTS uom varchar(32),
    ADD COLUMN IF NOT EXISTS received_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_incoming_dn_items_uniq_code ON incoming_dn_items (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_incoming_dn_items_packing   ON incoming_dn_items (packing_number);

-- -----------------------------------------------------------------------------
-- New: incoming_receiving_scans (append-only)
-- Why new: needed for audit + idempotent QR scanning + variance analysis.
-- Reuse incoming_dn_items.id (uuid) as FK.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS incoming_receiving_scans (
    id                  bigserial PRIMARY KEY,
    incoming_dn_item_id uuid      NOT NULL REFERENCES incoming_dn_items(id) ON DELETE CASCADE,
    idempotency_key     varchar(128) NULL UNIQUE,
    scan_ref            text      NOT NULL,
    qty                 numeric(15,4) NOT NULL CHECK (qty > 0),
    weight_kg           numeric(15,4) NULL CHECK (weight_kg IS NULL OR weight_kg >= 0),
    scanned_at          timestamptz NOT NULL DEFAULT now(),
    scanned_by          varchar(255),

    -- prevent duplicate scan_ref for same DN line
    CONSTRAINT incoming_receiving_scans_dn_item_scanref_uq UNIQUE (incoming_dn_item_id, scan_ref)
);

CREATE INDEX IF NOT EXISTS idx_incoming_receiving_scans_dn_item ON incoming_receiving_scans (incoming_dn_item_id);
CREATE INDEX IF NOT EXISTS idx_incoming_receiving_scans_time    ON incoming_receiving_scans (scanned_at DESC);

-- -----------------------------------------------------------------------------
-- Link QC task to incoming DN item (reuse existing qc_tasks)
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS qc_tasks
    ADD COLUMN IF NOT EXISTS incoming_dn_item_id uuid;

-- Optional FK: keep as soft link first to avoid breaking existing data.
-- If you want strict FK later, add it after backfill:
--   ALTER TABLE qc_tasks ADD CONSTRAINT fk_qc_tasks_incoming_dn_item
--       FOREIGN KEY (incoming_dn_item_id) REFERENCES incoming_dn_items(id);

CREATE INDEX IF NOT EXISTS idx_qc_tasks_incoming_dn_item ON qc_tasks (incoming_dn_item_id);
