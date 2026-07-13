package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/import_file/models"
	"github.com/ganasa18/go-template/internal/import_file/repository"
	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type ImportService interface {
	GenerateTemplatePrls(ctx context.Context) (*bytes.Buffer, error)
	GenerateTemplateKanban(ctx context.Context) (*bytes.Buffer, error)
	GenerateTemplateWO(ctx context.Context) (*bytes.Buffer, error)

	ParsingPRL(ctx context.Context, filePath string) ([]models.ImportDataRequest, error)
	ParsingKanban(ctx context.Context, filePath string) ([]models.CreateKanbanParameterRequest, error)
	ParsingWO(ctx context.Context, filePath string) ([]woModels.CreateWorkOrderRequest, error)

	BulkInsertPRL(ctx context.Context, data []models.ImportDataRequest, filePath string) (*models.BulkInsertResponse, error)
	BulkInsertKanban(ctx context.Context, data []models.CreateKanbanParameterRequest, filePath string) (*models.BulkInsertResponse, error)
}

type importService struct {
	repo repository.ImportRepository
}

func New(repo repository.ImportRepository) ImportService {
	return &importService{repo: repo}
}

func (s *importService) BulkInsertPRL(ctx context.Context, data []models.ImportDataRequest, filePath string) (*models.BulkInsertResponse, error) {
	var failedRows []models.FailedImport
	success := 0

	customerCache := make(map[string]*models.Customer)
	itemCache := make(map[string]*models.Item)

	// CustomerID -> PRL ID
	customerPRLMap := make(map[string]string)

	year := time.Now().Format("2006")
	baseNumber, err := s.repo.GetMaxPRLNumber(ctx, year)
	if err != nil {
		return nil, err
	}

	prlCounter := baseNumber

	for i, item := range data {
		customerName := strings.TrimSpace(item.CustomerName)
		uniqCode := strings.TrimSpace(item.UniqCode)

		createFailedRow := func(msg string) models.FailedImport {
			return models.FailedImport{
				RowNumber:      i + 2,
				CustomerName:   customerName,
				UniqCode:       uniqCode,
				ProductModel:   item.ProductModel,
				PartName:       item.PartName,
				PartNumber:     item.PartNumber,
				ForecastPeriod: item.ForecastPeriod,
				Quantity:       int(item.Quantity),
				ErrorMessage:   msg,
			}
		}

		if customerName == "" || uniqCode == "" {
			failedRows = append(failedRows, createFailedRow("customer / uniq_code kosong"))
			continue
		}

		// CUSTOMER CACHE
		customer, ok := customerCache[customerName]
		if !ok {
			c, err := s.repo.GetLatestCustomerByName(ctx, customerName)
			if err != nil || c == nil {
				failedRows = append(failedRows, createFailedRow("customer tidak ditemukan"))
				continue
			}
			customerCache[customerName] = c
			customer = c
		}

		// ITEM CACHE
		itm, ok := itemCache[uniqCode]
		if !ok {
			iData, err := s.repo.GetItemByUniqCode(ctx, uniqCode)
			if err != nil || iData == nil {
				failedRows = append(failedRows, createFailedRow("item tidak ditemukan"))
				continue
			}
			itemCache[uniqCode] = iData
			itm = iData
		}

		// ==========================================
		// SATU CUSTOMER = SATU PRL ID
		// ==========================================
		prlID, exists := customerPRLMap[customer.CustomerID]
		if !exists {
			prlCounter++

			prlID = fmt.Sprintf("PRL-%s-%s-%03d", year, customer.CustomerID, prlCounter)

			customerPRLMap[customer.CustomerID] = prlID
		}

		// INSERT
		err = s.repo.InsertPRL(ctx, &models.PRL{
			UUID:           uuid.New().String(),
			PRLID:          prlID,
			CustomerUUID:   customer.UUID,
			CustomerCode:   customer.CustomerID,
			CustomerName:   customer.CustomerName,
			UniqBomUUID:    &itm.UUID,
			UniqCode:       uniqCode,
			ProductModel:   item.ProductModel,
			PartName:       item.PartName,
			PartNumber:     item.PartNumber,
			ForecastPeriod: item.ForecastPeriod,
			Quantity:       item.Quantity,
			Status:         "pending",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})

		if err != nil {
			failedRows = append(failedRows, createFailedRow(err.Error()))
			continue
		}

		success++
	}

	if len(failedRows) > 0 {
		err := s.appendErrorToExcel(filePath, failedRows)
		if err != nil {
			return nil, fmt.Errorf("gagal menulis error ke excel: %v", err)
		}
	}

	return &models.BulkInsertResponse{
		Success:    success,
		Failed:     len(failedRows),
		FailedFile: filePath,
	}, nil
}

func (s *importService) appendErrorToExcel(filePath string, failedRows []models.FailedImport) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0) // Ambil sheet pertama

	// 1. Tambah Header "Keterangan Error" di kolom terakhir (Misal kolom K atau L)
	// Sesuaikan indeks kolomnya, di sini saya asumsikan kolom terakhir adalah 'I'
	errCol := "I"
	f.SetCellValue(sheetName, errCol+"1", "Keterangan Error")

	// 2. Isi pesan error berdasarkan barisnya
	for _, row := range failedRows {
		cell := fmt.Sprintf("%s%d", errCol, row.RowNumber)
		f.SetCellValue(sheetName, cell, row.ErrorMessage)
	}

	// 3. Simpan perubahan ke file yang sama
	if err := f.Save(); err != nil {
		return err
	}

	return nil
}

func (s *importService) BulkInsertKanban(ctx context.Context, data []models.CreateKanbanParameterRequest, filePath string) (*models.BulkInsertResponse, error) {

	var failedRows []models.FailedImportKanban
	success := 0

	itemCache := make(map[string]*models.Item)

	totalKanban, err := s.repo.CountKanban(ctx)
	if err != nil {
		return nil, err
	}

	counter := totalKanban

	for i, item := range data {

		uniqCode := strings.TrimSpace(item.ItemUniqCode)

		createFailedRow := func(msg string) models.FailedImportKanban {
			return models.FailedImportKanban{
				RowNumber:    i + 2,
				ItemUniqCode: uniqCode,
				KanbanQty:    item.KanbanQty,
				MinStock:     item.MinStock,
				MaxStock:     item.MaxStock,
				Status:       item.Status,
				ErrorMessage: msg,
			}
		}

		if uniqCode == "" {
			failedRows = append(failedRows, createFailedRow("item_uniq_code kosong"))
			continue
		}

		// cek item master
		if _, ok := itemCache[uniqCode]; !ok {
			itm, err := s.repo.GetItemByUniqCode(ctx, uniqCode)
			if err != nil || itm == nil {
				failedRows = append(failedRows, createFailedRow("item tidak ditemukan"))
				continue
			}

			itemCache[uniqCode] = itm
		}

		// cek duplicate
		exist, err := s.repo.IsKanbanExist(ctx, uniqCode)
		if err != nil {
			failedRows = append(failedRows, createFailedRow(err.Error()))
			continue
		}

		if exist {
			failedRows = append(failedRows, createFailedRow("kanban sudah ada"))
			continue
		}

		counter++

		kanbanNumber := fmt.Sprintf(
			"KBN-%d-%04d",
			time.Now().Year(),
			counter,
		)

		kanban := models.KanbanParameter{
			KanbanNumber: kanbanNumber,
			ItemUniqCode: uniqCode,
			KanbanQty:    item.KanbanQty,
			MinStock:     item.MinStock,
			MaxStock:     item.MaxStock,
			Status:       item.Status,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err = s.repo.CreateKanban(ctx, &kanban)
		if err != nil {
			failedRows = append(failedRows, createFailedRow(err.Error()))
			continue
		}

		success++
	}

	if len(failedRows) > 0 {
		if err := s.appendKanbanErrorToExcel(filePath, failedRows); err != nil {
			return nil, fmt.Errorf("gagal menulis error ke excel: %v", err)
		}
	}

	return &models.BulkInsertResponse{
		Success:    success,
		Failed:     len(failedRows),
		FailedFile: filePath,
	}, nil
}

func (s *importService) appendKanbanErrorToExcel(filePath string, failedRows []models.FailedImportKanban) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	// Template Kanban punya 5 kolom
	// A=item_uniq_code
	// B=kanban_qty
	// C=min_stock
	// D=max_stock
	// E=status
	// F=Keterangan Error

	errCol := "F"

	f.SetCellValue(sheet, errCol+"1", "Keterangan Error")

	for _, row := range failedRows {
		cell := fmt.Sprintf("%s%d", errCol, row.RowNumber)
		f.SetCellValue(sheet, cell, row.ErrorMessage)
	}

	return f.Save()
}
