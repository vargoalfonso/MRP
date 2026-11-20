CREATE TABLE IF NOT EXISTS public.supplier_performance_snapshots (
    id                        BIGINT GENERATED ALWAYS AS IDENTITY,
    snapshot_uuid             UUID            NOT NULL,
    supplier_uuid             UUID            NOT NULL,
    supplier_code             VARCHAR(64)     NOT NULL,
    supplier_name             VARCHAR(255)    NOT NULL,
    evaluation_period_type    VARCHAR(16)     NOT NULL,
    evaluation_period_value   VARCHAR(32)     NOT NULL,
    evaluation_date           DATE            NOT NULL,
    total_deliveries          INTEGER         NOT NULL DEFAULT 0,
    on_time_deliveries        INTEGER         NOT NULL DEFAULT 0,
    late_deliveries           INTEGER         NOT NULL DEFAULT 0,
    otd_percentage            NUMERIC(7,2)    NOT NULL DEFAULT 0,
    average_delay_days        NUMERIC(10,2)   NOT NULL DEFAULT 0,
    quality_inspection_count  INTEGER         NOT NULL DEFAULT 0,
    accepted_quantity         NUMERIC(18,4)   NOT NULL DEFAULT 0,
    rejected_quantity         NUMERIC(18,4)   NOT NULL DEFAULT 0,
    inspected_quantity        NUMERIC(18,4)   NOT NULL DEFAULT 0,
    quality_percentage        NUMERIC(7,2)    NOT NULL DEFAULT 0,
    total_purchase_value      NUMERIC(18,2)   NOT NULL DEFAULT 0,
    computed_score            NUMERIC(7,2)    NOT NULL DEFAULT 0,
    system_grade              VARCHAR(1)      NOT NULL,
    final_grade               VARCHAR(1)      NOT NULL,
    status_label              VARCHAR(32)     NOT NULL,
    poor_delivery_performance BOOLEAN         NOT NULL DEFAULT FALSE,
    qc_alert                  BOOLEAN         NOT NULL DEFAULT FALSE,
    supplier_review_required  BOOLEAN         NOT NULL DEFAULT FALSE,
    is_grade_overridden       BOOLEAN         NOT NULL DEFAULT FALSE,
    override_grade            VARCHAR(1)      NULL,
    override_remarks          TEXT            NULL,
    override_by               VARCHAR(255)    NULL,
    override_at               TIMESTAMPTZ     NULL,
    computed_at               TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    data_through_at           TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    logic_version             VARCHAR(32)     NOT NULL,
    formula_otd               TEXT            NOT NULL,
    formula_quality           TEXT            NOT NULL,
    formula_grade             TEXT            NOT NULL,
    formula_notes             TEXT            NULL,
    created_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at                TIMESTAMPTZ     NULL,

    CONSTRAINT supplier_performance_snapshots_pkey PRIMARY KEY (id),
    CONSTRAINT supplier_performance_snapshots_uuid_key UNIQUE (snapshot_uuid),
    CONSTRAINT supplier_performance_snapshots_period_key UNIQUE (supplier_uuid, evaluation_period_type, evaluation_period_value, logic_version),
    CONSTRAINT supplier_performance_period_type_check CHECK (evaluation_period_type IN ('monthly', 'quarterly', 'yearly')),
    CONSTRAINT supplier_performance_system_grade_check CHECK (system_grade IN ('A', 'B', 'C')),
    CONSTRAINT supplier_performance_final_grade_check CHECK (final_grade IN ('A', 'B', 'C')),
    CONSTRAINT supplier_performance_override_grade_check CHECK (override_grade IS NULL OR override_grade IN ('A', 'B', 'C'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_period
    ON public.supplier_performance_snapshots (evaluation_period_type, evaluation_period_value);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_supplier
    ON public.supplier_performance_snapshots (supplier_uuid);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_status
    ON public.supplier_performance_snapshots (status_label);

CREATE INDEX IF NOT EXISTS idx_supplier_performance_deleted_at
    ON public.supplier_performance_snapshots (deleted_at);
