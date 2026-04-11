-- Migration 0019: rollback

DROP INDEX IF EXISTS idx_po_budget_entries_po_budget_ref;

ALTER TABLE po_budget_entries
  DROP COLUMN IF EXISTS po_budget_ref;
