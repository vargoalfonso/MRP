package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/bulkimport"
	"github.com/xuri/excelize/v2"
)

// ---------------------------------------------------------------------------
// Work Order import template (Sheet 1: input, Sheet 2: master data + bantuan)
// ---------------------------------------------------------------------------

const (
	// Sheet 2 name: master data + petunjuk pengisian.
	workOrderMasterSheetName = "Master Data"

	// Kanban number rule for bulk import: KBN-BLK-<increment>.
	kanbanBlkPrefix = "KBN-BLK"

	// Max master data rows rendered into the template.
	workOrderMasterDataLimit = 1000
)

// formatKanbanBlkNumber builds KBN-BLK-0001 style numbers.
func formatKanbanBlkNumber(seq int) string {
	return fmt.Sprintf("%s-%04d", kanbanBlkPrefix, seq)
}

// workOrderMasterRow is one parent uniq row shown on the Master Data sheet.
type workOrderMasterRow struct {
	UniqCode     string   `gorm:"column:uniq_code"`
	PartNumber   *string  `gorm:"column:part_number"`
	PartName     *string  `gorm:"column:part_name"`
	Model        *string  `gorm:"column:model"`
	UOM          *string  `gorm:"column:uom"`
	ProcessName  *string  `gorm:"column:process_name"`
	KanbanQty    *int     `gorm:"column:kanban_qty"`
	KanbanNumber *string  `gorm:"column:kanban_number"`
	ChildCount   int      `gorm:"column:child_count"`
	StockQty     *float64 `gorm:"column:stock_qty"`
}

const workOrderMasterDataSQL = `
SELECT i.uniq_code,
       i.part_number,
       i.part_name,
       i.model,
       i.uom                         AS uom,
       fp.process_name               AS process_name,
       kp.kanban_qty                 AS kanban_qty,
       kp.kanban_number              AS kanban_number,
       COALESCE(bl.child_count, 0)   AS child_count,
       NULL::numeric                 AS stock_qty
FROM   items i
LEFT   JOIN LATERAL (
           SELECT COUNT(DISTINCT b.child_item_id) AS child_count
           FROM   bom_lines b
           WHERE  b.parent_item_id = i.id
       ) bl ON TRUE
LEFT   JOIN LATERAL (
           SELECT pp.process_name
           FROM   routing_headers rh
           JOIN   routing_operations ro ON ro.routing_header_id = rh.id
           JOIN   process_parameters pp ON pp.id = ro.process_id
           WHERE  rh.item_id = i.id
           ORDER  BY ro.op_seq ASC, rh.id DESC
           LIMIT  1
       ) fp ON TRUE
LEFT   JOIN LATERAL (
           SELECT k.kanban_number, k.kanban_qty
           FROM   kanban_parameters k
           WHERE  k.item_uniq_code = i.uniq_code
             AND  k.status ILIKE 'active'
           ORDER  BY k.id DESC
           LIMIT  1
       ) kp ON TRUE
WHERE  i.deleted_at IS NULL
  AND  (i.status IS NULL OR i.status ILIKE 'active')
  %s
ORDER  BY i.uniq_code
LIMIT  ?`

// listWorkOrderMasterData returns the list of parent uniq codes (items that own
// at least one BOM line / BOM header). When the workspace has no BOM yet, it
// falls back to every active item so the template is never empty.
func (s *service) listWorkOrderMasterData(ctx context.Context) ([]workOrderMasterRow, error) {
	parentFilter := `AND (
       EXISTS (SELECT 1 FROM bom_lines b2 WHERE b2.parent_item_id = i.id)
    OR EXISTS (SELECT 1 FROM bom_item bi WHERE bi.item_id = i.id)
  )`

	var rows []workOrderMasterRow
	sql := fmt.Sprintf(workOrderMasterDataSQL, parentFilter)
	if err := s.db.WithContext(ctx).Raw(sql, workOrderMasterDataLimit).Scan(&rows).Error; err != nil {
		return nil, apperror.InternalWrap("failed to load work order master data", err)
	}
	if len(rows) == 0 {
		sql = fmt.Sprintf(workOrderMasterDataSQL, "")
		if err := s.db.WithContext(ctx).Raw(sql, workOrderMasterDataLimit).Scan(&rows).Error; err != nil {
			return nil, apperror.InternalWrap("failed to load work order master data", err)
		}
	}
	return rows, nil
}

// nextKanbanBlkSeq returns the next increment for the KBN-BLK-<n> rule.
func (s *service) nextKanbanBlkSeq(ctx context.Context) (int, error) {
	var maxSeq int
	sql := `SELECT COALESCE(MAX(CAST(SUBSTRING(kanban_number FROM '[0-9]+$') AS INTEGER)), 0)
	        FROM kanban_parameters
	        WHERE kanban_number ~ '^KBN-BLK-[0-9]+$'`
	if err := s.db.WithContext(ctx).Raw(sql).Scan(&maxSeq).Error; err != nil {
		return 0, apperror.InternalWrap("failed to read last kanban number", err)
	}
	return maxSeq + 1, nil
}

// ensureKanbanParamForImport guarantees the item has an active kanban parameter
// before the WO is created, mirroring the manual flow (which requires the
// kanban parameter to already exist). Auto-generated numbers follow the
// KBN-BLK-<increment> rule.
func (s *service) ensureKanbanParamForImport(ctx context.Context, itemUniqCode string, kanbanQty int) error {
	code := strings.TrimSpace(itemUniqCode)
	if code == "" || kanbanQty <= 0 {
		return nil
	}

	kp, err := s.getKanbanParam(ctx, s.db, code)
	if err != nil {
		return err
	}
	if kp != nil && strings.TrimSpace(kp.KanbanNumber) != "" {
		return nil
	}

	seq, err := s.nextKanbanBlkSeq(ctx)
	if err != nil {
		return err
	}
	number := formatKanbanBlkNumber(seq)

	insert := `INSERT INTO kanban_parameters (kanban_number, item_uniq_code, kanban_qty, min_stock, max_stock, status, created_at, updated_at)
	           VALUES (?, ?, ?, 0, 0, 'active', NOW(), NOW())`
	if err := s.db.WithContext(ctx).Exec(insert, number, code, kanbanQty).Error; err != nil {
		return apperror.InternalWrap("failed to create kanban parameter", err)
	}
	return nil
}

// masterSheetHeaders: kolom A-E sengaja disusun sama persis dengan urutan
// kolom F-J pada sheet "Work Orders" supaya user tinggal copy-paste.
var workOrderMasterSheetHeaders = []string{
	"item_uniq_code",
	"quantity",
	"uom",
	"process_name",
	"kanban_qty",
	"part_number",
	"part_name",
	"model",
	"kanban_number",
	"jumlah_child_bom",
}

var workOrderMasterSheetHelp = []string{
	"CARA PAKAI (copy-paste ke sheet \"Work Orders\")",
	"1. Pilih baris parent uniq yang ingin dibuatkan Work Order pada tabel di bawah.",
	"2. Blok kolom A sampai E (item_uniq_code, quantity, uom, process_name, kanban_qty) lalu Copy.",
	"3. Paste ke sheet \"Work Orders\" mulai dari kolom F (item_uniq_code) pada baris kosong pertama.",
	"4. Isi quantity sesuai kebutuhan, lalu lengkapi kolom A-E di sheet \"Work Orders\" (wo_type, reference_wo, created_date, target_date, notes).",
	"5. Hapus baris contoh di sheet \"Work Orders\" sebelum upload.",
	"",
	"ATURAN NOMOR KANBAN: " + kanbanBlkPrefix + "-<increment 4 digit>, contoh " + kanbanBlkPrefix + "-0001, " + kanbanBlkPrefix + "-0002.",
	"Nomor kanban dibuat OTOMATIS oleh sistem saat import (tidak perlu diisi manual).",
	"Kolom kanban_number di bawah hanya informasi: nilai yang sudah ada di master, atau usulan nomor berikutnya untuk item yang belum punya kanban.",
	"Nomor kanban per item WO tetap mengikuti aturan manual: <kanban_number>-<item_uniq_code>-<wo_number>-<NN>.",
	"kanban_qty = jumlah pcs per kanban. Bila 0/kosong, sistem memakai nilai dari master kanban parameter.",
}

type woTemplateStyles struct {
	title     int
	help      int
	header    int
	copyHead  int
	cell      int
	cellAlt   int
	copyCell  int
	copyAlt   int
	sample    int
	numberFmt int
}

func buildWoTemplateStyles(f *excelize.File) woTemplateStyles {
	border := []excelize.Border{
		{Type: "left", Color: "D9D9D9", Style: 1},
		{Type: "top", Color: "D9D9D9", Style: 1},
		{Type: "bottom", Color: "D9D9D9", Style: 1},
		{Type: "right", Color: "D9D9D9", Style: 1},
	}

	title, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "1F3864"},
	})
	help, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Color: "3B3B3B"},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	header, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F4E79"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	copyHead, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2E7D32"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})
	cell, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    border,
	})
	cellAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F7FA"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    border,
	})
	copyCell, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F5E9"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    border,
	})
	copyAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DCEFDC"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    border,
	})
	sample, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: "9C6500"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF2CC"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    border,
	})

	return woTemplateStyles{
		title:    title,
		help:     help,
		header:   header,
		copyHead: copyHead,
		cell:     cell,
		cellAlt:  cellAlt,
		copyCell: copyCell,
		copyAlt:  copyAlt,
		sample:   sample,
	}
}

func strOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

// generateWorkOrderImportTemplateWithMaster builds the 2-sheet template.
func generateWorkOrderImportTemplateWithMaster(master []workOrderMasterRow, nextKanbanSeq int) ([]byte, error) {
	f := excelize.NewFile()
	st := buildWoTemplateStyles(f)

	// ------------------------------------------------------------------
	// Sheet 1 - Work Orders (input)
	// ------------------------------------------------------------------
	_ = f.SetSheetName("Sheet1", workOrderImportSheetName)
	for i, h := range workOrderImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(workOrderImportSheetName, cell, h)
		_ = f.SetCellStyle(workOrderImportSheetName, cell, cell, st.header)
	}
	f.SetRowHeight(workOrderImportSheetName, 1, 22)

	today := time.Now()
	sampleUniq := "UNIQ-001"
	sampleUOM := "pcs"
	sampleProcess := "Cutting"
	sampleKanbanQty := 0
	if len(master) > 0 {
		sampleUniq = master[0].UniqCode
		if v := strOrEmpty(master[0].UOM); v != "" {
			sampleUOM = v
		}
		if v := strOrEmpty(master[0].ProcessName); v != "" {
			sampleProcess = v
		}
		if master[0].KanbanQty != nil && *master[0].KanbanQty > 0 {
			sampleKanbanQty = *master[0].KanbanQty
		}
	}

	sample := []interface{}{
		"New",
		"",
		today.Format("2006-01-02"),
		today.AddDate(0, 0, 7).Format("2006-01-02"),
		"CONTOH - hapus baris ini sebelum upload",
		sampleUniq,
		100,
		sampleUOM,
		sampleProcess,
		sampleKanbanQty,
	}
	for i, v := range sample {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(workOrderImportSheetName, cell, v)
		_ = f.SetCellStyle(workOrderImportSheetName, cell, cell, st.sample)
	}

	_ = f.SetColWidth(workOrderImportSheetName, "A", "A", 12)
	_ = f.SetColWidth(workOrderImportSheetName, "B", "B", 16)
	_ = f.SetColWidth(workOrderImportSheetName, "C", "D", 14)
	_ = f.SetColWidth(workOrderImportSheetName, "E", "E", 38)
	_ = f.SetColWidth(workOrderImportSheetName, "F", "F", 22)
	_ = f.SetColWidth(workOrderImportSheetName, "G", "G", 12)
	_ = f.SetColWidth(workOrderImportSheetName, "H", "H", 10)
	_ = f.SetColWidth(workOrderImportSheetName, "I", "I", 20)
	_ = f.SetColWidth(workOrderImportSheetName, "J", "J", 12)

	lastCol, _ := excelize.ColumnNumberToName(len(workOrderImportHeaders))
	f.AutoFilter(workOrderImportSheetName, "A1:"+lastCol+"1", nil)
	f.SetPanes(workOrderImportSheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Header comments (bantuan singkat di sheet utama).
	hints := map[string]string{
		"A1": "wo_type: New / Assembly / Rework / Addendum. Kosong = New.",
		"B1": "reference_wo: nomor WO referensi (opsional, wajib untuk Rework/Addendum).",
		"C1": "created_date: format YYYY-MM-DD.",
		"D1": "target_date: format YYYY-MM-DD.",
		"E1": "notes: catatan bebas (opsional).",
		"F1": "item_uniq_code: ambil dari sheet Master Data (kolom A).",
		"G1": "quantity: total qty yang akan diproduksi (wajib > 0).",
		"H1": "uom: satuan, ikut master item.",
		"I1": "process_name: proses pertama, ikut routing di master.",
		"J1": "kanban_qty: pcs per kanban. 0 = pakai master kanban parameter.",
	}
	for cell, text := range hints {
		f.AddComment(workOrderImportSheetName, excelize.Comment{
			Cell:   cell,
			Author: "MRP",
			Paragraph: []excelize.RichTextRun{
				{Text: text},
			},
		})
	}

	// ------------------------------------------------------------------
	// Sheet 2 - Master Data (bantuan + list parent uniq)
	// ------------------------------------------------------------------
	if _, err := f.NewSheet(workOrderMasterSheetName); err != nil {
		return nil, apperror.InternalWrap("failed to create master data sheet", err)
	}

	_ = f.SetCellValue(workOrderMasterSheetName, "A1", "MASTER DATA - PARENT UNIQ (Work Order Bulk Import)")
	_ = f.SetCellStyle(workOrderMasterSheetName, "A1", "A1", st.title)
	f.SetRowHeight(workOrderMasterSheetName, 1, 24)

	row := 3
	for _, line := range workOrderMasterSheetHelp {
		cell := fmt.Sprintf("A%d", row)
		_ = f.SetCellValue(workOrderMasterSheetName, cell, line)
		_ = f.SetCellStyle(workOrderMasterSheetName, cell, cell, st.help)
		row++
	}

	headerRow := row + 1
	for i, h := range workOrderMasterSheetHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(workOrderMasterSheetName, cell, h)
		style := st.header
		if i < 5 {
			style = st.copyHead
		}
		_ = f.SetCellStyle(workOrderMasterSheetName, cell, cell, style)
	}
	f.SetRowHeight(workOrderMasterSheetName, headerRow, 22)

	seq := nextKanbanSeq
	if seq <= 0 {
		seq = 1
	}

	dataRow := headerRow + 1
	for idx, m := range master {
		kanbanNumber := strOrEmpty(m.KanbanNumber)
		if kanbanNumber == "" {
			kanbanNumber = formatKanbanBlkNumber(seq) + " (auto)"
			seq++
		}
		kanbanQty := 0
		if m.KanbanQty != nil {
			kanbanQty = *m.KanbanQty
		}

		values := []interface{}{
			m.UniqCode,
			"",
			strOrEmpty(m.UOM),
			strOrEmpty(m.ProcessName),
			kanbanQty,
			strOrEmpty(m.PartNumber),
			strOrEmpty(m.PartName),
			strOrEmpty(m.Model),
			kanbanNumber,
			m.ChildCount,
		}

		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, dataRow)
			_ = f.SetCellValue(workOrderMasterSheetName, cell, v)

			var style int
			switch {
			case i < 5 && idx%2 == 0:
				style = st.copyCell
			case i < 5:
				style = st.copyAlt
			case idx%2 == 0:
				style = st.cell
			default:
				style = st.cellAlt
			}
			_ = f.SetCellStyle(workOrderMasterSheetName, cell, cell, style)
		}
		dataRow++
	}

	if len(master) == 0 {
		cell := fmt.Sprintf("A%d", dataRow)
		_ = f.SetCellValue(workOrderMasterSheetName, cell, "(belum ada master item aktif)")
		_ = f.SetCellStyle(workOrderMasterSheetName, cell, cell, st.cell)
		dataRow++
	}

	_ = f.SetColWidth(workOrderMasterSheetName, "A", "A", 24)
	_ = f.SetColWidth(workOrderMasterSheetName, "B", "B", 12)
	_ = f.SetColWidth(workOrderMasterSheetName, "C", "C", 10)
	_ = f.SetColWidth(workOrderMasterSheetName, "D", "D", 22)
	_ = f.SetColWidth(workOrderMasterSheetName, "E", "E", 12)
	_ = f.SetColWidth(workOrderMasterSheetName, "F", "F", 24)
	_ = f.SetColWidth(workOrderMasterSheetName, "G", "G", 34)
	_ = f.SetColWidth(workOrderMasterSheetName, "H", "H", 18)
	_ = f.SetColWidth(workOrderMasterSheetName, "I", "I", 26)
	_ = f.SetColWidth(workOrderMasterSheetName, "J", "J", 18)

	masterLastCol, _ := excelize.ColumnNumberToName(len(workOrderMasterSheetHeaders))
	f.AutoFilter(
		workOrderMasterSheetName,
		fmt.Sprintf("A%d:%s%d", headerRow, masterLastCol, headerRow),
		nil,
	)
	f.SetPanes(workOrderMasterSheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      headerRow,
		TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
		ActivePane:  "bottomLeft",
	})

	// Dropdown di sheet utama: item_uniq_code & process_name diambil dari master.
	if len(master) > 0 {
		lastDataRow := headerRow + len(master)

		dvUniq := excelize.NewDataValidation(true)
		dvUniq.Sqref = "F2:F5000"
		dvUniq.SetSqrefDropList(fmt.Sprintf("'%s'!$A$%d:$A$%d", workOrderMasterSheetName, headerRow+1, lastDataRow))
		dvUniq.SetError(excelize.DataValidationErrorStyleWarning, "item_uniq_code", "Pilih item_uniq_code dari sheet Master Data.")
		f.AddDataValidation(workOrderImportSheetName, dvUniq)

		dvType := excelize.NewDataValidation(true)
		dvType.Sqref = "A2:A5000"
		dvType.SetDropList([]string{"New", "Assembly", "Rework", "Addendum"})
		dvType.SetError(excelize.DataValidationErrorStyleWarning, "wo_type", "Pilih New / Assembly / Rework / Addendum.")
		f.AddDataValidation(workOrderImportSheetName, dvType)
	}

	f.SetActiveSheet(0)

	return bulkimport.ToBytes(f)
}
