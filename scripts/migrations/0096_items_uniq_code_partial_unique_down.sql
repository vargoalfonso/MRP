-- 0096_items_uniq_code_partial_unique_down.sql
--
-- Rollback: kembalikan column-level UNIQUE constraint bawaan 0003 dan
-- lookup index non-unique. CATATAN untuk DBA: jika sudah ada duplicate
-- `uniq_code` di antara baris soft-delete + aktif, ALTER TABLE di bawah
-- akan gagal. Bersihkan/dekorasi ulang uniq_code baris soft-delete
-- (misal tambahkan suffix `-deleted-<id>`) sebelum menjalankan down.

BEGIN;

DROP INDEX IF EXISTS uq_items_uniq_code_active;

CREATE INDEX IF NOT EXISTS idx_items_uniq_code
	ON items(uniq_code)
	WHERE deleted_at IS NULL;

ALTER TABLE items
	ADD CONSTRAINT items_uniq_code_key UNIQUE (uniq_code);

COMMIT;
