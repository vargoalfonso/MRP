package service

import (
	"context"
	"time"

	"github.com/ganasa18/go-template/internal/main_dashboard/models"
	"github.com/ganasa18/go-template/internal/main_dashboard/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/concurrency"
)

const (
	TopCustomersLimit = 5
	UniqProgressLimit = 5
	TopSuppliersLimit = 5
)

type SummaryParams struct {
	Period    string
	StartDate string
	EndDate   string
}

type IService interface {
	GetSummary(ctx context.Context, req SummaryParams) (*models.Summary, error)
	GetRawMaterialSummary(ctx context.Context, req SummaryParams) (*models.RawMaterialSummary, error)
	GetListTables(ctx context.Context, schema string, limit int) (*models.ListTablesResponse, error)
}

type service struct {
	repo   repository.IRepository
	fanout concurrency.FanoutOptions
}

func New(repo repository.IRepository, fanout concurrency.FanoutOptions) IService {
	return &service{repo: repo, fanout: fanout}
}

func (s *service) GetSummary(ctx context.Context, req SummaryParams) (*models.Summary, error) {
	period, err := resolvePeriod(req.Period, req.StartDate, req.EndDate, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var (
		deliveryKPI          *models.DeliveryKPI
		currentProductionKPI *models.CurrentProductionKPI
		totalProductionKPI   *models.TotalProductionKPI
		poRawMaterialKPI     *models.PORawMaterialKPI
		deliveryPerf         *models.DeliveryPerformance
		productionPerf       *models.ProductionPerformance
		topCustomers         []models.TopCustomer
		uniqProgress         []models.UniqProgress
	)

	err = concurrency.Run(ctx, []concurrency.Task{
		func(c context.Context) error {
			result, repoErr := s.repo.GetDeliveryKPI(c, period.CurrentRange, period.PreviousRange)
			deliveryKPI = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetCurrentProductionKPI(c, period.CurrentRange, period.PreviousRange)
			currentProductionKPI = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetTotalProductionKPI(c, period.CurrentRange, period.PreviousRange)
			totalProductionKPI = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetPORawMaterialKPI(c, period.CurrentRange, period.PreviousRange)
			poRawMaterialKPI = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetDeliveryPerformance(c)
			deliveryPerf = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetProductionPerformance(c, period.CurrentRange)
			productionPerf = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetTopCustomers(c, period.CurrentRange, TopCustomersLimit)
			topCustomers = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetUniqProgress(c, UniqProgressLimit)
			uniqProgress = result
			return repoErr
		},
	}, s.fanout)
	if err != nil {
		return nil, err
	}

	return &models.Summary{
		AsOf: time.Now().UTC(),
		Period: models.PeriodMeta{
			Type:      period.Type,
			StartDate: period.CurrentRange.StartDate.Format("2006-01-02"),
			EndDate:   period.CurrentRange.EndDate.Format("2006-01-02"),
		},
		KPIs: models.KPIBundle{
			TotalDeliveries:   *deliveryKPI,
			CurrentProduction: *currentProductionKPI,
			TotalProduction:   *totalProductionKPI,
			PORawMaterial:     *poRawMaterialKPI,
		},
		DeliveryPerformance:   *deliveryPerf,
		ProductionPerformance: *productionPerf,
		TopCustomers:          topCustomers,
		CurrentUniqProgress:   uniqProgress,
	}, nil
}

func (s *service) GetRawMaterialSummary(ctx context.Context, req SummaryParams) (*models.RawMaterialSummary, error) {
	period, err := resolvePeriod(req.Period, req.StartDate, req.EndDate, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var (
		poSummary    *models.POSummary
		categoryDist []models.CategoryDistribution
		topSuppliers []models.TopSupplier
	)

	err = concurrency.Run(ctx, []concurrency.Task{
		func(c context.Context) error {
			result, repoErr := s.repo.GetPOSummary(c, period.CurrentRange)
			poSummary = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetCategoryDistribution(c)
			categoryDist = result
			return repoErr
		},
		func(c context.Context) error {
			result, repoErr := s.repo.GetTopSuppliers(c, TopSuppliersLimit)
			topSuppliers = result
			return repoErr
		},
	}, s.fanout)
	if err != nil {
		return nil, err
	}

	return &models.RawMaterialSummary{
		AsOf:                 time.Now().UTC(),
		POSummary:            *poSummary,
		CategoryDistribution: categoryDist,
		TopSuppliers:         topSuppliers,
	}, nil
}

func (s *service) GetListTables(ctx context.Context, schema string, limit int) (*models.ListTablesResponse, error) {
	return s.repo.ListTables(ctx, schema, limit)
}

type resolvedPeriod struct {
	Type          string
	CurrentRange  models.DateRange
	PreviousRange models.DateRange
}

func resolvePeriod(period, startDateRaw, endDateRaw string, now time.Time) (*resolvedPeriod, error) {
	if period == "" {
		period = "current_month"
	}

	start, end, err := currentRange(period, startDateRaw, endDateRaw, now)
	if err != nil {
		return nil, err
	}

	days := int(end.Sub(start).Hours()/24) + 1
	prevEnd := start.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -(days - 1))

	return &resolvedPeriod{
		Type: period,
		CurrentRange: models.DateRange{
			StartDate: start,
			EndDate:   end,
		},
		PreviousRange: models.DateRange{
			StartDate: prevStart,
			EndDate:   prevEnd,
		},
	}, nil
}

func currentRange(period, startDateRaw, endDateRaw string, now time.Time) (time.Time, time.Time, error) {
	today := dateOnly(now)
	switch period {
	case "today":
		return today, today, nil
	case "current_week":
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1))
		end := start.AddDate(0, 0, 6)
		return start, end, nil
	case "current_month":
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	case "current_quarter":
		quarterStartMonth := ((int(today.Month())-1)/3)*3 + 1
		start := time.Date(today.Year(), time.Month(quarterStartMonth), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 3, -1)
		return start, end, nil
	case "current_year":
		start := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(today.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
		return start, end, nil
	case "custom":
		if startDateRaw == "" || endDateRaw == "" {
			return time.Time{}, time.Time{}, apperror.BadRequest("start_date and end_date are required when period=custom")
		}
		start, err := time.Parse("2006-01-02", startDateRaw)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.BadRequest("invalid start_date format, expected YYYY-MM-DD")
		}
		end, err := time.Parse("2006-01-02", endDateRaw)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.BadRequest("invalid end_date format, expected YYYY-MM-DD")
		}
		if end.Before(start) {
			return time.Time{}, time.Time{}, apperror.BadRequest("end_date must be greater than or equal to start_date")
		}
		return dateOnly(start), dateOnly(end), nil
	default:
		return time.Time{}, time.Time{}, apperror.BadRequest("invalid period: use today|current_week|current_month|current_quarter|current_year|custom")
	}
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
