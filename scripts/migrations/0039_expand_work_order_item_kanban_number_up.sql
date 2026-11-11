-- =============================================================================
-- Migration: 0039_expand_work_order_item_kanban_number_up.sql
-- Feature  : Expand work_order_items.kanban_number length
-- Notes    : New format embeds WO number for uniqueness
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
    ALTER COLUMN kanban_number TYPE VARCHAR(150);
