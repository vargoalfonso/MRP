-- 0087_add_qc_finish_reason_to_qc_tasks_up.sql
-- Menyimpan keterangan defect (NG) & scrap dari Final Inspection (round 3).
-- Catatan: runner migrasi hanya menjalankan STATEMENT PERTAMA per file,
-- jadi semua kolom ditambahkan dalam satu ALTER TABLE.
ALTER TABLE qc_tasks
    ADD COLUMN IF NOT EXISTS defect_reason VARCHAR(255),
    ADD COLUMN IF NOT EXISTS scrap_reason VARCHAR(255);
