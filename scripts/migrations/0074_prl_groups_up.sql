-- Up: create prl_groups table and sequence/trigger for prl_id
CREATE SEQUENCE IF NOT EXISTS prl_groups_global_seq;

CREATE TABLE IF NOT EXISTS prl_groups (
    id bigserial PRIMARY KEY,
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    prl_id varchar(50) NOT NULL UNIQUE,
    remarks text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL
);

CREATE INDEX IF NOT EXISTS idx_prl_groups_prl_id ON prl_groups(prl_id);

-- function to set prl_id on insert
CREATE OR REPLACE FUNCTION set_prl_groups_prl_id() RETURNS trigger AS $$
DECLARE
  seqval bigint;
  yr int := EXTRACT(YEAR FROM now())::int;
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' THEN
    seqval := nextval('prl_groups_global_seq');
    NEW.prl_id := format('PRL-%s-%s', yr, lpad(seqval::text, 4, '0'));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_prl_groups_prl_id ON prl_groups;
CREATE TRIGGER trg_set_prl_groups_prl_id
BEFORE INSERT ON prl_groups
FOR EACH ROW EXECUTE FUNCTION set_prl_groups_prl_id();
