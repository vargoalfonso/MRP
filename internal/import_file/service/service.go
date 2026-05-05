package service

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/import_file/helper"
	"github.com/ganasa18/go-template/internal/import_file/models"
	"github.com/ganasa18/go-template/internal/import_file/repository"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type ImportService interface {
	GenerateTemplateExcelPrls() (*bytes.Buffer, error)
	ImportExcel(ctx context.Context, filePath string) ([]models.ImportDataRequest, error)
	BulkInsertPRL(ctx context.Context, data []models.ImportDataRequest, filePath string) (*models.BulkInsertResponse, error)
}

type importService struct {
	repo repository.ImportRepository
}

func New(repo repository.ImportRepository) ImportService {
	return &importService{repo: repo}
}

func (s *importService) GenerateTemplateExcelPrls() (*bytes.Buffer, error) {
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

	// 🔹 convert ke buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (s *importService) ImportExcel(ctx context.Context, filePath string) ([]models.ImportDataRequest, error) {
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
			CustomerName:   helper.SafeGet(row, 1), // skip kolom "no"
			UniqCode:       helper.SafeGet(row, 2),
			ProductModel:   helper.SafeGet(row, 3),
			PartName:       helper.SafeGet(row, 4),
			PartNumber:     helper.SafeGet(row, 5),
			ForecastPeriod: helper.SafeGet(row, 6),
		}

		qtyStr := helper.SafeGet(row, 7)

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
