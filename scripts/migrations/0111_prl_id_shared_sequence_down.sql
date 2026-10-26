-- +migrate Down
-- Kembalikan trigger ke versi awal (0073 dan 0074). Nilai sequence tidak diubah.
CREATE OR REPLACE FUNCTION set_prl_id() RETURNS trigger AS $$
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' OR NEW.prl_id = 'PENDING' THEN
    NEW.prl_id := FORMAT('PRL-%s-%s', TO_CHAR(NOW(),'YYYY'), LPAD(NEXTVAL('prl_global_seq')::text, 4, '0'));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

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
