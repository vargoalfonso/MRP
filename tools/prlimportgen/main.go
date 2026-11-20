package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/prlimportgen <customer_code> <output>")
		os.Exit(1)
	}

	customerCode := os.Args[1]
	outputPath := os.Args[2]

	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	headers := []string{"customer_code", "uniq_code", "forecast_period", "quantity"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		_ = file.SetCellValue(sheet, cell, header)
	}

	rows := [][]interface{}{
		{customerCode, "LV7-001", "2026-Q1", 1200},
		{customerCode, "LV7-002", "2026-Q2", 900},
	}

	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			_ = file.SetCellValue(sheet, cell, value)
		}
	}

	if err := file.SaveAs(outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
