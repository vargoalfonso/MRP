package service

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/ganasa18/go-template/internal/procurement/models"
)

// splitChild is one child material extracted from a budget entry's detail_jsonb,
// already resolved to a single supplier and a single stage quantity.
type splitChild struct {
	entryID       int64
	parentUniq    string
	childUniqCode string
	uniqCode      string // uniq / material_grade used as the line's item_uniq_code
	partName      string
	partNumber    string
	uom           string
	weightKg      *float64
	materialGrade string
	materialSpec  map[string]interface{}
	// supplier for this child (legacy BIGINT id + name)
	supplierID   *int64
	supplierName string
	// childQty is the child's full budget quantity (before stage split).
	childQty float64
}

// budgetDetail mirrors the po_budget detail_jsonb shape we care about here.
type budgetDetail struct {
	Parents []budgetParent `json:"parents"`
	// Children is the flat backward-compat list; only used when parents is empty.
	Children []budgetChild `json:"children"`
}

type budgetParent struct {
	UniqCode   string         `json:"uniq_code"`
	PartName   string         `json:"part_name"`
	PartNumber string         `json:"part_number"`
	Children   []budgetChild  `json:"children"`
}

type budgetChild struct {
	Uniq                string                 `json:"uniq"`
	UniqCode            string                 `json:"uniq_code"`
	PartName            string                 `json:"part_name"`
	PartNumber          string                 `json:"part_number"`
	Uom                 string                 `json:"uom"`
	Quantity            float64                `json:"quantity"`
	WeightKg            *float64               `json:"weight_kg"`
	MaterialSpec        map[string]interface{} `json:"material_spec"`
	ExistingRawMaterial string                 `json:"existing_raw_material"`
	Suppliers           []budgetChildSupplier  `json:"suppliers"`
}

type budgetChildSupplier struct {
	SupplierID   *int64  `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	Quantity     float64 `json:"quantity"`
}

// extractSplitChildren flattens an entry's detail_jsonb into per-child rows.
// Returns nil when the entry has no usable child breakdown (caller then falls
// back to the flat entry-level PO1/PO2 quantities).
func extractSplitChildren(e models.POBudgetEntry) []splitChild {
	raw := []byte(e.DetailJSON)
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return nil
	}

	var detail budgetDetail
	if err := json.Unmarshal(raw, &detail); err != nil {
		return nil
	}

	// Prefer the multi-parent shape; fall back to the flat children list.
	type parentBucket struct {
		uniqCode string
		children []budgetChild
	}
	buckets := make([]parentBucket, 0)
	if len(detail.Parents) > 0 {
		for _, p := range detail.Parents {
			buckets = append(buckets, parentBucket{uniqCode: p.UniqCode, children: p.Children})
		}
	} else if len(detail.Children) > 0 {
		buckets = append(buckets, parentBucket{uniqCode: e.UniqCode, children: detail.Children})
	} else {
		return nil
	}

	out := make([]splitChild, 0)
	for _, bucket := range buckets {
		parentUniq := strings.TrimSpace(bucket.uniqCode)
		if parentUniq == "" {
			parentUniq = e.UniqCode
		}
		for _, c := range bucket.children {
			// Resolve the line's uniq: prefer material grade / uniq, then child uniq_code.
			grade := materialGradeOf(c)
			lineUniq := strings.TrimSpace(c.Uniq)
			if lineUniq == "" {
				lineUniq = grade
			}
			if lineUniq == "" {
				lineUniq = strings.TrimSpace(c.UniqCode)
			}

			// One split row per supplier assigned to this child. Supplier qty is
			// the child budget share for that supplier (falls back to child qty).
			suppliers := c.Suppliers
			if len(suppliers) == 0 {
				suppliers = []budgetChildSupplier{{SupplierName: supplierNameOf(e), SupplierID: e.SupplierID, Quantity: c.Quantity}}
			}
			for _, sup := range suppliers {
				qty := sup.Quantity
				if qty <= 0 {
					qty = c.Quantity
				}
				if qty <= 0 {
					continue
				}
				out = append(out, splitChild{
					entryID:       e.ID,
					parentUniq:    parentUniq,
					childUniqCode: strings.TrimSpace(c.UniqCode),
					uniqCode:      lineUniq,
					partName:      strings.TrimSpace(c.PartName),
					partNumber:    strings.TrimSpace(c.PartNumber),
					uom:           strings.TrimSpace(c.Uom),
					weightKg:      c.WeightKg,
					materialGrade: grade,
					materialSpec:  c.MaterialSpec,
					supplierID:    sup.SupplierID,
					supplierName:  strings.TrimSpace(sup.SupplierName),
					childQty:      qty,
				})
			}
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func materialGradeOf(c budgetChild) string {
	if c.MaterialSpec != nil {
		if v, ok := c.MaterialSpec["material_grade"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		if v, ok := c.MaterialSpec["grade"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func supplierNameOf(e models.POBudgetEntry) string {
	if e.SupplierName != nil {
		return strings.TrimSpace(*e.SupplierName)
	}
	return ""
}

// stageQty returns the split quantity for a child at the given stage,
// rounded to a whole unit.
func stageQty(childQty float64, stage int, po1Pct, po2Pct float64) float64 {
	pct := po1Pct
	if stage == 2 {
		pct = po2Pct
	}
	return math.Round(childQty * pct / 100)
}
