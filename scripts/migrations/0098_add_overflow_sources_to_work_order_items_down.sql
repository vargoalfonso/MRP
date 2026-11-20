-- [overflow-topup] rollback kolom overflow_sources
ALTER TABLE work_order_items
    DROP COLUMN IF EXISTS overflow_sources;
