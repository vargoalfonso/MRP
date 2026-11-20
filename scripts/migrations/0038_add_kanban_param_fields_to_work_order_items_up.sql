-- =============================================================================
-- Migration: 0038_add_kanban_param_fields_to_work_order_items_up.sql
-- Feature  : Store kanban parameter identity on WO items
-- Notes    : Keep WO item UUID as QR identity; store base KBN separately for traceability
-- =============================================================================

ALTER TABLE IF EXISTS work_order_items
    ADD COLUMN IF NOT EXISTS kanban_param_number VARCHAR(50),
    ADD COLUMN IF NOT EXISTS kanban_seq INT;
