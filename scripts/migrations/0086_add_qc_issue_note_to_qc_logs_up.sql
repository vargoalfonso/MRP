-- 0086_add_qc_issue_note_to_qc_logs_up.sql
-- Menyimpan jenis issue + detail bebas ("Other") dari QC Process round 1 & 2.
-- Catatan: runner migrasi hanya menjalankan STATEMENT PERTAMA per file,
-- jadi semua kolom ditambahkan dalam satu ALTER TABLE.
ALTER TABLE qc_logs
    ADD COLUMN IF NOT EXISTS issue_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS issue_note VARCHAR(255);
