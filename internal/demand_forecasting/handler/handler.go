package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/demand_forecasting/models"
	"github.com/ganasa18/go-template/internal/demand_forecasting/repository"
	"github.com/ganasa18/go-template/internal/demand_forecasting/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/forecastingclient"
	userPkg "github.com/ganasa18/go-template/pkg/auth"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/ganasa18/go-template/pkg/validator"
)

type HTTPHandler struct {
	svc service.Service
}

func New(svc service.Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// ---------------------------------------------------------------------------
// Upload Dataset
// ---------------------------------------------------------------------------

func (h *HTTPHandler) UploadDataset(c *app.Context) *app.CostumeResponse {
	if err := c.Request.ParseMultipartForm(128 << 20); err != nil { // 128 MB
		return app.NewError(c, apperror.BadRequest("failed to parse multipart form: "+err.Error()))
	}

	req := models.UploadDatasetRequest{}
	req.RequestID = c.PostForm("request_id")
	req.Domain = c.PostForm("domain")
	req.SourceMode = c.PostForm("source_mode")
	req.Name = c.PostForm("name")
	req.Version = c.PostForm("version")
	req.Freq = c.PostForm("freq")
	req.Scope = c.PostForm("scope")
	req.Tenant = c.PostForm("tenant")
	req.Uniq = c.PostForm("uniq")

	if req.RequestID == "" {
		return app.NewError(c, apperror.BadRequest("request_id is required"))
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return app.NewError(c, apperror.BadRequest("file is required"))
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return app.NewError(c, apperror.BadRequest("failed to read file: "+err.Error()))
	}

	resp, err := h.svc.UploadDataset(c.Request.Context(), req, header.Filename, fileBytes)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Train Global
// ---------------------------------------------------------------------------

func (h *HTTPHandler) TrainGlobal(c *app.Context) *app.CostumeResponse {
	var req models.TrainGlobalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return app.NewError(c, apperror.BadRequest("invalid request body: "+err.Error()))
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	userID, err := mustUserID(c)
	if err != nil {
		return app.NewError(c, err)
	}

	resp, err := h.svc.TrainGlobal(c.Request.Context(), req, userID)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusCreated,
		Message:   "training triggered",
		Data:      resp,
	}
}

// ---------------------------------------------------------------------------
// Train Custom
// ---------------------------------------------------------------------------

func (h *HTTPHandler) TrainCustom(c *app.Context) *app.CostumeResponse {
	var req models.TrainCustomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return app.NewError(c, apperror.BadRequest("invalid request body: "+err.Error()))
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	userID, err := mustUserID(c)
	if err != nil {
		return app.NewError(c, err)
	}

	resp, err := h.svc.TrainCustom(c.Request.Context(), req, userID)
	if err != nil {
		return app.NewError(c, err)
	}

	return &app.CostumeResponse{
		RequestID: c.APIReqID,
		Status:    http.StatusCreated,
		Message:   "training triggered",
		Data:      resp,
	}
}

// ---------------------------------------------------------------------------
// Get Training Run Detail
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetTrainingRun(c *app.Context) *app.CostumeResponse {
	trainingRunID := c.Param("id")
	if trainingRunID == "" {
		return app.NewError(c, apperror.BadRequest("training run id is required"))
	}

	resp, err := h.svc.GetTrainingRun(c.Request.Context(), trainingRunID)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// List Training Runs
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListTrainingRuns(c *app.Context) *app.CostumeResponse {
	pg := pagination.ForecastingPagination(c)
	pagination := repository.PaginationInput{
		Limit:  pg.Limit,
		Page:   pg.Page,
		Offset: pg.Offset(),
	}

	filters := repository.TrainingRunFilters{
		Scope:  c.Query("scope"),
		Tenant: c.Query("tenant"),
		Uniq:   c.Query("uniq"),
		Domain: c.Query("domain"),
		Status: c.Query("status"),
	}

	resp, err := h.svc.ListTrainingRuns(c.Request.Context(), filters, pagination)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// List Datasets
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListDatasets(c *app.Context) *app.CostumeResponse {
	name := c.Query("name")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	resp, err := h.svc.ListDatasets(c.Request.Context(), name, limit)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// List Model Versions
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListModelVersions(c *app.Context) *app.CostumeResponse {
	opts := forecastingclient.ListModelVersionsOptions{
		Scope:  c.Query("scope"),
		Tenant: c.Query("tenant"),
		Uniq:   c.Query("uniq"),
		Status: c.Query("status"),
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			opts.Limit = n
		}
	}

	resp, err := h.svc.ListModelVersions(c.Request.Context(), opts)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// List Deployments
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListDeployments(c *app.Context) *app.CostumeResponse {
	opts := forecastingclient.ListDeploymentsOptions{
		Stage:  c.Query("stage"),
		Scope:  c.Query("scope"),
		Tenant: c.Query("tenant"),
		Uniq:   c.Query("uniq"),
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			opts.Limit = n
		}
	}

	resp, err := h.svc.ListDeployments(c.Request.Context(), opts)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Promote Model
// ---------------------------------------------------------------------------

func (h *HTTPHandler) PromoteModel(c *app.Context) *app.CostumeResponse {
	var req models.PromoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return app.NewError(c, apperror.BadRequest("invalid request body: "+err.Error()))
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	resp, err := h.svc.PromoteModel(c.Request.Context(), req)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Reload Model
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ReloadModel(c *app.Context) *app.CostumeResponse {
	domain := c.Query("domain")
	if domain == "" {
		domain = "dn"
	}

	if err := h.svc.ReloadModel(c.Request.Context(), domain); err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, map[string]interface{}{"message": "model reloaded", "domain": domain})
}

// ---------------------------------------------------------------------------
// Predict
// ---------------------------------------------------------------------------

func (h *HTTPHandler) Predict(c *app.Context) *app.CostumeResponse {
	var req models.PredictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return app.NewError(c, apperror.BadRequest("invalid request body: "+err.Error()))
	}
	if errs := validator.Validate(req); errs != nil {
		return &app.CostumeResponse{
			RequestID: c.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "validation failed",
			Data:      errs,
		}
	}

	userID, err := mustUserID(c)
	if err != nil {
		return app.NewError(c, err)
	}

	resp, err := h.svc.Predict(c.Request.Context(), req, userID)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Get Inference Result Detail
// ---------------------------------------------------------------------------

func (h *HTTPHandler) GetInferenceResult(c *app.Context) *app.CostumeResponse {
	id := c.Param("id")
	if id == "" {
		return app.NewError(c, apperror.BadRequest("id is required"))
	}

	resp, err := h.svc.GetInferenceResult(c.Request.Context(), id)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// List Inference Results
// ---------------------------------------------------------------------------

func (h *HTTPHandler) ListInferenceResults(c *app.Context) *app.CostumeResponse {
	pgInput := pagination.InferenceResultPagination(c)
	pagination := repository.PaginationInput{
		Limit:  pgInput.Limit,
		Page:   pgInput.Page,
		Offset: pgInput.Offset(),
	}

	filters := repository.InferenceResultFilters{
		Domain:         c.Query("domain"),
		Tenant:         c.Query("tenant"),
		ItemID:         c.Query("item_id"),
		Status:         c.Query("status"),
		ModelVersionID: c.Query("model_version_id"),
	}

	resp, err := h.svc.ListInferenceResults(c.Request.Context(), filters, pagination)
	if err != nil {
		return app.NewError(c, err)
	}

	return app.NewSuccess(c, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustUserID(c *app.Context) (string, error) {
	userCtx := userPkg.MustExtractUserContext(c)
	if userCtx.UserID == "" {
		return "", apperror.Unauthorized("user not authenticated")
	}
	return userCtx.UserID, nil
}
