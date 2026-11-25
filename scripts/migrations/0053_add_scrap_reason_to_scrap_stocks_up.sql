-- Migration 0053: Add disposal_reason to scrap_stocks
ALTER TABLE scrap_stocks
    ADD COLUMN IF NOT EXISTS disposal_reason VARCHAR(128);
