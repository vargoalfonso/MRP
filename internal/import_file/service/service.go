package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ganasa18/go-template/internal/import_file/helper"
	"github.com/ganasa18/go-template/internal/import_file/models"
	"github.com/ganasa18/go-template/internal/import_file/repository"
	"github.com/xuri/excelize/v2"
)

type ImportService interface {
	ImportExcel(ctx context.Context, filePath string) ([]models.ImportData, error)
}

type importService struct {
	repo repository.ImportRepository
}

func New(repo repository.ImportRepository) ImportService {
	return &importService{repo: repo}
}

func (s *importService) ImportExcel(ctx context.Context, filePath string) ([]models.ImportData, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed open excel: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("sheet tidak ditemukan")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed get rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("data kosong")
	}

	headers := rows[0]

	var result []models.ImportData

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		uniq := helper.SafeGet(row, 0)
		if uniq == "" {
			continue
		}

		for colIdx := 1; colIdx < len(headers); colIdx++ {
			valStr := helper.SafeGet(row, colIdx)
			if valStr == "" {
				continue
			}

			value, err := strconv.ParseFloat(helper.CleanNumber(valStr), 64)
			if err != nil {
				continue
			}

			period := helper.ParsePeriod(headers[colIdx])
			if period == "" {
				continue
			}

			result = append(result, models.ImportData{
				Uniq:   uniq,
				Period: period,
				Value:  value,
			})
		}
	}

	return result, nil
}
