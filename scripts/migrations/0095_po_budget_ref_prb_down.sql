-- Migration 0095: rollback — PRB kembali ke POB
-- Membalik prefix `po_budget_ref` dari PRB-{YYYY}-{TYPE}-{id}
-- ke format lama POB-{YYYY}-{TYPE}-{id} (identik dengan migrasi 0019).

DROP INDEX IF EXISTS idx_po_budget_entries_po_budget_ref;

ALTER TABLE po_budget_entries
  DROP COLUMN IF EXISTS po_budget_ref;

ALTER TABLE po_budget_entries
  ADD COLUMN po_budget_ref varchar(32)
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
