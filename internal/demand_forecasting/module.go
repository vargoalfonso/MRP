package demand_forecasting

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	dfHandler "github.com/ganasa18/go-template/internal/demand_forecasting/handler"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base         *baseHandler.BaseHTTPHandler
	handler      *dfHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *dfHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	g := r.Group("/api/v1/forecasting")
	g.Use(auth)
	{
		// Dataset upload
		g.POST("/datasets/upload",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "create"),
			m.base.RunAction(m.handler.UploadDataset))

		// Training
		g.POST("/train/global",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "create"),
			m.base.RunAction(m.handler.TrainGlobal))
		g.POST("/train/custom",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "create"),
			m.base.RunAction(m.handler.TrainCustom))

		// Training runs (list + detail)
		g.GET("/training-runs",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.ListTrainingRuns))
		g.GET("/training-runs/:id",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.GetTrainingRun))

		// Dataset / model / deployment lookups (proxy only)
		g.GET("/datasets",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.ListDatasets))
		g.GET("/model-versions",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.ListModelVersions))
		g.GET("/deployments",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.ListDeployments))

		// Promote / Reload
		g.POST("/promote",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "approve"),
			m.base.RunAction(m.handler.PromoteModel))
		g.POST("/reload",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "approve"),
			m.base.RunAction(m.handler.ReloadModel))

		// Predict
		g.POST("/predict",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "create"),
			m.base.RunAction(m.handler.Predict))

		// Inference results (DB-backed)
		g.GET("/inference-results",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.ListInferenceResults))
		g.GET("/inference-results/:id",
			roleMiddleware.RequirePermission(m.roleService, "forecasting", "view"),
			m.base.RunAction(m.handler.GetInferenceResult))
	}
}
