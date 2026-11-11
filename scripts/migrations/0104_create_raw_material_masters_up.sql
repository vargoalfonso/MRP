-- Canonical raw-material specification shared by BOMs and inventory pools.
CREATE TABLE IF NOT EXISTS raw_material_masters (
    id BIGSERIAL PRIMARY KEY,
    material_code VARCHAR(64) NOT NULL,
    material_name VARCHAR(255) NOT NULL,
    material_grade VARCHAR(100),
    form VARCHAR(32),
    type_material VARCHAR(50) NOT NULL DEFAULT 'raw',
    width_mm NUMERIC(18,4),
    diameter_mm NUMERIC(18,4),
    thickness_mm NUMERIC(18,4),
    length_mm NUMERIC(18,4),
    weight_kg NUMERIC(18,6),
    uom VARCHAR(32),
    spec_key VARCHAR(512) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Active',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_raw_material_masters_status CHECK (status IN ('Active','Inactive'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_material_masters_code_active
    ON raw_material_masters (UPPER(material_code)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_material_masters_spec_active
    ON raw_material_masters (spec_key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_raw_material_masters_grade_form
    ON raw_material_masters (material_grade, form) WHERE deleted_at IS NULL;

ALTER TABLE item_material_specs
    ADD COLUMN IF NOT EXISTS raw_material_master_id BIGINT NULL
    REFERENCES raw_material_masters(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_item_material_specs_rm_master
    ON item_material_specs(raw_material_master_id);

ALTER TABLE raw_materials
    ADD COLUMN IF NOT EXISTS raw_material_master_id BIGINT NULL
    REFERENCES raw_material_masters(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_raw_materials_rm_master
    ON raw_materials(raw_material_master_id) WHERE deleted_at IS NULL;

-- Safe initial backfill: one master for every distinct, non-empty normalized BOM specification.
WITH specs AS (
    SELECT DISTINCT
        UPPER(TRIM(COALESCE(material_grade, grade, ''))) AS material_grade,
        UPPER(TRIM(COALESCE(form, ''))) AS form,
        LOWER(TRIM(COALESCE(type_material, 'raw'))) AS type_material,
        width_mm, diameter_mm, thickness_mm, length_mm, weight_kg,
        CONCAT_WS('|',
            UPPER(TRIM(COALESCE(material_grade, grade, ''))),
            UPPER(TRIM(COALESCE(form, ''))),
            LOWER(TRIM(COALESCE(type_material, 'raw'))),
            COALESCE(ROUND(width_mm::numeric,4)::text,''),
            COALESCE(ROUND(diameter_mm::numeric,4)::text,''),
            COALESCE(ROUND(thickness_mm::numeric,4)::text,''),
            COALESCE(ROUND(length_mm::numeric,4)::text,''),
            COALESCE(ROUND(weight_kg::numeric,6)::text,'')
        ) AS spec_key
    FROM item_material_specs
    WHERE LOWER(COALESCE(type_material, '')) = 'raw'
      AND COALESCE(NULLIF(TRIM(material_grade),''), NULLIF(TRIM(grade),'')) IS NOT NULL
), numbered AS (
    SELECT *, ROW_NUMBER() OVER (ORDER BY spec_key) AS rn FROM specs
)
INSERT INTO raw_material_masters (
    material_code, material_name, material_grade, form, type_material,
    width_mm, diameter_mm, thickness_mm, length_mm, weight_kg, spec_key
)
SELECT
    'RM-' || LPAD(rn::text, 5, '0'),
    CONCAT_WS(' ', NULLIF(material_grade,''), NULLIF(form,'')),
    NULLIF(material_grade,''), NULLIF(form,''), type_material,
    width_mm, diameter_mm, thickness_mm, length_mm, weight_kg, spec_key
FROM numbered
ON CONFLICT DO NOTHING;

UPDATE item_material_specs ims
SET raw_material_master_id = rmm.id
FROM raw_material_masters rmm
WHERE ims.raw_material_master_id IS NULL
  AND rmm.deleted_at IS NULL
  AND rmm.spec_key = CONCAT_WS('|',
      UPPER(TRIM(COALESCE(ims.material_grade, ims.grade, ''))),
      UPPER(TRIM(COALESCE(ims.form, ''))),
      LOWER(TRIM(COALESCE(ims.type_material, 'raw'))),
      COALESCE(ROUND(ims.width_mm::numeric,4)::text,''),
      COALESCE(ROUND(ims.diameter_mm::numeric,4)::text,''),
      COALESCE(ROUND(ims.thickness_mm::numeric,4)::text,''),
      COALESCE(ROUND(ims.length_mm::numeric,4)::text,''),
      COALESCE(ROUND(ims.weight_kg::numeric,6)::text,'')
  );

-- Link existing inventory when its item has an unambiguous latest specification.
UPDATE raw_materials rm
SET raw_material_master_id = x.raw_material_master_id
FROM (
    SELECT DISTINCT ON (i.id) i.id AS item_id, ims.raw_material_master_id
    FROM items i
    JOIN item_revisions ir ON ir.item_id = i.id
    JOIN item_material_specs ims ON ims.item_revision_id = ir.id
    WHERE ims.raw_material_master_id IS NOT NULL
    ORDER BY i.id, ir.id DESC
) x
WHERE rm.raw_material_master_id IS NULL
  AND rm.item_id = x.item_id;
