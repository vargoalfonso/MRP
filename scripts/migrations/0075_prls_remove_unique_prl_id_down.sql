-- Down: recreate unique constraint on prls.prl_id (if desired)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes WHERE tablename = 'prls' AND indexname = 'prls_prl_id_key'
    ) THEN
        CREATE UNIQUE INDEX prls_prl_id_key ON prls(prl_id);
    END IF;
    DROP INDEX IF EXISTS idx_prls_prl_id;
END$$;
