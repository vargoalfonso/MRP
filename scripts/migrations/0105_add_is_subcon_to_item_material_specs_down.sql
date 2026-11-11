UPDATE item_material_specs SET type_material = 'subcon' WHERE is_subcon = true;
ALTER TABLE item_material_specs DROP COLUMN IF EXISTS is_subcon;
