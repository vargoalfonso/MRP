
CREATE TABLE IF NOT EXISTS public.delivery_note_logs (
    id bigint NOT NULL,
    dn_id bigint NOT NULL,
    dn_item_id bigint NOT NULL,
    item_uniq_code varchar(100) NOT NULL,
    packing_number varchar(100),
    scan_type varchar(20) NOT NULL,
    qty numeric(15,2) NOT NULL,
    from_location varchar(50),
    to_location varchar(50),
    reference_type varchar(50),
    reference_id bigint,
    created_by varchar(100),
    created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.machine_pattern_params (
    id bigint NOT NULL,
    fast_moving_threshold numeric(18,4) NOT NULL DEFAULT 1000,
    slow_moving_threshold numeric(18,4) NOT NULL DEFAULT 1000,
    pattern_min_minutes numeric(18,4) NOT NULL DEFAULT 48,
    default_working_days integer NOT NULL DEFAULT 25,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.production_scanin_drafts (
    id bigint NOT NULL,
    wo_id bigint NOT NULL,
    wo_item_id bigint NOT NULL,
    current_step integer NOT NULL DEFAULT 1,
    payload jsonb NOT NULL,
    updated_by varchar(255),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.rm_source (
    id uuid NOT NULL,
    name varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS public.rm_types (
    id uuid NOT NULL,
    name varchar(255)
);

CREATE TABLE IF NOT EXISTS public.rm_uniq (
    id uuid NOT NULL,
    code varchar(255)
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'delivery_note_logs_pkey') THEN
        ALTER TABLE public.delivery_note_logs ADD CONSTRAINT delivery_note_logs_pkey PRIMARY KEY (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'machine_pattern_params_pkey') THEN
        ALTER TABLE public.machine_pattern_params ADD CONSTRAINT machine_pattern_params_pkey PRIMARY KEY (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'production_scanin_drafts_pkey') THEN
        ALTER TABLE public.production_scanin_drafts ADD CONSTRAINT production_scanin_drafts_pkey PRIMARY KEY (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'rm_source_pkey') THEN
        ALTER TABLE public.rm_source ADD CONSTRAINT rm_source_pkey PRIMARY KEY (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'rm_types_pkey') THEN
        ALTER TABLE public.rm_types ADD CONSTRAINT rm_types_pkey PRIMARY KEY (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'rm_uniq_pkey') THEN
        ALTER TABLE public.rm_uniq ADD CONSTRAINT rm_uniq_pkey PRIMARY KEY (id);
    END IF;
    IF to_regclass('public.production_issues') IS NOT NULL
       AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'production_issues_uuid_key') THEN
        ALTER TABLE public.production_issues ADD CONSTRAINT production_issues_uuid_key UNIQUE (uuid);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_production_scanin_drafts_key') THEN
        ALTER TABLE public.production_scanin_drafts ADD CONSTRAINT uq_production_scanin_drafts_key UNIQUE (wo_id, wo_item_id, current_step);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_rm_source_name_unique ON public.rm_source (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_rm_types_name_unique ON public.rm_types (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_rm_uniq_code_unique ON public.rm_uniq (code);
CREATE INDEX IF NOT EXISTS idx_production_scanin_drafts_wo ON public.production_scanin_drafts (wo_id, current_step);
