-- Migration: 0025_fix_qc_tasks_incoming_dn_item_id_up.sql
-- Fix: qc_tasks.incoming_dn_item_id was created as UUID in older schema.
--      Migration 0015 used CREATE TABLE IF NOT EXISTS so it was skipped.
--      Alter the column to bigint to match delivery_note_items.id.

ALTER TABLE qc_tasks
    ALTER COLUMN incoming_dn_item_id TYPE bigint
    USING NULL;

-- Re-apply NOT NULL default and add other columns if missing (0015 guards).
ALTER TABLE qc_tasks
    ADD COLUMN IF NOT EXISTS good_quantity   int,
    ADD COLUMN IF NOT EXISTS ng_quantity     int,
    ADD COLUMN IF NOT EXISTS scrap_quantity  int,
    ADD COLUMN IF NOT EXISTS round          int          NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS round_results  jsonb        NOT NULL DEFAULT '[]';
