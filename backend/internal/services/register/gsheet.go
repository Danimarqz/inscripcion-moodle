package register

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var courseLabels = map[string]string{
	"ingesa":      "INGESA",
	"sergas":      "SERGAS",
	"gva":         "VALENCIA",
	"aragon":      "ARAGON",
	"sescam":      "SESCAM",
	"osakidetza":  "OSAKIDETZA",
	"sacyl":       "SACYL",
	"osasunbidea": "OSASUNBIDEA",
	"seris":       "SERIS",
	"cantabria":   "SCS",
	"canarias":    "CANARIAS",
	"sms":         "SMS",
	"sespa":       "SESPA",
	"imserso":     "IMSERSO",
	"ses":         "SES",
	"era":         "ERA",
	"ib-salut":    "IB SALUT",
	"xunta":       "XUNTA",
	"sas":         "SAS",
	"sermas":      "SERMAS",
	"sepad":       "SEPAD",
}

var (
	amountRegex = regexp.MustCompile(`(\d+[\.,]?\d*)`)
	groupRegex  = regexp.MustCompile(`\(([^)]+)\)`)
)

func postRegistrationToGSheet(ctx context.Context, client *http.Client, endpoint string, data Data) error {
	if endpoint == "" {
		return errors.New("API_GSHEET no esta configurado")
	}
	payload := map[string]string{
		"CURSO":                mapCourse(data.Course),
		"GRUPO":                extractGroup(data.Modality),
		"IMPORTE/MES":          extractAmount(data.Modality),
		"PRIMER PAGO":          formatFirstPaymentMonth(data.StartDate),
		"OBSERVACIONES":        buildObservaciones(data.Payment),
		"FECHA ALTA":           time.Now().Format("02/01/2006"),
		"Nombre":               normalizeString(data.Name),
		"Apellido":             normalizeString(data.Surname),
		"Nacimiento":           formatBirthDate(data.DOB),
		"DNI":                  normalizeString(data.DNI),
		"Domicilio":            normalizeString(data.Address),
		"Provincia":            normalizeString(data.City),
		"Localidad":            normalizeString(data.Locality),
		"CP":                   normalizeString(data.Postal),
		"País":                 normalizeString(data.Country),
		"Teléfono":             normalizeString(data.Phone),
		"Email":                normalizeString(data.Email),
		"IBAN":                 normalizeString(data.IBAN),
		"¿Cómo nos conociste?": normalizeString(data.Discover),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode registration payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google sheets responded with %d", resp.StatusCode)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("invalid response from Google Sheets: %w", err)
	}

	if status, _ := response["status"].(string); strings.ToLower(status) != "ok" {
		msg, _ := response["message"].(string)
		if msg == "" {
			msg = "respuesta inesperada del script de Google Sheets"
		}
		return fmt.Errorf("google sheets error: %s", msg)
	}

	return nil
}

func mapCourse(course string) string {
	key := strings.TrimSpace(strings.ToLower(course))
	if label, ok := courseLabels[key]; ok {
		return label
	}
	return course
}

func formatBirthDate(dob string) string {
	if dob == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", dob); err == nil {
		return parsed.Format("02/01/2006")
	}
	return dob
}

func formatFirstPaymentMonth(start string) string {
	if start == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01", start); err == nil {
		return parsed.Format("01/2006")
	}
	return start
}

func extractGroup(modality string) string {
	if modality == "" {
		return ""
	}
	if matches := groupRegex.FindStringSubmatch(modality); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	return modality
}

func extractAmount(modality string) string {
	if modality == "" {
		return ""
	}
	if matches := amountRegex.FindStringSubmatch(modality); len(matches) == 2 {
		amount := strings.ReplaceAll(matches[1], ",", ".")
		return fmt.Sprintf("%s EUR/mes", amount)
	}
	return modality
}

func buildObservaciones(payment string) string {
	if payment == "" {
		return ""
	}
	return "Forma de pago: " + payment
}

func normalizeString(value string) string {
	return strings.TrimSpace(value)
}
