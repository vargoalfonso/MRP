-- 0084_add_qty_opname_to_dn_items_up.sql
-- [so-packing] Stock opname TIDAK BOLEH menimpa delivery_note_items.quantity,
-- karena kolom itu adalah angka rencana DN sekaligus acuan "Qty maksimal".
-- Hasil hitung fisik per packing kini disimpan di qty_opname.

ALTER TABLE delivery_note_items
    ADD COLUMN IF NOT EXISTS qty_opname NUMERIC(15,4);

ALTER TABLE delivery_note_items
    ADD COLUMN IF NOT EXISTS qty_opname_at TIMESTAMPTZ;

-- Pemulihan data yang sudah terlanjur tertimpa oleh patch sebelumnya.
-- stock_opname_entries.max_qty menyimpan nilai quantity ASLI (dibaca saat entry
-- dibuat, sebelum approve menimpanya), jadi bisa dipakai untuk restore.
UPDATE delivery_note_items dni
SET quantity  = sub.max_qty,
    qty_opname = sub.counted_qty
FROM (
    SELECT DISTINCT ON (packing_number)
           packing_number,
           max_qty,
           counted_qty
    FROM stock_opname_entries
    WHERE packing_number IS NOT NULL
      AND max_qty IS NOT NULL
      AND status = 'approved'
    ORDER BY packing_number, id DESC
) sub
WHERE dni.packing_number = sub.packing_number
  AND dni.quantity <> sub.max_qty;
