-- Restore uom_id FK (data loss on down — uom string cannot be reliably mapped back)
ALTER TABLE items ADD COLUMN IF NOT EXISTS uom_id bigint NOT NULL DEFAULT 1;
ALTER TABLE items DROP COLUMN IF EXISTS uom;

ALTER TABLE bom_lines ADD COLUMN IF NOT EXISTS uom_id bigint;
ALTER TABLE bom_lines DROP COLUMN IF EXISTS uom;
