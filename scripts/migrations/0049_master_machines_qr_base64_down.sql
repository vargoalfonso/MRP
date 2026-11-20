-- =============================================================================
-- Migration: 0049_master_machines_qr_base64_down.sql
-- Module   : Master Machines
-- Purpose  : Drop qr_image_base64
-- =============================================================================

ALTER TABLE master_machines
    DROP COLUMN IF EXISTS qr_image_base64;
