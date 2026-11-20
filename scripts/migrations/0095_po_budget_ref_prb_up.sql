-- Migration 0095: Rename PO Budget reference prefix POB -> PRB
-- Format lama:  POB-{YYYY}-{TYPE}-{id}, contoh POB-2025-RM-000123
-- Format baru:  PRB-{YYYY}-{TYPE}-{id}, contoh PRB-2025-RM-000123
--
-- Kolom `po_budget_ref` adalah GENERATED ALWAYS AS ... STORED, sehingga
-- ekspresinya tidak bisa diubah pakai ALTER COLUMN. Cara yang aman:
--   1. DROP unique index yang mereferensikan kolom.
--   2. DROP kolom generated (data lama otomatis ikut ter-drop, tapi tidak
--      masalah karena nilainya di-recompute dari kolom lain).
--   3. ADD kolom lagi dengan ekspresi yang sama tapi prefix 'PRB-'.
--   4. CREATE kembali unique index.
--
-- Catatan: Postgres mewajibkan ekspresi generated column bersifat IMMUTABLE,
-- jadi kita tetap menghindari to_char(...) dan tetap memakai lpad + EXTRACT
-- persis seperti migrasi 0019.

DROP INDEX IF EXISTS idx_po_budget_entries_po_budget_ref;

ALTER TABLE po_budget_entries
  DROP COLUMN IF EXISTS po_budget_ref;

ALTER TABLE po_budget_entries
  ADD COLUMN po_budget_ref varchar(32)
  GENERATED ALWAYS AS (
    'PRB-' || lpad((EXTRACT(YEAR FROM period_date)::int)::text, 4, '0') || '-' ||
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
