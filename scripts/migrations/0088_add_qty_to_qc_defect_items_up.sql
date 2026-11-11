-- 0088_add_qty_to_qc_defect_items_up.sql
-- Menyimpan Qty eksplisit per baris issue (QC Process round 1 & 2) dan per
-- baris Reason/Info (round 3) di action-ui.
-- Catatan: runner migrasi hanya menjalankan STATEMENT PERTAMA per file,
-- jadi semua kolom ditambahkan dalam satu ALTER TABLE.
ALTER TABLE qc_defect_items
    ADD COLUMN IF NOT EXISTS qty NUMERIC(15,4) NOT NULL DEFAULT 0;
