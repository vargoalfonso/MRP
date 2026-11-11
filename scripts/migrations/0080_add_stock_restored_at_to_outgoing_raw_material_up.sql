-- Migration 0080: add stock_restored_at to outgoing_raw_material
-- Supports the manual "restore stock" action for outgoing RM transactions.
-- A non-null value means the transaction quantity has already been returned to
-- raw_materials stock, and guards against a second (double) restore.

ALTER TABLE outgoing_raw_material
    ADD COLUMN IF NOT EXISTS stock_restored_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outgoing_raw_material_stock_restored_at
    ON outgoing_raw_material (stock_restored_at);
