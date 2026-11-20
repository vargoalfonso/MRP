DROP INDEX IF EXISTS idx_raw_materials_rm_master;
ALTER TABLE raw_materials DROP COLUMN IF EXISTS raw_material_master_id;
DROP INDEX IF EXISTS idx_item_material_specs_rm_master;
ALTER TABLE item_material_specs DROP COLUMN IF EXISTS raw_material_master_id;
DROP TABLE IF EXISTS raw_material_masters;
