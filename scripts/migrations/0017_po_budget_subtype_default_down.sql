-- =============================================================================
-- Migration: 0017_po_budget_subtype_default_down.sql
-- =============================================================================

DROP INDEX IF EXISTS idx_po_budget_entries_budget_subtype;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'po_budget_entries'
      AND column_name = 'budget_subtype'
  ) THEN
    ALTER TABLE po_budget_entries
      ALTER COLUMN budget_subtype DROP NOT NULL,
      ALTER COLUMN budget_subtype DROP DEFAULT;
  END IF;
END $$;
