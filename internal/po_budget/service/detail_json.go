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
// allocations) belonging to one PRL into a single detail_jsonb payload.
//
// Shape (multi-parent):
//
//	{
//	  "parent":   { ... first PRL row, backward compat ... },
//	  "children": [ ... flat list of all items, backward compat ... ],
//	  "parents": [
//	    { "prl_row_id": 313, "uniq_code": "SWMB", ..., "children": [ ... ] },
//	    { "prl_row_id": 314, "uniq_code": "SWM-B", ..., "children": [ ... ] }
//	  ]
//	}
//
// Each entry in parents[] is one UNIQ item of the PRL (prl_item_id) with its
// own children, so the FE can render the PRL → parent UNIQ → child tree.
func buildBulkEntryDetailJSON(
	budgetType string,
	parent models.PRLRow,
	parentRows map[int64]models.PRLRow,
	items []models.BulkItemInput,
) (datatypes.JSON, error) {
	if !shouldAttachPRLChildren(budgetType) {
		return emptyDetailJSON(), nil
	}

	toChildEntry := func(item models.BulkItemInput) map[string]interface{} {
		childUniqCode := strings.TrimSpace(valueOrEmpty(item.ChildUniqCode))
		if childUniqCode == "" {
			childUniqCode = strings.TrimSpace(item.UniqCode)
		}

		model := strings.TrimSpace(valueOrEmpty(item.Model))
		if model == "" {
			model = strings.TrimSpace(valueOrEmpty(item.ProductModel))
		}

		suppliers := make([]map[string]interface{}, 0, len(item.Suppliers))
		var itemQty float64
		for _, sup := range item.Suppliers {
			itemQty += sup.Quantity
			suppliers = append(suppliers, map[string]interface{}{
				"supplier_id":   supplierIDValue(sup.SupplierID),
				"supplier_name": strings.TrimSpace(sup.SupplierName),
				"quantity":      sup.Quantity,
			})
		}

		return map[string]interface{}{
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
		}
	}

	// Flat children (backward compat) + grouped parents[] by prl_item_id.
	children := make([]map[string]interface{}, 0, len(items))
	parentOrder := make([]int64, 0, len(items))
	childrenByParent := make(map[int64][]map[string]interface{}, len(items))
	parentUniqFallback := make(map[int64]string, len(items))

	for _, item := range items {
		entry := toChildEntry(item)
		children = append(children, entry)

		if _, ok := childrenByParent[item.PrlItemID]; !ok {
			parentOrder = append(parentOrder, item.PrlItemID)
			parentUniqFallback[item.PrlItemID] = strings.TrimSpace(item.UniqCode)
		}
		childrenByParent[item.PrlItemID] = append(childrenByParent[item.PrlItemID], entry)
	}

	parents := make([]map[string]interface{}, 0, len(parentOrder))
	for _, rowID := range parentOrder {
		row, ok := parentRows[rowID]
		uniqCode := strings.TrimSpace(valueOrEmpty(row.UniqCode))
		if uniqCode == "" {
			uniqCode = parentUniqFallback[rowID]
		}

		entry := map[string]interface{}{
			"prl_row_id":  rowID,
			"uniq_code":   uniqCode,
			"part_name":   strings.TrimSpace(valueOrEmpty(row.PartName)),
			"part_number": strings.TrimSpace(valueOrEmpty(row.PartNumber)),
			"children":    childrenByParent[rowID],
		}
		if ok {
			entry["quantity"] = row.Quantity
		}
		parents = append(parents, entry)
	}

	payload := map[string]interface{}{
		"prl_id": strings.TrimSpace(parent.PrlID),
		"parent": map[string]interface{}{
			"prl_id":      strings.TrimSpace(parent.PrlID),
			"prl_row_id":  parent.ID,
			"uniq_code":   strings.TrimSpace(valueOrEmpty(parent.UniqCode)),
			"part_name":   strings.TrimSpace(valueOrEmpty(parent.PartName)),
			"part_number": strings.TrimSpace(valueOrEmpty(parent.PartNumber)),
		},
		"children": children,
		"parents":  parents,
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
