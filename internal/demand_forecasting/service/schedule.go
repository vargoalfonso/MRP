package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/pkg/apperror"
)

// Training schedule (System Settings -> Automation).
//
// ERP hanya menyimpan kapan DN / PRL di-train. Script forecasting membaca jadwal
// ini dan melaporkan hasil run lewat ReportScheduleRun.

const defaultScheduleTimezone = "Asia/Jakarta"

var (
	runTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

	scheduleDomains     = map[string]bool{"dn": true, "prl": true}
	scheduleFrequencies = map[string]bool{"daily": true, "weekly": true, "monthly": true}
	weekdays            = map[string]time.Weekday{
		"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
		"wednesday": time.Wednesday, "thursday": time.Thursday,
		"friday": time.Friday, "saturday": time.Saturday,
	}
	runStatuses = map[string]bool{"RUNNING": true, "SUCCEEDED": true, "FAILED": true}
)

func (s *service) ListSchedules(ctx context.Context) ([]models.ForecastingSchedule, error) {
	return s.repo.ListSchedules(ctx)
}

func (s *service) UpsertSchedule(ctx context.Context, domain string, req models.UpsertScheduleRequest, updatedBy string) (*models.ForecastingSchedule, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !scheduleDomains[domain] {
		return nil, apperror.BadRequest("domain harus 'dn' atau 'prl'")
	}

	req.Frequency = strings.ToLower(strings.TrimSpace(req.Frequency))
	if !scheduleFrequencies[req.Frequency] {
		return nil, apperror.BadRequest("frequency harus daily, weekly, atau monthly")
	}
	// Data PRL bulanan -> hanya boleh monthly.
	if domain == "prl" && req.Frequency != "monthly" {
		return nil, apperror.BadRequest("data PRL bulanan, frequency hanya boleh monthly")
	}

	req.RunTime = strings.TrimSpace(req.RunTime)
	if !runTimePattern.MatchString(req.RunTime) {
		return nil, apperror.BadRequest("run_time harus berformat HH:mm (24 jam)")
	}

	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = defaultScheduleTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, apperror.BadRequest("timezone tidak dikenali: " + tz)
	}

	model := strings.TrimSpace(req.SelectedModel)
	if model == "" {
		model = "Chronos2"
	}

	// Hanya simpan field yang relevan dengan frekuensi.
	var dow *string
	var dom *int
	switch req.Frequency {
	case "weekly":
		d := strings.ToLower(strings.TrimSpace(deref(req.DayOfWeek)))
		if _, ok := weekdays[d]; !ok {
			return nil, apperror.BadRequest("day_of_week wajib diisi (monday..sunday) untuk weekly")
		}
		dow = &d
	case "monthly":
		if req.DayOfMonth == nil || *req.DayOfMonth < 1 || *req.DayOfMonth > 31 {
			return nil, apperror.BadRequest("day_of_month wajib diisi (1-31) untuk monthly")
		}
		v := *req.DayOfMonth
		dom = &v
	}

	sched := &models.ForecastingSchedule{
		Domain:        domain,
		IsActive:      req.IsActive,
		Frequency:     req.Frequency,
		RunTime:       req.RunTime,
		DayOfWeek:     dow,
		DayOfMonth:    dom,
		Timezone:      tz,
		SelectedModel: model,
		UpdatedAt:     time.Now().UTC(),
		UpdatedBy:     &updatedBy,
	}
	if sched.IsActive {
		next := computeNextRun(sched, time.Now().In(loc))
		sched.NextRunAt = &next
	}

	if err := s.repo.UpsertSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return s.repo.FindSchedule(ctx, domain)
}

// ReportScheduleRun dipanggil script forecasting untuk melaporkan hasil run.
func (s *service) ReportScheduleRun(ctx context.Context, domain string, rep models.ScheduleRunReport) (*models.ForecastingSchedule, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !scheduleDomains[domain] {
		return nil, apperror.BadRequest("domain harus 'dn' atau 'prl'")
	}
	status := strings.ToUpper(strings.TrimSpace(rep.Status))
	if !runStatuses[status] {
		return nil, apperror.BadRequest("status harus RUNNING, SUCCEEDED, atau FAILED")
	}

	runAt, err := parseReportTime(rep.StartedAt)
	if err != nil {
		return nil, apperror.BadRequest("started_at harus berformat RFC3339")
	}
	if status != "RUNNING" && rep.FinishedAt != nil && strings.TrimSpace(*rep.FinishedAt) != "" {
		finished, err := parseReportTime(*rep.FinishedAt)
		if err != nil {
			return nil, apperror.BadRequest("finished_at harus berformat RFC3339")
		}
		runAt = finished
	}

	current, err := s.repo.FindSchedule(ctx, domain)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, apperror.NotFound("jadwal untuk domain ini belum dikonfigurasi")
	}

	fields := map[string]interface{}{
		"last_run_at":          runAt,
		"last_run_status":      status,
		"last_run_message":     rep.Message,
		"last_training_run_id": rep.TrainingRunID,
	}
	// Run selesai -> hitung jadwal berikutnya dari sekarang.
	if status != "RUNNING" && current.IsActive {
		loc, lerr := time.LoadLocation(current.Timezone)
		if lerr != nil {
			loc, _ = time.LoadLocation(defaultScheduleTimezone)
		}
		if loc == nil {
			loc = time.UTC
		}
		fields["next_run_at"] = computeNextRun(current, time.Now().In(loc))
	}

	if err := s.repo.UpdateScheduleRun(ctx, domain, fields); err != nil {
		return nil, err
	}
	return s.repo.FindSchedule(ctx, domain)
}

func parseReportTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, v)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// computeNextRun menghitung run berikutnya (setelah `now`) di zona waktu `now`.
// Sama dengan preview di UI: bulan yang lebih pendek berjalan di hari terakhir.
func computeNextRun(sc *models.ForecastingSchedule, now time.Time) time.Time {
	var h, m int
	fmt.Sscanf(sc.RunTime, "%d:%d", &h, &m)
	loc := now.Location()
	at := func(y int, mo time.Month, d int) time.Time {
		return time.Date(y, mo, d, h, m, 0, 0, loc)
	}

	switch sc.Frequency {
	case "daily":
		t := at(now.Year(), now.Month(), now.Day())
		if t.After(now) {
			return t
		}
		return t.AddDate(0, 0, 1)

	case "weekly":
		target := time.Monday
		if sc.DayOfWeek != nil {
			if wd, ok := weekdays[*sc.DayOfWeek]; ok {
				target = wd
			}
		}
		t := at(now.Year(), now.Month(), now.Day())
		t = t.AddDate(0, 0, (int(target)-int(now.Weekday())+7)%7)
		if !t.After(now) {
			t = t.AddDate(0, 0, 7)
		}
		return t

	default: // monthly
		dom := 1
		if sc.DayOfMonth != nil {
			dom = *sc.DayOfMonth
		}
		inMonth := func(first time.Time) time.Time {
			last := first.AddDate(0, 1, -1).Day()
			d := dom
			if d > last {
				d = last
			}
			return at(first.Year(), first.Month(), d)
		}
		firstThis := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		if t := inMonth(firstThis); t.After(now) {
			return t
		}
		return inMonth(firstThis.AddDate(0, 1, 0))
	}
}
