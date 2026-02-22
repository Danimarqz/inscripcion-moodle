package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

type QueuedEmail struct {
	Cfg         *config.Config
	To          []string
	Subject     string
	Body        string
	Attachments []Attachment
	Bcc         []string
}

var emailQueue chan QueuedEmail

func InitWorkerPool(workers int) {
	emailQueue = make(chan QueuedEmail, 1000)
	for range workers {
		go func() {
			for job := range emailQueue {
				if err := SendEmail(job.Cfg, job.To, job.Subject, job.Body, job.Attachments, job.Bcc); err != nil {
					log.Printf("email worker error: failed to send email to %v: %v", job.To, err)
				}
			}
		}()
	}
}

func EnqueueEmail(cfg *config.Config, to []string, subject, body string, attachments []Attachment, bcc []string) error {
	if emailQueue == nil {
		log.Println("WARNING: Email worker pool not initialized, sending synchronously falling back")
		return SendEmail(cfg, to, subject, body, attachments, bcc)
	}

	select {
	case emailQueue <- QueuedEmail{
		Cfg:         cfg,
		To:          to,
		Subject:     subject,
		Body:        body,
		Attachments: attachments,
		Bcc:         bcc,
	}:
		return nil
	default:
		return errors.New("email queue is full, unable to enqueue message")
	}
}

func SendEmail(cfg *config.Config, to []string, subject, body string, attachments []Attachment, bcc []string) error {
	if cfg == nil {
		return errors.New("email config is required")
	}
	if len(to) == 0 {
		return errors.New("at least one to recipient is required")
	}
	if strings.TrimSpace(body) == "" {
		return errors.New("email body is required")
	}

	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "OpositaTCAE - Notificación"
	}

	addr := fmt.Sprintf("%s:%d", cfg.SMTPServer, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPServer)

	recipients := make([]string, 0, len(to)+len(bcc))
	for _, recipient := range to {
		normalized := strings.TrimSpace(recipient)
		if normalized == "" {
			continue
		}
		recipients = append(recipients, normalized)
	}
	for _, recipient := range bcc {
		normalized := strings.TrimSpace(recipient)
		if normalized == "" {
			continue
		}
		recipients = append(recipients, normalized)
	}
	if len(recipients) == 0 {
		return errors.New("no valid recipients specified")
	}

	msg, err := buildMessage(cfg.SMTPUser, strings.Join(to, ", "), subject, body, attachments, bcc)
	if err != nil {
		return err
	}
	return smtp.SendMail(addr, auth, cfg.SMTPUser, recipients, msg)
}

func buildMessage(from, to, subject, body string, attachments []Attachment, bcc []string) ([]byte, error) {
	boundary := fmt.Sprintf("BOUNDARY-%d", time.Now().UnixNano())
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	if len(bcc) > 0 {
		fmt.Fprintf(&buf, "Bcc: %s\r\n", strings.Join(bcc, ", "))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&buf, "%s\r\n", body)

	for _, attachment := range attachments {
		contentType := attachment.ContentType
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/octet-stream"
		}
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: %s\r\n", contentType)
		fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=%q\r\n", attachment.Filename)
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n\r\n")
		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buf.WriteString(encoded[i:end] + "\r\n")
		}
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}
