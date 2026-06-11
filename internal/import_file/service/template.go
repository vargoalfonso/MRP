package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"
)

func (s *importService) GenerateTemplatePrls(ctx context.Context) (*bytes.Buffer, error) {
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

	// 🔹 add master sheet
	masterSheetName := "Master"
	f.NewSheet(masterSheetName)

	// set data for master sheet
	// Kolom B: No, Kolom C: Supplier, Kolom E: No, Kolom F: Product
	f.SetCellValue(masterSheetName, "B1", "No")
	f.SetCellValue(masterSheetName, "C1", "Supplier")
	f.SetCellValue(masterSheetName, "E1", "No")
	f.SetCellValue(masterSheetName, "F1", "Product")

	// set header style untuk master sheet
	for _, col := range []string{"B", "C", "E", "F"} {
		cell := col + "1"
		f.SetCellStyle(masterSheetName, cell, cell, style)
	}

	// 🔹 fetch suppliers dari database
	suppliers, err := s.repo.GetAllSuppliers(ctx)
	if err != nil {
		return nil, err
	}

	row := 2
	for _, supplier := range suppliers {
		f.SetCellValue(masterSheetName, "B"+fmt.Sprint(row), supplier["no"])
		f.SetCellValue(masterSheetName, "C"+fmt.Sprint(row), supplier["supplier_name"])
		row++
	}

	// 🔹 fetch supplier items dari database
	items, err := s.repo.GetAllSupplierItems(ctx)
	if err != nil {
		return nil, err
	}

	row = 2
	for _, item := range items {
		f.SetCellValue(masterSheetName, "E"+fmt.Sprint(row), item["no"])
		f.SetCellValue(masterSheetName, "F"+fmt.Sprint(row), item["product_name"])
		row++
	}

	// set column width untuk master sheet
	f.SetColWidth(masterSheetName, "B", "F", 20)

	// freeze header pada master sheet
	f.SetPanes(masterSheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "B2",
	})

	// 🔹 convert ke buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf, nil
}
