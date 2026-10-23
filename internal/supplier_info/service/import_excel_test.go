package service

import (
	"bytes"
	"context"
	"testing"

	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	supplierItemModels "github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/bulkimport"
	"github.com/xuri/excelize/v2"
)

// fakeRepo adalah implementasi in-memory siRepo.IRepository untuk test.
type fakeRepo struct {
	items   []supplierItemModels.SupplierItem
	infos   []siModels.SupplierInfo
	created []*siModels.SupplierInfo
}

func (f *fakeRepo) Create(_ context.Context, info *siModels.SupplierInfo) error {
	f.created = append(f.created, info)
	f.infos = append(f.infos, *info)
	return nil
}
func (f *fakeRepo) FindByUUID(context.Context, string) (*siModels.SupplierInfo, error) {
	return nil, nil
}
func (f *fakeRepo) List(context.Context, siModels.ListSupplierInfoQuery) ([]siModels.SupplierInfo, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepo) Update(context.Context, *siModels.SupplierInfo) error { return nil }
func (f *fakeRepo) Delete(context.Context, *siModels.SupplierInfo) error { return nil }
func (f *fakeRepo) FindSupplierItem(context.Context, string, string, string) (*supplierItemModels.SupplierItem, error) {
	return nil, nil
}
func (f *fakeRepo) ExistsByUniqSupplierType(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (f *fakeRepo) ListActiveSupplierItems(context.Context) ([]supplierItemModels.SupplierItem, error) {
	return f.items, nil
}
func (f *fakeRepo) ListAll(context.Context) ([]siModels.SupplierInfo, error) { return f.infos, nil }

func strPtr(s string) *string { return &s }

func newTestRepo() *fakeRepo {
	return &fakeRepo{
		items: []supplierItemModels.SupplierItem{
			{UniqCode: strPtr("M11"), SupplierName: "PT METALINDO PRIMA", Type: "subcon", Status: "active"},
			{UniqCode: strPtr("M11"), SupplierName: "PT INDONESIA STEEL TUBE WORKS", Type: "raw_material", Status: "active"},
			{UniqCode: strPtr("A1"), SupplierName: "PT ABC", Type: "indirect", Status: "active"},
			{UniqCode: strPtr("A2"), SupplierName: "PT DEF", Type: "raw_material", Status: "active"},
			{UniqCode: strPtr("B7"), SupplierName: "PT GHI", Type: "raw_material", Status: "active"},
		},
		// B7 sudah terdaftar
		infos: []siModels.SupplierInfo{{Uniq: "B7", SupplierName: "PT GHI", Type: "RM", Status: "active"}},
	}
}

func buildWorkbook(t *testing.T, rows [][]string) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetSheetName("Sheet1", importSheetName)
	for r, row := range rows {
		for c, v := range row {
			if v == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := f.SetCellValue(importSheetName, cell, v); err != nil {
				t.Fatalf("set cell: %v", err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestNormalizeImportType(t *testing.T) {
	cases := map[string]string{"rm": "RM", "RAW_MATERIAL": "RM", "Irm": "IRM", "indirect": "IRM", "SUBCON": "SUBCON", "": ""}
	for in, want := range cases {
		got, ok := normalizeImportType(in)
		if !ok || got != want {
			t.Errorf("normalizeImportType(%q) = %q,%v; want %q,true", in, got, ok, want)
		}
	}
	if _, ok := normalizeImportType("xyz"); ok {
		t.Error("type 'xyz' seharusnya tidak valid")
	}
}

func TestNormalizeImportStatus(t *testing.T) {
	if got, ok := normalizeImportStatus(""); !ok || got != "active" {
		t.Errorf("status kosong harus active, got %q,%v", got, ok)
	}
	if got, ok := normalizeImportStatus("Inactive"); !ok || got != "inactive" {
		t.Errorf("got %q,%v", got, ok)
	}
	if _, ok := normalizeImportStatus("foo"); ok {
		t.Error("status 'foo' seharusnya tidak valid")
	}
}

func TestImportFromExcel_MixedRows(t *testing.T) {
	repo := newTestRepo()
	s := &svc{repo: repo}

	rows := [][]string{
		{"error_field", "UNIQ", "Supplier Name", "Type", "UNIQ ZAHIR", "Status"}, // header + kolom error_field diabaikan
		{"", "M11", "PT METALINDO PRIMA", "SUBCON", "0011", "active"},            // ok
		{"", "M11", "", "", "", ""},                                              // ambigu
		{"", "M11", "PT INDONESIA STEEL TUBE WORKS", "RM", "", ""},               // ok
		{"", "A1", "", "", "", ""},                                               // ok (auto-resolve), status default active
		{"", "ZZZ", "", "", "", ""},                                              // tidak ditemukan
		{"", "M11", "PT METALINDO PRIMA", "SUBCON", "", ""},                      // duplikat di file
		{"", "A2", "", "", "", "foo"},                                            // status invalid
		{"", "B7", "", "", "", ""},                                               // sudah terdaftar
	}

	res, err := s.ImportFromExcel(context.Background(), buildWorkbook(t, rows), "x.xlsx")
	if err != nil {
		t.Fatalf("ImportFromExcel error: %v", err)
	}

	if res.Total != 8 || res.SuccessCount != 3 || res.FailedCount != 5 {
		t.Fatalf("total/success/failed = %d/%d/%d; want 8/3/5 (errors: %+v)", res.Total, res.SuccessCount, res.FailedCount, res.Errors)
	}
	if res.Status != bulkimport.StatusPartial {
		t.Errorf("status = %s; want partial", res.Status)
	}
	if res.ErrorToken == "" {
		t.Error("ErrorToken seharusnya terisi saat ada baris gagal")
	}
	if len(repo.created) != 3 {
		t.Fatalf("created = %d; want 3", len(repo.created))
	}

	first := repo.created[0]
	if first.Uniq != "M11" || first.SupplierName != "PT METALINDO PRIMA" || first.Type != "SUBCON" {
		t.Errorf("baris pertama salah: %+v", first)
	}
	if first.UniqZahir == nil || *first.UniqZahir != "0011" {
		t.Errorf("uniq_zahir = %v; want 0011", first.UniqZahir)
	}
	second := repo.created[1]
	if second.SupplierName != "PT INDONESIA STEEL TUBE WORKS" || second.Type != "RM" {
		t.Errorf("baris kedua salah: %+v", second)
	}
	third := repo.created[2]
	if third.Uniq != "A1" || third.Type != "IRM" || third.Status != "active" {
		t.Errorf("baris ketiga salah: %+v", third)
	}

	// Nomor baris pada error mengacu ke baris Excel asli.
	wantRows := map[int]bool{3: true, 6: true, 7: true, 8: true, 9: true}
	for _, e := range res.Errors {
		if !wantRows[e.Row] {
			t.Errorf("error di baris tak terduga %d: %s", e.Row, e.Message)
		}
	}
}

func TestImportFromExcel_AllSuccess(t *testing.T) {
	s := &svc{repo: newTestRepo()}
	rows := [][]string{
		{"uniq", "supplier_name", "type", "uniq_zahir", "status"},
		{"A1", "", "", "Z-1", "inactive"},
	}
	res, err := s.ImportFromExcel(context.Background(), buildWorkbook(t, rows), "x.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != bulkimport.StatusSuccess || res.SuccessCount != 1 || res.ErrorToken != "" {
		t.Errorf("hasil tidak sesuai: %+v", res)
	}
}

func TestImportFromExcel_MissingUniqColumn(t *testing.T) {
	s := &svc{repo: newTestRepo()}
	rows := [][]string{{"foo", "bar"}, {"a", "b"}}
	if _, err := s.ImportFromExcel(context.Background(), buildWorkbook(t, rows), "x.xlsx"); err == nil {
		t.Error("seharusnya error karena kolom uniq tidak ada")
	}
}

func TestDownloadImportTemplate(t *testing.T) {
	s := &svc{repo: newTestRepo()}
	data, err := s.DownloadImportTemplate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("template bukan xlsx valid: %v", err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 3 || got[0] != importSheetName {
		t.Errorf("sheet = %v", got)
	}
	ref, _ := f.GetRows(importRefSheet)
	if len(ref) != 1+5 { // header + 5 kombinasi supplier_item
		t.Errorf("baris referensi = %d; want 6", len(ref))
	}
}
