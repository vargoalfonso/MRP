-- Subcon adalah flag proses (boolean), terpisah dari kategori material (raw | indirect).
ALTER TABLE item_material_specs
    ADD COLUMN IF NOT EXISTS is_subcon BOOLEAN NOT NULL DEFAULT false;

-- Backfill: baris lama yang type_material = 'subcon' jadi is_subcon = true.
-- Kategori aslinya (raw/indirect) sudah tertimpa, jadi coba pulihkan dari raw material master
-- bila ada; kalau tidak ada dibiarkan NULL supaya user memilih ulang.
UPDATE item_material_specs ims
SET is_subcon = true,
    type_material = (
        SELECT rmm.type_material
        FROM raw_material_masters rmm
        WHERE rmm.id = ims.raw_material_master_id
          AND rmm.type_material IN ('raw', 'indirect')
    )
WHERE LOWER(TRIM(COALESCE(ims.type_material, ''))) = 'subcon';
