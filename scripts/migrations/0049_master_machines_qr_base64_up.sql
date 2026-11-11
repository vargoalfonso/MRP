-- =============================================================================
-- Migration: 0049_master_machines_qr_base64_up.sql
-- Module   : Master Machines
-- Purpose  : Store machine QR code as base64 data URL
-- Notes    : If earlier column image_base64 exists, rename it.
-- =============================================================================

DO $$
BEGIN
    -- If qr_image_base64 already exists, do nothing.
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'master_machines'
          AND column_name = 'qr_image_base64'
    ) THEN
        RETURN;
    END IF;

    -- If legacy image_base64 exists, rename to qr_image_base64.
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'master_machines'
          AND column_name = 'image_base64'
    ) THEN
        EXECUTE 'ALTER TABLE master_machines RENAME COLUMN image_base64 TO qr_image_base64';
    ELSE
        EXECUTE 'ALTER TABLE master_machines ADD COLUMN qr_image_base64 TEXT NULL';
    END IF;
END $$;
