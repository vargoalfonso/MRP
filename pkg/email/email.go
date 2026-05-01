package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func SendEmail(to, subject, body string) error {
	host := firstEnv("SMTP_HOST", "MAIL_HOST")
	port := firstEnv("SMTP_PORT", "MAIL_PORT")
	fromEmail := firstEnv("SMTP_EMAIL", "MAIL_USER")
	password := firstEnv("SMTP_PASSWORD", "MAIL_PASS")

	auth := smtp.PlainAuth("", fromEmail, password, host)

	msg := []byte(fmt.Sprintf(
		"Subject: %s\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"MIME-version: 1.0;\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n"+
			"%s",
		subject, fromEmail, to, body,
	))

	addr := fmt.Sprintf("%s:%s", host, port)
	if strings.TrimSpace(port) == "465" {
		return sendMailTLS(addr, host, auth, fromEmail, []string{to}, msg)
	}

	return smtp.SendMail(addr, auth, fromEmail, []string{to}, msg)
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}

	return writer.Close()
}
