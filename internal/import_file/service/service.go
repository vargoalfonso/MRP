package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/import_file/models"
	"github.com/ganasa18/go-template/internal/import_file/repository"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type ImportService interface {
	GenerateTemplatePrls() (*bytes.Buffer, error)

	ParsingPRL(ctx context.Context, filePath string) ([]models.ImportDataRequest, error)

	BulkInsertPRL(ctx context.Context, data []models.ImportDataRequest, filePath string) (*models.BulkInsertResponse, error)
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

	year := time.Now().Format("2006")
	baseNumber, err := s.repo.GetMaxPRLNumber(ctx, year)
	if err != nil {
		return nil, err
	}

	for i, item := range data {
		customerName := strings.TrimSpace(item.CustomerName)
		uniqCode := strings.TrimSpace(item.UniqCode)

		// Template helper untuk failed row agar tidak repetitif
		createFailedRow := func(msg string) models.FailedImport {
			return models.FailedImport{
				RowNumber:      i + 2, // Baris di excel (asumsi header di baris 1)
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

		// GENERATE PRL ID
		prlID := fmt.Sprintf("PRL-%s-%03d", year, baseNumber+int64(i)+1)

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

	// =========================
	// MODIFIKASI FILE ASLI (Jika ada yang gagal)
	// =========================
	if len(failedRows) > 0 {
		// Panggil fungsi untuk nambahin kolom "Error" ke file yang diupload tadi
		err := s.appendErrorToExcel(filePath, failedRows)
		if err != nil {
			return nil, fmt.Errorf("gagal menulis error ke excel: %v", err)
		}
	}

	return &models.BulkInsertResponse{
		Success:    success,
		Failed:     len(failedRows),
		FailedFile: filePath, // Balikin path yang sama
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
