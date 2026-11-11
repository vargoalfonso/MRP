-- =============================================================================
-- Migration: 0040_add_work_orders_created_by_up.sql
-- Feature  : Work Order audit fields
-- Notes    : Store creator identity from login
-- =============================================================================

ALTER TABLE IF EXISTS work_orders
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_work_orders_created_by ON work_orders (created_by);
