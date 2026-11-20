-- =============================================================================
-- Migration: 0040_add_work_orders_created_by_down.sql
-- Rollback : Drop WO creator audit fields
-- =============================================================================

DROP INDEX IF EXISTS idx_work_orders_created_by;

ALTER TABLE IF EXISTS work_orders
    DROP COLUMN IF EXISTS created_by_name,
    DROP COLUMN IF EXISTS created_by;
