-- Migration 0019: Add PO Budget reference code
-- Adds a human-friendly identifier to each po_budget_entries row.
-- Format: POB-{YYYY}-{TYPE}-{id}, e.g. POB-2025-RM-000123

-- NOTE: Postgres requires generated column expressions to be IMMUTABLE.
-- We avoid to_char(...), which is STABLE (depends on locale settings).
ALTER TABLE po_budget_entries
  ADD COLUMN IF NOT EXISTS po_budget_ref varchar(32)
  GENERATED ALWAYS AS (
    'POB-' || lpad((EXTRACT(YEAR FROM period_date)::int)::text, 4, '0') || '-' ||
    CASE budget_type
      WHEN 'raw_material' THEN 'RM'
      WHEN 'subcon' THEN 'SC'
      WHEN 'indirect' THEN 'IB'
      ELSE 'UNK'
    END ||
    '-' || lpad(id::text, 6, '0')
  ) STORED;

CREATE UNIQUE INDEX IF NOT EXISTS idx_po_budget_entries_po_budget_ref
  ON po_budget_entries (po_budget_ref);
