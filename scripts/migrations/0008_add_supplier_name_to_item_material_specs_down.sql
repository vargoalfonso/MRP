-- Rollback denormalized supplier_name
DROP INDEX IF EXISTS idx_item_material_specs_supplier_name;

ALTER TABLE item_material_specs DROP COLUMN IF EXISTS supplier_name;