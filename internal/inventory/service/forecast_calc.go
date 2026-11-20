package service

// Rumus pure untuk safety stock / stockdays bertipe "forecast".
//
// Aturan mengikuti service forecasting (dev-mrp-forecasting, 19 Nov 2026) dan
// recalculateRMStatus di repository, supaya angka yang ditulis scheduler ke DB
// dan angka yang dihitung ERP saat itu juga tidak saling bertentangan:
//   - safety_stock_qty dari forecast = forecast harian, TANPA Constanta.
//   - threshold = safety stock (10 bila safety stock <= 0). cycle_pengiriman
//     tidak dipakai.
//   - buy bila stock < threshold (ketat), bukan <=.

// forecastDefaultThreshold adalah threshold bila safety stock forecast <= 0
// (sama dengan fallback safetyStock = 10 di recalculateRMStatus).
const forecastDefaultThreshold = 10.0

// forecastThreshold mengembalikan threshold untuk safety stock forecast.
func forecastThreshold(safetyStockQty float64) float64 {
	if safetyStockQty > 0 {
		return safetyStockQty
	}
	return forecastDefaultThreshold
}

// thresholdFor memilih rumus threshold. forecastRule true hanya bila
// calculation_type safety stock = forecast DAN nilainya berhasil diambil dari
// forecasting; bila gagal diambil, rumus lama (safety stock x cycle) tetap
// dipakai agar tidak muncul threshold 10 yang tidak berasal dari data.
func thresholdFor(forecastRule bool, safetyStockQty float64, cyclePengiriman int) float64 {
	if forecastRule {
		return forecastThreshold(safetyStockQty)
	}
	return safetyStockQty * float64(cyclePengiriman)
}

// shouldBuy: forecast memakai stock < threshold, tipe lain tetap stock <= threshold.
func shouldBuy(forecastRule bool, stockQty, threshold float64) bool {
	if forecastRule {
		return stockQty < threshold
	}
	return stockQty <= threshold
}

// stockStatus mengklasifikasikan stok terhadap dasar safety stock.
// base <= 0 berarti tidak ada safety stock -> "normal".
func stockStatus(stockQty, base, highRatio float64) string {
	switch {
	case base > 0 && stockQty < base:
		return "low_on_stock"
	case base > 0 && stockQty > base*highRatio:
		return "overstock"
	default:
		return "normal"
	}
}
