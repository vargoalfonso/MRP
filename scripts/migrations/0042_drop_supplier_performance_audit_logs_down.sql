CREATE TABLE IF NOT EXISTS public.supplier_performance_audit_logs (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY,
    snapshot_uuid           UUID            NOT NULL,
    supplier_uuid           UUID            NOT NULL,
    evaluation_period_type  VARCHAR(16)     NOT NULL,
    evaluation_period_value VARCHAR(32)     NOT NULL,
    action                  VARCHAR(64)     NOT NULL,
    old_grade               VARCHAR(1)      NULL,
    new_grade               VARCHAR(1)      NULL,
    remarks                 TEXT            NULL,
    logic_version           VARCHAR(32)     NULL,
    actor                   VARCHAR(255)    NULL,
    metadata                JSONB           NOT NULL DEFAULT '{}'::jsonb,
    occurred_at             TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_performance_audit_logs_pkey PRIMARY KEY (id),
    CONSTRAINT supplier_performance_audit_period_type_check CHECK (evaluation_period_type IN ('daily')),
    CONSTRAINT supplier_performance_audit_old_grade_check CHECK (old_grade IS NULL OR old_grade IN ('A', 'B', 'C')),
    CONSTRAINT supplier_performance_audit_new_grade_check CHECK (new_grade IS NULL OR new_grade IN ('A', 'B', 'C'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_audit_supplier_period
    ON public.supplier_performance_audit_logs (supplier_uuid, evaluation_period_type, evaluation_period_value, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_audit_snapshot_uuid
    ON public.supplier_performance_audit_logs (snapshot_uuid);
