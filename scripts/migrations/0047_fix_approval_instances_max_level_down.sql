-- =============================================================================
-- Migration: 0047_fix_approval_instances_max_level_down.sql
-- Purpose  : Revert max_level constraint back to BETWEEN 2 AND 4.
-- =============================================================================

ALTER TABLE approval_instances
    DROP CONSTRAINT IF EXISTS approval_instances_max_level_check;

ALTER TABLE approval_instances
    ADD CONSTRAINT approval_instances_max_level_check CHECK (max_level BETWEEN 2 AND 4);
