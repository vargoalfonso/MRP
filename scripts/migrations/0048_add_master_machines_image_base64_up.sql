-- =============================================================================
-- Migration: 0048_add_master_machines_image_base64_up.sql
-- Module   : Master Machines
-- Purpose  : Add optional base64 image column
-- =============================================================================

ALTER TABLE master_machines
    ADD COLUMN IF NOT EXISTS image_base64 TEXT NULL;
