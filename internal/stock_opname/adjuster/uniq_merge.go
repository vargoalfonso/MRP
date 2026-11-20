package adjuster

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// mergeUniqWithBOM melengkapi daftar uniq yang SUDAH ada di inventory (live) dengan
// item BOM aktif, supaya uniq yang belum ada di inventory tetap bisa dipilih di form
// stock opname (tanpa validasi). Uniq live didahulukan (membawa stok sistem asli);
// duplikat dibuang dan hasil dibatasi limit. Uniq baru dibuat di inventory saat
// opname disetujui (lihat ApplyAdjustment tiap adjuster).
func mergeUniqWithBOM(ctx context.Context, tx *gorm.DB, q string, limit int, live []UniqSnapshotResult) ([]UniqSnapshotResult, error) {
	if limit <= 0 {
		limit = 20
	}
	bom, err := SearchWIPBOMUniqs(ctx, tx, q, limit)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(live)+len(bom))
	out := make([]UniqSnapshotResult, 0, limit)
	for _, group := range [][]UniqSnapshotResult{live, bom} {
		for _, r := range group {
			key := strings.ToUpper(strings.TrimSpace(r.UniqCode))
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, r)
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}
