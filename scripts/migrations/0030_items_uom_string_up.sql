-- Replace uom_id (int FK to uom_parameters) with uom (varchar) on items table.
-- Migrate existing values by joining uom_parameters.
ALTER TABLE items ADD COLUMN IF NOT EXISTS uom varchar(32);

UPDATE items
SET uom = up.name
FROM uom_parameters up
WHERE up.id = items.uom_id;

ALTER TABLE items DROP COLUMN IF EXISTS uom_id;

-- bom_lines also has uom_id FK
ALTER TABLE bom_lines ADD COLUMN IF NOT EXISTS uom varchar(32);
ALTER TABLE bom_lines DROP COLUMN IF EXISTS uom_id;
