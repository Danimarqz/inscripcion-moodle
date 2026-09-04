package register

import (
	"fmt"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/services/email"
)

func sendEmails(cfg *config.Config, recipient string, pdf []byte, name, surname string, moodleError bool, moodleUsername string, dni string, gsheetConflict, gsheetFailed bool) error {
	subject := "Confirmación de inscripción"
	errorMessage := ""
	errorMessageUser := ""
	if moodleError && moodleUsername != "" {
		errorMessageUser = fmt.Sprintf("\nYa estabas registrado/a en Moodle, tu usuario es: %s\nTu contraseña ha sido actualizada a tu DNI: %s\n", moodleUsername, strings.ToUpper(strings.TrimSpace(dni)))
	} else if moodleError {
		errorMessage = fmt.Sprintf("\nEl email %s ya estaba usado en Moodle. Por favor, revísalo.\n", recipient)
	}

	body := fmt.Sprintf("Hola %s %s,\nGracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.%s\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname, errorMessageUser)

	adminSubject := subject
	adminWarning := ""
	if moodleError || gsheetConflict {
		adminSubject = "⚠️ (REPETIDO) " + subject
		adminWarning = "\n[AVISO: Este usuario ya estaba registrado en Moodle o en la Hoja de Inscripciones y ha vuelto a intentar matricularse. Revisa sus datos.]\n"
	}
	if gsheetFailed {
		adminSubject = "⚠️ (FALTA EN LA HOJA) " + subject
		adminWarning += "\n[AVISO: No se pudo escribir el alta en la Hoja de Inscripciones. Añade la fila a mano con los datos del PDF adjunto.]\n"
	}

	adminBody := fmt.Sprintf("Hola %s %s,\n%s%s\nInfo enviada al usuario: %s, Gracias por completar tu inscripción. Adjuntamos el PDF generado con tus datos.\n\nUn saludo,\nEquipo de OpositaTCAE", name, surname, adminWarning, errorMessage, errorMessageUser)

	attachments := []email.Attachment{
		{
			Filename:    "inscripcion.pdf",
			Content:     pdf,
			ContentType: "application/pdf",
		},
	}

	if err := email.EnqueueEmail(cfg, []string{recipient}, subject, body, attachments, nil); err != nil {
		return err
	}
	return email.EnqueueEmail(cfg, []string{cfg.AdminEmail}, adminSubject, adminBody, attachments, nil)
}
