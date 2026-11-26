package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/po_budget/models"
)

// ResolveItemTypeFlagsByUniq mengembalikan type_material mentah + is_subcon per
// uniq_code (key lowercase). Aturan pemilihan revisi sama dengan
// ResolveTypeMaterialByUniq (revisi current, fallback ke revisi terbaru), tetapi
// is_subcon TIDAK lagi menimpa type_material, sehingga item raw yang juga
// subcon bisa dimasukkan ke kedua tab.
func (r *repo) ResolveItemTypeFlagsByUniq(ctx context.Context, uniqCodes []string) (map[string]models.ItemTypeFlags, error) {
	out := map[string]models.ItemTypeFlags{}
	if len(uniqCodes) == 0 {
		return out, nil
	}

	type row struct {
		UniqCode     string
		TypeMaterial string
		IsSubcon     bool
	}
	const selectCols = `
		i.uniq_code,
		COALESCE(NULLIF(TRIM(COALESCE(ims.type_material, '')), ''), '') AS type_material,
		COALESCE(ims.is_subcon, false) AS is_subcon`

	var currentRows []row
	err := r.db.WithContext(ctx).
		Table("items AS i").
		Select(selectCols).
		Joins("JOIN bom_item bi ON bi.item_id = i.id AND bi.is_current = true").
		Joins("JOIN item_material_specs ims ON ims.item_revision_id = bi.root_item_revision_id").
		Where("i.deleted_at IS NULL AND i.uniq_code IN ?", uniqCodes).
		Scan(&currentRows).Error
	if err != nil {
		return nil, err
	}
	resolved := make(map[string]bool, len(currentRows))
	for _, rw := range currentRows {
		key := strings.ToLower(strings.TrimSpace(rw.UniqCode))
		resolved[key] = true
		if rw.TypeMaterial == "" && !rw.IsSubcon {
			continue
		}
		out[key] = models.ItemTypeFlags{
			TypeMaterial: strings.ToLower(strings.TrimSpace(rw.TypeMaterial)),
			IsSubcon:     rw.IsSubcon,
		}
	}

	remaining := make([]string, 0, len(uniqCodes))
	for _, code := range uniqCodes {
		if !resolved[strings.ToLower(strings.TrimSpace(code))] {
			remaining = append(remaining, code)
		}
	}
	if len(remaining) > 0 {
		var latestRows []row
		err := r.db.WithContext(ctx).
			Table("items AS i").
			Select("DISTINCT ON (i.uniq_code) " + selectCols).
			Joins("JOIN item_revisions ir ON ir.item_id = i.id").
			Joins("JOIN item_material_specs ims ON ims.item_revision_id = ir.id").
			Where("i.deleted_at IS NULL AND i.uniq_code IN ? AND (ims.is_subcon = true OR NULLIF(TRIM(COALESCE(ims.type_material, '')), '') IS NOT NULL)", remaining).
			Order("i.uniq_code, ir.id DESC").
			Scan(&latestRows).Error
		if err != nil {
			return nil, err
		}
		for _, rw := range latestRows {
			out[strings.ToLower(strings.TrimSpace(rw.UniqCode))] = models.ItemTypeFlags{
				TypeMaterial: strings.ToLower(strings.TrimSpace(rw.TypeMaterial)),
				IsSubcon:     rw.IsSubcon,
			}
		}
	}
	return out, nil
}
