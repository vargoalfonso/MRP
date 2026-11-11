-- =============================================================================
-- Migration: 0014_po_feature_up.sql
-- Feature  : Purchase Order (PO)
-- Strategy : Reuse existing legacy table `purchase_orders`.
--            Add missing tables/columns for PO line-items + integration mapping.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- Bridge: map legacy supplier (BIGINT) <-> suppliers UUID, if you use both.
-- New table (additive, no impact to legacy tables).
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS supplier_legacy_map (
    legacy_supplier_id bigint PRIMARY KEY,
    supplier_uuid      uuid   NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT supplier_legacy_map_supplier_uuid_uq UNIQUE (supplier_uuid)
);

-- -----------------------------------------------------------------------------
-- Improve PO header (legacy): purchase_orders
-- Only ADD columns/indexes (safe, backward compatible).
-- -----------------------------------------------------------------------------
ALTER TABLE IF EXISTS purchase_orders
    ADD COLUMN IF NOT EXISTS po_date date,
    ADD COLUMN IF NOT EXISTS expected_delivery_date date,
    ADD COLUMN IF NOT EXISTS currency varchar(8),
    ADD COLUMN IF NOT EXISTS total_amount numeric(18,2),
    ADD COLUMN IF NOT EXISTS external_system varchar(64),
    ADD COLUMN IF NOT EXISTS external_po_number varchar(128),
    ADD COLUMN IF NOT EXISTS created_by varchar(255),
    ADD COLUMN IF NOT EXISTS updated_by varchar(255);

-- Helpful indexes for PO board screens
CREATE INDEX IF NOT EXISTS idx_purchase_orders_period   ON purchase_orders (period);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier ON purchase_orders (supplier_id, period);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_status   ON purchase_orders (status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_number ON purchase_orders (po_number);

-- -----------------------------------------------------------------------------
-- New: purchase_order_items
-- Why new: existing legacy schema documents PO header only; UI needs line-level.
-- Links to legacy `purchase_orders.po_id`.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS purchase_order_items (
    id            bigserial PRIMARY KEY,
    po_id         bigint NOT NULL REFERENCES purchase_orders(po_id) ON DELETE CASCADE,
    line_no       int    NOT NULL DEFAULT 1,

    item_uniq_code varchar(64) NOT NULL,
    product_model  varchar(255),
    material_type  varchar(64),
    part_name      varchar(255),
    part_number    varchar(128),
    uom            varchar(32),
    weight_kg      numeric(15,4),
    description    text,

    ordered_qty    numeric(15,4) NOT NULL CHECK (ordered_qty > 0),
    unit_price     numeric(18,6) NULL CHECK (unit_price IS NULL OR unit_price >= 0),
    amount         numeric(18,2) GENERATED ALWAYS AS (COALESCE(unit_price,0) * ordered_qty) STORED,

    packing_number varchar(64),
    pcs_per_kanban int,

    status         varchar(32) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','closed','cancelled')),

    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT purchase_order_items_po_line_uq UNIQUE (po_id, line_no)
);

CREATE INDEX IF NOT EXISTS idx_po_items_po_id   ON purchase_order_items (po_id);
CREATE INDEX IF NOT EXISTS idx_po_items_uniq    ON purchase_order_items (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_po_items_status  ON purchase_order_items (status);
