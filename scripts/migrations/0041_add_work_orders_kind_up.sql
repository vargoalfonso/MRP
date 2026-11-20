-- =============================================================================
-- Migration: 0041_add_work_orders_kind_up.sql
-- Feature  : Work Order kind flag (one table for multiple WO flows)
-- Notes    : Distinguish standard WO vs bulk WO vs RM processing WO
-- =============================================================================

ALTER TABLE IF EXISTS work_orders
ADD COLUMN IF NOT EXISTS wo_kind VARCHAR(32) NOT NULL DEFAULT 'standard';

CREATE INDEX IF NOT EXISTS idx_work_orders_wo_kind ON work_orders (wo_kind);