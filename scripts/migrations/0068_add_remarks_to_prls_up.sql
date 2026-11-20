-- Add remarks column to prls table
ALTER TABLE prls
ADD COLUMN IF NOT EXISTS remarks text;

-- Backfill empty string for existing rows (optional)
-- UPDATE prls SET remarks = NULL WHERE remarks = '';

-- Note: This migration is additive and safe to run in production.
