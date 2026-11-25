-- Migration 0110: Work order automation logs
-- Menyimpan baris yang gagal masuk / gagal terkirim dari integrasi Raigine
-- (robot automation) untuk Work Order. Padanan customer_order_automation_logs (0090).
-- wo_number sengaja tanpa FK: kegagalan bisa terjadi sebelum WO sempat terbentuk.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS work_order_automation_logs (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid                 UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    wo_number            VARCHAR(64),
    automation_job_id    VARCHAR(128),
    robot_name           VARCHAR(255),
    row_no               INTEGER NOT NULL DEFAULT 0,
    item_uniq_code       VARCHAR(100),
    part_name            VARCHAR(255),
    description          TEXT,
    qty                  NUMERIC(15,4) NOT NULL DEFAULT 0,
    failure_reason       TEXT,
    special_instructions TEXT,
    source               VARCHAR(64) NOT NULL DEFAULT 'automation',
    status               VARCHAR(32) NOT NULL DEFAULT 'failed',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wo_automation_logs_wo_number
    ON work_order_automation_logs (wo_number);

CREATE INDEX IF NOT EXISTS idx_wo_automation_logs_job_id
    ON work_order_automation_logs (automation_job_id);

CREATE INDEX IF NOT EXISTS idx_wo_automation_logs_item_uniq_code
    ON work_order_automation_logs (item_uniq_code);

CREATE INDEX IF NOT EXISTS idx_wo_automation_logs_created_at
    ON work_order_automation_logs (created_at DESC);
