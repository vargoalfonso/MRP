-- +migrate Up
-- Fix: PRL tunggal baru "menempel" ke grup PRL lama (mis. 1 UNIQ jadi +65 UNIQ).
--
-- Penyebab: ID PRL dibuat dari DUA sequence berbeda dengan format yang sama
--   * PRL tunggal (trigger set_prl_id)        -> prl_global_seq
--   * PRL grup / multi-UNIQ (prl_groups)      -> prl_groups_global_seq
-- Keduanya menghasilkan "PRL-<tahun>-<nomor>", sehingga nomor yang sama bisa
-- terbit dua kali (mis. PRL-2026-0007). List PRL mengelompokkan baris
-- berdasarkan prl_id, jadi PRL tunggal ikut tergabung dengan grup lama.
--
-- Perbaikan: semua ID PRL memakai satu sequence (prl_groups_global_seq), nilainya
-- dinaikkan di atas semua nomor yang sudah ada, dan trigger melewati ID yang sudah
-- dipakai. Migrasi ini TIDAK mengubah data lama (lihat scripts/maintenance).

-- 1) Naikkan sequence bersama di atas nomor tertinggi yang sudah dipakai.
DO $$
DECLARE
  max_used bigint;
BEGIN
  SELECT COALESCE(MAX(NULLIF(substring(prl_id from '[0-9]+$'), '')::bigint), 0)
    INTO max_used
  FROM (
    SELECT prl_id FROM prls
    UNION ALL
    SELECT prl_id FROM prl_groups
  ) ids;

  PERFORM setval(
    'prl_groups_global_seq',
    GREATEST(max_used, (SELECT last_value FROM prl_groups_global_seq)),
    true
  );
END$$;

-- 2) PRL tunggal: pakai sequence bersama dan lewati ID yang sudah ada.
CREATE OR REPLACE FUNCTION set_prl_id() RETURNS trigger AS $$
DECLARE
  candidate text;
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' OR NEW.prl_id = 'PENDING' THEN
    LOOP
      candidate := FORMAT('PRL-%s-%s', TO_CHAR(NOW(),'YYYY'), LPAD(NEXTVAL('prl_groups_global_seq')::text, 4, '0'));
      EXIT WHEN NOT EXISTS (SELECT 1 FROM prls WHERE prl_id = candidate)
            AND NOT EXISTS (SELECT 1 FROM prl_groups WHERE prl_id = candidate);
    END LOOP;
    NEW.prl_id := candidate;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 3) PRL grup: sama, lewati ID yang sudah dipakai PRL tunggal lama.
CREATE OR REPLACE FUNCTION set_prl_groups_prl_id() RETURNS trigger AS $$
DECLARE
  candidate text;
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' THEN
    LOOP
      candidate := format('PRL-%s-%s', EXTRACT(YEAR FROM now())::int, lpad(nextval('prl_groups_global_seq')::text, 4, '0'));
      EXIT WHEN NOT EXISTS (SELECT 1 FROM prls WHERE prl_id = candidate)
            AND NOT EXISTS (SELECT 1 FROM prl_groups WHERE prl_id = candidate);
    END LOOP;
    NEW.prl_id := candidate;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
