-- =============================================================
-- 0085 : kolom detail issue (free text) untuk jenis issue "Lainnya"
--
-- Runner migrasi pada project ini hanya menjalankan STATEMENT PERTAMA
-- per file, jadi file ini sengaja hanya berisi satu statement.
--
-- Aman dijalankan berulang kali (IF NOT EXISTS).
-- =============================================================
ALTER TABLE production_issues ADD COLUMN IF NOT EXISTS issue_note TEXT;
