-- Migration: 0079_add_prl_type_to_prls_down.sql

ALTER TABLE prls
DROP CONSTRAINT IF EXISTS prls_prl_type_ck;

ALTER TABLE prls
DROP COLUMN IF EXISTS prl_type;
