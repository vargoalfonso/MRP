-- +migrate Up
-- Jadwal training Demand Forecasting (System Settings -> Automation).
-- ERP hanya menyimpan "kapan DN / PRL di-train"; script forecasting membaca jadwal ini
-- dan melaporkan hasil run kembali (last_run_*). Maksimal 1 baris per domain.
CREATE TABLE IF NOT EXISTS forecasting_schedules (
    domain            VARCHAR(16) PRIMARY KEY CHECK (domain IN ('dn', 'prl')),
    is_active         BOOLEAN      NOT NULL DEFAULT FALSE,
    frequency         VARCHAR(16)  NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    run_time          VARCHAR(5)   NOT NULL DEFAULT '02:00',
    day_of_week       VARCHAR(16),
    day_of_month      INT CHECK (day_of_month IS NULL OR (day_of_month BETWEEN 1 AND 31)),
    timezone          VARCHAR(64)  NOT NULL DEFAULT 'Asia/Jakarta',
    selected_model    VARCHAR(64)  NOT NULL DEFAULT 'Chronos2',
    last_run_at       TIMESTAMPTZ,
    last_run_status   VARCHAR(16),
    last_run_message  TEXT,
    last_training_run_id VARCHAR(128),
    next_run_at       TIMESTAMPTZ,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by        VARCHAR(255)
);
