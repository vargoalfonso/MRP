-- QPU parent (header BOM). Default 1 = parent dihitung 1 unit acuan.
ALTER TABLE bom_item
    ADD COLUMN IF NOT EXISTS qty_per_uniq NUMERIC(18,6) NOT NULL DEFAULT 1;
