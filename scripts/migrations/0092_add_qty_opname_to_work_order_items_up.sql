-- 0092_add_qty_opname_to_work_order_items_up.sql
-- [so-fg-packing] Stock opname Finished Goods di-scan per kanban/packing list
-- Work Order (work_order_items.kanban_number). Sama seperti delivery_note_items
-- (migrasi 0084), hasil hitung fisik per packing WO disimpan di qty_opname
-- supaya delta antar-opname akurat dan work_order_items.total_good_qty (angka
-- produksi WO) TIDAK tertimpa.

ALTER TABLE work_order_items
    ADD COLUMN IF NOT EXISTS qty_opname NUMERIC(15,4);

ALTER TABLE work_order_items
    ADD COLUMN IF NOT EXISTS qty_opname_at TIMESTAMPTZ;
