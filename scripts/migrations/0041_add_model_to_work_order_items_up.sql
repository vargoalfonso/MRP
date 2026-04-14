-- =============================================================================
-- Migration: 0041_add_model_to_work_order_items_up.sql
-- Feature  : Store denormalized item model on WO items (copied from items table at create time)
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
ADD COLUMN IF NOT EXISTS model VARCHAR(128);