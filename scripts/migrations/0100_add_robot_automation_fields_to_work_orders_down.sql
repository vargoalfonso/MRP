DROP INDEX IF EXISTS uq_work_orders_automation_job_id;
DROP INDEX IF EXISTS idx_work_orders_source_system;
ALTER TABLE IF EXISTS work_orders
    DROP COLUMN IF EXISTS robot_name,
    DROP COLUMN IF EXISTS automation_job_id,
    DROP COLUMN IF EXISTS source_system;
