ALTER TABLE item_material_specs
    ADD COLUMN IF NOT EXISTS customer_cycle VARCHAR(100) NULL;
