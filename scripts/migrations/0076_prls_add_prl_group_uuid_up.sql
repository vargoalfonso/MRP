-- Up: add prl_group_uuid column to prls
ALTER TABLE prls
  ADD COLUMN IF NOT EXISTS prl_group_uuid uuid NULL;

CREATE INDEX IF NOT EXISTS idx_prls_prl_group_uuid ON prls(prl_group_uuid);

-- Down
-- ALTER TABLE prls DROP COLUMN IF EXISTS prl_group_uuid;
-- DROP INDEX IF EXISTS idx_prls_prl_group_uuid;
