ALTER TABLE supplier_item
DROP CONSTRAINT IF EXISTS supplier_item_supplier_uniq_type_key;

ALTER TABLE supplier_item
ADD CONSTRAINT supplier_item_supplier_uniq_key UNIQUE (supplier_uuid, uniq_code);

ALTER TABLE supplier_item
DROP COLUMN IF EXISTS percentage;
