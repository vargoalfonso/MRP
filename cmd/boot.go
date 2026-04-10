package cmd

import (
	"fmt"
	"log/slog"

	appconf "github.com/ganasa18/go-template/config"
	"github.com/ganasa18/go-template/http/server"
	userModule "github.com/ganasa18/go-template/internal/access_control"
	userHandler "github.com/ganasa18/go-template/internal/access_control/handler"
	userRepository "github.com/ganasa18/go-template/internal/access_control/repository"
	userService "github.com/ganasa18/go-template/internal/access_control/service"
	approvalWorkflowModule "github.com/ganasa18/go-template/internal/approval_workflow"
	approvalWorkflowHandler "github.com/ganasa18/go-template/internal/approval_workflow/handler"
	approvalWorkflowRepository "github.com/ganasa18/go-template/internal/approval_workflow/repository"
	approvalWorkflowService "github.com/ganasa18/go-template/internal/approval_workflow/service"
	authModule "github.com/ganasa18/go-template/internal/auth"
	authHandler "github.com/ganasa18/go-template/internal/auth/handler"
	authRepository "github.com/ganasa18/go-template/internal/auth/repository"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseModule "github.com/ganasa18/go-template/internal/base"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	bomModule "github.com/ganasa18/go-template/internal/billmaterial"
	bomHandler "github.com/ganasa18/go-template/internal/billmaterial/handler"
	bomRepository "github.com/ganasa18/go-template/internal/billmaterial/repository"
	bomService "github.com/ganasa18/go-template/internal/billmaterial/service"
	departementModule "github.com/ganasa18/go-template/internal/departement"
	departementHandler "github.com/ganasa18/go-template/internal/departement/handler"
	departementRepository "github.com/ganasa18/go-template/internal/departement/repository"
	departementService "github.com/ganasa18/go-template/internal/departement/service"
	employeeModule "github.com/ganasa18/go-template/internal/employee"
	employeeHandler "github.com/ganasa18/go-template/internal/employee/handler"
	employeeRepository "github.com/ganasa18/go-template/internal/employee/repository"
	employeeService "github.com/ganasa18/go-template/internal/employee/service"
	globalParameterModule "github.com/ganasa18/go-template/internal/global_parameter"
	globalParameterHandler "github.com/ganasa18/go-template/internal/global_parameter/handler"
	globalParameterRepository "github.com/ganasa18/go-template/internal/global_parameter/repository"
	globalParameterService "github.com/ganasa18/go-template/internal/global_parameter/service"
	kanbanModule "github.com/ganasa18/go-template/internal/kanban"
	kanbanHandler "github.com/ganasa18/go-template/internal/kanban/handler"
	kanbanRepository "github.com/ganasa18/go-template/internal/kanban/repository"
	kanbanService "github.com/ganasa18/go-template/internal/kanban/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	poBudgetModule "github.com/ganasa18/go-template/internal/po_budget"
	procModule "github.com/ganasa18/go-template/internal/procurement"
	procHandler "github.com/ganasa18/go-template/internal/procurement/handler"
	procRepository "github.com/ganasa18/go-template/internal/procurement/repository"
	procService "github.com/ganasa18/go-template/internal/procurement/service"
	poBudgetHandler "github.com/ganasa18/go-template/internal/po_budget/handler"
	poBudgetRepository "github.com/ganasa18/go-template/internal/po_budget/repository"
	poBudgetService "github.com/ganasa18/go-template/internal/po_budget/service"
	poSplitSettingModule "github.com/ganasa18/go-template/internal/po_split_setting"
	poSplitSettingHandler "github.com/ganasa18/go-template/internal/po_split_setting/handler"
	poSplitSettingRepository "github.com/ganasa18/go-template/internal/po_split_setting/repository"
	poSplitSettingService "github.com/ganasa18/go-template/internal/po_split_setting/service"
	processParameterModule "github.com/ganasa18/go-template/internal/process_parameter"
	processParameterHandler "github.com/ganasa18/go-template/internal/process_parameter/handler"
	processParameterRepository "github.com/ganasa18/go-template/internal/process_parameter/repository"
	processParameterService "github.com/ganasa18/go-template/internal/process_parameter/service"
	roleModule "github.com/ganasa18/go-template/internal/role"
	roleHandler "github.com/ganasa18/go-template/internal/role/handler"
	roleRepository "github.com/ganasa18/go-template/internal/role/repository"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	safetyStockModule "github.com/ganasa18/go-template/internal/safety_stock_parameter"
	safetyStockHandler "github.com/ganasa18/go-template/internal/safety_stock_parameter/handler"
	safetyStockRepo "github.com/ganasa18/go-template/internal/safety_stock_parameter/repository"
	safetyStockService "github.com/ganasa18/go-template/internal/safety_stock_parameter/service"
	typeParameterModule "github.com/ganasa18/go-template/internal/type_parameter"
	typeParameterHandler "github.com/ganasa18/go-template/internal/type_parameter/handler"
	typeParameterRepository "github.com/ganasa18/go-template/internal/type_parameter/repository"
	typeParameterService "github.com/ganasa18/go-template/internal/type_parameter/service"
	UnitMeasureModule "github.com/ganasa18/go-template/internal/unit_measurement"
	UnitMeasureHandler "github.com/ganasa18/go-template/internal/unit_measurement/handler"
	UnitMeasureRepository "github.com/ganasa18/go-template/internal/unit_measurement/repository"
	UnitMeasureService "github.com/ganasa18/go-template/internal/unit_measurement/service"
	uploadModule "github.com/ganasa18/go-template/internal/upload"
	uploadHandler "github.com/ganasa18/go-template/internal/upload/handler"
	uploadRepository "github.com/ganasa18/go-template/internal/upload/repository"
	uploadService "github.com/ganasa18/go-template/internal/upload/service"
)

// initHTTP wires every module inside the modular monolith and returns an HTTP server.
func initHTTP(cfg *appconf.Config) (*server.Server, error) {
	// --- Database ---
	db, err := appconf.NewDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("database init: %w", err)
	}
	slog.Info("database connected", slog.String("name", cfg.DBName))

	// --- Redis (only required for stateful JWT mode) ---
	var authSvc authService.Authenticator
	// if cfg.IsStateful() {
	// 	rdb, err := appconf.NewRedis(cfg)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("redis init: %w", err)
	// 	}
	// 	slog.Info("redis connected", slog.String("addr", cfg.RedisHost+":"+cfg.RedisPort))

	// 	authRepo := authRepository.New(db)
	// 	authSvc = authService.New(cfg, authRepo, rdb)
	// } else {
	// }
	authRepo := authRepository.New(db)
	authSvc = authService.New(cfg, authRepo, nil)

	roleRepo := roleRepository.New(db)
	roleSvc := roleService.New(roleRepo)

	departementRepo := departementRepository.New(db)
	departementSvc := departementService.New(departementRepo)
	departementHTTPHandler := departementHandler.New(departementSvc)

	baseHTTPHandler := baseHandler.NewBaseHTTPHandler(db)
	authHTTPHandler := authHandler.New(authSvc)
	roleHTTPHandler := roleHandler.New(roleSvc)

	employeeRepo := employeeRepository.New(db)
	employeeSvc := employeeService.New(employeeRepo)
	employeeHTTPHandler := employeeHandler.New(employeeSvc)

	userRepo := userRepository.New(db)
	userSvc := userService.New(userRepo, roleRepo, authRepo, employeeRepo, departementRepo)
	userHTTPHandler := userHandler.New(userSvc, authSvc)

	// BOM module
	bomRepo := bomRepository.New(db)
	bomSvc := bomService.New(bomRepo)
	bomHTTPHandler := bomHandler.New(bomSvc)

	// Upload module (chunked / resumable)
	uploadRepo := uploadRepository.New(db)
	uploadSvc := uploadService.New(uploadRepo, bomRepo)
	uploadHTTPHandler := uploadHandler.New(uploadSvc)

	// PO Budget module
	poBudgetRepo := poBudgetRepository.New(db)
	poBudgetSvc := poBudgetService.New(poBudgetRepo)
	poBudgetHTTPHandler := poBudgetHandler.New(poBudgetSvc)

	// Procurement module
	procRepo := procRepository.New(db)
	procSvc := procService.New(procRepo)
	procHTTPHandler := procHandler.New(procSvc)

	safetyStockRepository := safetyStockRepo.New(db)
	safetyStockService := safetyStockService.New(safetyStockRepository)
	safetyStockHandler := safetyStockHandler.New(safetyStockService)

	typeParameterRepo := typeParameterRepository.New(db)
	typeParameterSvc := typeParameterService.New(typeParameterRepo)
	typeParameterHTTPHandler := typeParameterHandler.New(typeParameterSvc)

	unitMeasureRepo := UnitMeasureRepository.New(db)
	unitMeasureSvc := UnitMeasureService.New(unitMeasureRepo)
	unitMeasureHTTPHandler := UnitMeasureHandler.New(unitMeasureSvc)

	poSplitSettingRepo := poSplitSettingRepository.New(db)
	poSplitSettingSvc := poSplitSettingService.New(poSplitSettingRepo)
	poSplitSettingHTTPHandler := poSplitSettingHandler.New(poSplitSettingSvc)

	approvalWorkflowRepo := approvalWorkflowRepository.New(db)
	approvalWorkflowSvc := approvalWorkflowService.New(approvalWorkflowRepo)
	approvalWorkflowHTTPHandler := approvalWorkflowHandler.New(approvalWorkflowSvc)

	globalParameterRepo := globalParameterRepository.New(db)
	globalParameterSvc := globalParameterService.New(globalParameterRepo)
	globalParameterHTTPHandler := globalParameterHandler.New(globalParameterSvc)

	processParameterRepo := processParameterRepository.New(db)
	processParameterSvc := processParameterService.New(processParameterRepo)
	processParameterHTTPHandler := processParameterHandler.New(processParameterSvc)

	kanbanRepo := kanbanRepository.New(db)
	kanbanSvc := kanbanService.New(kanbanRepo)
	kanbanHTTPHandler := kanbanHandler.New(kanbanSvc)

	modules := []appmodule.HTTPModule{
		baseModule.NewHTTPModule(baseHTTPHandler),
		authModule.NewHTTPModule(cfg, baseHTTPHandler, authHTTPHandler, authSvc),
		bomModule.NewHTTPModule(cfg, baseHTTPHandler, bomHTTPHandler, authSvc, roleSvc, bomSvc),
		uploadModule.NewHTTPModule(baseHTTPHandler, uploadHTTPHandler, uploadSvc),
		roleModule.NewHTTPModule(cfg, baseHTTPHandler, roleHTTPHandler, authSvc, roleSvc),
		departementModule.NewHTTPModule(cfg, baseHTTPHandler, departementHTTPHandler, authSvc, roleSvc, departementSvc),
		employeeModule.NewHTTPModule(cfg, baseHTTPHandler, employeeHTTPHandler, authSvc, roleSvc, employeeSvc),
		userModule.NewHTTPModule(cfg, baseHTTPHandler, userHTTPHandler, authSvc, roleSvc, userSvc),
		poBudgetModule.NewHTTPModule(cfg, baseHTTPHandler, poBudgetHTTPHandler, authSvc, roleSvc),
		procModule.NewHTTPModule(cfg, baseHTTPHandler, procHTTPHandler, authSvc, roleSvc),
		safetyStockModule.NewHTTPModule(cfg, baseHTTPHandler, safetyStockHandler, authSvc, roleSvc, safetyStockService),
		typeParameterModule.NewHTTPModule(cfg, baseHTTPHandler, typeParameterHTTPHandler, authSvc, roleSvc, typeParameterSvc),
		UnitMeasureModule.NewHTTPModule(cfg, baseHTTPHandler, unitMeasureHTTPHandler, authSvc, roleSvc, unitMeasureSvc),
		poSplitSettingModule.NewHTTPModule(cfg, baseHTTPHandler, poSplitSettingHTTPHandler, authSvc, roleSvc, poSplitSettingSvc),
		approvalWorkflowModule.NewHTTPModule(cfg, baseHTTPHandler, approvalWorkflowHTTPHandler, authSvc, roleSvc, approvalWorkflowSvc),
		globalParameterModule.NewHTTPModule(cfg, baseHTTPHandler, globalParameterHTTPHandler, authSvc, roleSvc, globalParameterSvc),
		processParameterModule.NewHTTPModule(cfg, baseHTTPHandler, processParameterHTTPHandler, authSvc, roleSvc, processParameterSvc),
		kanbanModule.NewHTTPModule(cfg, baseHTTPHandler, kanbanHTTPHandler, authSvc, roleSvc, kanbanSvc),
	}

	// --- Server ---
	srv := server.New(cfg, modules)
	return srv, nil
}
