package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParsingWO(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wo-template.xlsx")

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"wo_type", "reference_wo", "created_date", "target_date", "notes", "item_uniq_code", "quantity", "uom", "process_name", "kanban_qty"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	rows := [][]interface{}{
		{"New", "", "2026-04-14", "2026-04-20", "WO harian shift 1", "EMA-LV7-001", 6, "Piece", "Bending", 20},
		{"New", "", "2026-04-15", "2026-04-21", "WO harian shift 2", "EMA-LV7-002", 3, "Piece", "Cutting", 15},
	}
	for rIdx, row := range rows {
		for cIdx, v := range row {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rIdx+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("save workbook: %v", err)
	}

	svc := New(nil)
	reqs, err := svc.ParsingWO(context.Background(), filePath)
	if err != nil {
		t.Fatalf("parsing WO should succeed: %v", err)
	}

	if len(reqs) != 2 {
		t.Fatalf("expected 2 work order requests, got %d", len(reqs))
	}
	if reqs[0].WOType != "New" {
		t.Fatalf("expected first WOType New, got %s", reqs[0].WOType)
	}
	if len(reqs[0].Items) != 1 {
		t.Fatalf("expected first request to contain 1 item, got %d", len(reqs[0].Items))
	}
	if reqs[0].Items[0].ItemUniqCode != "EMA-LV7-001" {
		t.Fatalf("expected first item uniq code EMA-LV7-001, got %s", reqs[0].Items[0].ItemUniqCode)
	}
}
