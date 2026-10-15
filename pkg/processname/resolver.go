// Package processname resolves the current display name (process_parameters.process_name)
// of a routing step stored in work_order_items.process_flow_json.
//
// The flow JSON is a snapshot taken at WO creation, so its process_name keeps
// whatever label the process had back then (often equal to the process code).
// Renaming a process in System Parameters must still show the new name, so
// pages look the name up live instead of trusting the snapshot.
package processname

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// Resolver maps a snapshot step to the current process_name.
// The zero value is usable and simply returns the snapshot name.
type Resolver struct {
	byID   map[int64]string
	byCode map[string]string
}

// Load reads all process parameters. On error it returns a zero Resolver so
// callers can fall back to the snapshot name without failing the request.
func Load(ctx context.Context, db *gorm.DB) (Resolver, error) {
	var rows []struct {
		ID          int64  `gorm:"column:id"`
		ProcessCode string `gorm:"column:process_code"`
		ProcessName string `gorm:"column:process_name"`
	}
	if err := db.WithContext(ctx).Table("process_parameters").
		Select("id, process_code, process_name").Scan(&rows).Error; err != nil {
		return Resolver{}, err
	}
	r := Resolver{
		byID:   make(map[int64]string, len(rows)),
		byCode: make(map[string]string, len(rows)),
	}
	for _, row := range rows {
		name := strings.TrimSpace(row.ProcessName)
		if name == "" {
			continue
		}
		r.byID[row.ID] = name
		if code := strings.TrimSpace(row.ProcessCode); code != "" {
			r.byCode[code] = name
		}
	}
	return r, nil
}

// Name returns the current process_name for a step. Lookup order: process_id,
// process_code, then the snapshot name treated as a code (older snapshots have
// no id/code). Falls back to the snapshot name.
func (r Resolver) Name(processID int64, processCode, snapshotName string) string {
	if processID > 0 {
		if n, ok := r.byID[processID]; ok {
			return n
		}
	}
	if n, ok := r.byCode[strings.TrimSpace(processCode)]; ok {
		return n
	}
	snapshotName = strings.TrimSpace(snapshotName)
	if n, ok := r.byCode[snapshotName]; ok {
		return n
	}
	return snapshotName
}
