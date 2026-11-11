-- Drop FK constraint on item_material_specs.supplier_id (suppliers.id is bigint, not uuid).
-- Keep supplier_id as varchar for reference; supplier_name is stored alongside it.
ALTER TABLE item_material_specs
    DROP CONSTRAINT IF EXISTS item_material_specs_supplier_id_fkey;

ALTER TABLE item_material_specs
    ALTER COLUMN supplier_id TYPE varchar(64) USING supplier_id::text;
