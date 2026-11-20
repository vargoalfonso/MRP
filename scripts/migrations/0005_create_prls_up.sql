CREATE TABLE IF NOT EXISTS public.prls
(
    id              BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid            UUID         NOT NULL,
    prl_id          VARCHAR(32)  NOT NULL,
    customer_uuid   UUID         NOT NULL,
    customer_code   VARCHAR(32)  NOT NULL,
    customer_name   VARCHAR(255) NOT NULL,
    uniq_bom_uuid   UUID         NOT NULL,
    uniq_code       VARCHAR(100) NOT NULL,
    product_model   VARCHAR(255) NOT NULL,
    part_name       VARCHAR(255) NOT NULL,
    part_number     VARCHAR(150) NOT NULL,
    forecast_period VARCHAR(7)   NOT NULL,
    quantity        BIGINT       NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    approved_at     TIMESTAMPTZ  NULL,
    rejected_at     TIMESTAMPTZ  NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  NULL,

    CONSTRAINT prls_pkey PRIMARY KEY (id),
    CONSTRAINT prls_uuid_key UNIQUE (uuid),
    CONSTRAINT prls_prl_id_key UNIQUE (prl_id),
    CONSTRAINT prls_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT prls_forecast_period_check CHECK (forecast_period ~ '^[0-9]{4}-Q[1-4]$')
);

CREATE INDEX IF NOT EXISTS idx_prls_deleted_at ON public.prls (deleted_at);
CREATE INDEX IF NOT EXISTS idx_prls_prl_id ON public.prls (prl_id);
CREATE INDEX IF NOT EXISTS idx_prls_customer_uuid ON public.prls (customer_uuid);
CREATE INDEX IF NOT EXISTS idx_prls_forecast_period ON public.prls (forecast_period);
CREATE INDEX IF NOT EXISTS idx_prls_status ON public.prls (status);
CREATE INDEX IF NOT EXISTS idx_prls_uniq_code ON public.prls (uniq_code);
