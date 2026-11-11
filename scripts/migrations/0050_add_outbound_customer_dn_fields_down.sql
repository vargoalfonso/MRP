-- =============================================================================
-- Migration: 0050_add_outbound_customer_dn_fields_down.sql
-- Feature  : Rollback customer delivery scheduling + customer delivery notes
-- =============================================================================

DROP TABLE IF EXISTS delivery_note_logs_customer;
DROP TABLE IF EXISTS delivery_note_items_customer;
DROP TABLE IF EXISTS delivery_notes_customer;
DROP TABLE IF EXISTS delivery_schedule_items_customer;
DROP TABLE IF EXISTS delivery_schedules_customer;
