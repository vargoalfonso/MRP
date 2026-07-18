-- Migration 0080 (down): remove stock_restored_at from outgoing_raw_material

DROP INDEX IF EXISTS idx_outgoing_raw_material_stock_restored_at;

ALTER TABLE outgoing_raw_material
    DROP COLUMN IF EXISTS stock_restored_at;
