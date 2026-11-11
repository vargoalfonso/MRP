-- 0096_items_uniq_code_partial_unique_up.sql
--
-- Konteks: Sebelumnya kolom `items.uniq_code` didefinisikan sebagai
-- `VARCHAR(64) NOT NULL UNIQUE` di migrasi 0003. Constraint UNIQUE tingkat
-- kolom berlaku untuk SEMUA baris tabel, termasuk baris yang sudah
-- soft-delete (deleted_at IS NOT NULL), sehingga user tidak bisa membuat
-- item baru dengan `uniq_code` yang sama dengan item yang sudah dihapus
-- lewat web (contoh: BT208 muncul di list tapi tidak bisa dipakai ulang).
--
-- Perubahan: constraint UNIQUE tingkat kolom di-drop, digantikan dengan
-- partial UNIQUE index yang hanya menjamin keunikan pada baris AKTIF
-- (deleted_at IS NULL). Baris soft-delete diabaikan oleh index ini.
--
-- Efek samping yang diinginkan:
--   - Membuat item baru dengan uniq_code yang sama dengan item yang
--     sudah soft-delete akan berhasil.
--   - Duplicate uniq_code di antara item aktif tetap diblokir.
--
-- Catatan: migration ini idempotent dan aman dijalankan ulang.

BEGIN;

-- 1. Hapus column-level UNIQUE constraint bawaan 0003.
--    Nama constraint auto-generate oleh PostgreSQL: `items_uniq_code_key`.
ALTER TABLE items
	DROP CONSTRAINT IF EXISTS items_uniq_code_key;

-- 2. Hapus index lookup non-unique lama; digantikan oleh partial unique index.
DROP INDEX IF EXISTS idx_items_uniq_code;

-- 3. Buat partial UNIQUE index untuk baris aktif saja.
CREATE UNIQUE INDEX IF NOT EXISTS uq_items_uniq_code_active
	ON items(uniq_code)
	WHERE deleted_at IS NULL;

COMMIT;
