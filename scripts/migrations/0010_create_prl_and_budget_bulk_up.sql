-- Migration 0010: PRL Forecasts + PO Budget bulk support
-- -----------------------------------------------------------------------

-- -----------------------------------------------------------------------
-- prl_forecasts
-- Header PRL per customer per period.
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS prl_forecasts (
    id           bigserial    PRIMARY KEY,
    prl_number   varchar(64)  NOT NULL,
    customer_id  bigint,
    customer_name varchar(255),
    period       varchar(32)  NOT NULL,   -- "October 2025"
    period_date  date         NOT NULL,
    status       varchar(32)  NOT NULL DEFAULT 'Active'
                     CHECK (status IN ('Active','Inactive','Closed')),
    notes        text,
    created_by   varchar(255),
    created_at   timestamptz  NOT NULL DEFAULT now(),
    updated_at   timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT prl_forecasts_number_period_uq UNIQUE (prl_number, period_date)
);

CREATE INDEX IF NOT EXISTS idx_prl_forecasts_period_date  ON prl_forecasts (period_date DESC);
CREATE INDEX IF NOT EXISTS idx_prl_forecasts_customer_id  ON prl_forecasts (customer_id);
CREATE INDEX IF NOT EXISTS idx_prl_forecasts_status       ON prl_forecasts (status);

-- -----------------------------------------------------------------------
-- prl_forecast_items
-- One row per UNIQ within a PRL.
-- `quantity` is the PRL budget ceiling — no supplier allocation for this
-- item may exceed it (enforced in application layer).
-- -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS prl_forecast_items (
    id                    bigserial      PRIMARY KEY,
    prl_id                bigint         NOT NULL REFERENCES prl_forecasts(id) ON DELETE CASCADE,
    uniq_code             varchar(64)    NOT NULL,
    part_name             varchar(255),
    part_number           varchar(128),
    weight_kg             numeric(15,4),
    quantity              numeric(15,4)  NOT NULL CHECK (quantity > 0),  -- budget ceiling
    existing_raw_material varchar(255),
    uom                   varchar(32),
    created_at            timestamptz    NOT NULL DEFAULT now(),
    updated_at            timestamptz    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_prl_forecast_items_prl_id    ON prl_forecast_items (prl_id);
CREATE INDEX IF NOT EXISTS idx_prl_forecast_items_uniq_code ON prl_forecast_items (uniq_code);

-- -----------------------------------------------------------------------
-- Alter po_budget_entries
-- Add PRL linkage + budget ceiling + subtype for bulk-from-PRL workflow.
-- -----------------------------------------------------------------------
ALTER TABLE po_budget_entries
    ADD COLUMN IF NOT EXISTS prl_id          bigint
        REFERENCES prl_forecasts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS prl_item_id     bigint
        REFERENCES prl_forecast_items(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS budget_qty      numeric(15,4),   -- PRL item qty ceiling (snapshot at creation)
    ADD COLUMN IF NOT EXISTS budget_subtype  varchar(32)      -- adhoc | regular | NULL (single entry)
        CHECK (budget_subtype IN ('adhoc','regular') OR budget_subtype IS NULL);

CREATE INDEX IF NOT EXISTS idx_po_budget_entries_prl_id ON po_budget_entries (prl_id);
