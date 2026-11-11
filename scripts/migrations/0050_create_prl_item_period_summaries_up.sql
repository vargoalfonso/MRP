CREATE TABLE IF NOT EXISTS prl_item_period_summaries (
    id              BIGSERIAL PRIMARY KEY,
    forecast_period VARCHAR(100) NOT NULL,
    item_uniq_code  VARCHAR(100) NOT NULL,
    prl_total_qty   NUMERIC(18, 4) NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_prl_period_item UNIQUE (forecast_period, item_uniq_code)
);

CREATE INDEX IF NOT EXISTS idx_prl_period_summaries_item ON prl_item_period_summaries (item_uniq_code);
