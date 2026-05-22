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
	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.CreateEmployee, error)
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

func (s *service) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.CreateEmployee, error) {
	var employee *models.CreateEmployee

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

			link := fmt.Sprintf(
				"%s/set-password?token=%s",
				os.Getenv("BASE_URL"),
				token,
			)

			// format tanggal jam
			currentDate := time.Now().Format("Mon, Jan 02 2006 15:04")

			subject := "Set Password Account"

			body := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>Set Password</title>
</head>

<body style="
	margin:0;
	padding:0;
	background:#f3f4f6;
	font-family:Arial,sans-serif;
">

	<table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
		<tr>
			<td align="center">

				<table width="620" cellpadding="0" cellspacing="0" style="
					background:#ffffff;
					border-radius:14px;
					overflow:hidden;
					box-shadow:0 4px 18px rgba(0,0,0,0.06);
				">

					<!-- HEADER -->
					<tr>
						<td style="
							padding:26px 34px;
							background:#ffffff;
							border-bottom:1px solid #e5e7eb;
						">

							<table width="100%" cellpadding="0" cellspacing="0">
								<tr>

									<!-- LOGO -->
									<td align="left">
										<img
											src="https://raigine.com/wp-content/themes/company-profile-theme/images/logo-raigine.png"
											alt="Raigine Logo"
											width="150"
											style="display:block;"
										/>
									</td>

									<!-- DATE -->
									<td align="right" style="
										font-size:14px;
										color:#9ca3af;
										font-weight:500;
									">
										%s
									</td>

								</tr>
							</table>

						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:50px 46px;">

							<h1 style="
								margin-top:0;
								margin-bottom:28px;
								font-size:44px;
								line-height:1.2;
								color:#111827;
								font-weight:700;
							">
								Change Your Password
							</h1>

							<h3 style="
								margin:0 0 20px 0;
								font-size:22px;
								color:#111827;
							">
								Password Reset Request 🔐
							</h3>

							<p style="
								font-size:17px;
								line-height:1.9;
								color:#374151;
								margin:0 0 20px 0;
							">
								Hi %s,
							</p>

							<p style="
								font-size:16px;
								line-height:1.9;
								color:#4b5563;
								margin:0 0 18px 0;
							">
								We received a request to reset your password for your Raigine account.
							</p>

							<div style="text-align:center; margin:42px 0;">
								<a href="%s"
									style="
										background:#1d4ed8;
										color:#ffffff;
										text-decoration:none;
										padding:15px 38px;
										border-radius:8px;
										display:inline-block;
										font-size:16px;
										font-weight:600;
									">
									Reset Password
								</a>
							</div>

							<p style="
								font-size:14px;
								color:#6b7280;
								line-height:1.8;
							">
								This link will expire in 24 hours.
							</p>

						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="
							background:#f9fafb;
							padding:30px 40px;
							border-top:1px solid #e5e7eb;
						">

							<table width="100%" cellpadding="0" cellspacing="0">
								<tr>

									<td align="left">
										<img
											src="https://raigine.com/wp-content/themes/company-profile-theme/images/logo-raigine.png"
											width="90"
											style="
												display:block;
												margin-bottom:14px;
											"
										/>

										<p style="
											margin:0;
											font-size:13px;
											color:#9ca3af;
											line-height:1.7;
										">
											Copyright © 2026 <br/>
											All rights reserved
										</p>
									</td>

								</tr>
							</table>

						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>

</body>
</html>
`,
				currentDate, // %s pertama = tanggal
				name,        // %s kedua = nama
				link,        // %s ketiga = link button
			)

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
