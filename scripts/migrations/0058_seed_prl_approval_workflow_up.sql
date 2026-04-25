-- =============================================================================
-- Migration: 0058_seed_prl_approval_workflow_up.sql
-- Purpose  : Ensure PRL has an active approval workflow so PRL submit and
--            approve/reject can use approval_instances end-to-end.
--
-- Default roles use "admin" for both levels so local/dev can test the full
-- multi-level flow immediately. Adjust later via approval_workflows API if the
-- final business roles differ (for example PPIC leader / manager planning).
-- =============================================================================

UPDATE approval_workflows
SET
    level_1_role = 'admin',
    level_2_role = 'admin',
    level_3_role = NULL,
    level_4_role = NULL,
    status = 'active',
    updated_at = NOW()
WHERE action_name = 'prl';

INSERT INTO approval_workflows (
    action_name,
    level_1_role,
    level_2_role,
    level_3_role,
    level_4_role,
    status,
    created_by,
    created_at,
    updated_at
)
SELECT
    'prl',
    'admin',
    'admin',
    NULL,
    NULL,
    'active',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1
    FROM approval_workflows
    WHERE action_name = 'prl'
);
