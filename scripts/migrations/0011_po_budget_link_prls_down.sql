-- Rollback 0011: remove linkage columns to prls

ALTER TABLE po_budget_entries
    DROP COLUMN IF EXISTS prl_ref,
    DROP COLUMN IF EXISTS prl_row_id;

DROP INDEX IF EXISTS idx_po_budget_entries_prl_ref;
DROP INDEX IF EXISTS idx_po_budget_entries_prl_row_id;
