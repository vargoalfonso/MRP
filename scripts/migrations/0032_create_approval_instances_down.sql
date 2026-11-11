-- =============================================================================
-- Migration: 0032_create_approval_instances_down.sql
-- Rollback : drop approval_instances + indexes
-- =============================================================================

DROP INDEX IF EXISTS idx_approval_instances_progress;

DROP INDEX IF EXISTS idx_approval_instances_workflow_status;

DROP INDEX IF EXISTS idx_approval_instances_action_ref;

DROP TABLE IF EXISTS approval_instances;