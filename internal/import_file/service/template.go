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
	f.SetCellValue(masterSheetName, "F1", "Customer")
	f.SetCellValue(masterSheetName, "H1", "No")
	f.SetCellValue(masterSheetName, "I1", "Product")

	// set header style untuk master sheet
	for _, col := range []string{"B", "C", "E", "F", "H", "I"} {
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

	// 🔹 fetch customers dari database
	customers, err := s.repo.GetAllCustomers(ctx)
	if err != nil {
		return nil, err
	}

	row = 2
	for _, customer := range customers {
		f.SetCellValue(masterSheetName, "E"+fmt.Sprint(row), customer["no"])
		f.SetCellValue(masterSheetName, "F"+fmt.Sprint(row), customer["customer_name"])
		row++
	}

	// 🔹 fetch supplier items dari database
	items, err := s.repo.GetAllSupplierItems(ctx)
	if err != nil {
		return nil, err
	}

	row = 2
	for _, item := range items {
		f.SetCellValue(masterSheetName, "H"+fmt.Sprint(row), item["no"])
		f.SetCellValue(masterSheetName, "I"+fmt.Sprint(row), item["product_name"])
		row++
	}

	// set column width untuk master sheet
	f.SetColWidth(masterSheetName, "B", "I", 20)

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

func (s *importService) GenerateTemplateKanban(ctx context.Context) (*bytes.Buffer, error) {
	f := excelize.NewFile()

	templateSheet := "Template"
	masterSheet := "Master"

	f.SetSheetName("Sheet1", templateSheet)
	f.NewSheet(masterSheet)

	// =========================
	// Header Style
	// =========================
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	})

	// =========================
	// TEMPLATE SHEET
	// =========================

	headers := []string{
		"item_uniq_code",
		"kanban_qty",
		"min_stock",
		"max_stock",
		"status",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(templateSheet, cell, h)
		f.SetCellStyle(templateSheet, cell, cell, headerStyle)
	}

	// contoh data
	example := []interface{}{
		"EMA-001-LV2",
		100,
		20,
		200,
		"ACTIVE",
	}

	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(templateSheet, cell, v)
	}

	f.SetColWidth(templateSheet, "A", "E", 20)

	f.SetPanes(templateSheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
	})

	// =========================
	// MASTER SHEET
	// =========================

	masterHeaders := []string{
		"No",
		"Kanban Number",
		"Item Uniq Code",
		"Kanban Qty",
		"Min Stock",
		"Max Stock",
		"Status",
		"Created At",
	}

	for i, h := range masterHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(masterSheet, cell, h)
		f.SetCellStyle(masterSheet, cell, cell, headerStyle)
	}

	kanbans, err := s.repo.GetAllKanban(ctx)
	if err != nil {
		return nil, err
	}

	for i, k := range kanbans {
		row := i + 2

		f.SetCellValue(masterSheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(masterSheet, fmt.Sprintf("B%d", row), k.KanbanNumber)
		f.SetCellValue(masterSheet, fmt.Sprintf("C%d", row), k.ItemUniqCode)
		f.SetCellValue(masterSheet, fmt.Sprintf("D%d", row), k.KanbanQty)
		f.SetCellValue(masterSheet, fmt.Sprintf("E%d", row), k.MinStock)
		f.SetCellValue(masterSheet, fmt.Sprintf("F%d", row), k.MaxStock)
		f.SetCellValue(masterSheet, fmt.Sprintf("G%d", row), k.Status)
		f.SetCellValue(masterSheet, fmt.Sprintf("H%d", row), k.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	f.SetColWidth(masterSheet, "A", "H", 20)

	f.SetPanes(masterSheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
	})

	f.SetActiveSheet(0)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf, nil
}
