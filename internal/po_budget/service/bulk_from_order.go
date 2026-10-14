package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/po_budget/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// BulkCreateFromPRL — jalur sumber Customer PO / Delivery Note / Special Order
// ---------------------------------------------------------------------------
//
// Wizard "Add New PO Budget" boleh memakai dokumen dari menu
// "Customer PO DN SO" (tabel customer_order_documents) sebagai sumber, tanpa
// PRL. Frontend mengirim prl_id = "po:<uuid>" | "dn:<uuid>" | "so:<uuid>" dan
// tiap item tidak punya prl_item_id.
//
// Perbedaan dengan jalur PRL:
//   - tidak ada baris PRL yang dimuat dan prl_row_id disimpan NULL;
//   - tidak ada pengecekan plafon alokasi terhadap qty PRL;
//   - prl_ref diisi nomor dokumen (PO / DN / SO) sebagai referensi.
//
// Pengelompokan per tipe budget (raw_material / indirect / subcon), split PO,
// detail_jsonb, log, dan approval sama persis dengan jalur PRL.

type orderDocHeader struct {
	ID             int64  `gorm:"column:id"`
	UUID           string `gorm:"column:uuid"`
	DocumentType   string `gorm:"column:document_type"`
	DocumentNumber string `gorm:"column:document_number"`
	CustomerName   string `gorm:"column:customer_name_snapshot"`
}

type orderDocItem struct {
	ItemUniqCode string  `gorm:"column:item_uniq_code"`
	PartName     string  `gorm:"column:part_name"`
	PartNumber   string  `gorm:"column:part_number"`
	Model        *string `gorm:"column:model"`
	Quantity     float64 `gorm:"column:quantity"`
}

// parseOrderSourceRef mengenali prl_id berbentuk "po:<uuid>", "dn:<uuid>"
// atau "so:<uuid>" dan mengembalikan uuid dokumennya.
func parseOrderSourceRef(prlID string) (string, bool) {
	v := strings.TrimSpace(prlID)
	lower := strings.ToLower(v)
	for _, prefix := range []string{"po:", "dn:", "so:"} {
		if strings.HasPrefix(lower, prefix) {
			id := strings.TrimSpace(v[len(prefix):])
			return id, id != ""
		}
	}
	return "", false
}

func orderUniqKey(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (s *svc) bulkCreateFromOrderDocument(ctx context.Context, req models.BulkFromPRLRequest, docUUID, createdBy string) (*models.BulkFromPRLResult, error) {
	periodDate, err := parsePeriod(req.Period)
	if err != nil {
		return nil, apperror.BadRequest("invalid period format, use 'Month YYYY', e.g. 'October 2025'")
	}

	var doc orderDocHeader
	if err := s.db.WithContext(ctx).
		Table("customer_order_documents").
		Where("uuid = ? AND deleted_at IS NULL", docUUID).
		Take(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest(fmt.Sprintf("dokumen order %s tidak ditemukan", docUUID))
		}
		return nil, apperror.InternalWrap("database error", err)
	}

	var docItems []orderDocItem
	if err := s.db.WithContext(ctx).
		Table("customer_order_document_items").
		Where("document_id = ?", doc.ID).
		Order("line_no").
		Find(&docItems).Error; err != nil {
		return nil, apperror.InternalWrap("database error", err)
	}
	docItemByUniq := make(map[string]orderDocItem, len(docItems))
	for _, it := range docItems {
		key := orderUniqKey(it.ItemUniqCode)
		if _, exists := docItemByUniq[key]; !exists {
			docItemByUniq[key] = it
		}
	}

	// prl_ref berupa varchar(32): pakai nomor dokumen (dipotong bila perlu).
	ref := strings.TrimSpace(doc.DocumentNumber)
	if len(ref) > 32 {
		ref = ref[:32]
	}
	customerName := strings.TrimSpace(doc.CustomerName)
	var customerNamePtr *string
	if customerName != "" {
		customerNamePtr = &customerName
	}
	docLabel := strings.ToUpper(strings.TrimSpace(doc.DocumentType))
	if docLabel == "" {
		docLabel = "ORDER"
	}

	result := &models.BulkFromPRLResult{}
	if len(req.Items) == 0 {
		return result, nil
	}

	// Tipe budget tiap item diturunkan dari item_material_specs.type_material
	// (sama seperti jalur PRL; URL :type sengaja diabaikan).
	typeLookupCodes := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		typeLookupCodes = append(typeLookupCodes, itemTypeUniqCode(item))
	}
	typeMap, err := s.repo.ResolveTypeMaterialByUniq(ctx, typeLookupCodes)
	if err != nil {
		return nil, apperror.InternalWrap("resolve type_material failed", err)
	}

	type orderGroup struct {
		budgetType string
		items      []models.BulkItemInput
	}
	groupOrder := make([]string, 0, len(req.Items))
	groups := make(map[string]*orderGroup, len(req.Items))

	for idx, item := range req.Items {
		itemType := budgetTypeFromMaterial(typeMap[strings.ToLower(strings.TrimSpace(itemTypeUniqCode(item)))])

		// raw_material / indirect digabung per (dokumen, tipe); subcon tidak
		// digabung: satu item = satu entry.
		key := ref + "|" + itemType
		if itemType == budgetTypeSubcon {
			key = fmt.Sprintf("%s|%s|%d", ref, itemType, idx)
		}
		g, ok := groups[key]
		if !ok {
			g = &orderGroup{budgetType: itemType}
			groups[key] = g
			groupOrder = append(groupOrder, key)
		}
		g.items = append(g.items, item)
	}

	subtype := req.BudgetSubtype
	var entries []models.POBudgetEntry

	for _, key := range groupOrder {
		g := groups[key]
		if len(g.items) == 0 {
			continue
		}

		entryBudgetType := g.budgetType
		attachChildren := shouldAttachPRLChildren(entryBudgetType)
		head := g.items[0]
		headDoc, hasHeadDoc := docItemByUniq[orderUniqKey(head.UniqCode)]

		uniqCode := strings.TrimSpace(head.UniqCode)
		if uniqCode == "" && hasHeadDoc {
			uniqCode = headDoc.ItemUniqCode
		}

		// Baris "induk" sintetis dari item dokumen (pengganti baris PRL).
		parent := models.PRLRow{
			PrlID:        ref,
			CustomerName: customerNamePtr,
			UniqCode:     &uniqCode,
		}
		if hasHeadDoc {
			pn := headDoc.PartNumber
			pname := headDoc.PartName
			parent.PartNumber = &pn
			parent.PartName = &pname
			parent.ProductModel = headDoc.Model
		}

		productModel := parent.ProductModel
		if attachChildren && head.ProductModel != nil {
			productModel = head.ProductModel
		} else if productModel == nil {
			productModel = head.ProductModel
		}
		partName := parent.PartName
		if attachChildren && head.PartName != nil {
			partName = head.PartName
		} else if partName == nil {
			partName = head.PartName
		}
		partNumber := parent.PartNumber
		if attachChildren && head.PartNumber != nil {
			partNumber = head.PartNumber
		} else if partNumber == nil {
			partNumber = head.PartNumber
		}

		po1Pct, po2Pct, err := s.resolveSplit(ctx, entryBudgetType, head.Po1Pct, head.Po2Pct)
		if err != nil {
			return nil, err
		}

		var totalQty, totalSalesPlan, orderQty float64
		seenUniq := make(map[string]struct{}, len(g.items))
		var uniqSupplierID *int64
		var uniqSupplierName string
		supplierMixed := false
		firstSupplierSeen := false

		for _, item := range g.items {
			totalSalesPlan += item.SalesPlan

			// Qty dokumen dihitung sekali per uniq.
			uk := orderUniqKey(item.UniqCode)
			if _, dup := seenUniq[uk]; !dup {
				seenUniq[uk] = struct{}{}
				if d, ok := docItemByUniq[uk]; ok {
					orderQty += d.Quantity
				}
			}

			for _, sup := range item.Suppliers {
				totalQty += sup.Quantity
				if !firstSupplierSeen {
					uniqSupplierID = sup.SupplierID
					uniqSupplierName = sup.SupplierName
					firstSupplierSeen = true
				} else if !supplierMixed {
					sameID := (sup.SupplierID == nil && uniqSupplierID == nil) ||
						(sup.SupplierID != nil && uniqSupplierID != nil && *sup.SupplierID == *uniqSupplierID)
					if !sameID || sup.SupplierName != uniqSupplierName {
						supplierMixed = true
					}
				}
			}
		}
		if orderQty <= 0 {
			orderQty = totalQty
		}
		parent.Quantity = orderQty

		supplierID := uniqSupplierID
		supplierName := uniqSupplierName
		if supplierMixed {
			supplierID = nil
			supplierName = "Multiple"
		}

		// Semua item sumber order memakai PrlItemID 0 -> satu "parent" di
		// detail_jsonb yang mewakili dokumen.
		detailJSON, detailErr := buildBulkEntryDetailJSON(entryBudgetType, parent, map[int64]models.PRLRow{0: parent}, g.items)
		if detailErr != nil {
			return nil, apperror.InternalWrap("build bulk detail_jsonb failed", detailErr)
		}

		cb := createdBy
		prlRef := ref
		entries = append(entries, models.POBudgetEntry{
			BudgetType:      entryBudgetType,
			CustomerID:      nil,
			CustomerName:    customerNamePtr,
			UniqCode:        uniqCode,
			ProductModel:    productModel,
			MaterialType:    nil,
			PartName:        partName,
			PartNumber:      partNumber,
			Quantity:        totalQty,
			Uom:             head.Uom,
			WeightKg:        head.WeightKg,
			SupplierID:      supplierID,
			SupplierName:    strPtr(supplierName),
			Period:          req.Period,
			PeriodDate:      periodDate,
			SalesPlan:       totalSalesPlan,
			PurchaseRequest: totalQty,
			Po1Pct:          po1Pct,
			Po2Pct:          po2Pct,
			Prl:             orderQty,
			PrlRef:          &prlRef,
			PrlRowID:        nil,
			BudgetSubtype:   &subtype,
			DetailJSON:      detailJSON,
			Status:          "Pending",
			CreatedBy:       &cb,
		})
	}

	if len(entries) > 0 {
		if err := s.repo.BulkCreateEntries(ctx, entries); err != nil {
			return nil, apperror.InternalWrap("database error", err)
		}
		for _, e := range entries {
			if e.ID == 0 {
				return nil, apperror.Internal("bulk create po budget returned empty entry id")
			}

			cb := createdBy
			createdNote := fmt.Sprintf("Bulk from %s %s — %s", docLabel, doc.DocumentNumber, e.BudgetType)
			submittedNote := fmt.Sprintf("Submitted for approval — %s", e.BudgetType)
			_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Created", Username: &cb, Notes: &createdNote})
			_ = s.repo.CreateLog(ctx, &models.POBudgetEntryLog{EntryID: e.ID, Action: "Submitted", Username: &cb, Notes: &submittedNote})

			if _, err := approval.CreateInstance(ctx, s.db, approval.CreateInstanceParams{
				ActionName:     "po-budget",
				ReferenceTable: "po_budget_entries",
				ReferenceID:    e.ID,
				SubmittedBy:    createdBy,
				MinLevels:      1,
			}); err != nil {
				return nil, err
			}
		}
	}

	result.Created = len(entries)
	return result, nil
}
