package forecastingclient

import (
	"context"
	"net/url"
)

// InventoryPlanItem adalah satu baris hasil scheduler harian forecasting
// (GET /inventory-plan). Field angka bisa null, jadi memakai pointer.
type InventoryPlanItem struct {
	PlanDate             string   `json:"plan_date"`
	InventoryType        string   `json:"inventory_type"`
	UniqCode             string   `json:"uniq_code"`
	ModelVersionID       *string  `json:"model_version_id"`
	ForecastStart        *string  `json:"forecast_start"`
	ForecastEnd          *string  `json:"forecast_end"`
	ForecastQty          *float64 `json:"forecast_qty"`
	WorkingDays          *int     `json:"working_days"`
	ForecastDaily        *float64 `json:"forecast_daily"`
	SafetyStockConstanta *float64 `json:"safety_stock_constanta"`
	SafetyStockQty       *float64 `json:"safety_stock_qty"`
	StockQty             *float64 `json:"stock_qty"`
	CyclePengiriman      *int     `json:"cycle_pengiriman"`
	Threshold            *float64 `json:"threshold"`
	KanbanStd            *float64 `json:"kanban_std"`
	Status               *string  `json:"status"`
	BuyNotBuy            *string  `json:"buy_not_buy"`
	KanbansNeeded        *int     `json:"kanbans_needed"`
	BuyQty               *float64 `json:"buy_qty"`
	Note                 *string  `json:"note"`
	CreatedAt            *string  `json:"created_at"`
}

// GetInventoryPlanOptions adalah filter untuk GET /inventory-plan.
type GetInventoryPlanOptions struct {
	UniqCodes     []string // boleh banyak; kosong = semua baris
	InventoryType string   // raw_material | indirect_material | finished_goods | subcon
	PlanDate      string   // YYYY-MM-DD; kosong = plan terbaru
}

// GetInventoryPlan membaca hasil scheduler safety stock / buy-not-buy.
// Respons API berupa JSON array langsung; belum ada data -> slice kosong.
func (c *Client) GetInventoryPlan(ctx context.Context, opts GetInventoryPlanOptions) ([]InventoryPlanItem, error) {
	q := url.Values{}
	for _, u := range opts.UniqCodes {
		if u != "" {
			q.Add("uniq_code", u)
		}
	}
	if opts.InventoryType != "" {
		q.Set("inventory_type", opts.InventoryType)
	}
	if opts.PlanDate != "" {
		q.Set("plan_date", opts.PlanDate)
	}
	path := "/inventory-plan"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	resp := []InventoryPlanItem{}
	if err := c.do(ctx, "GET", path, nil, "", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
