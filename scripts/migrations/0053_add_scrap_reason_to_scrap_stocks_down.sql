-- Rollback 0053
ALTER TABLE scrap_stocks
    DROP COLUMN IF EXISTS disposal_reason;
