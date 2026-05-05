package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/import_file/helper"
	"github.com/ganasa18/go-template/internal/import_file/models"
	"github.com/xuri/excelize/v2"
)

func cleanNumber(val string) string {
	val = strings.TrimSpace(val)

	// hapus spasi
	val = strings.ReplaceAll(val, " ", "")

	// handle format indo
	if strings.Contains(val, ",") {
		val = strings.ReplaceAll(val, ".", "")
		val = strings.ReplaceAll(val, ",", ".")
	}

	// hapus selain angka & titik
	re := regexp.MustCompile(`[^\d\.]`)
	val = re.ReplaceAllString(val, "")

	return val
}

func (s *importService) ParsingPRL(ctx context.Context, filePath string) ([]models.ImportDataRequest, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file kosong")
	}

	var result []models.ImportDataRequest

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		item := models.ImportDataRequest{
			CustomerName:   helper.SafeGet(row, 1), // skip kolom "no"
			UniqCode:       helper.SafeGet(row, 2),
			ProductModel:   helper.SafeGet(row, 3),
			PartName:       helper.SafeGet(row, 4),
			PartNumber:     helper.SafeGet(row, 5),
			ForecastPeriod: helper.SafeGet(row, 6),
		}

		qtyStr := helper.SafeGet(row, 7)

		qty, err := strconv.ParseFloat(cleanNumber(qtyStr), 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: quantity tidak valid (%s)", i+2, qtyStr)
		}

		item.Quantity = qty

		if item.CustomerName == "" {
			continue
		}

		result = append(result, item)
	}

	return result, nil
}
