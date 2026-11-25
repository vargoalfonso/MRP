-- =============================================================================
-- Migration: 0047_fix_approval_instances_max_level_up.sql
-- Purpose  : Allow max_level = 1 on approval_instances.
--            Original constraint was BETWEEN 2 AND 4, changed to BETWEEN 1 AND 4.
-- =============================================================================

ALTER TABLE approval_instances
    DROP CONSTRAINT IF EXISTS approval_instances_max_level_check;

ALTER TABLE approval_instances
    ADD CONSTRAINT approval_instances_max_level_check CHECK (max_level BETWEEN 1 AND 4);
