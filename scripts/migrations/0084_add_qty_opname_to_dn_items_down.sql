-- 0084_add_qty_opname_to_dn_items_down.sql
ALTER TABLE delivery_note_items DROP COLUMN IF EXISTS qty_opname_at;
ALTER TABLE delivery_note_items DROP COLUMN IF EXISTS qty_opname;
