-- inventory_demand_periode_summaries
-- One snapshot row per item per day for the global active demand period.
-- Rebuilt by POST /api/v1/admin/jobs/rebuild-prl-period-summaries.
CREATE TABLE IF NOT EXISTS inventory_demand_periode_summaries (
    id                              BIGSERIAL    PRIMARY KEY,
    uniq_code                       VARCHAR(100) NOT NULL,
    active_periode                  VARCHAR(100) NOT NULL,
    snapshot_date                   DATE         NOT NULL,
    working_days_periode_used       VARCHAR(100) NOT NULL DEFAULT '',
    working_days_used               INT          NOT NULL DEFAULT 0,
    safety_stock_calc_type_active   VARCHAR(50),
    safety_stock_constanta_active   NUMERIC(18, 4),
    stockdays_calc_type_active      VARCHAR(50),
    stockdays_constanta_active      NUMERIC(18, 4),
    prl_sum                         NUMERIC(18, 4) NOT NULL DEFAULT 0,
    po_customer_sum                 NUMERIC(18, 4) NOT NULL DEFAULT 0,
    total_demand_sum                NUMERIC(18, 4) NOT NULL DEFAULT 0,
    rebuilt_at                      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_inv_demand_periode_summary
        UNIQUE (uniq_code, active_periode, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_inv_demand_summary_uniq_code
    ON inventory_demand_periode_summaries (uniq_code);

CREATE INDEX IF NOT EXISTS idx_inv_demand_summary_active_periode
    ON inventory_demand_periode_summaries (active_periode);

CREATE INDEX IF NOT EXISTS idx_inv_demand_summary_snapshot_date
    ON inventory_demand_periode_summaries (snapshot_date);
