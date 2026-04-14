-- =============================================================================
-- Migration: 0038_add_kanban_param_fields_to_work_order_items_down.sql
-- Rollback : Drop WO item kanban parameter fields
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
    DROP COLUMN IF EXISTS kanban_seq,
    DROP COLUMN IF EXISTS kanban_param_number;
