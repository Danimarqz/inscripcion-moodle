package register

import (
	"fmt"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/services/email"
)

func sendEmails(cfg *config.Config, recipient string, pdf []byte, name, surname string, moodleError bool) error {
	subject := "Confirmación de inscripción"
	errorMessage := ""
	if moodleError {
		errorMessage = fmt.Sprintf("\nEl email %s ya estaba usado en Moodle. Por favor, revísalo.\n", recipient)
	}

	body := fmt.Sprintf("Hola %s %s,\nGracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname)
	adminBody := fmt.Sprintf("Hola %s %s,\n%s\nGracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname, errorMessage)

	attachments := []email.Attachment{
		{
			Filename:    "inscripcion.pdf",
			Content:     pdf,
			ContentType: "application/pdf",
		},
	}

	if err := email.SendEmail(cfg, []string{recipient}, subject, body, attachments, nil); err != nil {
		return err
	}
	return email.SendEmail(cfg, []string{cfg.AdminEmail}, subject, adminBody, attachments, nil)
}
