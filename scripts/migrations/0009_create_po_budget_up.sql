-- Migration 0009: PO Budget Management
-- Tables: po_split_settings, po_budget_entries

-- -----------------------------------------------------------------------
-- po_split_settings
-- Parameterize PO1 / PO2 percentages per budget_type.
-- Default: PO1=60%, PO2=40% (kanban packing logic).
-- A single PO is split into two payment stages (e.g. 60% upfront, 40% on delivery).
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS po_split_settings (
    id          bigserial PRIMARY KEY,
    budget_type varchar(32)    NOT NULL,                        -- raw_material | subcon | indirect
    po1_pct     numeric(5,2)   NOT NULL DEFAULT 60.00
                    CHECK (po1_pct >= 0 AND po1_pct <= 100),
    po2_pct     numeric(5,2)   NOT NULL DEFAULT 40.00
                    CHECK (po2_pct >= 0 AND po2_pct <= 100),
    description text,
    created_at  timestamptz    NOT NULL DEFAULT now(),
    updated_at  timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT po_split_settings_budget_type_unique UNIQUE (budget_type),
    CONSTRAINT po_split_pct_sum CHECK (po1_pct + po2_pct = 100)
);

-- Seed default split settings
INSERT INTO po_split_settings (budget_type, po1_pct, po2_pct, description)
VALUES
  ('raw_material', 60, 40, 'Default kanban packing split for raw material PO'),
  ('subcon',       60, 40, 'Default kanban packing split for subcon PO'),
  ('indirect',     60, 40, 'Default kanban packing split for indirect PO')
ON CONFLICT (budget_type) DO NOTHING;

-- -----------------------------------------------------------------------
-- po_budget_entries
-- One row = one budget line for a Uniq / customer / period combination.
-- Multiple rows with the same uniq_code + period are aggregated for reporting.
-- PO1 and PO2 are STORED (not computed) so each invoice can have independent
-- overrides; the default is derived from po_split_settings at creation time.
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS po_budget_entries (
    id              bigserial PRIMARY KEY,
    budget_type     varchar(32)    NOT NULL
                        CHECK (budget_type IN ('raw_material','subcon','indirect')),

  
    customer_id     bigint,
    customer_name   varchar(255),

    uniq_code       varchar(64)    NOT NULL,
    product_model   varchar(255),
    material_type   varchar(64),           -- Pipe | Steel Plate | Wire | Add On | etc.
    part_name       varchar(255),
    part_number     varchar(128),

    quantity        numeric(15,4)  NOT NULL DEFAULT 0,
    uom             varchar(32),           -- Kg | pcs | sheet
    weight_kg       numeric(15,4),

    description     text,

    supplier_id     uuid,
    supplier_name   varchar(255),

    period          varchar(32)    NOT NULL,   -- display label, e.g. "October 2025"
    period_date     date           NOT NULL,   -- first day of month, used for ordering/filtering

    sales_plan         numeric(15,4) NOT NULL DEFAULT 0,
    purchase_request   numeric(15,4) NOT NULL DEFAULT 0,   -- PR (Units)

    po1_pct         numeric(5,2)   NOT NULL DEFAULT 60,
    po2_pct         numeric(5,2)   NOT NULL DEFAULT 40,
    po1_qty         numeric(15,4)  GENERATED ALWAYS AS (purchase_request * po1_pct / 100) STORED,
    po2_qty         numeric(15,4)  GENERATED ALWAYS AS (purchase_request * po2_pct / 100) STORED,
    total_po        numeric(15,4)  GENERATED ALWAYS AS (purchase_request * (po1_pct + po2_pct) / 100) STORED,

    prl             numeric(15,4)  NOT NULL DEFAULT 0,


    status          varchar(32)    NOT NULL DEFAULT 'Draft'
                        CHECK (status IN ('Draft','Pending','Approved','Rejected')),
    approved_by     varchar(255),
    approved_at     timestamptz,

    created_by      varchar(255),
    updated_by      varchar(255),
    created_at      timestamptz    NOT NULL DEFAULT now(),
    updated_at      timestamptz    NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_budget_type    ON po_budget_entries (budget_type);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_uniq_code      ON po_budget_entries (uniq_code);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_period_date    ON po_budget_entries (period_date DESC);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_customer_id    ON po_budget_entries (customer_id);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_status         ON po_budget_entries (status);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_uniq_period    ON po_budget_entries (uniq_code, period_date, budget_type);
