package service

import (
	"errors"
	"strings"

	"github.com/ganasa18/go-template/internal/safety_stock_parameter/constant"
)

func normalizeCalculationType(calcType string) string {
	raw := strings.ToLower(strings.TrimSpace(calcType))
	switch {
	case raw == "days" || strings.Contains(raw, "* days"):
		return string(constant.CalcDays)
	case raw == "percentage" || strings.Contains(raw, "percentage"):
		return string(constant.CalcPercentage)
	case raw == "forecast" || strings.Contains(raw, "forecast"):
		return string(constant.CalcForecast)
	default:
		return strings.TrimSpace(calcType)
	}
}

func validateCalculationType(calcType string) error {
	switch normalizeCalculationType(calcType) {
	case string(constant.CalcDays),
		string(constant.CalcPercentage),
		string(constant.CalcForecast):
		return nil
	default:
		return errors.New("invalid calculation type")
	}
}
