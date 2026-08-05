-- 0092_add_qty_opname_to_work_order_items_down.sql
ALTER TABLE work_order_items DROP COLUMN IF EXISTS qty_opname_at;
ALTER TABLE work_order_items DROP COLUMN IF EXISTS qty_opname;
