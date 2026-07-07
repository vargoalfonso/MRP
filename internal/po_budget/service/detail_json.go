package service

import (
	"encoding/json"
	"strings"

	"github.com/ganasa18/go-template/internal/po_budget/models"
	"gorm.io/datatypes"
)

func emptyDetailJSON() datatypes.JSON {
	return datatypes.JSON([]byte(`{}`))
}

// buildBulkEntryDetailJSON merges every UNIQ item (and each item's supplier
// allocations) belonging to one PRL into a single detail_jsonb payload, so
// that one PRL always produces one po_budget_entries row with a
// children[] array holding all of its BOM/UNIQ rows.
func buildBulkEntryDetailJSON(
	budgetType string,
	parent models.PRLRow,
	items []models.BulkItemInput,
) (datatypes.JSON, error) {
	if !shouldAttachPRLChildren(budgetType) {
		return emptyDetailJSON(), nil
	}

	children := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		childUniqCode := strings.TrimSpace(valueOrEmpty(item.ChildUniqCode))
		if childUniqCode == "" {
			childUniqCode = strings.TrimSpace(item.UniqCode)
		}

		model := strings.TrimSpace(valueOrEmpty(item.Model))
		if model == "" {
			model = strings.TrimSpace(valueOrEmpty(item.ProductModel))
		}

		suppliers := make([]map[string]interface{}, 0, len(item.Suppliers))
		for _, sup := range item.Suppliers {
			suppliers = append(suppliers, map[string]interface{}{
				"supplier_id":   supplierIDValue(sup.SupplierID),
				"supplier_name": strings.TrimSpace(sup.SupplierName),
				"quantity":      sup.Quantity,
			})
		}

		var itemQty float64
		for _, sup := range item.Suppliers {
			itemQty += sup.Quantity
		}

		children = append(children, map[string]interface{}{
			"uniq":                  strings.TrimSpace(item.UniqCode),
			"uniq_code":             childUniqCode,
			"part_name":             strings.TrimSpace(valueOrEmpty(item.PartName)),
			"part_number":           strings.TrimSpace(valueOrEmpty(item.PartNumber)),
			"model":                 model,
			"qty_per_uniq":          valueOrZero(item.QtyPerUniq),
			"weight_kg":             valueOrZero(item.WeightKg),
			"quantity":              itemQty,
			"existing_raw_material": strings.TrimSpace(valueOrEmpty(item.ExistingRawMaterial)),
			"uom":                   strings.TrimSpace(valueOrEmpty(item.Uom)),
			"material_spec":         item.MaterialSpec,
			"suppliers":             suppliers,
		})
	}

	payload := map[string]interface{}{
		"parent": map[string]interface{}{
			"prl_id":      strings.TrimSpace(parent.PrlID),
			"prl_row_id":  parent.ID,
			"uniq_code":   strings.TrimSpace(valueOrEmpty(parent.UniqCode)),
			"part_name":   strings.TrimSpace(valueOrEmpty(parent.PartName)),
			"part_number": strings.TrimSpace(valueOrEmpty(parent.PartNumber)),
		},
		"children": children,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func supplierIDValue(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
