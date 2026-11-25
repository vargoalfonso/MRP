-- =============================================================================
-- Migration: 0041_add_model_to_work_order_items_down.sql
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
DROP COLUMN IF EXISTS model;