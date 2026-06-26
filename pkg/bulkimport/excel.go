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
			if val == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(colIdx+2, rowNum)
			if err := f.SetCellValue(e.Sheet, cell, val); err != nil {
				return nil, fmt.Errorf("write data cell %s on %s: %w", cell, e.Sheet, err)
			}
		}

		rowCounters[e.Sheet] = rowNum + 1
	}

	return f, nil
}

// SupplierRef is a lightweight supplier record used to populate the Suppliers
// reference sheet in the BOM import template.
type SupplierRef struct{ Code, Name string }

// GenerateBomErrorExcel builds the BOM import error file using the official
// template as a base (headers + sample rows intact) so users can see the
// expected format while correcting their data. Failed rows are appended
// after the last sample row.
func GenerateBomErrorExcel(errors []RowError) (*excelize.File, error) {
	f, err := BuildBomTemplate(nil)
	if err != nil {
		return nil, fmt.Errorf("build base template: %w", err)
	}

	// Detect how many rows the template already wrote so we don't hardcode it.
	existing, err := f.GetRows("Items")
	if err != nil {
		return nil, fmt.Errorf("read template rows: %w", err)
	}
	nextRow := len(existing) + 1

	for _, e := range errors {
		cellA, _ := excelize.CoordinatesToCellName(1, nextRow)
		if err := f.SetCellValue("Items", cellA, e.Message); err != nil {
			return nil, fmt.Errorf("write error_field row %d: %w", nextRow, err)
		}
		for colIdx, val := range e.RawData {
			if val == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(colIdx+2, nextRow)
			if err := f.SetCellValue("Items", cell, val); err != nil {
				return nil, fmt.Errorf("write data cell %s row %d: %w", cell, nextRow, err)
			}
		}
		nextRow++
	}

	return f, nil
}

const bomTemplateMaxRoutes = 7

type bomSampleRoute struct {
	opSeq, processCode, machineNum string
	cycleTimeSec, setupTimeMin     string
	machineStroke, toolingRef      string
}

type bomSampleRow struct {
	bomGroup, rowType, uniqCode, parentUniq string
	partName, partNumber, model, uom        string
	level, qtyPerUniq                       string
	status, description                     string
	materialGrade, grade, form              string
	widthMM, thicknessMM, lengthMM          string
	diameterMM, weightKG                    string
	supplierCode, customerCycle             string
	typeMaterial                            string
	routes                                  []bomSampleRoute
}

func (r bomSampleRow) toSlice(totalCols int) []string {
	vals := []string{
		"", // error_field (col A, always blank in template)
		r.bomGroup, r.rowType, r.uniqCode, r.parentUniq,
		r.partName, r.partNumber, r.model, r.uom,
		r.level, r.qtyPerUniq,
		r.status, r.description,
		r.materialGrade, r.grade, r.form,
		r.widthMM, r.thicknessMM, r.lengthMM,
		r.diameterMM, r.weightKG,
		r.supplierCode, r.customerCycle, r.typeMaterial,
	}
	for _, rt := range r.routes {
		vals = append(vals,
			rt.opSeq, rt.processCode, rt.machineNum,
			rt.cycleTimeSec, rt.setupTimeMin,
			rt.machineStroke, rt.toolingRef,
		)
	}
	for len(vals) < totalCols {
		vals = append(vals, "")
	}
	return vals
}

// BuildBomTemplate creates the BOM import template Excel.
// Sheet 1 "Items" has headers + sample rows.
// Sheet 2 "Suppliers" lists all active suppliers for reference (skipped when suppliers is nil).
// Route fields are inlined as numbered columns: op_seq_1 … tooling_ref_7.
func BuildBomTemplate(suppliers []SupplierRef) (*excelize.File, error) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Items")

	// Build header row
	headers := []string{
		"error_field",
		"bom_group", "row_type", "uniq_code", "parent_uniq_code",
		"part_name", "part_number", "model", "uom", "level",
		"qty_per_uniq",
		"status", "description",
		"material_grade", "grade", "form", "width_mm", "thickness_mm", "length_mm",
		"diameter_mm", "weight_kg", "supplier_code", "customer_cycle", "type_material",
	}
	for n := 1; n <= bomTemplateMaxRoutes; n++ {
		s := fmt.Sprintf("%d", n)
		headers = append(headers,
			"op_seq_"+s, "process_code_"+s, "machine_number_"+s,
			"cycle_time_sec_"+s, "setup_time_min_"+s, "machine_stroke_"+s, "tooling_ref_"+s,
		)
	}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err := f.SetCellValue("Items", cell, h); err != nil {
			return nil, err
		}
	}

	// helper: write a row, skipping empty cells
	writeRow := func(sheet string, rowNum int, vals []string) error {
		for colIdx, v := range vals {
			if v == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return err
			}
		}
		return nil
	}

	const supplierCode = "SUP-2024-001"

	samples := []bomSampleRow{
		// ROOT — Engine Mount Assembly (4 routes, all fully filled)
		{
			bomGroup: "EMA-LV7-001", rowType: "ROOT",
			uniqCode: "EMA-LV7-001",
			partName: "Engine Mount Assembly", partNumber: "EMA-001-LV7", model: "LV7", uom: "PCS",
			status: "Active", description: "Engine mount sub-assembly LV7 model",
			materialGrade: "STKM550", grade: "Grade-A", form: "Plate",
			widthMM: "200", thicknessMM: "5", lengthMM: "300", weightKG: "2.34",
			supplierCode: supplierCode, customerCycle: "daily",
			routes: []bomSampleRoute{
				{opSeq: "10", processCode: "STAMP", machineNum: "PM-A1-001", cycleTimeSec: "45", setupTimeMin: "10", machineStroke: "220 spm", toolingRef: "Dies"},
				{opSeq: "20", processCode: "WELD", machineNum: "M-WELD-01", cycleTimeSec: "60", setupTimeMin: "15", machineStroke: "200 spm", toolingRef: "JIG"},
				{opSeq: "30", processCode: "ASSY", machineNum: "M-ASSY-01", cycleTimeSec: "30", setupTimeMin: "5", machineStroke: "180 spm", toolingRef: "CF"},
				{opSeq: "40", processCode: "INSP", cycleTimeSec: "20", setupTimeMin: "5", toolingRef: "CF"},
			},
		},
		// CHILD L1-A — Main Bracket (3 routes, all fully filled)
		{
			bomGroup: "EMA-LV7-001", rowType: "CHILD",
			uniqCode: "MB-LV7-001-A", parentUniq: "EMA-LV7-001",
			partName: "Main Bracket", partNumber: "MB-001-LV7", model: "LV7", uom: "PCS",
			level: "1", qtyPerUniq: "1", status: "Active",
			materialGrade: "STKM550", form: "Plate",
			widthMM: "150", thicknessMM: "4", lengthMM: "250", weightKG: "1.5",
			supplierCode: supplierCode, typeMaterial: "raw",
			routes: []bomSampleRoute{
				{opSeq: "10", processCode: "STAMP", machineNum: "M-STAMP-02", cycleTimeSec: "30", setupTimeMin: "10", machineStroke: "180 spm", toolingRef: "Dies"},
				{opSeq: "20", processCode: "BEND", machineNum: "M-BEND-01", cycleTimeSec: "25", setupTimeMin: "8", machineStroke: "150 spm", toolingRef: "Dies"},
				{opSeq: "30", processCode: "PIERCE", cycleTimeSec: "20", setupTimeMin: "5", toolingRef: "CF"},
			},
		},
		// CHILD L2-A1 — Steel Sheet 4mm (2 routes, all fully filled)
		{
			bomGroup: "EMA-LV7-001", rowType: "CHILD",
			uniqCode: "SH-LV7-001-A1", parentUniq: "MB-LV7-001-A",
			partName: "Steel Sheet 4mm", partNumber: "SS-001-LV7", uom: "KG",
			level: "2", qtyPerUniq: "1.5", status: "Active",
			materialGrade: "SS400", form: "Plate",
			widthMM: "150", thicknessMM: "4", typeMaterial: "raw",
			routes: []bomSampleRoute{
				{opSeq: "10", processCode: "STAMP", machineNum: "PM-A2-001", cycleTimeSec: "15", setupTimeMin: "5", machineStroke: "160 spm", toolingRef: "Dies"},
				{opSeq: "20", processCode: "TRIM", cycleTimeSec: "10", setupTimeMin: "3", toolingRef: "CF"},
			},
		},
		// CHILD L1-B — Rubber Insulator (no routes)
		{
			bomGroup: "EMA-LV7-001", rowType: "CHILD",
			uniqCode: "RI-LV7-001-B", parentUniq: "EMA-LV7-001",
			partName: "Rubber Insulator", partNumber: "RI-002-LV7", model: "LV7", uom: "PCS",
			level: "1", qtyPerUniq: "2", status: "Active",
			materialGrade: "SS400", form: "Plate",
			widthMM: "150", thicknessMM: "4", weightKG: "1.5",
			supplierCode: supplierCode, typeMaterial: "subcon",
		},
		// CHILD L1-C — Bolt M10 x 30 (no routes)
		{
			bomGroup: "EMA-LV7-001", rowType: "CHILD",
			uniqCode: "BLT-LV7-001-C", parentUniq: "EMA-LV7-001",
			partName: "Bolt M10 x 30", partNumber: "BLT-003-LV7", model: "LV7", uom: "PCS",
			level: "1", qtyPerUniq: "4", status: "Active",
			materialGrade: "SS400", form: "Plate",
			widthMM: "150", thicknessMM: "4", weightKG: "1.5",
			supplierCode: supplierCode, typeMaterial: "indirect",
		},
	}

	for i, s := range samples {
		if err := writeRow("Items", i+2, s.toSlice(len(headers))); err != nil {
			return nil, err
		}
	}

	// Sheet 2 — Suppliers reference list
	if len(suppliers) > 0 {
		f.NewSheet("Suppliers")
		_ = f.SetCellValue("Suppliers", "A1", "supplier_code")
		_ = f.SetCellValue("Suppliers", "B1", "supplier_name")
		for i, sp := range suppliers {
			rowNum := i + 2
			_ = f.SetCellValue("Suppliers", fmt.Sprintf("A%d", rowNum), sp.Code)
			_ = f.SetCellValue("Suppliers", fmt.Sprintf("B%d", rowNum), sp.Name)
		}
	}

	return f, nil
}
