package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
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
	// if req.ReportsToID != nil && *req.ReportsToID <= 0 {
	// 	req.ReportsToID = nil
	// }

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

		// if req.ReportsToID != nil {
		// 	if _, err := txRepo.FindByID(ctx, *req.ReportsToID); err != nil {
		// 		return apperror.BadRequest("reports_to_id tidak valid")
		// 	}
		// }

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
				url.QueryEscape(token),
			)

			currentDate := time.Now().Format("Mon, Jan 02 2006 15:04")

			subject := "Set Password Account"

			body := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Reset Password</title>
</head>

<body style="
  margin:0;
  padding:0;
  background:#efefef;
  font-family:Arial, Helvetica, sans-serif;
">

  <table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#efefef;">
    <tr>
      <td align="center" style="padding:24px 12px;">

        <!-- MAIN CONTAINER -->
        <table width="620" cellpadding="0" cellspacing="0" border="0" style="
          background:#ffffff;
          border:1px solid #dcdcdc;
        ">

          <!-- HEADER -->
          <tr>
            <td style="
              padding:28px 34px;
              border-bottom:1px solid #dddddd;
            ">

              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>

                  <!-- LOGO -->
                  <td align="left">
                    <img
                      src="https://raigine.com/wp-content/themes/company-profile-theme/images/logo-raigine.png"
                      alt="Raigine"
                      width="150"
                      style="display:block;"
                    />
                  </td>

                  <!-- DATE -->
                  <td align="right" style="
                    font-size:14px;
                    color:#9ca3af;
                  ">
                    %s
                  </td>

                </tr>
              </table>

            </td>
          </tr>

          <!-- BODY -->
          <tr>
            <td style="padding:46px 42px;">

              <!-- TITLE -->
              <h1 style="
                margin:0 0 34px 0;
                font-size:58px;
                line-height:1.15;
                color:#111111;
                font-weight:700;
                letter-spacing:-1px;
              ">
                Change Your Password by the Forgot Password Link Below!
              </h1>

              <!-- SUBTITLE -->
              <h3 style="
                margin:0 0 26px 0;
                font-size:28px;
                color:#1f2937;
                font-weight:700;
              ">
                Password Reset Request 🔐
              </h3>

              <!-- CONTENT -->
              <p style="
                margin:0 0 18px 0;
                font-size:18px;
                line-height:1.7;
                color:#374151;
              ">
                Hi %s,
              </p>

              <p style="
                margin:0 0 26px 0;
                font-size:17px;
                line-height:1.8;
                color:#4b5563;
              ">
                We received a request to reset your password for your rAIgine account.
              </p>

              <p style="
                margin:0 0 24px 0;
                font-size:17px;
                line-height:1.8;
                color:#4b5563;
              ">
                Please note that this password reset link will expire in 24 hours.
                If you do not reset it within that time, you'll need to request a new one.
              </p>

              <p style="
                margin:0 0 34px 0;
                font-size:17px;
                line-height:1.8;
                color:#4b5563;
              ">
                To proceed, simply click the button below to reset your password:
              </p>

              <!-- BUTTON -->
              <div style="text-align:center; margin:42px 0 46px 0;">

                <a href="%s"
                  style="
                    background:#0b57d0;
                    color:#ffffff;
                    text-decoration:none;
                    padding:16px 42px;
                    border-radius:4px;
                    display:inline-block;
                    font-size:18px;
                    font-weight:500;
                    min-width:150px;
                  ">
                  Reset Password
                </a>

              </div>

              <!-- FOOT TEXT -->
              <p style="
                margin:0 0 14px 0;
                font-size:16px;
                line-height:1.9;
                color:#4b5563;
              ">
                If you have any questions or need help, feel free to contact us via WhatsApp at 0881010745346 or by email at
                <a href="mailto:techops@raigine.com"
                  style="
                    color:#374151;
                    text-decoration:underline;
                  ">
                  techops@raigine.com
                </a>
              </p>

              <p style="
                margin:40px 0 0 0;
                font-size:17px;
                line-height:1.9;
                color:#374151;
              ">
                Let's get started,<br/>
                rAIgine Team
              </p>

            </td>
          </tr>

          <!-- FOOTER -->
          <tr>
            <td style="
              background:#f5f5f5;
              padding:34px 40px;
              border-top:1px solid #dddddd;
            ">

              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>

                  <!-- LEFT -->
                  <td align="left" valign="top">

                    <img
                      src="https://raigine.com/wp-content/themes/company-profile-theme/images/logo-raigine.png"
                      width="80"
                      style="
                        display:block;
                        margin-bottom:18px;
                      "
                    />

                    <p style="
                      margin:0;
                      font-size:13px;
                      line-height:1.8;
                      color:#9ca3af;
                    ">
                      Copyright © 2024 <br/>
                      All rights reserved
                    </p>

                  </td>

                  <!-- RIGHT -->
                  <td align="right" valign="top">

                    <table cellpadding="0" cellspacing="0">
                      <tr>

                        <td style="padding-left:16px;">
                          <a href="#">
                            <img
                              src="https://cdn-icons-png.flaticon.com/512/174/174857.png"
                              width="18"
                              style="display:block;"
                            />
                          </a>
                        </td>

                        <td style="padding-left:16px;">
                          <a href="#">
                            <img
                              src="https://cdn-icons-png.flaticon.com/512/733/733547.png"
                              width="18"
                              style="display:block;"
                            />
                          </a>
                        </td>

                        <td style="padding-left:16px;">
                          <a href="#">
                            <img
                              src="https://cdn-icons-png.flaticon.com/512/2111/2111463.png"
                              width="18"
                              style="display:block;"
                            />
                          </a>
                        </td>

                      </tr>
                    </table>

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
				currentDate,
				name,
				link,
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
