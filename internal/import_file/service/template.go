package service

import (
	"bytes"

	"github.com/xuri/excelize/v2"
)

func (s *importService) GenerateTemplatePrls() (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheetName := "Template"

	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"no",
		"customer_name",
		"uniq_code",
		"product_model",
		"part_name",
		"part_number",
		"forecast_period",
		"quantity",
	}

	// 🔹 styling header
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	})

	// set header
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, style)
	}

	// 🔹 set column width biar enak dilihat
	f.SetColWidth(sheetName, "A", "H", 20)

	// 🔹 kasih contoh data (optional tapi recommended)
	example := []interface{}{
		1,
		"PT Customer Beta",
		"EMA-001-LV2",
		"LV2",
		"Engine Mount Assembly",
		"EMA-001-LV2",
		"Mei 2026",
		100,
	}

	for i, val := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, val)
	}

	// 🔹 freeze header (biar enak scroll)
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "A2",
	})

	// 🔹 convert ke buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf, nil
}
