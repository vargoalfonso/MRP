package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func SendEmail(to []string, subject, body string) error {

	host := firstEnv("MAIL_HOST")
	port := firstEnv("MAIL_PORT")
	username := firstEnv("MAIL_USER")
	password := firstEnv("MAIL_PASS")
	fromEmail := firstEnv("MAIL_FROM")
	fromName := firstEnv("MAIL_NAME")

	addr := fmt.Sprintf("%s:%s", host, port)

	auth := smtp.PlainAuth("", username, password, host)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", fromName, fromEmail),
		"To":           strings.Join(to, ","),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
	}

	var message string

	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	message += "\r\n" + body

	if port == "465" {
		return sendMailTLS(
			addr,
			host,
			auth,
			fromEmail,
			to,
			[]byte(message),
		)
	}

	return smtp.SendMail(
		addr,
		auth,
		fromEmail,
		to,
		[]byte(message),
	)
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
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
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	return w.Close()
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
