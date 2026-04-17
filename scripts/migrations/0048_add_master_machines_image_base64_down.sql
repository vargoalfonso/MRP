-- =============================================================================
-- Migration: 0048_add_master_machines_image_base64_down.sql
-- Module   : Master Machines
-- Purpose  : Drop optional base64 image column
-- =============================================================================

ALTER TABLE master_machines
    DROP COLUMN IF EXISTS image_base64;
