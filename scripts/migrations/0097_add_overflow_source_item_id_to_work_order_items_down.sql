-- 0097 down
DROP INDEX IF EXISTS idx_work_order_items_overflow_source;
ALTER TABLE work_order_items
    DROP COLUMN IF EXISTS overflow_source_item_id;
