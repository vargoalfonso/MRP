-- Add denormalized supplier_name to material specs so read APIs do not need supplier join
ALTER TABLE item_material_specs
ADD COLUMN IF NOT EXISTS supplier_name VARCHAR(255);

-- Backfill from suppliers for existing rows
UPDATE item_material_specs ims
SET
    supplier_name = s.supplier_name
FROM suppliers s
WHERE
    ims.supplier_id = s.id
    AND (
        ims.supplier_name IS NULL
        OR ims.supplier_name = ''
    );

CREATE INDEX IF NOT EXISTS idx_item_material_specs_supplier_name ON item_material_specs (supplier_name);