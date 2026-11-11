package repository

import (
	"context"
	"strings"

	invModels "github.com/ganasa18/go-template/internal/inventory/models"
	"gorm.io/gorm"
)

func (r *repo) ListRawMaterialMasters(ctx context.Context, search string, limit, offset int) ([]invModels.RawMaterialMaster, int64, error) {
	q := r.db.WithContext(ctx).Model(&invModels.RawMaterialMaster{}).Where("deleted_at IS NULL")
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + s + "%"
		q = q.Where("material_code ILIKE ? OR material_name ILIKE ? OR material_grade ILIKE ? OR spec_key ILIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []invModels.RawMaterialMaster
	if err := q.Order("material_code ASC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	for i := range items {
		r.db.WithContext(ctx).Table("item_material_specs").Where("raw_material_master_id = ?", items[i].ID).Count(&items[i].BOMUsageCount)
		r.db.WithContext(ctx).Table("raw_materials").Where("raw_material_master_id = ? AND deleted_at IS NULL", items[i].ID).Select("COALESCE(SUM(stock_qty),0)").Scan(&items[i].InventoryQty)
	}
	return items, total, nil
}

func (r *repo) GetRawMaterialMaster(ctx context.Context, id int64) (*invModels.RawMaterialMaster, error) {
	var item invModels.RawMaterialMaster
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&item).Error; err != nil {
		return nil, err
	}
	r.db.WithContext(ctx).Table("item_material_specs").Where("raw_material_master_id = ?", id).Count(&item.BOMUsageCount)
	r.db.WithContext(ctx).Table("raw_materials").Where("raw_material_master_id = ? AND deleted_at IS NULL", id).Select("COALESCE(SUM(stock_qty),0)").Scan(&item.InventoryQty)
	return &item, nil
}

func (r *repo) CreateRawMaterialMaster(ctx context.Context, item *invModels.RawMaterialMaster) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *repo) UpdateRawMaterialMaster(ctx context.Context, id int64, updates map[string]interface{}) (*invModels.RawMaterialMaster, error) {
	if err := r.db.WithContext(ctx).Model(&invModels.RawMaterialMaster{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.GetRawMaterialMaster(ctx, id)
}

func (r *repo) DeleteRawMaterialMaster(ctx context.Context, id int64, deletedBy string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var refs int64
		if err := tx.Table("item_material_specs").Where("raw_material_master_id = ?", id).Count(&refs).Error; err != nil {
			return err
		}
		if refs > 0 {
			return gorm.ErrInvalidData
		}
		return tx.Model(&invModels.RawMaterialMaster{}).Where("id = ? AND deleted_at IS NULL", id).
			Updates(map[string]interface{}{"deleted_at": gorm.Expr("NOW()"), "updated_by": deletedBy, "updated_at": gorm.Expr("NOW()")}).Error
	})
}

func (r *repo) ListRawMaterialPlanning(ctx context.Context) ([]invModels.RawMaterialPlanningItem, error) {
	// Recursive explosion multiplies QPU across the active/current BOM tree, then pools demand by master id.
	const query = `
WITH RECURSIVE active_bom AS (
    SELECT bi.id AS bom_id, bi.item_id AS root_item_id, i.uniq_code AS root_uniq
    FROM bom_item bi JOIN items i ON i.id = bi.item_id
    WHERE bi.is_current = TRUE AND bi.status IN ('Released','Active')
), pattern_output AS (
    SELECT mp.uniq_code,
           SUM(CASE WHEN COALESCE(mp.cycle_time_sec,0) > 0 AND COALESCE(mp.min_output,0) > 0
                    THEN mp.min_output / mp.cycle_time_sec
                    ELSE COALESCE(mp.prl_reference,0) / NULLIF(COALESCE(mp.working_days,1),0) * GREATEST(COALESCE(mp.pattern_value,1),1)
               END) AS daily_output
    FROM machine_patterns mp
    WHERE LOWER(COALESCE(mp.status,'active')) = 'active' AND mp.deleted_at IS NULL
    GROUP BY mp.uniq_code
), explosion AS (
    SELECT ab.bom_id, ab.root_uniq, bl.child_item_id, bl.child_item_revision_id,
           bl.qty_per_uniq * (1 + COALESCE(bl.scrap_factor,0)) AS effective_qpu,
           ARRAY[bl.parent_item_id, bl.child_item_id]::bigint[] AS path
    FROM active_bom ab JOIN bom_lines bl ON bl.bom_item_id = ab.bom_id AND bl.parent_item_id = ab.root_item_id
    UNION ALL
    SELECT e.bom_id, e.root_uniq, bl.child_item_id, bl.child_item_revision_id,
           e.effective_qpu * bl.qty_per_uniq * (1 + COALESCE(bl.scrap_factor,0)), e.path || bl.child_item_id
    FROM explosion e JOIN bom_lines bl ON bl.bom_item_id = e.bom_id AND bl.parent_item_id = e.child_item_id
    WHERE NOT bl.child_item_id = ANY(e.path)
), mapped AS (
    SELECT e.root_uniq, e.effective_qpu, ims.raw_material_master_id
    FROM explosion e
    LEFT JOIN LATERAL (
        SELECT ims2.raw_material_master_id
        FROM item_material_specs ims2
        WHERE ims2.item_revision_id = COALESCE(e.child_item_revision_id,
            (SELECT MAX(ir.id) FROM item_revisions ir WHERE ir.item_id = e.child_item_id))
        LIMIT 1
    ) ims ON TRUE
    WHERE ims.raw_material_master_id IS NOT NULL
), demand AS (
    SELECT m.raw_material_master_id AS master_id,
           SUM(COALESCE(po.daily_output,0) * m.effective_qpu) AS daily_demand,
           COUNT(DISTINCT m.root_uniq) AS source_count
    FROM mapped m LEFT JOIN pattern_output po ON po.uniq_code = m.root_uniq
    GROUP BY m.raw_material_master_id
), stock AS (
    SELECT raw_material_master_id AS master_id, SUM(stock_qty) AS stock_qty,
           MAX(COALESCE(safety_stock_qty,0)) AS safety_stock
    FROM raw_materials WHERE deleted_at IS NULL AND raw_material_master_id IS NOT NULL
    GROUP BY raw_material_master_id
), targets AS (
    SELECT ims.raw_material_master_id AS master_id, MAX(COALESCE(sp.constanta,0)) AS target_days
    FROM item_material_specs ims
    JOIN item_revisions ir ON ir.id = ims.item_revision_id
    JOIN items i ON i.id = ir.item_id
    LEFT JOIN stockdays_parameters sp ON sp.item_uniq_code = i.uniq_code AND LOWER(COALESCE(sp.inventory_type,'')) IN ('raw','raw_material','raw_materials')
    WHERE ims.raw_material_master_id IS NOT NULL GROUP BY ims.raw_material_master_id
), calc AS (
    SELECT rmm.id AS master_id, rmm.material_code, rmm.material_name, rmm.material_grade, rmm.uom,
           COALESCE(d.daily_demand,0) AS daily_demand, COALESCE(s.stock_qty,0) AS beginning_stock,
           COALESCE(s.stock_qty,0)-COALESCE(d.daily_demand,0) AS ending_stock,
           COALESCE(s.safety_stock,0) AS safety_stock, COALESCE(t.target_days,0) AS target_days,
           GREATEST(COALESCE(s.safety_stock,0), COALESCE(d.daily_demand,0)*COALESCE(t.target_days,0)) AS minimum_level,
           COALESCE(d.source_count,0) AS source_count
    FROM raw_material_masters rmm
    LEFT JOIN demand d ON d.master_id=rmm.id LEFT JOIN stock s ON s.master_id=rmm.id LEFT JOIN targets t ON t.master_id=rmm.id
    WHERE rmm.deleted_at IS NULL AND rmm.status='Active'
)
SELECT master_id, material_code, material_name, material_grade, uom, daily_demand, beginning_stock, ending_stock,
       safety_stock, target_days AS target_stock_days,
       CASE WHEN daily_demand>0 THEN ending_stock/daily_demand ELSE 0 END AS actual_stock_days,
       minimum_level AS minimum_stock_level, GREATEST(0,minimum_level-ending_stock) AS recommended_buy_qty,
       CASE WHEN daily_demand=0 THEN 'DATA_INCOMPLETE'
            WHEN ending_stock < minimum_level OR (ending_stock/daily_demand) < target_days THEN 'BUY' ELSE 'NOT_BUY' END AS decision,
       source_count AS demand_source_count
FROM calc ORDER BY material_code`
	var rows []invModels.RawMaterialPlanningItem
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
