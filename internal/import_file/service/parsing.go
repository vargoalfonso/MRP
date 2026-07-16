package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ganasa18/go-template/internal/import_file/helper"
	"github.com/ganasa18/go-template/internal/import_file/models"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
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
			CustomerName:   helper.SafeGet(row, 0),
			UniqCode:       helper.SafeGet(row, 1),
			ProductModel:   helper.SafeGet(row, 2),
			PartName:       helper.SafeGet(row, 3),
			PartNumber:     helper.SafeGet(row, 4),
			ForecastPeriod: helper.SafeGet(row, 5),
		}

		qtyStr := helper.SafeGet(row, 6)

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

func (s *importService) ParsingKanban(ctx context.Context, filePath string) ([]models.CreateKanbanParameterRequest, error) {
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

	// mapping header
	headers := make(map[string]int)
	for i, h := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var result []models.CreateKanbanParameterRequest

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// skip row kosong
		if len(row) == 0 || strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}

		itemCode := strings.TrimSpace(helper.SafeGet(row, headers["item_uniq_code"]))
		if itemCode == "" {
			continue
		}

		item := models.CreateKanbanParameterRequest{
			ItemUniqCode: itemCode,
			Status:       strings.ToUpper(strings.TrimSpace(helper.SafeGet(row, headers["status"]))),
		}

		if item.Status == "" {
			item.Status = "ACTIVE"
		}

		// kanban_qty
		if val := strings.TrimSpace(helper.SafeGet(row, headers["kanban_qty"])); val != "" {
			qty, err := strconv.Atoi(cleanNumber(val))
			if err != nil {
				return nil, fmt.Errorf("row %d: kanban_qty tidak valid", i+1)
			}
			item.KanbanQty = qty
		}

		// min_stock
		if val := strings.TrimSpace(helper.SafeGet(row, headers["min_stock"])); val != "" {
			minStock, err := strconv.Atoi(cleanNumber(val))
			if err != nil {
				return nil, fmt.Errorf("row %d: min_stock tidak valid", i+1)
			}
			item.MinStock = minStock
		}

		// max_stock
		if val := strings.TrimSpace(helper.SafeGet(row, headers["max_stock"])); val != "" {
			maxStock, err := strconv.Atoi(cleanNumber(val))
			if err != nil {
				return nil, fmt.Errorf("row %d: max_stock tidak valid", i+1)
			}
			item.MaxStock = maxStock
		}

		result = append(result, item)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("tidak ada data yang dapat diimport")
	}

	return result, nil
}

func (s *importService) ParsingWO(ctx context.Context, filePath string) ([]woModels.CreateWorkOrderRequest, error) {
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

	headers := make(map[string]int)
	for i, h := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var result []woModels.CreateWorkOrderRequest

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}

		itemCode := strings.TrimSpace(helper.SafeGet(row, headers["item_uniq_code"]))
		if itemCode == "" {
			continue
		}

		req := woModels.CreateWorkOrderRequest{
			WOType: strings.TrimSpace(helper.SafeGet(row, headers["wo_type"])),
		}
		if req.WOType == "" {
			req.WOType = "New"
		}
		if referenceWO := strings.TrimSpace(helper.SafeGet(row, headers["reference_wo"])); referenceWO != "" {
			req.ReferenceWO = &referenceWO
		}
		if createdDate := strings.TrimSpace(helper.SafeGet(row, headers["created_date"])); createdDate != "" {
			req.CreatedDate = &createdDate
		}
		if targetDate := strings.TrimSpace(helper.SafeGet(row, headers["target_date"])); targetDate != "" {
			req.TargetDate = &targetDate
		}
		if notes := strings.TrimSpace(helper.SafeGet(row, headers["notes"])); notes != "" {
			req.Notes = &notes
		}

		item := woModels.CreateWorkOrderItem{ItemUniqCode: strings.TrimSpace(itemCode)}

		quantityStr := strings.TrimSpace(helper.SafeGet(row, headers["quantity"]))
		if quantityStr != "" {
			qty, err := strconv.ParseFloat(cleanNumber(quantityStr), 64)
			if err != nil {
				return nil, fmt.Errorf("row %d: quantity tidak valid (%s)", i+1, quantityStr)
			}
			item.Quantity = qty
		}

		if uom := strings.TrimSpace(helper.SafeGet(row, headers["uom"])); uom != "" {
			item.UOM = &uom
		}
		if process := strings.TrimSpace(helper.SafeGet(row, headers["process_name"])); process != "" {
			item.ProcessName = &process
		}
		if kanbanQty := strings.TrimSpace(helper.SafeGet(row, headers["kanban_qty"])); kanbanQty != "" {
			parsed, err := strconv.Atoi(cleanNumber(kanbanQty))
			if err != nil {
				return nil, fmt.Errorf("row %d: kanban_qty tidak valid", i+1)
			}
			item.KanbanQty = parsed
		}

		req.Items = []woModels.CreateWorkOrderItem{item}
		result = append(result, req)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("tidak ada item dalam file")
	}

	return result, nil
}
