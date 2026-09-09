-- +migrate Down
DROP TRIGGER IF EXISTS trg_set_prl_id ON public.prls;
DROP FUNCTION IF EXISTS set_prl_id();
DROP SEQUENCE IF EXISTS prl_global_seq;
