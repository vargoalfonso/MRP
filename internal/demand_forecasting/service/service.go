package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/internal/demand_forecasting/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/forecastingclient"
	"github.com/google/uuid"
)

type Service interface {
	// Dataset
	UploadDataset(ctx context.Context, req models.UploadDatasetRequest, fileName string, fileBytes []byte) (*forecastingclient.UploadDatasetResponse, error)

	// Training
	TrainGlobal(ctx context.Context, req models.TrainGlobalRequest, createdBy string) (*forecastingclient.TrainResponse, error)
	TrainCustom(ctx context.Context, req models.TrainCustomRequest, createdBy string) (*forecastingclient.TrainResponse, error)
	GetTrainingRun(ctx context.Context, trainingRunID string) (*models.TrainingRunResponse, error)
	ListTrainingRuns(ctx context.Context, filters repository.TrainingRunFilters, pagination repository.PaginationInput) (*models.TrainingRunListResponse, error)

	// Lookup
	ListDatasets(ctx context.Context, name string, limit int) ([]forecastingclient.DatasetListItem, error)
	ListModelVersions(ctx context.Context, opts forecastingclient.ListModelVersionsOptions) ([]forecastingclient.ModelVersionItem, error)
	ListDeployments(ctx context.Context, opts forecastingclient.ListDeploymentsOptions) ([]forecastingclient.DeploymentItem, error)

	// Promote
	PromoteModel(ctx context.Context, req models.PromoteRequest) (*forecastingclient.PromoteResponse, error)
	ReloadModel(ctx context.Context, domain string) error

	// Predict
	Predict(ctx context.Context, req models.PredictRequest, createdBy string) (*models.PredictResponse, error)

	// Inference History
	GetInferenceResult(ctx context.Context, uuid string) (*models.InferenceResultResponse, error)
	ListInferenceResults(ctx context.Context, filters repository.InferenceResultFilters, pagination repository.PaginationInput) (*models.InferenceResultListResponse, error)
}

type service struct {
	repo   repository.IRepository
	client *forecastingclient.Client
}

func New(repo repository.IRepository, client *forecastingclient.Client) Service {
	return &service{repo: repo, client: client}
}

// ---------------------------------------------------------------------------
// Dataset
// ---------------------------------------------------------------------------

func (s *service) UploadDataset(ctx context.Context, req models.UploadDatasetRequest, fileName string, fileBytes []byte) (*forecastingclient.UploadDatasetResponse, error) {
	fields := map[string]string{
		"request_id": req.RequestID,
	}
	if req.Domain != "" {
		fields["domain"] = req.Domain
	}
	if req.SourceMode != "" {
		fields["source_mode"] = req.SourceMode
	}
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.Version != "" {
		fields["version"] = req.Version
	}
	if req.Freq != "" {
		fields["freq"] = req.Freq
	}
	if req.Scope != "" {
		fields["scope"] = req.Scope
	}
	if req.Tenant != "" {
		fields["tenant"] = req.Tenant
	}
	if req.Uniq != "" {
		fields["uniq"] = req.Uniq
	}

	resp, err := s.client.UploadDataset(ctx, fields, fileName, fileBytes)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Training
// ---------------------------------------------------------------------------

func (s *service) TrainGlobal(ctx context.Context, req models.TrainGlobalRequest, createdBy string) (*forecastingclient.TrainResponse, error) {
	// Call external API
	externalResp, err := s.client.TrainGlobal(ctx, forecastingclient.TrainGlobalRequest{
		RequestID: req.RequestID,
		Domain:    req.Domain,
		DatasetID: req.DatasetID,
		FineTune:  req.FineTune,
		TimeLimit: req.TimeLimit,
		Presets:   req.Presets,
	})
	if err != nil {
		return nil, err
	}

	// Persist to DB
	run := &models.ForecastingTrainingRun{
		UUID:          uuid.New().String(),
		RequestID:     req.RequestID,
		TrainingRunID: externalResp.TrainingRunID,
		Domain:        req.Domain,
		Scope:         "global",
		DatasetID:     req.DatasetID,
		FineTune:      req.FineTune,
		TimeLimit:     req.TimeLimit,
		Presets:       req.Presets,
		JobName:       externalResp.JobName,
		Region:        externalResp.Region,
		OperationName: externalResp.OperationName,
		Status:        externalResp.Status,
		CreatedBy:     createdBy,
	}

	if err := s.repo.CreateTrainingRun(ctx, run); err != nil {
		return nil, err
	}

	return externalResp, nil
}

func (s *service) TrainCustom(ctx context.Context, req models.TrainCustomRequest, createdBy string) (*forecastingclient.TrainResponse, error) {
	externalResp, err := s.client.TrainCustom(ctx, forecastingclient.TrainCustomRequest{
		RequestID: req.RequestID,
		Domain:    req.Domain,
		DatasetID: req.DatasetID,
		Tenant:    req.Tenant,
		Uniq:      req.Uniq,
		FineTune:  req.FineTune,
		TimeLimit: req.TimeLimit,
		Presets:   req.Presets,
	})
	if err != nil {
		return nil, err
	}

	run := &models.ForecastingTrainingRun{
		UUID:          uuid.New().String(),
		RequestID:     req.RequestID,
		TrainingRunID: externalResp.TrainingRunID,
		Domain:        req.Domain,
		Scope:         "custom",
		Tenant:        req.Tenant,
		Uniq:          req.Uniq,
		DatasetID:     req.DatasetID,
		FineTune:      req.FineTune,
		TimeLimit:     req.TimeLimit,
		Presets:       req.Presets,
		JobName:       externalResp.JobName,
		Region:        externalResp.Region,
		OperationName: externalResp.OperationName,
		Status:        externalResp.Status,
		CreatedBy:     createdBy,
	}

	if err := s.repo.CreateTrainingRun(ctx, run); err != nil {
		return nil, err
	}

	return externalResp, nil
}

func (s *service) GetTrainingRun(ctx context.Context, trainingRunID string) (*models.TrainingRunResponse, error) {
	// 1. Try find by trainingRunID in our DB
	localRun, err := s.repo.FindTrainingRunByExternalID(ctx, trainingRunID)
	if err != nil && !apperror.IsNotFound(err) {
		return nil, err
	}

	// 2. Hit external API for latest status
	externalDetail, extErr := s.client.GetTrainingRun(ctx, trainingRunID)

	if extErr != nil && apperror.IsNotFound(extErr) {
		// External not found — if we have local record, return it
		if localRun != nil {
			resp := models.ToTrainingRunResponse(localRun)
			return &resp, nil
		}
		return nil, apperror.NotFound("training run not found")
	}
	if extErr != nil {
		return nil, extErr
	}

	// 3. Upsert local record
	upsertData := &repository.ExternalTrainingRunDetail{
		RequestID:      localRun.RequestID,
		Domain:         externalDetail.Domain,
		Status:         externalDetail.Status,
		TrainingRunID:  externalDetail.TrainingRunID,
		JobName:        externalDetail.JobName,
		Region:         externalDetail.Region,
		Scope:          "custom",
		Tenant:         externalDetail.Tenant,
		Uniq:           externalDetail.Uniq,
		DatasetID:      externalDetail.DatasetID,
		DatasetName:    externalDetail.DatasetName,
		DatasetVersion: externalDetail.DatasetVersion,
		SourceMode:     externalDetail.SourceMode,
		GCSURI:         externalDetail.GCSURI,
		RowCount:       externalDetail.RowCount,
		ItemCount:      externalDetail.ItemCount,
		ModelVersionID: externalDetail.ModelVersionID,
		OperationName:  externalDetail.OperationName,
		ErrorMessage:   externalDetail.ErrorMessage,
	}

	if localRun != nil {
		upsertData.Scope = localRun.Scope
		upsertData.Tenant = localRun.Tenant
		upsertData.Uniq = localRun.Uniq
		upsertData.RequestID = localRun.RequestID
	}

	// Update local record with latest external status
	if localRun != nil {
		localRun.Status = externalDetail.Status
		localRun.JobName = externalDetail.JobName
		localRun.Region = externalDetail.Region
		localRun.OperationName = externalDetail.OperationName
		localRun.ErrorMessage = externalDetail.ErrorMessage
		if externalDetail.ModelVersionID != nil {
			localRun.ModelVersionID = *externalDetail.ModelVersionID
		}
		lastResp, _ := json.Marshal(externalDetail)
		localRun.LastResponse = lastResp
		localRun.UpdatedAt = time.Now()
		if err := s.repo.UpdateTrainingRun(ctx, localRun); err != nil {
			return nil, err
		}
		resp := models.ToTrainingRunResponse(localRun)
		return &resp, nil
	}

	return nil, apperror.NotFound("training run not found")
}

func (s *service) ListTrainingRuns(ctx context.Context, filters repository.TrainingRunFilters, pagination repository.PaginationInput) (*models.TrainingRunListResponse, error) {
	runs, total, err := s.repo.ListTrainingRuns(ctx, filters, pagination)
	if err != nil {
		return nil, err
	}

	items := make([]models.TrainingRunResponse, len(runs))
	for i, run := range runs {
		items[i] = models.ToTrainingRunResponse(&run)
	}

	pages := 0
	if pagination.Limit > 0 {
		pages = int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	}

	return &models.TrainingRunListResponse{
		Items: items,
		Pagination: models.PaginationMeta{
			Total:      total,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: pages,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Lookup
// ---------------------------------------------------------------------------

func (s *service) ListDatasets(ctx context.Context, name string, limit int) ([]forecastingclient.DatasetListItem, error) {
	return s.client.ListDatasets(ctx, name, limit)
}

func (s *service) ListModelVersions(ctx context.Context, opts forecastingclient.ListModelVersionsOptions) ([]forecastingclient.ModelVersionItem, error) {
	return s.client.ListModelVersions(ctx, opts)
}

func (s *service) ListDeployments(ctx context.Context, opts forecastingclient.ListDeploymentsOptions) ([]forecastingclient.DeploymentItem, error) {
	return s.client.ListDeployments(ctx, opts)
}

// ---------------------------------------------------------------------------
// Promote
// ---------------------------------------------------------------------------

func (s *service) PromoteModel(ctx context.Context, req models.PromoteRequest) (*forecastingclient.PromoteResponse, error) {
	return s.client.PromoteModel(ctx, forecastingclient.PromoteRequest{
		RequestID:      req.RequestID,
		Domain:         req.Domain,
		ModelVersionID: req.ModelVersionID,
		Stage:          req.Stage,
		Scope:          req.Scope,
		Tenant:         req.Tenant,
		Uniq:           req.Uniq,
	})
}

func (s *service) ReloadModel(ctx context.Context, domain string) error {
	return s.client.ReloadActiveModel(ctx, domain)
}

// ---------------------------------------------------------------------------
// Predict
// ---------------------------------------------------------------------------

func (s *service) Predict(ctx context.Context, req models.PredictRequest, createdBy string) (*models.PredictResponse, error) {
	// Convert observations
	var observations []forecastingclient.ObservationItem
	for _, obs := range req.Observations {
		observations = append(observations, forecastingclient.ObservationItem{
			ItemID:     obs.ItemID,
			Timestamp:  obs.Timestamp,
			Target:     obs.Target,
			Covariates: obs.Covariates,
		})
	}

	var futureCovariates []forecastingclient.FutureCovariateItem
	for _, fc := range req.FutureCovariates {
		futureCovariates = append(futureCovariates, forecastingclient.FutureCovariateItem{
			CovariateName: fc.CovariateName,
			Values:        fc.Values,
		})
	}

	// Determine mode
	mode := "manual"
	if req.AutoObservations {
		mode = "auto"
	}

	// Call external API
	externalResp, err := s.client.Predict(ctx, forecastingclient.PredictRequest{
		RequestID:         req.RequestID,
		Domain:            req.Domain,
		Tenant:            req.Tenant,
		AutoObservations:  req.AutoObservations,
		ItemID:            req.ItemID,
		Horizon:           req.Horizon,
		LookbackPoints:    req.LookbackPoints,
		Observations:      observations,
		FutureCovariates:  futureCovariates,
	})
	if err != nil {
		// Still persist the failed attempt
		reqBytes, _ := json.Marshal(req)
		result := &models.ForecastingInferenceResult{
			UUID:           uuid.New().String(),
			RequestID:      req.RequestID,
			Domain:         req.Domain,
			Tenant:         req.Tenant,
			ItemID:         req.ItemID,
			ModelVersionID: "",
			Horizon:        req.Horizon,
			LookbackPoints: req.LookbackPoints,
			Mode:           mode,
			RequestPayload: reqBytes,
			ResponsePayload: []byte("{}"),
			Status:         "FAILED",
			ErrorMessage:   err.Error(),
			CreatedBy:      createdBy,
		}
		_ = s.repo.CreateInferenceResult(ctx, result)
		return nil, err
	}

	// Persist result
	reqBytes, _ := json.Marshal(req)
	respBytes, _ := json.Marshal(externalResp)

	result := &models.ForecastingInferenceResult{
		UUID:            uuid.New().String(),
		RequestID:       req.RequestID,
		Domain:          req.Domain,
		Tenant:          req.Tenant,
		ItemID:          req.ItemID,
		ModelVersionID:  externalResp.ModelVersionID,
		Horizon:         req.Horizon,
		LookbackPoints:  req.LookbackPoints,
		Mode:            mode,
		RequestPayload:  reqBytes,
		ResponsePayload: respBytes,
		Status:          "SUCCEEDED",
		CreatedBy:       createdBy,
	}

	if err := s.repo.CreateInferenceResult(ctx, result); err != nil {
		// Non-fatal — log but still return response
	}

	// Build response
	forecasts := make([]models.ForecastDTO, len(externalResp.Forecasts))
	for i, f := range externalResp.Forecasts {
		forecasts[i] = models.ForecastDTO{
			ItemID:    f.ItemID,
			Timestamp: f.Timestamp,
			Mean:      f.Mean,
			P10:       f.P10,
			P50:       f.P50,
			P90:       f.P90,
		}
	}

	return &models.PredictResponse{
		RequestID:      externalResp.RequestID,
		ModelVersionID: externalResp.ModelVersionID,
		Forecasts:      forecasts,
	}, nil
}

// ---------------------------------------------------------------------------
// Inference History
// ---------------------------------------------------------------------------

func (s *service) GetInferenceResult(ctx context.Context, uuid string) (*models.InferenceResultResponse, error) {
	result, err := s.repo.FindInferenceResultByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	resp := models.ToInferenceResultResponse(result)
	return &resp, nil
}

func (s *service) ListInferenceResults(ctx context.Context, filters repository.InferenceResultFilters, pagination repository.PaginationInput) (*models.InferenceResultListResponse, error) {
	results, total, err := s.repo.ListInferenceResults(ctx, filters, pagination)
	if err != nil {
		return nil, err
	}

	items := make([]models.InferenceResultResponse, len(results))
	for i, r := range results {
		items[i] = models.ToInferenceResultResponse(&r)
	}

	pages := 0
	if pagination.Limit > 0 {
		pages = int((total + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	}

	return &models.InferenceResultListResponse{
		Items: items,
		Pagination: models.PaginationMeta{
			Total:      total,
			Page:       pagination.Page,
			Limit:      pagination.Limit,
			TotalPages: pages,
		},
	}, nil
}
