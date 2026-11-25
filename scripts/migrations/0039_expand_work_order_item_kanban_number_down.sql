-- =============================================================================
-- Migration: 0039_expand_work_order_item_kanban_number_down.sql
-- Rollback : Shrink kanban_number length
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
    ALTER COLUMN kanban_number TYPE VARCHAR(64);
