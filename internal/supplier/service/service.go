package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier/models"
	supplierRepo "github.com/ganasa18/go-template/internal/supplier/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/email"
	"github.com/google/uuid"
)

type SupplierService interface {
	Create(ctx context.Context, req models.CreateSupplierRequest) (*models.Supplier, error)
	GetByUUID(ctx context.Context, uuid string) (*models.Supplier, error)
	List(ctx context.Context, query models.ListSupplierQuery) (*models.SupplierListResult, error)
	Update(ctx context.Context, uuid string, req models.UpdateSupplierRequest) (*models.Supplier, error)
	Delete(ctx context.Context, uuid string) error
}

type service struct {
	repo supplierRepo.IRepository
}

func New(repo supplierRepo.IRepository) SupplierService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req models.CreateSupplierRequest) (*models.Supplier, error) {
	materialCategory, err := normalizeMaterialCategory(req.MaterialCategory)
	if err != nil {
		return nil, err
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return nil, err
	}

	supplierCode := strings.ToUpper(strings.TrimSpace(req.SupplierCode))
	if supplierCode == "" {
		supplierCode = "TMP-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	}

	supplier := &models.Supplier{
		UUID:                 uuid.NewString(),
		SupplierCode:         supplierCode,
		SupplierName:         models.Trimmed(req.SupplierName),
		ContactPerson:        models.Trimmed(req.ContactPerson),
		ContactNumber:        models.Trimmed(req.ContactNumber),
		EmailAddress:         strings.ToLower(models.Trimmed(req.EmailAddress)),
		MaterialCategory:     materialCategory,
		FullAddress:          models.Trimmed(req.FullAddress),
		City:                 models.Trimmed(req.City),
		Province:             models.Trimmed(req.Province),
		Country:              models.Trimmed(req.Country),
		TaxIDNPWP:            models.Trimmed(req.TaxIDNPWP),
		BankName:             models.Trimmed(req.BankName),
		BankAccountNumber:    models.Trimmed(req.BankAccountNumber),
		BankAccountName:      models.Trimmed(req.BankAccountName),
		PaymentTerms:         models.Trimmed(req.PaymentTerms),
		DeliveryLeadTimeDays: req.DeliveryLeadTimeDays,
		Status:               status,
	}

	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}

	// Send notification email to supplier if email provided (async, non-blocking)
	if strings.TrimSpace(supplier.EmailAddress) != "" {

		go func(to, name string) {

			subject := "Selamat Datang Sebagai Supplier"

			body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Welcome Supplier</title>
</head>

<body style="
	margin:0;
	padding:0;
	background-color:#f3f4f6;
	font-family:Arial,sans-serif;
">

	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0" style="
					background:#ffffff;
					border-radius:14px;
					overflow:hidden;
					box-shadow:0 4px 18px rgba(0,0,0,0.06);
				">

					<!-- HEADER -->
					<tr>
						<td style="
							background:#029cde;
							padding:22px 30px;
						">

							<table width="100%%" cellpadding="0" cellspacing="0">
								<tr>

									<!-- LOGO -->
									<td align="left">
										<img
											src="https://raigine.com/wp-content/themes/company-profile-theme/images/logo-raigine.png"
											alt="Raigine Logo"
											width="145"
											style="display:block;"
										>
									</td>

									<!-- TITLE -->
									<td align="right" style="
										font-size:15px;
										font-weight:500;
										color:#d1d5db;
										letter-spacing:0.5px;
									">
										Supplier Registration
									</td>

								</tr>
							</table>

						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:48px 42px;">

							<h2 style="
								margin-top:0;
								margin-bottom:20px;
								color:#111827;
								font-size:30px;
							">
								Halo %s,
							</h2>

							<p style="
								font-size:16px;
								line-height:1.9;
								color:#4b5563;
								margin:0 0 18px 0;
							">
								Selamat! Anda telah berhasil terdaftar sebagai
								<b>supplier</b> di sistem kami.
							</p>

							<p style="
								font-size:16px;
								line-height:1.9;
								color:#4b5563;
								margin:0 0 18px 0;
							">
								Terima kasih telah bergabung dan menjadi bagian dari kerja sama kami.
							</p>

							<p style="
								font-size:16px;
								line-height:1.9;
								color:#4b5563;
								margin:0;
							">
								Kami berharap dapat menjalin kerja sama yang baik dan profesional ke depannya.
							</p>

							<!-- INFO BOX -->
							<div style="
								background:#f9fafb;
								border:1px solid #e5e7eb;
								border-radius:10px;
								padding:18px;
								margin-top:40px;
							">

								<p style="
									font-size:14px;
									color:#6b7280;
									line-height:1.8;
									margin:0;
								">
									Jika Anda memiliki pertanyaan lebih lanjut,
									silakan hubungi tim kami melalui email resmi perusahaan.
								</p>

							</div>

						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="
							background:#f9fafb;
							padding:22px;
							text-align:center;
							font-size:12px;
							color:#9ca3af;
							border-top:1px solid #e5e7eb;
						">
							© 2026 Raigine System. All rights reserved.
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>

</body>
</html>
`, supplier.SupplierName)

			if err := email.SendEmail(
				[]string{to},
				subject,
				body,
			); err != nil {

				// tidak menggagalkan proses create supplier
				fmt.Println("failed send supplier email:", err)
			}

		}(supplier.EmailAddress, supplier.SupplierName)
	}

	return supplier, nil
}

func (s *service) GetByUUID(ctx context.Context, uuid string) (*models.Supplier, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, apperror.BadRequest("supplier id is required")
	}

	return s.repo.FindByUUID(ctx, uuid)
}

func (s *service) List(ctx context.Context, query models.ListSupplierQuery) (*models.SupplierListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	status, err := normalizeOptionalStatus(query.Status)
	if err != nil {
		return nil, err
	}

	materialCategory, err := normalizeMaterialCategory(query.MaterialCategory)
	if err != nil {
		return nil, err
	}

	filters := models.SupplierListFilters{
		Search:           models.Trimmed(query.Search),
		Status:           status,
		MaterialCategory: materialCategory,
		UniqCode:         models.Trimmed(query.UniqCode),
		Page:             page,
		Limit:            limit,
		Offset:           (page - 1) * limit,
	}

	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &models.SupplierListResult{
		Items:      items,
		Pagination: models.NewPaginationMeta(page, limit, total),
	}, nil
}

func (s *service) Update(ctx context.Context, uuid string, req models.UpdateSupplierRequest) (*models.Supplier, error) {
	supplier, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	materialCategory, err := normalizeMaterialCategory(req.MaterialCategory)
	if err != nil {
		return nil, err
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return nil, err
	}

	supplier.SupplierName = models.Trimmed(req.SupplierName)
	supplier.ContactPerson = models.Trimmed(req.ContactPerson)
	supplier.ContactNumber = models.Trimmed(req.ContactNumber)
	supplier.EmailAddress = strings.ToLower(models.Trimmed(req.EmailAddress))
	supplier.MaterialCategory = materialCategory
	supplier.FullAddress = models.Trimmed(req.FullAddress)
	supplier.City = models.Trimmed(req.City)
	supplier.Province = models.Trimmed(req.Province)
	supplier.Country = models.Trimmed(req.Country)
	supplier.TaxIDNPWP = models.Trimmed(req.TaxIDNPWP)
	supplier.BankName = models.Trimmed(req.BankName)
	supplier.BankAccountNumber = models.Trimmed(req.BankAccountNumber)
	supplier.BankAccountName = models.Trimmed(req.BankAccountName)
	supplier.PaymentTerms = models.Trimmed(req.PaymentTerms)
	supplier.DeliveryLeadTimeDays = req.DeliveryLeadTimeDays
	supplier.Status = status

	if err := s.repo.Update(ctx, supplier); err != nil {
		return nil, err
	}

	return supplier, nil
}

func (s *service) Delete(ctx context.Context, uuid string) error {
	supplier, err := s.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, supplier)
}

func normalizeMaterialCategory(value string) (*string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" {
		return nil, nil
	}

	switch strings.ToLower(cleaned) {
	case "raw material":
		result := models.MaterialCategoryRawMaterial
		return &result, nil
	case "indirect raw material":
		result := models.MaterialCategoryIndirectRawMaterial
		return &result, nil
	case "subcon":
		result := models.MaterialCategorySubcon
		return &result, nil
	default:
		return nil, apperror.BadRequest("material_category must be empty or one of: Raw Material, Indirect Raw Material, Subcon")
	}
}

func normalizeStatus(value string) (string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if cleaned == "" {
		return "", apperror.BadRequest("status is required")
	}

	switch strings.ToLower(cleaned) {
	case "active":
		return models.SupplierStatusActive, nil
	case "inactive":
		return models.SupplierStatusInactive, nil
	default:
		return "", apperror.BadRequest("status must be one of: Active, Inactive")
	}
}

func normalizeOptionalStatus(value string) (*string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	status, err := normalizeStatus(value)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
