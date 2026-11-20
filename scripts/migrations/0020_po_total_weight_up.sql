-- =============================================================================
-- Migration: 0020_po_total_weight_up.sql
-- Changes:
--   1. Add total_weight column to purchase_orders
--   2. Extend po_budget_entries status CHECK to allow 'PO Generated'
--      (set when a budget entry has been used in a PO — duplicate guard)
-- =============================================================================

-- 1. total_weight on purchase_orders
ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS total_weight numeric(15,4);

-- 2. Extend po_budget_entries status constraint to include 'PO Generated'
ALTER TABLE po_budget_entries
    DROP CONSTRAINT IF EXISTS po_budget_entries_status_check;

ALTER TABLE po_budget_entries
    ADD CONSTRAINT po_budget_entries_status_check
        CHECK (status IN ('Draft','Pending','Approved','Rejected','PO Generated'));
