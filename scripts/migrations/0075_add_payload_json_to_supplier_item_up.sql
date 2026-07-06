ALTER TABLE supplier_item
ADD COLUMN IF NOT EXISTS payload_json JSONB;
