-- Migration: 0079_add_prl_type_to_prls_up.sql
-- Add prl_type flag (reguler | additional) to PRLs.

ALTER TABLE prls
ADD COLUMN IF NOT EXISTS prl_type varchar(20) NOT NULL DEFAULT 'reguler';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'prls_prl_type_ck'
    ) THEN
        ALTER TABLE prls
        ADD CONSTRAINT prls_prl_type_ck
        CHECK (prl_type IN ('reguler', 'additional'));
    END IF;
END $$;
