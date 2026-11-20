-- =============================================================================
-- Migration: 0020_po_total_weight_down.sql
-- Rollback : Undo changes from 0020_po_total_weight_up.sql
-- =============================================================================

ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS total_weight;

-- Revert po_budget_entries status constraint to original values
ALTER TABLE po_budget_entries
    DROP CONSTRAINT IF EXISTS po_budget_entries_status_check;

ALTER TABLE po_budget_entries
    ADD CONSTRAINT po_budget_entries_status_check
        CHECK (status IN ('Draft','Pending','Approved','Rejected'));
