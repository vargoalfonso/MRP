-- Migration 0010: Machine Pattern module
-- Creates two tables:
--   machine_pattern_params  — singleton row for global C parameters
--   machine_patterns        — one record per UNIQ-machine pair

-- ---------------------------------------------------------------------------
-- 1. Global parameters (singleton)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS machine_pattern_params (
    id                    BIGSERIAL PRIMARY KEY,
    fast_moving_threshold NUMERIC(18,4) NOT NULL DEFAULT 1000,  -- C: Fast Moving boundary
    slow_moving_threshold NUMERIC(18,4) NOT NULL DEFAULT 1000,  -- C: Slow Moving boundary
    pattern_min_minutes   NUMERIC(18,4) NOT NULL DEFAULT 48,    -- C: pattern calc threshold (minutes)
    default_working_days  INTEGER       NOT NULL DEFAULT 25,
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Seed the singleton row so GetParams always finds a record
INSERT INTO machine_pattern_params (id, fast_moving_threshold, slow_moving_threshold, pattern_min_minutes, default_working_days)
VALUES (1, 1000, 1000, 48, 25)
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2. Machine patterns
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS machine_patterns (
    id             BIGSERIAL PRIMARY KEY,
    uniq_code      VARCHAR(64)    NOT NULL,
    machine_id     BIGINT         NOT NULL REFERENCES master_machines(id),
    cycle_time_sec NUMERIC(18,4)  NOT NULL,   -- seconds, auto-filled from BOM routing
    prl_reference  NUMERIC(18,4)  NOT NULL,   -- total PRL qty for this UNIQ
    pattern_value  NUMERIC(18,4)  NOT NULL DEFAULT 1,
    working_days   INTEGER        NOT NULL DEFAULT 25,
    moving_type    VARCHAR(20)    NOT NULL DEFAULT 'Normal',  -- Fast Moving | Slow Moving | Normal
    min_output     NUMERIC(18,4)  NOT NULL DEFAULT 0,         -- PRL/wd * cycle_time * pattern
    status         VARCHAR(20)    NOT NULL DEFAULT 'Active',
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_machine_patterns_uniq_machine UNIQUE (uniq_code, machine_id)
);

CREATE INDEX IF NOT EXISTS idx_machine_patterns_uniq_code  ON machine_patterns (uniq_code);
CREATE INDEX IF NOT EXISTS idx_machine_patterns_machine_id ON machine_patterns (machine_id);
CREATE INDEX IF NOT EXISTS idx_machine_patterns_moving_type ON machine_patterns (moving_type);

-- Trigger: auto-update updated_at
CREATE OR REPLACE FUNCTION update_machine_patterns_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_machine_patterns_updated_at ON machine_patterns;
CREATE TRIGGER trg_machine_patterns_updated_at
    BEFORE UPDATE ON machine_patterns
    FOR EACH ROW EXECUTE FUNCTION update_machine_patterns_updated_at();
