package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/apperror"
)

// ListBomUsage mengembalikan FG/parent tempat tiap uniq dipakai pada BOM current.
// uniqCodes harus sudah UPPERCASE.
func (r *repository) ListBomUsage(ctx context.Context, uniqCodes []string) ([]models.BomUsageRow, error) {
	rows := make([]models.BomUsageRow, 0)
	if len(uniqCodes) == 0 {
		return rows, nil
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT * FROM (
			-- uniq dipakai sebagai child di BOM FG
			SELECT
				c.uniq_code            AS child_uniq_code,
				c.part_name            AS child_part_name,
				p.uniq_code            AS parent_uniq_code,
				p.part_name            AS parent_part_name,
				f.uniq_code            AS fg_uniq_code,
				f.part_number          AS fg_part_number,
				f.part_name            AS fg_part_name,
				f.model                AS fg_model,
				bl.level::int          AS level,
				bl.qty_per_uniq        AS qty_per_uniq,
				bi.version             AS bom_version,
				bi.status              AS bom_status
			FROM bom_lines bl
			JOIN bom_item bi ON bi.id = bl.bom_item_id AND bi.is_current = TRUE
			JOIN items c ON c.id = bl.child_item_id AND c.deleted_at IS NULL
			JOIN items f ON f.id = bi.item_id AND f.deleted_at IS NULL
			LEFT JOIN items p ON p.id = bl.parent_item_id AND p.deleted_at IS NULL
			WHERE UPPER(c.uniq_code) IN ?

			UNION ALL

			-- uniq itu sendiri adalah root BOM (FG)
			SELECT
				c.uniq_code, c.part_name,
				NULL::text, NULL::text,
				c.uniq_code, c.part_number, c.part_name, c.model,
				0, bi.qty_per_uniq, bi.version, bi.status
			FROM bom_item bi
			JOIN items c ON c.id = bi.item_id AND c.deleted_at IS NULL
			WHERE bi.is_current = TRUE AND UPPER(c.uniq_code) IN ?
		) u
		ORDER BY u.child_uniq_code, u.fg_uniq_code, u.level
		LIMIT 2000
	`, uniqCodes, uniqCodes).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("list bom usage failed", err)
	}
	return rows, nil
}

// ListBomMaterialGrades mengembalikan daftar material code yang ada di BOM
// (aturan "revisi terbaik" sama dengan ListUniqsByMaterialCode) beserta status
// ketersediaan Raw Material Master-nya.
func (r *repository) ListBomMaterialGrades(ctx context.Context, typeMaterials []string, search string) ([]models.BomMaterialGrade, error) {
	rows := make([]models.BomMaterialGrade, 0)
	if len(typeMaterials) == 0 {
		return rows, nil
	}
	term := strings.TrimSpace(search)
	like := "%" + term + "%"

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			g.material_grade AS material_grade,
			COUNT(*)         AS uniq_count,
			MIN(g.form)      AS form,
			EXISTS (
				SELECT 1 FROM raw_material_masters m
				WHERE m.deleted_at IS NULL
				  AND m.status = 'Active'
				  AND UPPER(TRIM(COALESCE(m.material_grade, ''))) = UPPER(g.material_grade)
			) AS has_master
		FROM (
			SELECT c.id, spec.material_grade AS material_grade, spec.form AS form
			FROM items c
			JOIN LATERAL (
				SELECT
					NULLIF(TRIM(COALESCE(ims.material_grade, '')), '') AS material_grade,
					ims.form, ims.type_material
				FROM item_revisions ir
				JOIN item_material_specs ims ON ims.item_revision_id = ir.id
				WHERE ir.item_id = c.id
				ORDER BY
					(NULLIF(TRIM(COALESCE(ims.material_grade, '')), '') IS NOT NULL) DESC,
					(ir.status = 'Released') DESC,
					ir.created_at DESC,
					ir.id DESC
				LIMIT 1
			) spec ON TRUE
			WHERE c.deleted_at IS NULL
			  AND spec.material_grade IS NOT NULL
			  AND LOWER(COALESCE(spec.type_material, '')) IN ?
			  AND (?::text = '' OR spec.material_grade ILIKE ?)
		) g
		GROUP BY g.material_grade
		ORDER BY g.material_grade
		LIMIT 1000
	`, typeMaterials, term, like).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("list bom material grades failed", err)
	}
	return rows, nil
}
