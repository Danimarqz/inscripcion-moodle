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

// pagoUnico une importe de la hoja y etiqueta del PDF: con dos mapas separados, anadir
// un codigo nuevo sin su etiqueta dejaba un alta valida cuyo PDF imprimia el slug.
type pagoUnico struct {
	amount string
	label  string
}

// pagoUnicoModalities traduce el value legible por maquina del formulario a su importe de
// pago unico y a su etiqueta legible; las modalidades mensuales siguen siendo texto libre.
var pagoUnicoModalities = map[string]pagoUnico{
	"ministerios-pago-unico-360": {amount: "360 EUR", label: "MINISTERIOS - PAGO ÚNICO - 360 €"},
}

// monthlyModality valida la forma de las modalidades mensuales, cuyo value sigue siendo el
// texto visible. Con una lista literal de precios, cambiarlos en el HTML rechazaba todas las
// altas mensuales hasta tocar tambien este fichero.
var monthlyModality = regexp.MustCompile(`^\d{2,3}€/mes \(\d+ temas/mes\)$`)

// ValidModality es el allowlist que evita que un POST arbitrario entre en la hoja de cobros.
func ValidModality(modality string) bool {
	if _, ok := pagoUnicoModalities[modality]; ok {
		return true
	}
	return monthlyModality.MatchString(modality)
}

func modalityLabel(modality string) string {
	if pu, ok := pagoUnicoModalities[modality]; ok {
		return pu.label
	}
	return modality
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
		"OBSERVACIONES":        buildObservaciones(data.Payment, data.Modality),
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
	if _, ok := pagoUnicoModalities[modality]; ok {
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
	if pu, ok := pagoUnicoModalities[modality]; ok {
		return pu.amount
	}
	if matches := amountRegex.FindStringSubmatch(modality); len(matches) == 2 {
		amount := strings.ReplaceAll(matches[1], ",", ".")
		return fmt.Sprintf("%s EUR/mes", amount)
	}
	return modality
}

func buildObservaciones(payment, modality string) string {
	var partes []string
	if pu, ok := pagoUnicoModalities[modality]; ok {
		partes = append(partes, "Pago único "+pu.amount)
	}
	if payment != "" {
		partes = append(partes, "Forma de pago: "+payment)
	}
	return strings.Join(partes, " | ")
}

func normalizeString(value string) string {
	return strings.TrimSpace(value)
}
