-- Add percentage column: supplier's share of supply for this item (0–100)
ALTER TABLE supplier_item
ADD COLUMN IF NOT EXISTS percentage NUMERIC(5,2) NULL;

-- Drop old unique key (supplier_uuid, uniq_code) — too strict, blocks same
-- supplier from supplying the same item under different types.
ALTER TABLE supplier_item
DROP CONSTRAINT IF EXISTS supplier_item_supplier_uniq_key;

-- New unique key: (supplier_uuid, uniq_code, type) — one row per supplier+item+type combo.
ALTER TABLE supplier_item
ADD CONSTRAINT supplier_item_supplier_uniq_type_key UNIQUE (supplier_uuid, uniq_code, type);
