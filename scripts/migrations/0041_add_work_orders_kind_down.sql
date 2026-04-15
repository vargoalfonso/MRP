-- =============================================================================
-- Migration: 0041_add_work_orders_kind_down.sql
-- Rollback : Drop WO kind flag
-- =============================================================================

DROP INDEX IF EXISTS idx_work_orders_wo_kind;

ALTER TABLE IF EXISTS work_orders
    DROP COLUMN IF EXISTS wo_kind;
