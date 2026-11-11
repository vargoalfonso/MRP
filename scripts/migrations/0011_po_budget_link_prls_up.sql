-- Migration 0011: Link PO budget entries to data-asli PRL (table: prls)
-- -----------------------------------------------------------------------
-- Existing bulk-from-PRL implementation (migration 0010) linked to
-- prl_forecasts/prl_forecast_items using bigint FKs.
--
-- Production uses table "prls" with prl_id (varchar(32)) as document key.
-- We add non-FK linkage columns to po_budget_entries so the PO Budget module
-- can enforce allocation ceilings and show PRL allocation correctly.

ALTER TABLE po_budget_entries
    ADD COLUMN IF NOT EXISTS prl_ref    varchar(32),
    ADD COLUMN IF NOT EXISTS prl_row_id bigint;

CREATE INDEX IF NOT EXISTS idx_po_budget_entries_prl_ref    ON po_budget_entries (prl_ref);
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_prl_row_id ON po_budget_entries (prl_row_id);
