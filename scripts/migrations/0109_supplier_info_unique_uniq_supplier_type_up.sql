-- 0109: supplier_info dipetakan per kombinasi (uniq, supplier_name, type).
-- Satu UNIQ boleh dimiliki beberapa supplier/type (lihat 0103), jadi duplikat
-- tidak lagi dicek hanya berdasarkan uniq. Index ini menjaga konsistensi di level DB.
CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_info_uniq_supplier_type
    ON supplier_info (uniq, supplier_name, type)
    WHERE deleted_at IS NULL;
