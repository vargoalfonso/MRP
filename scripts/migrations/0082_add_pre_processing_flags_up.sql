-- Migration: 0082_add_pre_processing_flags_up.sql
-- Add pre_processing flag to work_orders (RM processing) and raw_materials.
-- When an RM processing work order flagged pre_processing completes (scan done),
-- the source raw material stock is reduced and the processed target uniq is
-- registered into raw_materials with pre_processing = true.

ALTER TABLE work_orders
ADD COLUMN IF NOT EXISTS pre_processing boolean NOT NULL DEFAULT false;

ALTER TABLE raw_materials
ADD COLUMN IF NOT EXISTS pre_processing boolean NOT NULL DEFAULT false;
