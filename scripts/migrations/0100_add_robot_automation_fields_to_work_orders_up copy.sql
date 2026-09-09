-- Robot Automation can create standard work orders that wait in Robot Task.
ALTER TABLE IF EXISTS work_orders
    ADD COLUMN IF NOT EXISTS source_system VARCHAR(32) NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS automation_job_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS robot_name VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_work_orders_source_system
    ON work_orders (source_system);

CREATE UNIQUE INDEX IF NOT EXISTS uq_work_orders_automation_job_id
    ON work_orders (automation_job_id)
    WHERE automation_job_id IS NOT NULL;
