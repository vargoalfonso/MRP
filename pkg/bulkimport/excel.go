package bulkimport

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// SheetDef describes a single sheet to include in the error Excel file.
type SheetDef struct {
	Name    string
	Headers []string // column headers starting from col B (col A is reserved for error_field)
}

// GenerateErrorExcel builds an Excel workbook that contains only the failed rows.
// The first column (A) is always "error_field" and is filled with the error
// message for each row. Subsequent columns reproduce the original row data.
//
// errors must be pre-grouped: every RowError whose Sheet matches a SheetDef.Name
// is written into that sheet.
func GenerateErrorExcel(sheets []SheetDef, errors []RowError) (*excelize.File, error) {
	f := excelize.NewFile()

	// Track row counters per sheet (data starts at row 2, row 1 = header)
	rowCounters := make(map[string]int)

	// Create sheets and write headers
	for i, sd := range sheets {
		if i == 0 {
			f.SetSheetName("Sheet1", sd.Name)
		} else {
			f.NewSheet(sd.Name)
		}
		rowCounters[sd.Name] = 2 // data starts at row 2

		// Write header row
		if err := f.SetCellValue(sd.Name, "A1", "error_field"); err != nil {
			return nil, fmt.Errorf("write header A1 on %s: %w", sd.Name, err)
		}
		for colIdx, header := range sd.Headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+2, 1) // B, C, D …
			if err := f.SetCellValue(sd.Name, cell, header); err != nil {
				return nil, fmt.Errorf("write header %s on %s: %w", cell, sd.Name, err)
			}
		}
	}

	// Write error rows
	for _, e := range errors {
		rowNum := rowCounters[e.Sheet]
		if rowNum == 0 {
			continue // sheet not declared — skip
		}

		// Col A: error message
		cellA, _ := excelize.CoordinatesToCellName(1, rowNum)
		if err := f.SetCellValue(e.Sheet, cellA, e.Message); err != nil {
			return nil, fmt.Errorf("write error_field row %d on %s: %w", rowNum, e.Sheet, err)
		}

		// Cols B…: original data
		for colIdx, val := range e.RawData {
			cell, _ := excelize.CoordinatesToCellName(colIdx+2, rowNum)
			if err := f.SetCellValue(e.Sheet, cell, val); err != nil {
				return nil, fmt.Errorf("write data cell %s on %s: %w", cell, e.Sheet, err)
			}
		}

		rowCounters[e.Sheet] = rowNum + 1
	}

	return f, nil
}

// BuildBomTemplate creates the blank BOM import template Excel with the two
// standard sheets (Items, Routes) and their header rows.
func BuildBomTemplate() (*excelize.File, error) {
	f := excelize.NewFile()

	// ── Sheet: Items ──────────────────────────────────────────────────────────
	f.SetSheetName("Sheet1", "Items")

	itemHeaders := []string{
		"error_field",
		"bom_group", "row_type", "uniq_code", "parent_uniq_code",
		"part_name", "part_number", "uom", "level",
		"is_phantom", "status", "description",
		"material_grade", "form", "width_mm", "thickness_mm", "length_mm",
		"diameter_mm", "weight_kg",
	}
	for colIdx, h := range itemHeaders {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err := f.SetCellValue("Items", cell, h); err != nil {
			return nil, err
		}
	}

	// Sample success row (row 2)
	sampleItems := []string{
		"",                    // error_field
		"BOM-SAMPLE-001",      // bom_group
		"ROOT",                // row_type
		"SAMPLE-001",          // uniq_code
		"",                    // parent_uniq_code
		"Sample Assembly",     // part_name
		"SA-001",              // part_number
		"PCS",                 // uom
		"",                    // level
		"",                    // is_phantom
		"Active",              // status
		"Initial BOM version", // description
		"STKM550",             // material_grade
		"Plate",               // form
		"200",                 // width_mm
		"5",                   // thickness_mm
		"300",                 // length_mm
		"",                    // diameter_mm
		"",                    // weight_kg
	}
	for colIdx, v := range sampleItems {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		if err := f.SetCellValue("Items", cell, v); err != nil {
			return nil, err
		}
	}

	// Sample CHILD row (row 3)
	sampleChild := []string{
		"",               // error_field
		"BOM-SAMPLE-001", // bom_group
		"CHILD",          // row_type
		"SAMPLE-001-A",   // uniq_code
		"SAMPLE-001",     // parent_uniq_code
		"Sub Component",  // part_name
		"SC-001-A",       // part_number
		"PCS",            // uom
		"1",              // level
		"FALSE",          // is_phantom
		"Active",         // status
		"",               // description
		"SPCC",           // material_grade
		"Plate",          // form
		"100",            // width_mm
		"2",              // thickness_mm
		"150",            // length_mm
		"",               // diameter_mm
		"",               // weight_kg
	}
	for colIdx, v := range sampleChild {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 3)
		if err := f.SetCellValue("Items", cell, v); err != nil {
			return nil, err
		}
	}

	// ── Sheet: Routes ─────────────────────────────────────────────────────────
	f.NewSheet("Routes")

	routeHeaders := []string{
		"error_field",
		"uniq_code", "op_seq", "process_id", "machine_id",
		"cycle_time_sec", "setup_time_min", "machine_stroke", "tooling_ref",
	}
	for colIdx, h := range routeHeaders {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err := f.SetCellValue("Routes", cell, h); err != nil {
			return nil, err
		}
	}

	// Sample route row (row 2)
	sampleRoutes := [][]string{
		{"", "SAMPLE-001", "10", "1", "1", "28", "12", "220 spm", "Dies"},
		{"", "SAMPLE-001", "20", "6", "5", "50", "8", "", "JIG"},
		{"", "SAMPLE-001-A", "10", "1", "2", "15", "10", "180 spm", "Dies"},
	}
	for rowOffset, row := range sampleRoutes {
		for colIdx, v := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowOffset+2)
			if err := f.SetCellValue("Routes", cell, v); err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}
