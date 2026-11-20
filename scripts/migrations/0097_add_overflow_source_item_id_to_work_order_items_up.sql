-- 0097: link kanban overflow (dibuat di QC Round 3) ke kanban sumber.
ALTER TABLE work_order_items
    ADD COLUMN IF NOT EXISTS overflow_source_item_id BIGINT NULL;

CREATE INDEX IF NOT EXISTS idx_work_order_items_overflow_source
    ON work_order_items (overflow_source_item_id);
