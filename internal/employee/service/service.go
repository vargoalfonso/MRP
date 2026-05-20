package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/ganasa18/go-template/internal/employee/models"
	employeeRepo "github.com/ganasa18/go-template/internal/employee/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	email "github.com/ganasa18/go-template/pkg/email"
	"github.com/google/uuid"
)

type IEmployeeService interface {
	GetAll(ctx context.Context) ([]models.Employee, error)
	GetByID(ctx context.Context, id int64) (*models.Employee, error)
	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error)
	Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo     employeeRepo.IEmployeeRepository
	authRepo authRepo.IRepository
}

func New(repo employeeRepo.IEmployeeRepository, authRepo authRepo.IRepository) IEmployeeService {
	return &service{
		repo:     repo,
		authRepo: authRepo,
	}
}

//
// =========================
// CRUD
// =========================
//

func (s *service) GetAll(ctx context.Context) ([]models.Employee, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	if req.ReportsToID != nil {
		if _, err := s.repo.FindByID(ctx, *req.ReportsToID); err != nil {
			return nil, apperror.BadRequest("reports_to_id tidak valid")
		}
	}
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

//
// =========================
// 🔥 CREATE EMPLOYEE FLOW
// =========================
//

func (s *service) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {
	var employee *models.Employee

	if req.DepartmentID != nil && *req.DepartmentID <= 0 {
		req.DepartmentID = nil
	}
	if req.ReportsToID != nil && *req.ReportsToID <= 0 {
		req.ReportsToID = nil
	}

	err := s.repo.Tx(ctx, func(txRepo employeeRepo.IEmployeeRepository) error {

		// ==============================
		// 🔥 1. VALIDASI EMAIL
		// ==============================
		exist, err := txRepo.IsEmailExist(ctx, req.Email)
		if err != nil {
			return err
		}
		if exist {
			return apperror.Conflict("employee email sudah terdaftar")
		}

		_, err = s.authRepo.FindByEmail(ctx, req.Email)
		if err == nil {
			return apperror.Conflict("user email sudah terdaftar")
		}

		if req.ReportsToID != nil {
			if _, err := txRepo.FindByID(ctx, *req.ReportsToID); err != nil {
				return apperror.BadRequest("reports_to_id tidak valid")
			}
		}

		// ==============================
		// 🔥 2. CREATE EMPLOYEE
		// ==============================
		emp, err := txRepo.Create(ctx, req)
		if err != nil {
			return err
		}
		employee = emp

		// ==============================
		// 🔥 3. CREATE USER (DALAM TX)
		// ==============================
		user := authModels.User{
			Username:   req.Email,
			Email:      req.Email,
			Password:   "",
			Roles:      "user",
			IsVerified: false,
		}

		// 🔥 IMPORTANT: harus pakai TX
		if err := s.authRepo.Create(ctx, &user); err != nil {
			return err
		}

		// ==============================
		// 🔥 4. GENERATE TOKEN
		// ==============================
		token := uuid.NewString()

		activation := models.UserActivation{
			UserID:    user.ID,
			Token:     token,
			ExpiredAt: time.Now().Add(24 * time.Hour),
			Used:      false,
			CreatedAt: time.Now(),
		}

		if err := txRepo.SaveActivationToken(ctx, &activation); err != nil {
			return err
		}

		// ==============================
		// 🔥 5. SEND EMAIL (ASYNC)
		// ==============================
		go func(to, name, token string) {

			link := fmt.Sprintf("%s/set-password?token=%s", os.Getenv("BASE_URL"), token)

			subject := "Set Password Account"

			body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Set Password</title>
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
										Account Activation
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
								Akun Anda telah berhasil dibuat di sistem kami.
							</p>

							<p style="
								font-size:16px;
								line-height:1.9;
								color:#4b5563;
								margin:0;
							">
								Silakan klik tombol di bawah untuk membuat password akun Anda.
							</p>

							<!-- BUTTON -->
							<div style="
								text-align:center;
								margin:45px 0;
							">
								<a href="%s"
									style="
										background:#2563eb;
										color:white;
										padding:15px 34px;
										text-decoration:none;
										border-radius:10px;
										font-size:15px;
										font-weight:600;
										display:inline-block;
										box-shadow:0 4px 12px rgba(37,99,235,0.25);
									">
									Set Password
								</a>
							</div>

							<!-- INFO -->
							<p style="
								font-size:14px;
								color:#6b7280;
								line-height:1.8;
								margin-bottom:30px;
							">
								Link ini berlaku selama 24 jam.
							</p>

							<!-- FALLBACK -->
							<div style="
								background:#f9fafb;
								border:1px solid #e5e7eb;
								border-radius:10px;
								padding:18px;
							">

								<p style="
									font-size:13px;
									color:#6b7280;
									margin-top:0;
									margin-bottom:10px;
								">
									Jika tombol tidak dapat diklik, gunakan link berikut:
								</p>

								<a href="%s"
									style="
										font-size:14px;
										color:#2563eb;
										word-break:break-all;
										text-decoration:none;
									">
									%s
								</a>

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
`, name, link, link, link)

			if err := email.SendEmail(
				[]string{to},
				subject,
				body,
			); err != nil {
				log.Println("failed send email:", err)
			}

		}(req.Email, req.FullName, token)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return employee, nil
}
