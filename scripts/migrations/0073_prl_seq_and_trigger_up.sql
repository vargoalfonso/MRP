-- +migrate Up
-- Create a global sequence and trigger to populate PRL ID atomically
CREATE SEQUENCE IF NOT EXISTS prl_global_seq START 1;

-- Function to set prl_id before insert
CREATE OR REPLACE FUNCTION set_prl_id() RETURNS trigger AS $$
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' OR NEW.prl_id = 'PENDING' THEN
    -- format: PRL-<year>-<seq:03d>
    NEW.prl_id := FORMAT('PRL-%s-%s', TO_CHAR(NOW(),'YYYY'), LPAD(NEXTVAL('prl_global_seq')::text, 4, '0'));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop trigger if exists and create
DROP TRIGGER IF EXISTS trg_set_prl_id ON public.prls;
CREATE TRIGGER trg_set_prl_id
BEFORE INSERT ON public.prls
FOR EACH ROW
EXECUTE FUNCTION set_prl_id();

-- +migrate Down
DROP TRIGGER IF EXISTS trg_set_prl_id ON public.prls;
DROP FUNCTION IF EXISTS set_prl_id();
DROP SEQUENCE IF EXISTS prl_global_seq;
