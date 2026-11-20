-- =============================================================================
-- Migration: 0018_procurement_po_ext_up.sql
-- Feature  : Procurement PO — extend purchase_orders for new API layer
--
-- Changes:
--   0. Ensure purchase_orders base table exists (legacy table; may not exist in fresh DB)
--   1. Add po_stage (1=PO1, 2=PO2) to purchase_orders
--   2. Add po_budget_entry_id FK to po_budget_entries (new budget system)
--   3. Add po_budget_entry_id FK to purchase_order_items (trace line → budget)
--   4. Create purchase_order_logs (append-only audit trail per PO)
--   5. Ensure po_type CHECK constraint exists on purchase_orders
--   6. Add po_number unique constraint (idempotent)
--
-- Notes:
--   - purchase_orders.supplier_id  stays BIGINT (legacy supplier table).
--   - purchase_orders.po_budget_id stays BIGINT (legacy po_budget table).
--   - po_budget_entry_id is NEW column linking to the v2 po_budget_entries table.
--   - PO type mixing is prevented at application layer + enforced by po_type column.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 0. Ensure the legacy purchase_orders table exists.
--    In production this table is created by the old Express/Sequelize system.
--    In a fresh Go-only database we create it here so subsequent ALTERs don't fail.
-- po_type values: raw_material | indirect | subcon
-- (aligned with po_budget_entries.budget_type — single consistent value for frontend)
CREATE TABLE IF NOT EXISTS purchase_orders (
    po_id        bigserial    PRIMARY KEY,
    po_type      varchar(32)  NOT NULL CHECK (po_type IN ('raw_material','indirect','subcon')),
    period       varchar(32)  NOT NULL,
    po_number    varchar(128),
    po_budget_id bigint,
    supplier_id  bigint,                -- FK to legacy `supplier` table (BIGINT PK)
    total_incoming int         NOT NULL DEFAULT 0,
    dn_created   int          NOT NULL DEFAULT 0,
    dn_incoming  int          NOT NULL DEFAULT 0,
    status       varchar(32)  NOT NULL DEFAULT 'draft',
    created_at   timestamptz  NOT NULL DEFAULT now(),
    updated_at   timestamptz  NOT NULL DEFAULT now()
);

-- Ensure purchase_order_items base table exists (created in 0014; guard for fresh DB).
CREATE TABLE IF NOT EXISTS purchase_order_items (
    id             bigserial   PRIMARY KEY,
    po_id          bigint      NOT NULL REFERENCES purchase_orders(po_id) ON DELETE CASCADE,
    line_no        int         NOT NULL DEFAULT 1,
    item_uniq_code varchar(64) NOT NULL,
    product_model  varchar(255),
    material_type  varchar(64),
    part_name      varchar(255),
    part_number    varchar(128),
    uom            varchar(32),
    weight_kg      numeric(15,4),
    description    text,
    ordered_qty    numeric(15,4) NOT NULL CHECK (ordered_qty > 0),
    unit_price     numeric(18,6) NULL     CHECK (unit_price IS NULL OR unit_price >= 0),
    amount         numeric(18,2) GENERATED ALWAYS AS (COALESCE(unit_price,0) * ordered_qty) STORED,
    packing_number varchar(64),
    pcs_per_kanban int,
    status         varchar(32) NOT NULL DEFAULT 'open'
                   CHECK (status IN ('open','closed','cancelled')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT purchase_order_items_po_line_uq UNIQUE (po_id, line_no)
);

-- Also ensure the legacy supplier table exists (guard for fresh DB).
CREATE TABLE IF NOT EXISTS supplier (
    supplier_id  bigserial    PRIMARY KEY,
    supplier_name varchar(128) NOT NULL
);

-- 0014 columns — idempotent guard for fresh DB (migration 0014 may not have run).
ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS po_date               date,
    ADD COLUMN IF NOT EXISTS expected_delivery_date date,
    ADD COLUMN IF NOT EXISTS currency              varchar(8),
    ADD COLUMN IF NOT EXISTS total_amount          numeric(18,2),
    ADD COLUMN IF NOT EXISTS external_system       varchar(64),
    ADD COLUMN IF NOT EXISTS external_po_number    varchar(128),
    ADD COLUMN IF NOT EXISTS created_by            varchar(255),
    ADD COLUMN IF NOT EXISTS updated_by            varchar(255);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_period    ON purchase_orders (period);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier  ON purchase_orders (supplier_id, period);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_status    ON purchase_orders (status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_number ON purchase_orders (po_number);

CREATE INDEX IF NOT EXISTS idx_po_items_po_id  ON purchase_order_items (po_id);
CREATE INDEX IF NOT EXISTS idx_po_items_uniq   ON purchase_order_items (item_uniq_code);
CREATE INDEX IF NOT EXISTS idx_po_items_status ON purchase_order_items (status);

-- 1. po_stage: int 1 or 2 (PO1 / PO2 split stage)
ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS po_stage int CHECK (po_stage IN (1, 2));

-- 2. po_budget_entry_id: link PO header to the new po_budget_entries row
--    ON DELETE SET NULL so deleting a budget entry does not cascade to PO.
ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS po_budget_entry_id bigint
        REFERENCES po_budget_entries(id) ON DELETE SET NULL;

-- 3. po_budget_entry_id on purchase_order_items: trace each line item to its budget entry
ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS po_budget_entry_id bigint
        REFERENCES po_budget_entries(id) ON DELETE SET NULL;

-- Indexes for new columns
CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_stage
    ON purchase_orders (po_stage);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_po_budget_entry
    ON purchase_orders (po_budget_entry_id);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_type_period_stage
    ON purchase_orders (po_type, period, po_stage);

CREATE INDEX IF NOT EXISTS idx_po_items_budget_entry
    ON purchase_order_items (po_budget_entry_id);

-- 4. purchase_order_logs — append-only audit trail
--    action: Created | Updated | Cancelled | Approved | Rejected | DNCreated | ...
CREATE TABLE IF NOT EXISTS purchase_order_logs (
    id          bigserial    PRIMARY KEY,
    po_id       bigint       NOT NULL REFERENCES purchase_orders(po_id) ON DELETE CASCADE,
    action      varchar(64)  NOT NULL,
    notes       text,
    username    varchar(255),
    occurred_at timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_po_logs_po_id
    ON purchase_order_logs (po_id);

CREATE INDEX IF NOT EXISTS idx_po_logs_occurred_at
    ON purchase_order_logs (occurred_at DESC);

-- 5. Ensure po_type has the correct CHECK constraint.
--    Values match po_budget_entries.budget_type so frontend uses one consistent value.
DO $$
BEGIN
    BEGIN
        ALTER TABLE purchase_orders
            ADD CONSTRAINT purchase_orders_po_type_check
            CHECK (po_type IN ('raw_material', 'indirect', 'subcon'));
    EXCEPTION WHEN duplicate_object THEN
        NULL; -- constraint already exists, skip
    END;
END;
$$;

-- 6. Ensure po_number unique constraint
DO $$
BEGIN
    BEGIN
        ALTER TABLE purchase_orders
            ADD CONSTRAINT purchase_orders_po_number_uq UNIQUE (po_number);
    EXCEPTION WHEN duplicate_object THEN
        NULL;
    END;
END;
$$;
