-- Migration: 0082_add_pre_processing_flags_down.sql

ALTER TABLE raw_materials
DROP COLUMN IF EXISTS pre_processing;

ALTER TABLE work_orders
DROP COLUMN IF EXISTS pre_processing;
