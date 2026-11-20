package forecastingclient

import (
	"context"
	"net/url"
)

// InventoryPlanItem adalah satu baris hasil scheduler harian forecasting
// (GET /inventory-plan). Field angka bisa null, jadi memakai pointer.
//
// Endpoint ini KHUSUS safety stock (uniq yang parameter safety stock-nya
// "forecast"). Arti field sejak 19 Nov 2026:
//   - SafetyStockQty = ForecastDaily, tanpa Constanta.
//   - Threshold      = SafetyStockQty (10 bila SafetyStockQty <= 0).
//   - CyclePengiriman selalu null (tidak dipakai lagi).
//   - BuyNotBuy memakai stock < threshold.
//   - Untuk RM / IRM, ForecastQty = permintaan per bulan hasil akumulasi
//     semua FG yang memakainya di BOM.
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
	Unit                 *string  `json:"unit"`
	FGTotal              *int     `json:"fg_total"`         // jumlah FG yang memakai RM/IRM ini di BOM
	FGWithForecast       *int     `json:"fg_with_forecast"` // dari FG_total, berapa yang punya forecast
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
	resp := []InventoryPlanItem{}
	if err := c.do(ctx, "GET", "/inventory-plan"+opts.query(), nil, "", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// query menyusun query string ("" bila tidak ada filter, selain itu diawali "?").
func (o GetInventoryPlanOptions) query() string {
	q := url.Values{}
	for _, u := range o.UniqCodes {
		if u != "" {
			q.Add("uniq_code", u)
		}
	}
	if o.InventoryType != "" {
		q.Set("inventory_type", o.InventoryType)
	}
	if o.PlanDate != "" {
		q.Set("plan_date", o.PlanDate)
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// InventoryStockdaysItem adalah satu baris hasil GET /inventory-plan/stockdays.
// Khusus RM / IRM yang parameter stockdays-nya "forecast". Field angka bisa
// null, jadi memakai pointer.
type InventoryStockdaysItem struct {
	PlanDate       string   `json:"plan_date"`
	InventoryType  string   `json:"inventory_type"`
	UniqCode       string   `json:"uniq_code"`
	ModelVersionID *string  `json:"model_version_id"`
	ForecastStart  *string  `json:"forecast_start"`
	ForecastEnd    *string  `json:"forecast_end"`
	ForecastQty    *float64 `json:"forecast_qty"` // permintaan per bulan, akumulasi BOM
	WorkingDays    *int     `json:"working_days"`
	DailyUsage     *float64 `json:"daily_usage"` // forecast_qty / working_days
	StockQty       *float64 `json:"stock_qty"`
	StockDays      *int     `json:"stock_days"`
	Unit           *string  `json:"unit"`
	FGTotal        *int     `json:"fg_total"`
	FGWithForecast *int     `json:"fg_with_forecast"`
	Note           *string  `json:"note"`
	CreatedAt      *string  `json:"created_at"`
}

// GetInventoryStockdays membaca hasil scheduler stockdays dari forecasting
// (GET /inventory-plan/stockdays). Filter sama dengan GetInventoryPlan.
// Respons API berupa JSON array langsung; belum ada data -> slice kosong.
func (c *Client) GetInventoryStockdays(ctx context.Context, opts GetInventoryPlanOptions) ([]InventoryStockdaysItem, error) {
	resp := []InventoryStockdaysItem{}
	if err := c.do(ctx, "GET", "/inventory-plan/stockdays"+opts.query(), nil, "", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
