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
	byID   map[int64]string    // id -> process_name
	byCode map[string]int64    // process_code -> id
	names  map[string]struct{} // lower(process_name)
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
		byCode: make(map[string]int64, len(rows)),
		names:  make(map[string]struct{}, len(rows)),
	}
	for _, row := range rows {
		r.byID[row.ID] = strings.TrimSpace(row.ProcessName)
		if n := strings.ToLower(strings.TrimSpace(row.ProcessName)); n != "" {
			r.names[n] = struct{}{}
		}
		if code := strings.TrimSpace(row.ProcessCode); code != "" {
			r.byCode[code] = row.ID
		}
	}
	return r, nil
}

// ID returns the process_parameters id of a step. Lookup order: process_id,
// process_code, then the snapshot name treated as a code (older snapshots have
// no id/code). Returns 0 when it cannot be resolved.
func (r Resolver) ID(processID int64, processCode, snapshotName string) int64 {
	if _, ok := r.byID[processID]; ok && processID > 0 {
		return processID
	}
	if id, ok := r.byCode[strings.TrimSpace(processCode)]; ok {
		return id
	}
	if id, ok := r.byCode[strings.TrimSpace(snapshotName)]; ok {
		return id
	}
	return 0
}

// NameByID returns the current process_name of a process id ("" if unknown).
func (r Resolver) NameByID(id int64) string {
	return r.byID[id]
}

// Name returns the current process_name for a step, falling back to the
// snapshot name.
func (r Resolver) Name(processID int64, processCode, snapshotName string) string {
	if n := r.byID[r.ID(processID, processCode, snapshotName)]; n != "" {
		return n
	}
	return strings.TrimSpace(snapshotName)
}

// Known reports whether a step's process is registered in System Parameters
// (by id/code, or by its current name). Pseudo processes such as
// "RM Processing" are not.
func (r Resolver) Known(processID int64, processCode, snapshotName string) bool {
	if r.ID(processID, processCode, snapshotName) > 0 {
		return true
	}
	_, ok := r.names[strings.ToLower(strings.TrimSpace(snapshotName))]
	return ok
}

// SameProcess reports whether a machine registered for machineProcessID may run
// the given step: same process id, or the same current process_name
// (case-insensitive) as shown in System Parameters.
func (r Resolver) SameProcess(machineProcessID, processID int64, processCode, snapshotName string) bool {
	if machineProcessID <= 0 {
		return false
	}
	if id := r.ID(processID, processCode, snapshotName); id > 0 && id == machineProcessID {
		return true
	}
	machineName := strings.TrimSpace(r.byID[machineProcessID])
	stepName := r.Name(processID, processCode, snapshotName)
	return machineName != "" && strings.EqualFold(machineName, stepName)
}
