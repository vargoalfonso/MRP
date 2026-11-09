package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

// ListUniqsByMaterialCode mengembalikan semua item BOM yang material code-nya
// (item_material_specs.material_grade) sama dengan materialCode. Spec diambil
// dari revisi terbaik tiap item (yang punya material_grade, Released dulu,
// lalu terbaru) -- aturan yang sama dengan ListMaterialCodesByUniqCodes di
// po_budget, supaya hasilnya konsisten dengan yang tampil di UI.
func (r *repository) ListUniqsByMaterialCode(ctx context.Context, materialCode string, typeMaterials []string) ([]models.MaterialCodeUniq, error) {
	code := strings.TrimSpace(materialCode)
	if code == "" || len(typeMaterials) == 0 {
		return []models.MaterialCodeUniq{}, nil
	}

	rows := make([]models.MaterialCodeUniq, 0)
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			c.uniq_code                AS uniq_code,
			c.part_number              AS part_number,
			c.part_name                AS part_name,
			c.model                    AS model,
			COALESCE(c.uom, '')        AS uom,
			spec.material_grade        AS material_code,
			spec.grade                 AS grade,
			spec.form                  AS form,
			spec.width_mm              AS width_mm,
			spec.diameter_mm           AS diameter_mm,
			spec.thickness_mm          AS thickness_mm,
			spec.length_mm             AS length_mm,
			spec.weight_kg             AS weight_kg
		FROM items c
		JOIN LATERAL (
			SELECT
				NULLIF(TRIM(COALESCE(ims.material_grade, '')), '') AS material_grade,
				ims.grade, ims.form, ims.width_mm, ims.diameter_mm,
				ims.thickness_mm, ims.length_mm, ims.weight_kg, ims.type_material
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
		  AND UPPER(spec.material_grade) = UPPER(?)
		  AND LOWER(COALESCE(spec.type_material, '')) IN ?
		ORDER BY c.uniq_code
	`, code, typeMaterials).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("list uniq by material code failed", err)
	}
	return rows, nil
}

// ListRegisteredUniqs mengembalikan set uniq (UPPERCASE) yang SUDAH terdaftar
// untuk supplier + type tsb.
func (r *repository) ListRegisteredUniqs(ctx context.Context, supplierUUID, itemType string, uniqCodes []string) (map[string]bool, error) {
	out := make(map[string]bool, len(uniqCodes))
	if len(uniqCodes) == 0 {
		return out, nil
	}
	upper := make([]string, 0, len(uniqCodes))
	for _, u := range uniqCodes {
		upper = append(upper, strings.ToUpper(strings.TrimSpace(u)))
	}

	var found []string
	err := r.db.WithContext(ctx).Model(&models.SupplierItem{}).
		Where("supplier_uuid = ? AND type = ? AND deleted_at IS NULL AND UPPER(uniq_code) IN ?",
			supplierUUID, strings.ToLower(strings.TrimSpace(itemType)), upper).
		Pluck("UPPER(uniq_code)", &found).Error
	if err != nil {
		return nil, apperror.InternalWrap("check registered uniq failed", err)
	}
	for _, u := range found {
		out[u] = true
	}
	return out, nil
}

// CreateMany menyimpan semua item dalam SATU transaksi: kalau satu gagal,
// tidak ada yang tersimpan.
func (r *repository) CreateMany(ctx context.Context, items []*models.SupplierItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Create(item).Error; err != nil {
				return wrapSupplierItemDBError("bulk create supplier item failed", err)
			}
		}
		return nil
	})
}

// ResolveMasterGradesByName memetakan nama master raw material (ter-normalisasi:
// huruf besar, spasi dirapikan) ke material_grade-nya. Hanya nama yang grade-nya
// tunggal dan terisi yang dikembalikan; nama ambigu diabaikan.
func (r *repository) ResolveMasterGradesByName(ctx context.Context, nameKeys []string) (map[string]string, error) {
	out := make(map[string]string, len(nameKeys))
	if len(nameKeys) == 0 {
		return out, nil
	}

	type row struct {
		NameKey    string `gorm:"column:name_key"`
		Grade      string `gorm:"column:grade"`
		GradeCount int    `gorm:"column:grade_count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT name_key, MIN(grade) AS grade, COUNT(DISTINCT grade) AS grade_count
		FROM (
			SELECT
				UPPER(REGEXP_REPLACE(TRIM(material_name), '\s+', ' ', 'g')) AS name_key,
				NULLIF(TRIM(material_grade), '') AS grade
			FROM raw_material_masters
			WHERE deleted_at IS NULL
		) m
		WHERE name_key IN ? AND grade IS NOT NULL
		GROUP BY name_key
	`, nameKeys).Scan(&rows).Error
	if err != nil {
		return nil, apperror.InternalWrap("resolve master grades failed", err)
	}
	for _, rw := range rows {
		if rw.GradeCount == 1 && rw.Grade != "" {
			out[rw.NameKey] = rw.Grade
		}
	}
	return out, nil
}

// SupplierHasLegacyMaterialRow true bila supplier sudah punya baris LAMA yang
// uniq_code-nya berisi nama master dengan material code (grade) tsb -- baris
// seperti itu sudah berlaku untuk semua uniq ber-material-code itu.
func (r *repository) SupplierHasLegacyMaterialRow(ctx context.Context, supplierUUID, itemType, materialCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM supplier_item si
		WHERE si.supplier_uuid = ?
		  AND si.type = ?
		  AND si.deleted_at IS NULL
		  AND UPPER(REGEXP_REPLACE(TRIM(COALESCE(si.uniq_code, '')), '\s+', ' ', 'g')) IN (
			SELECT UPPER(REGEXP_REPLACE(TRIM(material_name), '\s+', ' ', 'g'))
			FROM raw_material_masters
			WHERE deleted_at IS NULL
			  AND UPPER(TRIM(COALESCE(material_grade, ''))) = UPPER(?)
		  )
	`, supplierUUID, strings.ToLower(strings.TrimSpace(itemType)), strings.TrimSpace(materialCode)).Scan(&count).Error
	if err != nil {
		return false, apperror.InternalWrap("check legacy material row failed", err)
	}
	return count > 0, nil
}
