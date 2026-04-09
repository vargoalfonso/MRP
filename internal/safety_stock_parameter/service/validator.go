package service

import (
	"errors"

	"github.com/ganasa18/go-template/internal/safety_stock_parameter/constant"
)

func validateCalculationType(calcType string) error {
	switch calcType {
	case string(constant.CalcDays),
		string(constant.CalcPercentage),
		string(constant.CalcForecast):
		return nil
	default:
		return errors.New("invalid calculation type")
	}
}
