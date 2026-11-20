CREATE TABLE IF NOT EXISTS public.uniq_bill_of_materials
(
    id            BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid          UUID         NOT NULL,
    uniq_code     VARCHAR(100) NOT NULL,
    product_model VARCHAR(255) NOT NULL,
    part_name     VARCHAR(255) NOT NULL,
    part_number   VARCHAR(150) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ  NULL,

    CONSTRAINT uniq_bill_of_materials_pkey PRIMARY KEY (id),
    CONSTRAINT uniq_bill_of_materials_uuid_key UNIQUE (uuid),
    CONSTRAINT uniq_bill_of_materials_uniq_code_key UNIQUE (uniq_code)
);

CREATE INDEX IF NOT EXISTS idx_uniq_bill_of_materials_deleted_at ON public.uniq_bill_of_materials (deleted_at);
CREATE INDEX IF NOT EXISTS idx_uniq_bill_of_materials_uniq_code ON public.uniq_bill_of_materials (uniq_code);

INSERT INTO public.uniq_bill_of_materials (uuid, uniq_code, product_model, part_name, part_number)
VALUES
    (uuid_generate_v4(), 'LV7-001', 'Camry 2024', 'Engine Mount Bracket', 'EM-001-LV7'),
    (uuid_generate_v4(), 'LV7-002', 'Camry 2024', 'Front Bumper Reinforcement', 'FB-002-LV7'),
    (uuid_generate_v4(), 'LV7-003', 'Innova 2025', 'Door Trim Panel', 'DT-003-LV7')
ON CONFLICT (uniq_code) DO NOTHING;
