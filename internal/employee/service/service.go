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
	background-color:#f4f6f9;
	font-family:Arial,sans-serif;
">

	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0" style="
					background:#ffffff;
					border-radius:12px;
					overflow:hidden;
					box-shadow:0 2px 10px rgba(0,0,0,0.08);
				">

					<!-- HEADER -->
					<tr>
						<td style="
							background:#2563eb;
							padding:30px;
							text-align:center;
							color:white;
						">
							<h1 style="
								margin:0;
								font-size:28px;
							">
								Welcome 👋
							</h1>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:40px;">

							<h2 style="
								margin-top:0;
								color:#111827;
							">
								Halo %s,
							</h2>

							<p style="
								font-size:16px;
								line-height:1.8;
								color:#4b5563;
							">
								Akun Anda telah berhasil dibuat.
							</p>

							<p style="
								font-size:16px;
								line-height:1.8;
								color:#4b5563;
							">
								Silakan klik tombol di bawah untuk membuat password akun Anda.
							</p>

							<div style="
								text-align:center;
								margin:40px 0;
							">
								<a href="%s"
									style="
										background:#2563eb;
										color:white;
										padding:14px 28px;
										text-decoration:none;
										border-radius:8px;
										font-size:16px;
										font-weight:bold;
										display:inline-block;
									">
									Set Password
								</a>
							</div>

							<p style="
								font-size:14px;
								color:#6b7280;
								line-height:1.8;
							">
								Link ini berlaku selama 24 jam.
							</p>

							<p style="
								font-size:14px;
								color:#9ca3af;
								line-height:1.8;
								margin-top:30px;
							">
								Jika tombol tidak dapat diklik, copy link berikut:
								<br><br>
								<a href="%s">%s</a>
							</p>

						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="
							background:#f9fafb;
							padding:20px;
							text-align:center;
							font-size:12px;
							color:#9ca3af;
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
