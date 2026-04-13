package repository

import (
	"fmt"
	"strings"
	"time"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	procModels "github.com/ganasa18/go-template/internal/procurement/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// inventoryHelper provides methods for posting to raw material inventory tables.
type inventoryHelper struct {
	db *gorm.DB
}

// postToInventoryByDNType routes to the appropriate inventory table based on DN type.
func (r *repo) postToInventoryByDNType(
	tx *gorm.DB,
	dnType string,
	itemUniqCode string,
	approvedQty int,
	weightKg *float64,
	uom *string,
	warehouseLocation *string,
	createdBy string,
) error {
	createdBy = normalizeActor(createdBy)

	switch strings.ToUpper(strings.TrimSpace(dnType)) {
	case "RM", "RAW MATERIAL":
		return r.upsertRawMaterial(tx, itemUniqCode, float64(approvedQty), weightKg, uom, warehouseLocation, createdBy)
	case "IB", "INDIRECT", "INDIRECT RAW MATERIAL":
		return r.upsertIndirectRawMaterial(tx, itemUniqCode, float64(approvedQty), weightKg, uom, warehouseLocation, createdBy)
	case "SC", "SUBCON", "SUBCON MATERIAL", "SUBCON RAW MATERIAL", "SUB CON", "SUB-CON":
		return r.upsertSubconInventory(tx, itemUniqCode, float64(approvedQty), weightKg, uom, createdBy)
	default:
		return nil
	}
}

// lookupPOItem does a best-effort lookup of part_number, part_name, material_type from purchase_order_items.
func lookupPOItem(tx *gorm.DB, itemUniqCode string) (partNumber, partName, materialType *string) {
	var poi procModels.PurchaseOrderItem
	if err := tx.Select("part_number, part_name, material_type").
		Where("item_uniq_code = ?", itemUniqCode).
		First(&poi).Error; err == nil {
		partNumber = poi.PartNumber
		partName = poi.PartName
		materialType = poi.MaterialType
	}
	return
}

// upsertRawMaterial creates or updates raw material entry and posts stock increase.
func (r *repo) upsertRawMaterial(
	tx *gorm.DB,
	itemUniqCode string,
	approvedQty float64,
	weightKg *float64,
	uom *string,
	warehouseLocation *string,
	createdBy string,
) error {
	var rm invModels.RawMaterial

	// Try to find existing entry
	result := tx.Where("uniq_code = ? AND deleted_at IS NULL", itemUniqCode).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&rm)

	if result.Error == gorm.ErrRecordNotFound {
		// CREATE new entry — enrich with PO item details
		partNumber, partName, materialType := lookupPOItem(tx, itemUniqCode)
		now := time.Now()
		rm = invModels.RawMaterial{
			UUID:       uuid.New(),
			UniqCode:   itemUniqCode,
			PartNumber: partNumber,
			PartName:   partName,
			RawMaterialType: func() string {
				if materialType != nil {
					return *materialType
				}
				return ""
			}(),
			RMSource:          "supplier",
			UOM:               uom,
			WarehouseLocation: warehouseLocation,
			StockQty:          approvedQty,
			Status:            "normal",
			BuyNotBuy:         "not_buy",
			CreatedBy:         &createdBy,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if weightKg != nil {
			rm.StockWeightKg = weightKg
		}

		if err := tx.Create(&rm).Error; err != nil {
			return fmt.Errorf("create raw_material: %w", err)
		}

		if err := writeInventoryMovementLog(tx, inventoryLogInput{
			Category:     "raw_material",
			MovementType: "incoming",
			UniqCode:     itemUniqCode,
			EntityID:     &rm.ID,
			QtyChange:    approvedQty,
			WeightChange: weightKg,
			SourceFlag:   strPtr("qc_approve"),
			LoggedBy:     &createdBy,
			LoggedAt:     now,
		}); err != nil {
			return fmt.Errorf("log raw_material transaction: %w", err)
		}

		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("query raw_material: %w", result.Error)
	}

	// UPDATE existing entry
	updates := map[string]interface{}{
		"stock_qty":  gorm.Expr("stock_qty + ?", approvedQty),
		"updated_by": &createdBy,
		"updated_at": time.Now(),
	}

	if weightKg != nil {
		updates["stock_weight_kg"] = gorm.Expr("COALESCE(stock_weight_kg, 0) + ?", *weightKg)
	}

	if err := tx.Model(&rm).Updates(updates).Error; err != nil {
		return fmt.Errorf("update raw_material stock: %w", err)
	}

	// Recalculate derived fields
	if err := r.recalculateRawMaterialStatus(tx, rm.ID); err != nil {
		return fmt.Errorf("recalculate raw_material status: %w", err)
	}

	// Log transaction
	now := time.Now()
	if err := writeInventoryMovementLog(tx, inventoryLogInput{
		Category:     "raw_material",
		MovementType: "incoming",
		UniqCode:     itemUniqCode,
		EntityID:     &rm.ID,
		QtyChange:    approvedQty,
		WeightChange: weightKg,
		SourceFlag:   strPtr("qc_approve"),
		LoggedBy:     &createdBy,
		LoggedAt:     now,
	}); err != nil {
		return fmt.Errorf("log raw_material transaction: %w", err)
	}

	return nil
}

// upsertIndirectRawMaterial creates or updates indirect raw material entry.
func (r *repo) upsertIndirectRawMaterial(
	tx *gorm.DB,
	itemUniqCode string,
	approvedQty float64,
	weightKg *float64,
	uom *string,
	warehouseLocation *string,
	createdBy string,
) error {
	var irm invModels.IndirectRawMaterial

	result := tx.Where("uniq_code = ? AND deleted_at IS NULL", itemUniqCode).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&irm)

	if result.Error == gorm.ErrRecordNotFound {
		partNumber, partName, _ := lookupPOItem(tx, itemUniqCode)
		now := time.Now()
		irm = invModels.IndirectRawMaterial{
			UUID:              uuid.New(),
			UniqCode:          itemUniqCode,
			PartNumber:        partNumber,
			PartName:          partName,
			UOM:               uom,
			WarehouseLocation: warehouseLocation,
			StockQty:          approvedQty,
			Status:            stringPtr("normal"),
			BuyNotBuy:         "not_buy",
			CreatedBy:         &createdBy,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if weightKg != nil {
			irm.StockWeightKg = weightKg
		}

		if err := tx.Create(&irm).Error; err != nil {
			return fmt.Errorf("create indirect_raw_material: %w", err)
		}

		if err := writeInventoryMovementLog(tx, inventoryLogInput{
			Category:     "indirect_raw_material",
			MovementType: "incoming",
			UniqCode:     itemUniqCode,
			EntityID:     &irm.ID,
			QtyChange:    approvedQty,
			WeightChange: weightKg,
			SourceFlag:   strPtr("qc_approve"),
			LoggedBy:     &createdBy,
			LoggedAt:     now,
		}); err != nil {
			return fmt.Errorf("log indirect_raw_material transaction: %w", err)
		}

		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("query indirect_raw_material: %w", result.Error)
	}

	// UPDATE existing entry
	updates := map[string]interface{}{
		"stock_qty":  gorm.Expr("stock_qty + ?", approvedQty),
		"updated_by": &createdBy,
		"updated_at": time.Now(),
	}

	if weightKg != nil {
		updates["stock_weight_kg"] = gorm.Expr("COALESCE(stock_weight_kg, 0) + ?", *weightKg)
	}

	if err := tx.Model(&irm).Updates(updates).Error; err != nil {
		return fmt.Errorf("update indirect_raw_material stock: %w", err)
	}

	// Recalculate derived fields
	if err := r.recalculateIndirectRawMaterialStatus(tx, irm.ID); err != nil {
		return fmt.Errorf("recalculate indirect_raw_material status: %w", err)
	}

	// Log transaction
	now := time.Now()
	if err := writeInventoryMovementLog(tx, inventoryLogInput{
		Category:     "indirect_raw_material",
		MovementType: "incoming",
		UniqCode:     itemUniqCode,
		EntityID:     &irm.ID,
		QtyChange:    approvedQty,
		WeightChange: weightKg,
		SourceFlag:   strPtr("qc_approve"),
		LoggedBy:     &createdBy,
		LoggedAt:     now,
	}); err != nil {
		return fmt.Errorf("log indirect_raw_material transaction: %w", err)
	}

	return nil
}

// upsertSubconInventory creates or updates subcon inventory entry.
func (r *repo) upsertSubconInventory(
	tx *gorm.DB,
	itemUniqCode string,
	approvedQty float64,
	weightKg *float64,
	uom *string,
	createdBy string,
) error {
	var si invModels.SubconInventory

	result := tx.Where("uniq_code = ? AND deleted_at IS NULL", itemUniqCode).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&si)

	if result.Error == gorm.ErrRecordNotFound {
		partNumber, partName, _ := lookupPOItem(tx, itemUniqCode)
		now := time.Now()
		si = invModels.SubconInventory{
			UUID:             uuid.New(),
			UniqCode:         itemUniqCode,
			PartNumber:       partNumber,
			PartName:         partName,
			StockAtVendorQty: approvedQty,
			Status:           "normal",
			CreatedBy:        &createdBy,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		_ = uom // SubconInventory has no UOM field

		if err := tx.Create(&si).Error; err != nil {
			return fmt.Errorf("create subcon_inventory: %w", err)
		}

		if err := writeInventoryMovementLog(tx, inventoryLogInput{
			Category:     "subcon",
			MovementType: "incoming",
			UniqCode:     itemUniqCode,
			EntityID:     &si.ID,
			QtyChange:    approvedQty,
			WeightChange: nil,
			SourceFlag:   strPtr("qc_approve"),
			LoggedBy:     &createdBy,
			LoggedAt:     now,
		}); err != nil {
			return fmt.Errorf("log subcon_inventory transaction: %w", err)
		}

		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("query subcon_inventory: %w", result.Error)
	}

	// UPDATE existing entry
	now := time.Now()
	updates := map[string]interface{}{
		"stock_at_vendor_qty": gorm.Expr("stock_at_vendor_qty + ?", approvedQty),
		"updated_by":          &createdBy,
		"updated_at":          now,
	}

	if err := tx.Model(&si).Updates(updates).Error; err != nil {
		return fmt.Errorf("update subcon_inventory stock: %w", err)
	}

	// Recalculate status
	if err := r.recalculateSubconInventoryStatus(tx, si.ID); err != nil {
		return fmt.Errorf("recalculate subcon_inventory status: %w", err)
	}

	// Log transaction
	if err := writeInventoryMovementLog(tx, inventoryLogInput{
		Category:     "subcon",
		MovementType: "incoming",
		UniqCode:     itemUniqCode,
		EntityID:     &si.ID,
		QtyChange:    approvedQty,
		WeightChange: nil,
		SourceFlag:   strPtr("qc_approve"),
		LoggedBy:     &createdBy,
		LoggedAt:     now,
	}); err != nil {
		return fmt.Errorf("log subcon_inventory transaction: %w", err)
	}

	return nil
}

// recalculateRawMaterialStatus updates status, stock_days, buy_not_buy fields.
func (r *repo) recalculateRawMaterialStatus(tx *gorm.DB, rmID int64) error {
	var rm invModels.RawMaterial
	if err := tx.First(&rm, "id = ?", rmID).Error; err != nil {
		return err
	}

	// Get safety stock threshold
	safetyStock := 10.0 // default from parameter
	if rm.SafetyStockQty != nil && *rm.SafetyStockQty > 0 {
		safetyStock = *rm.SafetyStockQty
	}

	// Get daily usage
	dailyUsage := 1.0 // default fallback
	if rm.DailyUsageQty != nil && *rm.DailyUsageQty > 0 {
		dailyUsage = *rm.DailyUsageQty
	}

	// Calculate stock days
	stockDays := int(rm.StockQty / dailyUsage)

	// Determine status
	status := "normal"
	if rm.StockQty < safetyStock {
		status = "low_on_stock"
	} else if rm.StockQty > safetyStock*2 {
		status = "overstock"
	}

	// Determine buy/not buy
	buyNotBuy := "not_buy"
	if rm.RawMaterialType == "ssp" {
		buyNotBuy = "n/a"
	} else if rm.StockQty < safetyStock {
		buyNotBuy = "buy"
	}

	// Update
	return tx.Model(&rm).Updates(map[string]interface{}{
		"status":      status,
		"stock_days":  stockDays,
		"buy_not_buy": buyNotBuy,
		"updated_at":  time.Now(),
	}).Error
}

// recalculateIndirectRawMaterialStatus similar to RawMaterial.
func (r *repo) recalculateIndirectRawMaterialStatus(tx *gorm.DB, irmID int64) error {
	var irm invModels.IndirectRawMaterial
	if err := tx.First(&irm, "id = ?", irmID).Error; err != nil {
		return err
	}

	safetyStock := 10.0
	if irm.SafetyStockQty != nil && *irm.SafetyStockQty > 0 {
		safetyStock = *irm.SafetyStockQty
	}

	dailyUsage := 1.0
	if irm.DailyUsageQty != nil && *irm.DailyUsageQty > 0 {
		dailyUsage = *irm.DailyUsageQty
	}

	stockDays := int(irm.StockQty / dailyUsage)

	status := stringPtr("normal")
	if irm.StockQty < safetyStock {
		status = stringPtr("low_on_stock")
	} else if irm.StockQty > safetyStock*2 {
		status = stringPtr("overstock")
	}

	buyNotBuy := "not_buy"
	if irm.StockQty < safetyStock {
		buyNotBuy = "buy"
	}

	return tx.Model(&irm).Updates(map[string]interface{}{
		"status":      status,
		"stock_days":  stockDays,
		"buy_not_buy": buyNotBuy,
		"updated_at":  time.Now(),
	}).Error
}

// recalculateSubconInventoryStatus updates subcon status.
func (r *repo) recalculateSubconInventoryStatus(tx *gorm.DB, siID int64) error {
	var si invModels.SubconInventory
	if err := tx.First(&si, "id = ?", siID).Error; err != nil {
		return err
	}

	safetyStock := 10.0
	if si.SafetyStockQty != nil && *si.SafetyStockQty > 0 {
		safetyStock = *si.SafetyStockQty
	}

	status := "normal"
	if si.StockAtVendorQty < safetyStock {
		status = "low_on_stock"
	} else if si.StockAtVendorQty > safetyStock*2 {
		status = "overstock"
	}

	// Calculate delta PO
	var deltaPO *float64
	if si.TotalPOQty != nil {
		delta := *si.TotalPOQty - si.StockAtVendorQty
		deltaPO = &delta
	}

	return tx.Model(&si).Updates(map[string]interface{}{
		"status":     status,
		"delta_po":   deltaPO,
		"updated_at": time.Now(),
	}).Error
}

// Helper: convert string to *string
func stringPtr(s string) *string {
	return &s
}

func strPtr(s string) *string {
	return &s
}

type inventoryLogInput struct {
	Category     string
	MovementType string
	UniqCode     string
	EntityID     *int64
	QtyChange    float64
	WeightChange *float64
	SourceFlag   *string
	DNNumber     *string
	ReferenceID  *string
	Notes        *string
	LoggedBy     *string
	LoggedAt     time.Time
}

func writeInventoryMovementLog(tx *gorm.DB, in inventoryLogInput) error {
	log := invModels.InventoryMovementLog{
		MovementCategory: in.Category,
		MovementType:     in.MovementType,
		UniqCode:         in.UniqCode,
		EntityID:         in.EntityID,
		QtyChange:        in.QtyChange,
		WeightChange:     in.WeightChange,
		SourceFlag:       in.SourceFlag,
		DNNumber:         in.DNNumber,
		ReferenceID:      in.ReferenceID,
		Notes:            in.Notes,
		LoggedBy:         in.LoggedBy,
		LoggedAt:         in.LoggedAt,
	}
	return tx.Create(&log).Error
}

func tableExists(tx *gorm.DB, table string) bool {
	var exists bool
	_ = tx.Raw("SELECT to_regclass(?) IS NOT NULL", table).Scan(&exists).Error
	return exists
}

func pickUniqColumn(tx *gorm.DB, table string) (string, error) {
	candidates := []string{"uniq", "item_uniq_code", "uniq_code"}
	for _, col := range candidates {
		var exists bool
		err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name = ? AND column_name = ?
		)`, table, col).Scan(&exists).Error
		if err == nil && exists {
			return col, nil
		}
	}
	return "", fmt.Errorf("cannot map uniq code for table %s", table)
}

func pickQtyColumn(tx *gorm.DB, table string) (string, error) {
	candidates := []string{"stock", "quantity", "qty"}
	for _, col := range candidates {
		var exists bool
		err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name = ? AND column_name = ?
		)`, table, col).Scan(&exists).Error
		if err == nil && exists {
			return col, nil
		}
	}
	return "", fmt.Errorf("cannot map quantity column for table %s", table)
}

// postToScrap stores rejected scrap quantity when scrap_stock table is available.
func postToScrap(tx *gorm.DB, itemUniqCode string, qtyScrap int, packingNumber *string) error {
	if qtyScrap <= 0 || !tableExists(tx, "scrap_stock") {
		return nil
	}

	uniqCol, err := pickUniqColumn(tx, "scrap_stock")
	if err != nil {
		return nil
	}
	qtyCol, err := pickQtyColumn(tx, "scrap_stock")
	if err != nil {
		return nil
	}

	res := tx.Table("scrap_stock").
		Where(fmt.Sprintf("%s = ?", uniqCol), itemUniqCode).
		Update(qtyCol, gorm.Expr(fmt.Sprintf("COALESCE(%s,0) + ?", qtyCol), qtyScrap))
	if res.Error != nil {
		return nil
	}

	if res.RowsAffected == 0 {
		row := map[string]interface{}{uniqCol: itemUniqCode, qtyCol: qtyScrap, "scrap_type": "Incoming"}
		if packingNumber != nil {
			row["packing_number"] = *packingNumber
		}
		_ = tx.Table("scrap_stock").Create(row).Error
	}
	return nil
}
