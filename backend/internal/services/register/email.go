package register

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

func sendEmails(cfg *config.Config, recipient string, pdf []byte, name, surname string, moodleError bool) error {
	subject := "Confirmación de inscripción"
	errorMessage := ""
	if moodleError {
		errorMessage = fmt.Sprintf("\nEl email %s ya estaba usado en Moodle. Por favor, revísalo.\n", recipient)
	}

	body := fmt.Sprintf("Hola %s %s,\nGracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname)
	adminBody := fmt.Sprintf("Hola %s %s,\n%s\nGracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname, errorMessage)

	if err := dispatchEmail(cfg, recipient, subject, body, pdf); err != nil {
		return err
	}
	return dispatchEmail(cfg, cfg.AdminEmail, subject, adminBody, pdf)
}

func dispatchEmail(cfg *config.Config, to, subject, body string, pdf []byte) error {
	msg, err := buildMessage(cfg.SMTPUser, to, subject, body, pdf)
	if err != nil {
		return err
	}
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPServer)
	addr := fmt.Sprintf("%s:%d", cfg.SMTPServer, cfg.SMTPPort)
	return smtp.SendMail(addr, auth, cfg.SMTPUser, []string{to}, msg)
}

func buildMessage(from, to, subject, body string, pdf []byte) ([]byte, error) {
	boundary := fmt.Sprintf("BOUNDARY-%d", time.Now().UnixNano())
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n", body)
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: application/pdf\r\n")
	fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=\"inscripcion.pdf\"\r\n")
	fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString(pdf)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end] + "\r\n")
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}
