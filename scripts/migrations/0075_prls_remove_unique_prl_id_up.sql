-- Up: remove unique constraint on prls.prl_id to allow multiple rows to share same PRL ID
DO $$
BEGIN
    -- If prl_id has a unique constraint, drop it by dropping the constraint on table prls
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'prls_prl_id_key'
    ) THEN
        EXECUTE 'ALTER TABLE prls DROP CONSTRAINT IF EXISTS prls_prl_id_key';
    END IF;
    -- create non-unique index
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE tablename = 'prls' AND indexname = 'idx_prls_prl_id'
    ) THEN
        CREATE INDEX idx_prls_prl_id ON prls(prl_id);
    END IF;
END$$;
