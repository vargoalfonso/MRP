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
	actionUIModule "github.com/ganasa18/go-template/internal/action_ui"
	actionUIHandler "github.com/ganasa18/go-template/internal/action_ui/handler"
	actionUIIncomingRepo "github.com/ganasa18/go-template/internal/action_ui/repository"
	actionUIProductionRepo "github.com/ganasa18/go-template/internal/action_ui/repository"
	actionUIRepo "github.com/ganasa18/go-template/internal/action_ui/repository"
	actionUIService "github.com/ganasa18/go-template/internal/action_ui/service"
	adminJobsModule "github.com/ganasa18/go-template/internal/admin_jobs"
	adminJobsHandler "github.com/ganasa18/go-template/internal/admin_jobs/handler"
	adminJobsRepo "github.com/ganasa18/go-template/internal/admin_jobs/repository"
	adminJobsService "github.com/ganasa18/go-template/internal/admin_jobs/service"
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
	customerModule "github.com/ganasa18/go-template/internal/customer"
	customerHandler "github.com/ganasa18/go-template/internal/customer/handler"
	customerRepository "github.com/ganasa18/go-template/internal/customer/repository"
	customerService "github.com/ganasa18/go-template/internal/customer/service"
	coModule "github.com/ganasa18/go-template/internal/customer_order"
	coHandler "github.com/ganasa18/go-template/internal/customer_order/handler"
	coRepository "github.com/ganasa18/go-template/internal/customer_order/repository"
	coService "github.com/ganasa18/go-template/internal/customer_order/service"
	deliveryNoteModule "github.com/ganasa18/go-template/internal/delivery_note"
	deliveryNoteHandler "github.com/ganasa18/go-template/internal/delivery_note/handler"
	deliveryNoteRepository "github.com/ganasa18/go-template/internal/delivery_note/repository"
	deliveryNoteService "github.com/ganasa18/go-template/internal/delivery_note/service"
	dscModule "github.com/ganasa18/go-template/internal/delivery_scheduling_customer"
	dscHandler "github.com/ganasa18/go-template/internal/delivery_scheduling_customer/handler"
	dscRepository "github.com/ganasa18/go-template/internal/delivery_scheduling_customer/repository"
	dscService "github.com/ganasa18/go-template/internal/delivery_scheduling_customer/service"
	departementModule "github.com/ganasa18/go-template/internal/departement"
	departementHandler "github.com/ganasa18/go-template/internal/departement/handler"
	departementRepository "github.com/ganasa18/go-template/internal/departement/repository"
	departementService "github.com/ganasa18/go-template/internal/departement/service"
	employeeModule "github.com/ganasa18/go-template/internal/employee"
	employeeHandler "github.com/ganasa18/go-template/internal/employee/handler"
	employeeRepository "github.com/ganasa18/go-template/internal/employee/repository"
	employeeService "github.com/ganasa18/go-template/internal/employee/service"
	finishedGoodsModule "github.com/ganasa18/go-template/internal/finished_goods"
	finishedGoodsHandler "github.com/ganasa18/go-template/internal/finished_goods/handler"
	finishedGoodsRepo "github.com/ganasa18/go-template/internal/finished_goods/repository"
	finishedGoodsService "github.com/ganasa18/go-template/internal/finished_goods/service"
	globalParameterModule "github.com/ganasa18/go-template/internal/global_parameter"
	globalParameterHandler "github.com/ganasa18/go-template/internal/global_parameter/handler"
	globalParameterRepository "github.com/ganasa18/go-template/internal/global_parameter/repository"
	globalParameterService "github.com/ganasa18/go-template/internal/global_parameter/service"
	inventoryModule "github.com/ganasa18/go-template/internal/inventory"
	inventoryHandler "github.com/ganasa18/go-template/internal/inventory/handler"
	inventoryRepo "github.com/ganasa18/go-template/internal/inventory/repository"
	inventoryService "github.com/ganasa18/go-template/internal/inventory/service"
	kanbanModule "github.com/ganasa18/go-template/internal/kanban"
	kanbanHandler "github.com/ganasa18/go-template/internal/kanban/handler"
	kanbanRepository "github.com/ganasa18/go-template/internal/kanban/repository"
	kanbanService "github.com/ganasa18/go-template/internal/kanban/service"
	masterMachineModule "github.com/ganasa18/go-template/internal/master_machine"
	masterMachineHandler "github.com/ganasa18/go-template/internal/master_machine/handler"
	masterMachineRepository "github.com/ganasa18/go-template/internal/master_machine/repository"
	masterMachineService "github.com/ganasa18/go-template/internal/master_machine/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	outgoingModule "github.com/ganasa18/go-template/internal/outgoing_material"
	outgoingHandler "github.com/ganasa18/go-template/internal/outgoing_material/handler"
	outgoingRepo "github.com/ganasa18/go-template/internal/outgoing_material/repository"
	outgoingService "github.com/ganasa18/go-template/internal/outgoing_material/service"
	poBudgetModule "github.com/ganasa18/go-template/internal/po_budget"
	poBudgetHandler "github.com/ganasa18/go-template/internal/po_budget/handler"
	poBudgetRepository "github.com/ganasa18/go-template/internal/po_budget/repository"
	poBudgetService "github.com/ganasa18/go-template/internal/po_budget/service"
	poSplitSettingModule "github.com/ganasa18/go-template/internal/po_split_setting"
	poSplitSettingHandler "github.com/ganasa18/go-template/internal/po_split_setting/handler"
	poSplitSettingRepository "github.com/ganasa18/go-template/internal/po_split_setting/repository"
	poSplitSettingService "github.com/ganasa18/go-template/internal/po_split_setting/service"
	prlModule "github.com/ganasa18/go-template/internal/prl"
	prlHandler "github.com/ganasa18/go-template/internal/prl/handler"
	prlRepository "github.com/ganasa18/go-template/internal/prl/repository"
	prlService "github.com/ganasa18/go-template/internal/prl/service"
	processParameterModule "github.com/ganasa18/go-template/internal/process_parameter"
	processParameterHandler "github.com/ganasa18/go-template/internal/process_parameter/handler"
	processParameterRepository "github.com/ganasa18/go-template/internal/process_parameter/repository"
	processParameterService "github.com/ganasa18/go-template/internal/process_parameter/service"
	procModule "github.com/ganasa18/go-template/internal/procurement"
	procHandler "github.com/ganasa18/go-template/internal/procurement/handler"
	procRepository "github.com/ganasa18/go-template/internal/procurement/repository"
	procService "github.com/ganasa18/go-template/internal/procurement/service"
	productionModule "github.com/ganasa18/go-template/internal/production"
	productionHandler "github.com/ganasa18/go-template/internal/production/handler"
	productionRepository "github.com/ganasa18/go-template/internal/production/repository"
	productionService "github.com/ganasa18/go-template/internal/production/service"
	qcModule "github.com/ganasa18/go-template/internal/qc"
	qcHandler "github.com/ganasa18/go-template/internal/qc/handler"
	qcRepo "github.com/ganasa18/go-template/internal/qc/repository"
	qcService "github.com/ganasa18/go-template/internal/qc/service"
	qcDashboardModule "github.com/ganasa18/go-template/internal/qc_dashboard"
	qcDashboardHandler "github.com/ganasa18/go-template/internal/qc_dashboard/handler"
	qcDashboardRepository "github.com/ganasa18/go-template/internal/qc_dashboard/repository"
	qcDashboardService "github.com/ganasa18/go-template/internal/qc_dashboard/service"
	roleModule "github.com/ganasa18/go-template/internal/role"
	roleHandler "github.com/ganasa18/go-template/internal/role/handler"
	roleRepository "github.com/ganasa18/go-template/internal/role/repository"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	safetyStockModule "github.com/ganasa18/go-template/internal/safety_stock_parameter"
	safetyStockHandler "github.com/ganasa18/go-template/internal/safety_stock_parameter/handler"
	safetyStockRepo "github.com/ganasa18/go-template/internal/safety_stock_parameter/repository"
	safetyStockService "github.com/ganasa18/go-template/internal/safety_stock_parameter/service"
	scrapModule "github.com/ganasa18/go-template/internal/scrap_stock"
	scrapHandler "github.com/ganasa18/go-template/internal/scrap_stock/handler"
	scrapRepo "github.com/ganasa18/go-template/internal/scrap_stock/repository"
	scrapService "github.com/ganasa18/go-template/internal/scrap_stock/service"
	shopFloorModule "github.com/ganasa18/go-template/internal/shop_floor"
	shopFloorHandler "github.com/ganasa18/go-template/internal/shop_floor/handler"
	shopFloorRepository "github.com/ganasa18/go-template/internal/shop_floor/repository"
	shopFloorService "github.com/ganasa18/go-template/internal/shop_floor/service"
	stockOpnameModule "github.com/ganasa18/go-template/internal/stock_opname"
	stockOpnameHandler "github.com/ganasa18/go-template/internal/stock_opname/handler"
	stockOpnameRepo "github.com/ganasa18/go-template/internal/stock_opname/repository"
	stockOpnameService "github.com/ganasa18/go-template/internal/stock_opname/service"
	supplierModule "github.com/ganasa18/go-template/internal/supplier"
	supplierHandler "github.com/ganasa18/go-template/internal/supplier/handler"
	supplierRepository "github.com/ganasa18/go-template/internal/supplier/repository"
	supplierService "github.com/ganasa18/go-template/internal/supplier/service"
	supplierItemModule "github.com/ganasa18/go-template/internal/supplier_item"
	supplierItemHandler "github.com/ganasa18/go-template/internal/supplier_item/handler"
	supplierItemRepository "github.com/ganasa18/go-template/internal/supplier_item/repository"
	supplierItemService "github.com/ganasa18/go-template/internal/supplier_item/service"
	spModule "github.com/ganasa18/go-template/internal/supplier_performance"
	spHandler "github.com/ganasa18/go-template/internal/supplier_performance/handler"
	spRepository "github.com/ganasa18/go-template/internal/supplier_performance/repository"
	spService "github.com/ganasa18/go-template/internal/supplier_performance/service"
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
	warehouseModule "github.com/ganasa18/go-template/internal/warehouse"
	warehouseHandler "github.com/ganasa18/go-template/internal/warehouse/handler"
	warehouseRepository "github.com/ganasa18/go-template/internal/warehouse/repository"
	warehouseService "github.com/ganasa18/go-template/internal/warehouse/service"
	wipModule "github.com/ganasa18/go-template/internal/wip"
	wipHandler "github.com/ganasa18/go-template/internal/wip/handler"
	wipRepository "github.com/ganasa18/go-template/internal/wip/repository"
	wipService "github.com/ganasa18/go-template/internal/wip/service"
	workOrderModule "github.com/ganasa18/go-template/internal/work_order"
	workOrderHandler "github.com/ganasa18/go-template/internal/work_order/handler"
	workOrderRepository "github.com/ganasa18/go-template/internal/work_order/repository"
	workOrderService "github.com/ganasa18/go-template/internal/work_order/service"
	"github.com/ganasa18/go-template/pkg/concurrency"
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
	customerRepo := customerRepository.New(db)
	customerSvc := customerService.New(customerRepo)
	customerHTTPHandler := customerHandler.New(customerSvc)
	prlRepo := prlRepository.New(db)
	prlSvc := prlService.New(prlRepo)
	prlHTTPHandler := prlHandler.New(prlSvc)
	supplierRepo := supplierRepository.New(db)
	supplierSvc := supplierService.New(supplierRepo)
	supplierHTTPHandler := supplierHandler.New(supplierSvc)
	supplierItemRepo := supplierItemRepository.New(db)
	supplierItemSvc := supplierItemService.New(supplierItemRepo)
	supplierItemHTTPHandler := supplierItemHandler.New(supplierItemSvc)
	warehouseRepo := warehouseRepository.New(db)
	warehouseSvc := warehouseService.New(warehouseRepo)
	warehouseHTTPHandler := warehouseHandler.New(warehouseSvc)
	roleHTTPHandler := roleHandler.New(roleSvc)

	employeeRepo := employeeRepository.New(db)
	employeeSvc := employeeService.New(employeeRepo, authRepo)
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
	uploadSvc := uploadService.New(uploadRepo, bomRepo, cfg.MaxChunkBytes)
	uploadHTTPHandler := uploadHandler.New(uploadSvc)

	// PO Budget module
	poBudgetRepo := poBudgetRepository.New(db)
	poBudgetSvc := poBudgetService.New(poBudgetRepo, db, cfg.RobotSplitURL)
	poBudgetHTTPHandler := poBudgetHandler.New(poBudgetSvc)

	// Procurement module
	procRepo := procRepository.New(db)
	procSvc := procService.New(procRepo)
	procHTTPHandler := procHandler.New(procSvc)

	// Action UI module (incoming scans)
	actionRepo := actionUIRepo.New(db)
	actionUIProductionRepo := actionUIProductionRepo.NewProductionRepository(db)
	actionUIIncomingRepo := actionUIIncomingRepo.NewIncomingRepository(db)
	actionSvc := actionUIService.New(actionRepo, actionUIProductionRepo, actionUIIncomingRepo, db)
	actionHTTPHandler := actionUIHandler.New(actionSvc)

	shopFloorRepo := shopFloorRepository.New(db)
	shopFloorSvc := shopFloorService.New(shopFloorRepo, concurrency.DefaultFanout)
	shopFloorHTTPHandler := shopFloorHandler.New(shopFloorSvc)

	// QC module (task list + approve/reject)
	qcRepository := qcRepo.New(db)
	qcSvc := qcService.New(qcRepository)
	qcHTTPHandler := qcHandler.New(qcSvc)
	qcDashboardRepo := qcDashboardRepository.New(db)
	qcDashboardSvc := qcDashboardService.New(qcDashboardRepo, concurrency.DefaultFanout)
	qcDashboardHTTPHandler := qcDashboardHandler.New(qcDashboardSvc)

	// Inventory module (RM database, Indirect RM, Subcon)
	invRepository := inventoryRepo.New(db)
	invSvc := inventoryService.New(invRepository)
	invHTTPHandler := inventoryHandler.New(invSvc)

	// Work Order module (Manufacturing)
	woRepo := workOrderRepository.New(db)
	woSvc := workOrderService.New(woRepo, db, invSvc)
	woHTTPHandler := workOrderHandler.New(woSvc)

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
	approvalWorkflowSvc := approvalWorkflowService.New(approvalWorkflowRepo, roleRepo)
	approvalWorkflowHTTPHandler := approvalWorkflowHandler.New(approvalWorkflowSvc)

	globalParameterRepo := globalParameterRepository.New(db)
	globalParameterSvc := globalParameterService.New(globalParameterRepo)
	globalParameterHTTPHandler := globalParameterHandler.New(globalParameterSvc)

	processParameterRepo := processParameterRepository.New(db)
	processParameterSvc := processParameterService.New(processParameterRepo)
	processParameterHTTPHandler := processParameterHandler.New(processParameterSvc)

	masterMachineRepo := masterMachineRepository.New(db)
	masterMachineSvc := masterMachineService.New(masterMachineRepo)
	masterMachineHTTPHandler := masterMachineHandler.New(masterMachineSvc)

	kanbanRepo := kanbanRepository.New(db)
	kanbanSvc := kanbanService.New(kanbanRepo)
	kanbanHTTPHandler := kanbanHandler.New(kanbanSvc)

	deliveryNoteRepo := deliveryNoteRepository.New(db)
	deliveryNoteSvc := deliveryNoteService.New(deliveryNoteRepo, db, approvalWorkflowRepo)
	deliveryNoteHTTPHandler := deliveryNoteHandler.New(deliveryNoteSvc)

	productionRepo := productionRepository.New(db)
	productionSvc := productionService.New(productionRepo)
	productionHTTPHandler := productionHandler.New(productionSvc)

	// Customer Order module (PO / DN / SO)
	coRepo := coRepository.New(db)
	coSvc := coService.New(coRepo)
	coHTTPHandler := coHandler.New(coSvc)

	// Admin Jobs module
	adminJobsRepository := adminJobsRepo.New(db)
	adminJobsSvc := adminJobsService.New(adminJobsRepository)
	adminJobsHTTPHandler := adminJobsHandler.New(adminJobsSvc)

	// Finished Goods module
	fgRepository := finishedGoodsRepo.New(db)
	fgSvc := finishedGoodsService.New(fgRepository, db)
	fgHTTPHandler := finishedGoodsHandler.New(fgSvc)

	// Scrap Stock module
	outgoingRepository := outgoingRepo.New(db)
	outgoingSvc := outgoingService.New(outgoingRepository, db)
	outgoingHTTPHandler := outgoingHandler.New(outgoingSvc)

	scrapRepository := scrapRepo.New(db)
	scrapSvc := scrapService.New(scrapRepository, db)
	scrapHTTPHandler := scrapHandler.New(scrapSvc)
	stockOpnameRepository := stockOpnameRepo.New(db)
	stockOpnameSvc := stockOpnameService.New(stockOpnameRepository, db, invSvc)
	stockOpnameHTTPHandler := stockOpnameHandler.New(stockOpnameSvc)

	// Delivery Scheduling Customer module (outbound customer delivery)
	dscRepo := dscRepository.New(db)
	dscSvc := dscService.New(dscRepo, db)
	dscHTTPHandler := dscHandler.New(dscSvc)

	// Supplier Performance module
	spRepo := spRepository.New(db)
	spSvc := spService.New(spRepo)
	spHTTPHandler := spHandler.New(spSvc)
	wipRepo := wipRepository.New(db)
	wipSvc := wipService.New(wipRepo)
	wipHTTPHandler := wipHandler.New(wipSvc)

	modules := []appmodule.HTTPModule{
		baseModule.NewHTTPModule(baseHTTPHandler),
		authModule.NewHTTPModule(cfg, baseHTTPHandler, authHTTPHandler, authSvc),
		customerModule.NewHTTPModule(cfg, baseHTTPHandler, customerHTTPHandler, authSvc),
		prlModule.NewHTTPModule(cfg, baseHTTPHandler, prlHTTPHandler, authSvc),
		supplierModule.NewHTTPModule(cfg, baseHTTPHandler, supplierHTTPHandler, authSvc),
		supplierItemModule.NewHTTPModule(cfg, baseHTTPHandler, supplierItemHTTPHandler, authSvc),
		bomModule.NewHTTPModule(cfg, baseHTTPHandler, bomHTTPHandler, authSvc, roleSvc, bomSvc),
		uploadModule.NewHTTPModule(baseHTTPHandler, uploadHTTPHandler, uploadSvc, authSvc, roleSvc),
		roleModule.NewHTTPModule(cfg, baseHTTPHandler, roleHTTPHandler, authSvc, roleSvc),
		warehouseModule.NewHTTPModule(cfg, baseHTTPHandler, warehouseHTTPHandler, authSvc),
		departementModule.NewHTTPModule(cfg, baseHTTPHandler, departementHTTPHandler, authSvc, roleSvc, departementSvc),
		employeeModule.NewHTTPModule(cfg, baseHTTPHandler, employeeHTTPHandler, authSvc, roleSvc, employeeSvc),
		userModule.NewHTTPModule(cfg, baseHTTPHandler, userHTTPHandler, authSvc, roleSvc, userSvc),
		poBudgetModule.NewHTTPModule(cfg, baseHTTPHandler, poBudgetHTTPHandler, authSvc, roleSvc),
		procModule.NewHTTPModule(cfg, baseHTTPHandler, procHTTPHandler, authSvc, roleSvc),
		actionUIModule.NewHTTPModule(cfg, baseHTTPHandler, actionHTTPHandler, authSvc, roleSvc),
		shopFloorModule.NewHTTPModule(cfg, baseHTTPHandler, shopFloorHTTPHandler, authSvc, roleSvc, shopFloorSvc),
		qcModule.NewHTTPModule(cfg, baseHTTPHandler, qcHTTPHandler, authSvc, roleSvc),
		qcDashboardModule.NewHTTPModule(cfg, baseHTTPHandler, qcDashboardHTTPHandler, authSvc, roleSvc, qcDashboardSvc),
		inventoryModule.NewHTTPModule(cfg, baseHTTPHandler, invHTTPHandler, authSvc, roleSvc, invSvc),
		workOrderModule.NewHTTPModule(cfg, baseHTTPHandler, woHTTPHandler, authSvc, roleSvc, woSvc),
		safetyStockModule.NewHTTPModule(cfg, baseHTTPHandler, safetyStockHandler, authSvc, roleSvc, safetyStockService),
		typeParameterModule.NewHTTPModule(cfg, baseHTTPHandler, typeParameterHTTPHandler, authSvc, roleSvc, typeParameterSvc),
		UnitMeasureModule.NewHTTPModule(cfg, baseHTTPHandler, unitMeasureHTTPHandler, authSvc, roleSvc, unitMeasureSvc),
		poSplitSettingModule.NewHTTPModule(cfg, baseHTTPHandler, poSplitSettingHTTPHandler, authSvc, roleSvc, poSplitSettingSvc),
		approvalWorkflowModule.NewHTTPModule(cfg, baseHTTPHandler, approvalWorkflowHTTPHandler, authSvc, roleSvc, approvalWorkflowSvc),
		globalParameterModule.NewHTTPModule(cfg, baseHTTPHandler, globalParameterHTTPHandler, authSvc, roleSvc, globalParameterSvc),
		processParameterModule.NewHTTPModule(cfg, baseHTTPHandler, processParameterHTTPHandler, authSvc, roleSvc, processParameterSvc),
		masterMachineModule.NewHTTPModule(cfg, baseHTTPHandler, masterMachineHTTPHandler, authSvc, roleSvc, masterMachineSvc),
		kanbanModule.NewHTTPModule(cfg, baseHTTPHandler, kanbanHTTPHandler, authSvc, roleSvc, kanbanSvc),
		deliveryNoteModule.NewHTTPModule(cfg, baseHTTPHandler, deliveryNoteHTTPHandler, authSvc, roleSvc, deliveryNoteSvc),
		productionModule.NewHTTPModule(cfg, baseHTTPHandler, productionHTTPHandler, authSvc, roleSvc, productionSvc),
		outgoingModule.NewHTTPModule(cfg, baseHTTPHandler, outgoingHTTPHandler, authSvc, roleSvc, outgoingSvc),
		scrapModule.NewHTTPModule(cfg, baseHTTPHandler, scrapHTTPHandler, authSvc, roleSvc, scrapSvc),
		stockOpnameModule.NewHTTPModule(cfg, baseHTTPHandler, stockOpnameHTTPHandler, authSvc, roleSvc, stockOpnameSvc),
		finishedGoodsModule.NewHTTPModule(cfg, baseHTTPHandler, fgHTTPHandler, authSvc, roleSvc, fgSvc),
		wipModule.NewHTTPModule(cfg, baseHTTPHandler, wipHTTPHandler, authSvc, roleSvc, wipSvc),
		adminJobsModule.NewHTTPModule(cfg, baseHTTPHandler, adminJobsHTTPHandler, adminJobsSvc),
		coModule.NewHTTPModule(cfg, baseHTTPHandler, coHTTPHandler, authSvc, roleSvc, coSvc),
		dscModule.NewHTTPModule(baseHTTPHandler, dscHTTPHandler, authSvc, roleSvc, dscSvc),
		spModule.NewHTTPModule(baseHTTPHandler, spHTTPHandler, authSvc, roleSvc),
	}

	// --- Server ---
	srv := server.New(cfg, modules)
	return srv, nil
}
