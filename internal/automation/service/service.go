package service

import (
	"context"

	"github.com/ganasa18/go-template/config"
	"github.com/ganasa18/go-template/internal/automation/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/raigineclient"
)

// IService is the automation integration service contract.
type IService interface {
	RunProcess(ctx context.Context, processPublicID string, req models.RunRequest) (map[string]interface{}, error)
	StopProcess(ctx context.Context, processPublicID string) (map[string]interface{}, error)
	ListProcesses(ctx context.Context, p raigineclient.ListParams) (map[string]interface{}, error)
	ListJobs(ctx context.Context, p raigineclient.ListParams) (map[string]interface{}, error)
	CreateSchedule(ctx context.Context, req models.ScheduleRequest) (map[string]interface{}, error)
}

type svc struct {
	client     *raigineclient.Client
	raigineCfg config.RaigineConfig
}

// New constructs the automation service.
func New(client *raigineclient.Client, raigineCfg config.RaigineConfig) IService {
	return &svc{client: client, raigineCfg: raigineCfg}
}

func (s *svc) ensure() error {
	if s.client == nil || !s.client.Enabled() {
		return apperror.New(503, apperror.CodeServiceUnavail,
			"automation platform (Raigine) is not configured; set RAIGINE_API_BASE_URL and a token or credentials")
	}
	return nil
}

func (s *svc) RunProcess(ctx context.Context, processPublicID string, req models.RunRequest) (map[string]interface{}, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	if processPublicID == "" {
		return nil, apperror.BadRequest("process id is required")
	}
	return s.client.RunProcess(ctx, processPublicID, raigineclient.RunProcessRequest{
		JobPublicID: req.JobPublicID,
	})
}

func (s *svc) StopProcess(ctx context.Context, processPublicID string) (map[string]interface{}, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	if processPublicID == "" {
		return nil, apperror.BadRequest("process id is required")
	}
	return s.client.StopProcess(ctx, processPublicID)
}

func (s *svc) ListProcesses(ctx context.Context, p raigineclient.ListParams) (map[string]interface{}, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	if p.FolderID == "" {
		p.FolderID = s.raigineCfg.DefaultFolderID
	}
	return s.client.ListProcesses(ctx, p)
}

func (s *svc) ListJobs(ctx context.Context, p raigineclient.ListParams) (map[string]interface{}, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	if p.FolderID == "" {
		p.FolderID = s.raigineCfg.DefaultFolderID
	}
	return s.client.ListJobs(ctx, p)
}

func (s *svc) CreateSchedule(ctx context.Context, req models.ScheduleRequest) (map[string]interface{}, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	folderID := req.FolderID
	if folderID == "" {
		folderID = s.raigineCfg.DefaultFolderID
	}
	timezone := req.Timezone
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	return s.client.CreateSchedule(ctx, raigineclient.CreateScheduleRequest{
		ScheduleName:        req.ScheduleName,
		AutomationProcessID: req.AutomationProcessID,
		TriggerPriority:     req.TriggerPriority,
		ExecutionFrequency:  req.ExecutionFrequency,
		ExecuteTimes:        req.ExecuteTimes,
		StartTime:           req.StartTime,
		Timezone:            timezone,
		FolderID:            folderID,
		RepeatInterval:      req.RepeatInterval,
		DaysOfWeek:          req.DaysOfWeek,
		DaysOfMonth:         req.DaysOfMonth,
		AutoDisable:         req.AutoDisable,
		DisableDate:         req.DisableDate,
		AutoEnd:             req.AutoEnd,
		EndDate:             req.EndDate,
		AlertIfStuck:        req.AlertIfStuck,
	})
}
