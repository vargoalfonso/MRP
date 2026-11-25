ALTER TABLE item_material_specs
    ALTER COLUMN supplier_id TYPE uuid USING supplier_id::uuid;

ALTER TABLE item_material_specs
    ADD CONSTRAINT item_material_specs_supplier_id_fkey
    FOREIGN KEY (supplier_id) REFERENCES suppliers(id);
