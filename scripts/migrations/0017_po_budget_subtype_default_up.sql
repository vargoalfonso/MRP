-- =============================================================================
-- Migration: 0017_po_budget_subtype_default_up.sql
-- Module   : PO Budget
-- Purpose  : Separate budget_type (raw_material/subcon/indirect) from budget_subtype
--            (regular/adhoc) by making budget_subtype non-null with default.
-- Notes    : Existing migration 0010 added budget_subtype as nullable. In case
--            the column doesn't exist yet (0010 not applied), this migration
--            will create it safely.
-- =============================================================================

-- 1) Ensure column exists
ALTER TABLE po_budget_entries
  ADD COLUMN IF NOT EXISTS budget_subtype varchar(32);

-- 2) Ensure constraint exists (adhoc|regular)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'po_budget_entries_budget_subtype_ck'
  ) THEN
    ALTER TABLE po_budget_entries
      ADD CONSTRAINT po_budget_entries_budget_subtype_ck
      CHECK (budget_subtype IN ('adhoc','regular'));
  END IF;
END $$;

-- 3) Backfill existing rows (incl. pre-existing NULL)
UPDATE po_budget_entries
SET budget_subtype = 'regular'
WHERE budget_subtype IS NULL;

-- 4) Enforce default + non-null going forward
ALTER TABLE po_budget_entries
  ALTER COLUMN budget_subtype SET DEFAULT 'regular',
  ALTER COLUMN budget_subtype SET NOT NULL;

-- 5) Index for filtering (list/aggregate)
CREATE INDEX IF NOT EXISTS idx_po_budget_entries_budget_subtype
  ON po_budget_entries (budget_subtype);
