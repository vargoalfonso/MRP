package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/bulkimport"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

const (
	importSheetName  = "Supplier Info"
	importRefSheet   = "Referensi Supplier Item"
	importGuideSheet = "Petunjuk"
	importMaxRows    = 5000
)

// importHeaders adalah urutan kolom pada template dan pada file error.
var importHeaders = []string{"uniq", "supplier_name", "type", "uniq_zahir", "status"}

// headerAliases memetakan nama header (sudah dinormalisasi) ke nama kolom baku.
var headerAliases = map[string]string{
	"uniq":          "uniq",
	"uniq_code":     "uniq",
	"supplier_name": "supplier_name",
	"supplier":      "supplier_name",
	"type":          "type",
	"tipe":          "type",
	"uniq_zahir":    "uniq_zahir",
	"zahir":         "uniq_zahir",
	"status":        "status",
	"error_field":   "error_field", // kolom A pada file error; diabaikan saat import ulang
}

func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	return strings.NewReplacer(" ", "_", "-", "_").Replace(h)
}

// normalizeImportType menerima RM/IRM/SUBCON maupun nilai asli supplier_item.type
// dan mengembalikan label tampilan. Nilai kosong diperbolehkan (ok=true, label="").
func normalizeImportType(t string) (label string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "":
		return "", true
	case "rm", "raw_material", "raw material":
		return "RM", true
	case "irm", "indirect", "indirect_raw_material", "indirect raw material":
		return "IRM", true
	case "subcon", "sub_con", "sub con":
		return "SUBCON", true
	default:
		return "", false
	}
}

// normalizeImportStatus: kosong dianggap active.
func normalizeImportStatus(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "active", "aktif":
		return "active", true
	case "inactive", "nonaktif", "non-aktif", "tidak aktif":
		return "inactive", true
	default:
		return "", false
	}
}

func comboKey(uniq, supplierName, typeLabel string) string {
	return strings.ToLower(strings.TrimSpace(uniq)) + "||" +
		strings.ToLower(strings.TrimSpace(supplierName)) + "||" +
		strings.ToUpper(strings.TrimSpace(typeLabel))
}

// itemRef adalah satu kombinasi (uniq, supplier, type) yang valid di supplier_item.
type itemRef struct {
	Uniq         string
	SupplierName string
	TypeLabel    string
}

func (r itemRef) key() string { return comboKey(r.Uniq, r.SupplierName, r.TypeLabel) }

func (r itemRef) String() string { return fmt.Sprintf("%s (%s)", r.SupplierName, r.TypeLabel) }

// importRow adalah satu baris data mentah dari Excel.
type importRow struct {
	rowNum       int
	uniq         string
	supplierName string
	typ          string
	uniqZahir    string
	status       string
}

func (r importRow) raw() []string {
	return []string{r.uniq, r.supplierName, r.typ, r.uniqZahir, r.status}
}

// resolveItem mencocokkan baris dengan supplier_item. Bila supplier_name dan/atau type
// dikosongkan, kombinasi dipilih otomatis hanya jika UNIQ tersebut tidak ambigu.
func resolveItem(row importRow, typeLabel string, byUniq map[string][]itemRef) (itemRef, string) {
	cands := byUniq[strings.ToLower(row.uniq)]
	if len(cands) == 0 {
		return itemRef{}, fmt.Sprintf("UNIQ '%s' tidak ditemukan di Supplier Item aktif", row.uniq)
	}

	filtered := make([]itemRef, 0, len(cands))
	for _, c := range cands {
		if row.supplierName != "" && !strings.EqualFold(c.SupplierName, row.supplierName) {
			continue
		}
		if typeLabel != "" && c.TypeLabel != typeLabel {
			continue
		}
		filtered = append(filtered, c)
	}

	switch len(filtered) {
	case 0:
		return itemRef{}, fmt.Sprintf("kombinasi UNIQ '%s' dengan supplier/type tersebut tidak ditemukan di Supplier Item aktif", row.uniq)
	case 1:
		return filtered[0], ""
	default:
		opts := make([]string, 0, 5)
		for i, c := range filtered {
			if i == 5 {
				opts = append(opts, "...")
				break
			}
			opts = append(opts, c.String())
		}
		return itemRef{}, fmt.Sprintf("UNIQ '%s' ambigu (%d supplier/type: %s), isi kolom supplier_name dan type", row.uniq, len(filtered), strings.Join(opts, "; "))
	}
}

func (s *svc) getErrorStore() (bulkimport.ErrorStore, error) {
	s.errorStoreMu.Lock()
	defer s.errorStoreMu.Unlock()
	if s.errorStore != nil {
		return s.errorStore, nil
	}
	store, err := bulkimport.NewFileStore(filepath.Join(os.TempDir(), "supplier_info_import_errors"))
	if err != nil {
		return nil, apperror.InternalWrap("init error store", err)
	}
	s.errorStore = store
	return store, nil
}

// parseImportRows membaca sheet data dan mengembalikan baris non-kosong.
func parseImportRows(r io.Reader) ([]importRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, apperror.BadRequest("file tidak dapat dibaca, pastikan format .xlsx yang valid")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, apperror.BadRequest("file tidak memiliki sheet")
	}
	sheet := sheets[0]
	for _, name := range sheets {
		if strings.EqualFold(strings.TrimSpace(name), importSheetName) {
			sheet = name
			break
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, apperror.BadRequest("gagal membaca sheet " + sheet)
	}
	if len(rows) == 0 {
		return nil, apperror.BadRequest("file kosong")
	}

	col := map[string]int{}
	for i, h := range rows[0] {
		if name, ok := headerAliases[normalizeHeader(h)]; ok {
			if _, dup := col[name]; !dup {
				col[name] = i
			}
		}
	}
	if _, ok := col["uniq"]; !ok {
		return nil, apperror.BadRequest("kolom 'uniq' tidak ditemukan di baris header (gunakan template)")
	}

	cell := func(row []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	out := make([]importRow, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		ir := importRow{
			rowNum:       i + 1,
			uniq:         cell(rows[i], "uniq"),
			supplierName: cell(rows[i], "supplier_name"),
			typ:          cell(rows[i], "type"),
			uniqZahir:    cell(rows[i], "uniq_zahir"),
			status:       cell(rows[i], "status"),
		}
		if ir.uniq == "" && ir.supplierName == "" && ir.typ == "" && ir.uniqZahir == "" && ir.status == "" {
			continue // baris kosong
		}
		out = append(out, ir)
		if len(out) > importMaxRows {
			return nil, apperror.BadRequest(fmt.Sprintf("maksimal %d baris per import", importMaxRows))
		}
	}
	if len(out) == 0 {
		return nil, apperror.BadRequest("tidak ada baris data pada file")
	}
	return out, nil
}

// ImportFromExcel membaca file .xlsx, memvalidasi tiap baris, lalu menyimpan baris yang
// valid. Baris bermasalah tidak menggagalkan baris lain (hasil: success/partial/failed)
// dan dikumpulkan ke file error yang bisa diunduh lewat token.
func (s *svc) ImportFromExcel(ctx context.Context, r io.Reader, filename string) (bulkimport.BulkResult, error) {
	rows, err := parseImportRows(r)
	if err != nil {
		return bulkimport.BulkResult{}, err
	}

	items, err := s.repo.ListActiveSupplierItems(ctx)
	if err != nil {
		return bulkimport.BulkResult{}, err
	}
	byUniq := make(map[string][]itemRef)
	seenItem := make(map[string]struct{})
	for _, it := range items {
		if it.UniqCode == nil || strings.TrimSpace(*it.UniqCode) == "" {
			continue
		}
		ref := itemRef{
			Uniq:         strings.TrimSpace(*it.UniqCode),
			SupplierName: strings.TrimSpace(it.SupplierName),
			TypeLabel:    normalizeTypeLabel(it.Type),
		}
		if _, dup := seenItem[ref.key()]; dup {
			continue
		}
		seenItem[ref.key()] = struct{}{}
		lu := strings.ToLower(ref.Uniq)
		byUniq[lu] = append(byUniq[lu], ref)
	}

	existing, err := s.repo.ListAll(ctx)
	if err != nil {
		return bulkimport.BulkResult{}, err
	}
	registered := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		registered[comboKey(e.Uniq, e.SupplierName, e.Type)] = struct{}{}
	}

	var rowErrs []bulkimport.RowError
	addErr := func(row importRow, field, msg string) {
		rowErrs = append(rowErrs, bulkimport.RowError{
			Sheet:   importSheetName,
			Row:     row.rowNum,
			Field:   field,
			Message: msg,
			RawData: row.raw(),
		})
	}

	seenInFile := make(map[string]int) // key -> nomor baris pertama
	success := 0

	for _, row := range rows {
		if row.uniq == "" {
			addErr(row, "uniq", "uniq wajib diisi")
			continue
		}
		typeLabel, ok := normalizeImportType(row.typ)
		if !ok {
			addErr(row, "type", fmt.Sprintf("type '%s' tidak valid (gunakan RM, IRM, atau SUBCON)", row.typ))
			continue
		}
		status, ok := normalizeImportStatus(row.status)
		if !ok {
			addErr(row, "status", fmt.Sprintf("status '%s' tidak valid (gunakan active atau inactive)", row.status))
			continue
		}

		ref, msg := resolveItem(row, typeLabel, byUniq)
		if msg != "" {
			addErr(row, "uniq", msg)
			continue
		}

		key := ref.key()
		if first, dup := seenInFile[key]; dup {
			addErr(row, "uniq", fmt.Sprintf("duplikat di file: %s sudah ada di baris %d", ref.Uniq+" - "+ref.String(), first))
			continue
		}
		seenInFile[key] = row.rowNum

		if _, exists := registered[key]; exists {
			addErr(row, "uniq", fmt.Sprintf("UNIQ '%s' untuk %s sudah terdaftar di supplier info", ref.Uniq, ref.String()))
			continue
		}

		info := &siModels.SupplierInfo{
			UUID:         uuid.NewString(),
			Uniq:         ref.Uniq,
			UniqZahir:    toOptStr(row.uniqZahir),
			SupplierName: ref.SupplierName,
			Type:         ref.TypeLabel,
			Status:       status,
		}
		if err := s.repo.Create(ctx, info); err != nil {
			m := "gagal menyimpan"
			if appErr, isApp := apperror.As(err); isApp {
				m = appErr.Message
			}
			addErr(row, "", m)
			continue
		}
		registered[key] = struct{}{}
		success++
	}

	result := bulkimport.BulkResult{
		Total:        len(rows),
		SuccessCount: success,
		FailedCount:  len(rowErrs),
		Errors:       rowErrs,
	}
	switch {
	case len(rowErrs) == 0:
		result.Status = bulkimport.StatusSuccess
	case success == 0:
		result.Status = bulkimport.StatusFailed
	default:
		result.Status = bulkimport.StatusPartial
	}

	if len(rowErrs) > 0 {
		f, err := bulkimport.GenerateErrorExcel(
			[]bulkimport.SheetDef{{Name: importSheetName, Headers: importHeaders}}, rowErrs)
		if err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("generate error excel", err)
		}
		defer f.Close()
		buf, err := f.WriteToBuffer()
		if err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("write error excel", err)
		}
		store, err := s.getErrorStore()
		if err != nil {
			return bulkimport.BulkResult{}, err
		}
		token, err := store.Save(buf.Bytes())
		if err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("save error excel", err)
		}
		result.ErrorToken = token
	}

	return result, nil
}

// DownloadImportErrors mengambil file error (sekali pakai, kedaluwarsa 30 menit).
func (s *svc) DownloadImportErrors(ctx context.Context, token string) ([]byte, error) {
	store, err := s.getErrorStore()
	if err != nil {
		return nil, err
	}
	data, err := store.Get(strings.TrimSpace(token))
	if err != nil {
		return nil, apperror.InternalWrap("read error file", err)
	}
	if data == nil {
		return nil, apperror.NotFound("file error tidak ditemukan atau sudah kedaluwarsa")
	}
	return data, nil
}

// DownloadImportTemplate membuat template: sheet data (kosong, dengan dropdown type &
// status), sheet petunjuk, dan sheet referensi berisi seluruh kombinasi Supplier Item aktif.
func (s *svc) DownloadImportTemplate(ctx context.Context) ([]byte, error) {
	items, err := s.repo.ListActiveSupplierItems(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	registered := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		registered[comboKey(e.Uniq, e.SupplierName, e.Type)] = struct{}{}
	}

	refs := make([]itemRef, 0, len(items))
	seen := make(map[string]struct{})
	for _, it := range items {
		if it.UniqCode == nil || strings.TrimSpace(*it.UniqCode) == "" {
			continue
		}
		ref := itemRef{
			Uniq:         strings.TrimSpace(*it.UniqCode),
			SupplierName: strings.TrimSpace(it.SupplierName),
			TypeLabel:    normalizeTypeLabel(it.Type),
		}
		if _, dup := seen[ref.key()]; dup {
			continue
		}
		seen[ref.key()] = struct{}{}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Uniq != refs[j].Uniq {
			return refs[i].Uniq < refs[j].Uniq
		}
		if refs[i].SupplierName != refs[j].SupplierName {
			return refs[i].SupplierName < refs[j].SupplierName
		}
		return refs[i].TypeLabel < refs[j].TypeLabel
	})

	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetSheetName("Sheet1", importSheetName)

	headStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8EEF9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	textStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 49}) // format Text: pertahankan angka 0 di depan

	for i, h := range importHeaders {
		cellName, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(importSheetName, cellName, h)
		_ = f.SetCellStyle(importSheetName, cellName, cellName, headStyle)
	}
	_ = f.SetColWidth(importSheetName, "A", "E", 26)
	_ = f.SetColStyle(importSheetName, "A:D", textStyle)

	typeDV := excelize.NewDataValidation(true)
	typeDV.Sqref = fmt.Sprintf("C2:C%d", importMaxRows+1)
	_ = typeDV.SetDropList([]string{"RM", "IRM", "SUBCON"})
	_ = f.AddDataValidation(importSheetName, typeDV)

	statusDV := excelize.NewDataValidation(true)
	statusDV.Sqref = fmt.Sprintf("E2:E%d", importMaxRows+1)
	_ = statusDV.SetDropList([]string{"active", "inactive"})
	_ = f.AddDataValidation(importSheetName, statusDV)

	// Sheet petunjuk
	_, _ = f.NewSheet(importGuideSheet)
	guide := []string{
		"PETUNJUK IMPORT SUPPLIER INFO",
		"",
		"Isi data pada sheet 'Supplier Info' mulai baris 2. Jangan mengubah nama kolom di baris 1.",
		"uniq          : WAJIB. Harus ada di Supplier Item aktif (lihat sheet 'Referensi Supplier Item').",
		"supplier_name : Nama supplier persis seperti di Supplier Item. Boleh kosong bila UNIQ hanya dimiliki satu supplier/type.",
		"type          : RM, IRM, atau SUBCON. Boleh kosong bila UNIQ hanya dimiliki satu supplier/type.",
		"uniq_zahir    : Opsional. Kode UNIQ ZAHIR dari supplier.",
		"status        : active atau inactive. Kosong dianggap active.",
		"",
		"Satu UNIQ bisa dimiliki beberapa supplier/type (mis. M11 sebagai RM dan SUBCON).",
		"Untuk UNIQ seperti itu, supplier_name dan type WAJIB diisi agar tidak ambigu.",
		"Kombinasi (uniq, supplier_name, type) yang sudah terdaftar akan ditolak (tidak menimpa data lama).",
		fmt.Sprintf("Maksimal %d baris per file. Baris yang gagal tidak membatalkan baris lain; unduh file error, perbaiki, lalu unggah ulang.", importMaxRows),
	}
	for i, line := range guide {
		cellName, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetCellValue(importGuideSheet, cellName, line)
	}
	_ = f.SetCellStyle(importGuideSheet, "A1", "A1", headStyle)
	_ = f.SetColWidth(importGuideSheet, "A", "A", 130)

	// Sheet referensi
	_, _ = f.NewSheet(importRefSheet)
	refHeaders := []string{"uniq", "supplier_name", "type", "sudah_terdaftar"}
	for i, h := range refHeaders {
		cellName, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(importRefSheet, cellName, h)
		_ = f.SetCellStyle(importRefSheet, cellName, cellName, headStyle)
	}
	for i, ref := range refs {
		row := i + 2
		reg := "Tidak"
		if _, ok := registered[ref.key()]; ok {
			reg = "Ya"
		}
		vals := []string{ref.Uniq, ref.SupplierName, ref.TypeLabel, reg}
		for c, v := range vals {
			cellName, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue(importRefSheet, cellName, v)
		}
	}
	_ = f.SetColWidth(importRefSheet, "A", "D", 28)

	f.SetActiveSheet(0) // sheet pertama = "Supplier Info"

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, apperror.InternalWrap("write template", err)
	}
	return buf.Bytes(), nil
}
