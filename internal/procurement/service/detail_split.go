package service

import (
	"encoding/json"
	"math"
	"strconv"
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
	// Suppliers: pembagian supplier level entri (entri tanpa child, mis. subcon /
	// indirect flat) beserta qty masing-masing.
	Suppliers []budgetChildSupplier `json:"suppliers"`
	Parents   []budgetParent        `json:"parents"`
	// Children is the flat backward-compat list; only used when parents is empty.
	Children []budgetChild `json:"children"`
}

type budgetParent struct {
	UniqCode   string        `json:"uniq_code"`
	PartName   string        `json:"part_name"`
	PartNumber string        `json:"part_number"`
	Children   []budgetChild `json:"children"`
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
			shareQtys := childSupplierQtys(suppliers, c.Quantity)
			for i, sup := range suppliers {
				qty := shareQtys[i]
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

// childSupplierQtys menentukan qty tiap supplier sebuah child.
//   - Ada supplier berqty > 0: pakai qty itu; supplier berqty 0 dilewati
//     (sebelumnya mendapat seluruh qty child sehingga qty terhitung ganda).
//   - Semua qty 0 dan hanya satu supplier: seluruh qty child.
//   - Semua qty 0 dan beberapa supplier: dibagi sama rata (dibulatkan, sisa ke
//     supplier terakhir) sehingga total tetap sama dengan qty child.
func childSupplierQtys(suppliers []budgetChildSupplier, childQty float64) []float64 {
	out := make([]float64, len(suppliers))
	positives := 0
	for i, s := range suppliers {
		if s.Quantity > 0 {
			out[i] = s.Quantity
			positives++
		}
	}
	if positives > 0 || childQty <= 0 || len(suppliers) == 0 {
		return out
	}
	if len(suppliers) == 1 {
		out[0] = childQty
		return out
	}
	var used float64
	for i := range suppliers {
		if i == len(suppliers)-1 {
			out[i] = childQty - used
			break
		}
		out[i] = math.Round(childQty / float64(len(suppliers)))
		used += out[i]
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

// flatSupplierShare adalah bagian satu supplier pada entri tanpa child.
type flatSupplierShare struct {
	supplierID   *int64
	supplierName string
	qty          float64
}

// extractFlatSupplierShares membaca detail_jsonb.suppliers entri tanpa child.
// Mengembalikan nil bila kurang dari DUA supplier berqty > 0 (entri tetap
// diproses dengan supplier level entri seperti sebelumnya).
func extractFlatSupplierShares(e models.POBudgetEntry) []flatSupplierShare {
	raw := []byte(e.DetailJSON)
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return nil
	}
	var detail budgetDetail
	if err := json.Unmarshal(raw, &detail); err != nil {
		return nil
	}

	out := make([]flatSupplierShare, 0, len(detail.Suppliers))
	index := map[string]int{}
	for _, s := range detail.Suppliers {
		name := strings.TrimSpace(s.SupplierName)
		if (name == "" && (s.SupplierID == nil || *s.SupplierID <= 0)) || s.Quantity <= 0 {
			continue
		}
		key := "name:" + strings.ToLower(name)
		if s.SupplierID != nil && *s.SupplierID > 0 {
			key = "id:" + strconv.FormatInt(*s.SupplierID, 10)
		}
		if i, ok := index[key]; ok {
			out[i].qty += s.Quantity
			continue
		}
		index[key] = len(out)
		out = append(out, flatSupplierShare{supplierID: s.SupplierID, supplierName: name, qty: s.Quantity})
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

// splitQtyByShares membagi total ke tiap share secara proporsional, dibulatkan ke
// bilangan bulat; share terakhir menerima sisanya supaya jumlahnya tetap sama.
func splitQtyByShares(total float64, shares []flatSupplierShare) []float64 {
	out := make([]float64, len(shares))
	var sum float64
	for _, sh := range shares {
		sum += sh.qty
	}
	if sum <= 0 || total <= 0 {
		return out
	}
	var used float64
	for i, sh := range shares {
		if i == len(shares)-1 {
			out[i] = total - used
			break
		}
		out[i] = math.Round(total * sh.qty / sum)
		used += out[i]
	}
	return out
}
