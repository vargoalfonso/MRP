package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/pkg/forecastingclient"
)

// Status run training terjadwal (last_run_*).
//
// Service forecasting tidak melapor ke POST /forecasting/service/schedules/:domain/runs
// (kredensial ADMIN_JOB_* tidak tersedia di sana). Jadi arahnya dibalik: tiap
// ListSchedules, ERP membaca status run terakhir dari forecasting
// (GET /admin/scheduler/status) lalu menyimpannya ke forecasting_schedules.
// Cara ini memakai forecastingclient yang sama dengan proxy datasets / model-versions
// dan tidak butuh Basic Auth tambahan.

// schedulerStatusTimeout membatasi panggilan status. Kalau forecasting lambat atau
// mati, daftar jadwal tetap tampil memakai nilai yang tersimpan di ERP.
var schedulerStatusTimeout = 5 * time.Second

// syncRunStatus best-effort: kegagalan membaca status tidak menggagalkan ListSchedules.
// Hasil gabungan juga langsung tercermin di `schedules` (slice yang akan dikembalikan).
func (s *service) syncRunStatus(ctx context.Context, schedules []models.ForecastingSchedule) {
	if len(schedules) == 0 {
		return
	}

	cctx, cancel := context.WithTimeout(ctx, schedulerStatusTimeout)
	defer cancel()
	statuses, err := s.client.ListSchedulerStatus(cctx)
	if err != nil {
		slog.Warn("forecasting schedules: read scheduler status failed, using stored values", slog.Any("error", err))
		return
	}

	byDomain := make(map[string]forecastingclient.SchedulerStatus, len(statuses))
	for _, st := range statuses {
		byDomain[strings.ToLower(strings.TrimSpace(st.Domain))] = st
	}

	now := time.Now()
	for i := range schedules {
		st, ok := byDomain[strings.ToLower(schedules[i].Domain)]
		if !ok {
			continue
		}
		fields, changed := applyRunStatus(&schedules[i], st, now)
		if !changed {
			continue
		}
		if err := s.repo.UpdateScheduleRun(ctx, schedules[i].Domain, fields); err != nil {
			// Nilai gabungan tetap dikembalikan ke FE; sinkron berikutnya akan mencoba lagi.
			slog.Warn("forecasting schedules: save run status failed",
				slog.String("domain", schedules[i].Domain), slog.Any("error", err))
		}
	}
}

// applyRunStatus menggabungkan status run dari forecasting ke jadwal `sc` (di memori)
// dan mengembalikan kolom yang perlu disimpan ke DB. changed=false bila tidak ada
// yang berubah atau datanya tidak valid.
//
// Aturan next_run_at sama dengan ReportScheduleRun: hanya dihitung ulang bila run
// sudah selesai (SUCCEEDED / FAILED) dan jadwal aktif. Dasar hitungnya adalah yang
// lebih baru antara waktu run dan waktu jadwal terakhir disimpan, supaya
//   - jadwal yang diubah setelah run terakhir tidak memicu run tambahan, dan
//   - run yang sudah jatuh tempo tapi belum jalan tidak terlewat.
func applyRunStatus(sc *models.ForecastingSchedule, st forecastingclient.SchedulerStatus, now time.Time) (map[string]interface{}, bool) {
	status := strings.ToUpper(strings.TrimSpace(deref(st.LastRunStatus)))
	if !runStatuses[status] {
		return nil, false
	}
	if st.LastRunAt == nil || strings.TrimSpace(*st.LastRunAt) == "" {
		return nil, false
	}
	runAt, err := parseReportTime(*st.LastRunAt)
	if err != nil {
		return nil, false
	}

	if sc.LastRunAt != nil && sc.LastRunAt.Equal(runAt) &&
		deref(sc.LastRunStatus) == status &&
		deref(sc.LastRunMessage) == deref(st.LastRunMessage) &&
		deref(sc.LastTrainingRunID) == deref(st.LastTrainingRunID) {
		return nil, false
	}

	fields := map[string]interface{}{
		"last_run_at":          runAt,
		"last_run_status":      status,
		"last_run_message":     st.LastRunMessage,
		"last_training_run_id": st.LastTrainingRunID,
	}

	if status != "RUNNING" && sc.IsActive {
		loc, lerr := time.LoadLocation(sc.Timezone)
		if lerr != nil {
			loc, _ = time.LoadLocation(defaultScheduleTimezone)
		}
		if loc == nil {
			loc = time.UTC
		}
		base := runAt
		if sc.UpdatedAt.After(base) {
			base = sc.UpdatedAt
		}
		next := computeNextRun(sc, base.In(loc))
		fields["next_run_at"] = next
		sc.NextRunAt = &next
	}

	sc.LastRunAt = &runAt
	sc.LastRunStatus = &status
	sc.LastRunMessage = st.LastRunMessage
	sc.LastTrainingRunID = st.LastTrainingRunID
	return fields, true
}
