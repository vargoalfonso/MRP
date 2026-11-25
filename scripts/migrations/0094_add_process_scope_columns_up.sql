-- 0094_add_process_scope_columns_up.sql
-- [proc-scope] / [wip-scope]
--
-- Satu work_order_item dipakai untuk SEMUA proses dalam satu WO item, sehingga
-- raw_material_logs (yang hanya berkunci wo_item_id) bocor antar-proses: log RM
-- proses 1 ikut terbaca sebagai material proses 2. Kolom process_name + step_seq
-- membuat log RM terpisah per step proses.
--
-- wip_items.from_process menyimpan proses yang MENGHASILKAN stok WIP, supaya
-- baris antrean (status = 'queue') ditampilkan sebagai hasil proses sebelumnya,
-- bukan seolah-olah proses berikutnya sudah berjalan.

ALTER TABLE raw_material_logs
    ADD COLUMN IF NOT EXISTS process_name VARCHAR(100);

ALTER TABLE raw_material_logs
    ADD COLUMN IF NOT EXISTS step_seq INTEGER;

CREATE INDEX IF NOT EXISTS idx_raw_material_logs_process_name
    ON raw_material_logs (process_name);

CREATE INDEX IF NOT EXISTS idx_raw_material_logs_step_seq
    ON raw_material_logs (step_seq);

ALTER TABLE wip_items
    ADD COLUMN IF NOT EXISTS from_process VARCHAR(100);

-- Catatan: baris lama sengaja dibiarkan NULL. Kode membacanya dengan
-- COALESCE(step_seq, 0) <= 1, jadi data lama otomatis dianggap milik step 1.
