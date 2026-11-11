-- Down: drop prl_groups table and associated objects
DROP TRIGGER IF EXISTS trg_set_prl_groups_prl_id ON prl_groups;
DROP FUNCTION IF EXISTS set_prl_groups_prl_id();
DROP TABLE IF EXISTS prl_groups;
DROP SEQUENCE IF EXISTS prl_groups_global_seq;
