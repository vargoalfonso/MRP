package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/internal/demand_forecasting/repository"
	"github.com/ganasa18/go-template/pkg/forecastingclient"
)

func schedStrp(s string) *string { return &s }
func schedIntp(i int) *int       { return &i }

var schedWIB, _ = time.LoadLocation("Asia/Jakarta")

func schedAt(y int, m time.Month, d, h, mi int) time.Time {
	return time.Date(y, m, d, h, mi, 0, 0, schedWIB)
}

func schedDaily(runTime string, updated time.Time) models.ForecastingSchedule {
	return models.ForecastingSchedule{Domain: "dn", IsActive: true, Frequency: "daily", RunTime: runTime, Timezone: "Asia/Jakarta", SelectedModel: "Chronos2", UpdatedAt: updated}
}

func schedStatus(runAt, st string, msg *string) forecastingclient.SchedulerStatus {
	return forecastingclient.SchedulerStatus{Domain: "dn", LastRunAt: &runAt, LastRunStatus: &st, LastRunMessage: msg, LastTrainingRunID: schedStrp("8838deac")}
}

func schedNextOf(t *testing.T, f map[string]interface{}) time.Time {
	v, ok := f["next_run_at"]
	if !ok {
		t.Fatalf("next_run_at tidak ada di fields: %v", f)
	}
	return v.(time.Time)
}

// Contoh dari catatan integrasi: last_run_at 2026-11-17T20:01:25Z = 18 Nov 03:01 WIB.
func TestApply_SucceededRecomputesNextFromRunTime(t *testing.T) {
	sc := schedDaily("03:00", schedAt(2026, 11, 13, 10, 12)) // disimpan 13 Nov, run_time 03:00
	f, changed := applyRunStatus(&sc, schedStatus("2026-11-17T20:01:25.420322+00:00", "SUCCEEDED", nil), time.Now())
	if !changed {
		t.Fatal("harus berubah")
	}
	if got := schedNextOf(t, f); !got.Equal(schedAt(2026, 11, 19, 3, 0)) {
		t.Fatalf("next = %v, mau 19 Nov 03:00 WIB", got.In(schedWIB))
	}
	if f["last_run_status"] != "SUCCEEDED" || f["last_training_run_id"].(*string) == nil {
		t.Fatalf("fields: %v", f)
	}
	if sc.LastRunStatus == nil || *sc.LastRunStatus != "SUCCEEDED" || sc.NextRunAt == nil || sc.LastRunAt == nil {
		t.Fatalf("merge di memori gagal: %+v", sc)
	}
}

// Jadwal diubah SETELAH run terakhir: tidak boleh jatuh ke slot lama yang sudah lewat.
func TestApply_ScheduleEditedAfterRunDoesNotTriggerExtraRun(t *testing.T) {
	// run 17 Nov 02:01 WIB, lalu user mengubah jadwal 17 Nov 20:00 WIB menjadi 18:00
	sc := schedDaily("18:00", schedAt(2026, 11, 17, 20, 0))
	f, _ := applyRunStatus(&sc, schedStatus(schedAt(2026, 11, 17, 2, 1).UTC().Format(time.RFC3339), "SUCCEEDED", nil), time.Now())
	if got := schedNextOf(t, f); !got.Equal(schedAt(2026, 11, 18, 18, 0)) {
		t.Fatalf("next = %v, mau 18 Nov 18:00 WIB (bukan 17 Nov 18:00 yang sudah lewat)", got.In(schedWIB))
	}
}

// Slot yang sudah jatuh tempo tapi belum dijalankan tidak boleh terlewat.
func TestApply_OverdueSlotIsKept(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	// run terakhir 17 Nov 02:01 WIB -> slot berikutnya 18 Nov 02:00, tak peduli "sekarang" sudah 18 Nov 02:05
	f, _ := applyRunStatus(&sc, schedStatus(schedAt(2026, 11, 17, 2, 1).UTC().Format(time.RFC3339), "SUCCEEDED", nil), schedAt(2026, 11, 18, 2, 5))
	if got := schedNextOf(t, f); !got.Equal(schedAt(2026, 11, 18, 2, 0)) {
		t.Fatalf("next = %v, mau 18 Nov 02:00 WIB (jatuh tempo, belum jalan)", got.In(schedWIB))
	}
}

func TestApply_RunningOnlyUpdatesLastRun(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	f, changed := applyRunStatus(&sc, schedStatus("2026-11-17T19:00:00Z", "RUNNING", nil), time.Now())
	if !changed {
		t.Fatal("harus berubah")
	}
	if _, has := f["next_run_at"]; has {
		t.Fatalf("RUNNING tidak boleh mengubah next_run_at: %v", f)
	}
	if sc.NextRunAt != nil {
		t.Fatal("next_run_at di memori tidak boleh berubah")
	}
}

func TestApply_FailedKeepsMessageAndRecomputes(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	f, _ := applyRunStatus(&sc, schedStatus(schedAt(2026, 11, 17, 2, 3).UTC().Format(time.RFC3339), "FAILED", schedStrp("pull gagal")), time.Now())
	if m := f["last_run_message"].(*string); m == nil || *m != "pull gagal" {
		t.Fatalf("message: %v", f["last_run_message"])
	}
	if f["last_run_status"] != "FAILED" {
		t.Fatalf("status: %v", f["last_run_status"])
	}
	_ = schedNextOf(t, f)
}

func TestApply_InactiveDoesNotSetNextRun(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	sc.IsActive = false
	f, _ := applyRunStatus(&sc, schedStatus("2026-11-17T19:00:00Z", "SUCCEEDED", nil), time.Now())
	if _, has := f["next_run_at"]; has {
		t.Fatalf("jadwal nonaktif tidak boleh dapat next_run_at: %v", f)
	}
}

func TestApply_UnchangedIsNoop(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	st := schedStatus("2026-11-17T19:00:00Z", "SUCCEEDED", nil)
	if _, changed := applyRunStatus(&sc, st, time.Now()); !changed {
		t.Fatal("pertama harus berubah")
	}
	if f, changed := applyRunStatus(&sc, st, time.Now()); changed || f != nil {
		t.Fatalf("kedua harus no-op, fields=%v", f)
	}
}

func TestApply_InvalidInputsAreIgnored(t *testing.T) {
	sc := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	bad := []forecastingclient.SchedulerStatus{
		{Domain: "dn"}, // kosong
		schedStatus("2026-11-17T19:00:00Z", "PENDING", nil),
		schedStatus("bukan-tanggal", "SUCCEEDED", nil),
		{Domain: "dn", LastRunStatus: schedStrp("SUCCEEDED")}, // tanpa last_run_at
	}
	for i, st := range bad {
		if _, changed := applyRunStatus(&sc, st, time.Now()); changed {
			t.Fatalf("kasus %d harus diabaikan", i)
		}
	}
	if sc.LastRunAt != nil {
		t.Fatal("jadwal tidak boleh berubah")
	}
}

func TestApply_WeeklyAndMonthlyUseScheduleRules(t *testing.T) {
	w := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	w.Frequency, w.DayOfWeek = "weekly", schedStrp("monday")
	f, _ := applyRunStatus(&w, schedStatus(schedAt(2026, 11, 16, 2, 1).UTC().Format(time.RFC3339), "SUCCEEDED", nil), time.Now())
	if got := schedNextOf(t, f); !got.Equal(schedAt(2026, 11, 23, 2, 0)) {
		t.Fatalf("weekly next = %v", got.In(schedWIB))
	}
	m := schedDaily("02:00", schedAt(2026, 11, 13, 10, 0))
	m.Frequency, m.DayOfMonth = "monthly", schedIntp(31)
	f, _ = applyRunStatus(&m, schedStatus(schedAt(2026, 10, 31, 2, 1).UTC().Format(time.RFC3339), "SUCCEEDED", nil), time.Now())
	if got := schedNextOf(t, f); !got.Equal(schedAt(2026, 11, 30, 2, 0)) { // bulan pendek -> hari terakhir
		t.Fatalf("monthly next = %v", got.In(schedWIB))
	}
}

// ---- integrasi: ListSchedules + server forecasting palsu ----

type schedFakeRepo struct {
	repository.IRepository // method lain tidak dipakai di tes ini
	schedules              []models.ForecastingSchedule
	updates                []map[string]interface{}
	domains                []string
}

func (r *schedFakeRepo) ListSchedules(context.Context) ([]models.ForecastingSchedule, error) {
	return append([]models.ForecastingSchedule(nil), r.schedules...), nil
}
func (r *schedFakeRepo) FindSchedule(context.Context, string) (*models.ForecastingSchedule, error) {
	return nil, nil
}
func (r *schedFakeRepo) UpsertSchedule(context.Context, *models.ForecastingSchedule) error {
	return nil
}
func (r *schedFakeRepo) UpdateScheduleRun(_ context.Context, domain string, f map[string]interface{}) error {
	r.domains, r.updates = append(r.domains, domain), append(r.updates, f)
	return nil
}

func schedSvcWith(repo *schedFakeRepo, srv *httptest.Server) *service {
	return &service{repo: repo, client: forecastingclient.New(forecastingclient.Options{BaseURL: srv.URL, BasicAuthUser: "u", BasicAuthPass: "p"})}
}

const schedStatusBody = `[{"domain":"dn","last_run_at":"2026-11-17T20:01:25.420322+00:00","last_run_status":"SUCCEEDED","last_run_message":null,"last_training_run_id":"8838deac-0490-4939-955c-83c82c98a27a"}]`

func TestListSchedules_MergesPersistsAndIsIdempotent(t *testing.T) {
	var hit string
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit, gotAuth = r.Method+" "+r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(schedStatusBody))
	}))
	defer srv.Close()

	dn := schedDaily("03:00", schedAt(2026, 11, 13, 10, 12))
	prl := schedDaily("03:00", schedAt(2026, 11, 13, 10, 12))
	prl.Domain, prl.Frequency, prl.DayOfMonth = "prl", "monthly", schedIntp(1) // tidak pernah dijalankan scheduler
	repo := &schedFakeRepo{schedules: []models.ForecastingSchedule{dn, prl}}
	s := schedSvcWith(repo, srv)

	got, err := s.ListSchedules(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("err=%v len=%d", err, len(got))
	}
	if hit != "GET /admin/scheduler/status" || gotAuth == "" {
		t.Fatalf("hit=%q auth=%q", hit, gotAuth)
	}
	if got[0].LastRunStatus == nil || *got[0].LastRunStatus != "SUCCEEDED" || got[0].LastRunAt == nil || got[0].NextRunAt == nil {
		t.Fatalf("dn belum digabung: %+v", got[0])
	}
	if got[1].LastRunStatus != nil || got[1].LastRunAt != nil {
		t.Fatalf("prl (belum pernah jalan) tidak boleh berubah: %+v", got[1])
	}
	if len(repo.updates) != 1 || repo.domains[0] != "dn" {
		t.Fatalf("harus menyimpan tepat 1 (dn): %v %v", repo.domains, repo.updates)
	}

	// Pemanggilan berikutnya dengan DB yang sudah tersinkron: tidak menulis lagi.
	repo.schedules = []models.ForecastingSchedule{got[0], prl}
	if _, err := s.ListSchedules(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("sinkron kedua harus no-op, update=%d", len(repo.updates))
	}
}

func TestListSchedules_ForecastingDownKeepsStoredValues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // tidak ada yang listen
	dn := schedDaily("03:00", schedAt(2026, 11, 13, 10, 12))
	repo := &schedFakeRepo{schedules: []models.ForecastingSchedule{dn}}
	got, err := schedSvcWith(repo, srv).ListSchedules(context.Background())
	if err != nil || len(got) != 1 || got[0].LastRunStatus != nil || len(repo.updates) != 0 {
		t.Fatalf("harus tetap sukses tanpa perubahan: err=%v got=%+v updates=%d", err, got, len(repo.updates))
	}
}

func TestListSchedules_SlowForecastingTimesOut(t *testing.T) {
	old := schedulerStatusTimeout
	schedulerStatusTimeout = 150 * time.Millisecond
	defer func() { schedulerStatusTimeout = old }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
		_, _ = w.Write([]byte(schedStatusBody))
	}))
	defer srv.Close()
	repo := &schedFakeRepo{schedules: []models.ForecastingSchedule{schedDaily("03:00", schedAt(2026, 11, 13, 10, 12))}}
	start := time.Now()
	got, err := schedSvcWith(repo, srv).ListSchedules(context.Background())
	if err != nil || len(got) != 1 || len(repo.updates) != 0 {
		t.Fatalf("err=%v updates=%d", err, len(repo.updates))
	}
	if time.Since(start) > time.Second {
		t.Fatalf("tidak boleh menunggu forecasting lebih dari timeout: %v", time.Since(start))
	}
}

func TestListSchedules_ErrorStatusDoesNotBreakList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()
	repo := &schedFakeRepo{schedules: []models.ForecastingSchedule{schedDaily("03:00", schedAt(2026, 11, 13, 10, 12))}}
	if got, err := schedSvcWith(repo, srv).ListSchedules(context.Background()); err != nil || len(got) != 1 || len(repo.updates) != 0 {
		t.Fatalf("err=%v len=%d updates=%d", err, len(got), len(repo.updates))
	}
}

func TestListSchedulerStatus_EmptyArrayAndNull(t *testing.T) {
	for _, body := range []string{"[]", "null"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		c := forecastingclient.New(forecastingclient.Options{BaseURL: srv.URL})
		got, err := c.ListSchedulerStatus(context.Background())
		srv.Close()
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("%s: err=%v got=%v", body, err, got)
		}
	}
}
