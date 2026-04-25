-- Ensure stock_opname and bom have active default approval workflows.

UPDATE approval_workflows
SET level_1_role = 'admin', level_2_role = 'admin', level_3_role = NULL, level_4_role = NULL, status = 'active', updated_at = NOW()
WHERE action_name IN ('stock_opname', 'bom');

INSERT INTO approval_workflows (action_name, level_1_role, level_2_role, level_3_role, level_4_role, status, created_by, created_at, updated_at)
SELECT 'stock_opname', 'admin', 'admin', NULL, NULL, 'active', 'system', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM approval_workflows WHERE action_name = 'stock_opname');

INSERT INTO approval_workflows (action_name, level_1_role, level_2_role, level_3_role, level_4_role, status, created_by, created_at, updated_at)
SELECT 'bom', 'admin', 'admin', NULL, NULL, 'active', 'system', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM approval_workflows WHERE action_name = 'bom');
