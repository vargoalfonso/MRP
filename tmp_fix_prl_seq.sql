CREATE SEQUENCE IF NOT EXISTS prl_global_seq START 1;
SELECT setval('prl_global_seq', 99, false);

CREATE OR REPLACE FUNCTION set_prl_id() RETURNS trigger AS $$
BEGIN
  IF NEW.prl_id IS NULL OR NEW.prl_id = '' OR NEW.prl_id = 'PENDING' THEN
    NEW.prl_id := FORMAT('PRL-%s-%s', TO_CHAR(NOW(),'YYYY'), LPAD(NEXTVAL('prl_global_seq')::text, 4, '0'));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_prl_id ON public.prls;
CREATE TRIGGER trg_set_prl_id
BEFORE INSERT ON public.prls
FOR EACH ROW
EXECUTE FUNCTION set_prl_id();

SELECT nextval('prl_global_seq') AS test_next;
