package register

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

var meses = map[string]string{
	"JANUARY":   "ENERO",
	"FEBRUARY":  "FEBRERO",
	"MARCH":     "MARZO",
	"APRIL":     "ABRIL",
	"MAY":       "MAYO",
	"JUNE":      "JUNIO",
	"JULY":      "JULIO",
	"AUGUST":    "AGOSTO",
	"SEPTEMBER": "SEPTIEMBRE",
	"OCTOBER":   "OCTUBRE",
	"NOVEMBER":  "NOVIEMBRE",
	"DECEMBER":  "DICIEMBRE",
}

func generatePDF(data Data) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	translator := pdf.UnicodeTranslatorFromDescriptor("cp1252")

	// Use constant logo path
	pdf.ImageOptions(
		logoPath,
		15,
		8,
		18,
		0,
		false,
		gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true},
		0,
		"",
	)

	pdf.SetTextColor(0, 51, 102)
	pdf.CellFormat(0, 10, translator("FORMULARIO DE SUSCRIPCIÓN"), "", 1, "C", false, 0, "")
	textBottom := pdf.GetY()
	pdf.SetDrawColor(0, 51, 102)
	pdf.SetLineWidth(0.5)
	pdf.Line(10, textBottom+2, 200, textBottom+2)
	pdf.Ln(6)

	pdf.SetFont("Arial", "B", 10)
	labeledLine(pdf, translator, "Fecha de solicitud:", time.Now().Format("02-01-2006"))

	addSectionTitle(pdf, translator, "Datos personales")
	labeledLine(pdf, translator, "Nombre:", data.Name)
	labeledLine(pdf, translator, "Apellido:", data.Surname)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(55, 8, translator("Fecha de nacimiento:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(35, 8, translator(formatDate(data.DOB)), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(20, 8, translator("DNI:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(70, 8, data.DNI, "", 1, "", false, 0, "")

	labeledLine(pdf, translator, "Dirección:", data.Address)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, translator("Ciudad:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(50, 8, translator(data.City), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(30, 8, translator("Localidad:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(70, 8, translator(data.Locality), "", 1, "", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, translator("Código postal:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(50, 8, translator(data.Postal), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(20, 8, translator("País:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(70, 8, translator(data.Country), "", 1, "", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, translator("Teléfono:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(50, 8, translator(data.Phone), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(20, 8, translator("Email:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(70, 8, translator(data.Email), "", 1, "", false, 0, "")

	addSectionTitle(pdf, translator, "Datos académicos")
	labeledLine(pdf, translator, "Curso:", data.Course)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, translator("Modalidad:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	// 62mm: "MINISTERIOS - PAGO ÚNICO - 360 €" mide 60,6mm en Arial 10; con 50 invadia la
	// columna "Forma de pago:".
	pdf.CellFormat(62, 8, translator(modalityLabel(data.Modality)), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, translator("Forma de pago:"), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(10, 8, translator(data.Payment), "", 1, "", false, 0, "")

	labeledLine(pdf, translator, "IBAN:", data.IBAN)
	labeledLine(pdf, translator, "Mes de inicio:", formatMonth(data.StartDate))

	pdf.SetFont("Arial", "I", 10)
	pdf.MultiCell(
		0,
		7,
		translator("Los cursos que se pagan mensualmente se abonarán obligatoriamente a través de domiciliación bancaria."),
		"",
		"L",
		false,
	)

	addSectionTitle(pdf, translator, "Consentimiento")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 8, translator("[X] He leído y acepto los términos y condiciones."), "", 1, "", false, 0, "")
	pdf.CellFormat(0, 8, translator("[X] He leído y acepto la política de privacidad."), "", 1, "", false, 0, "")

	addSectionTitle(pdf, translator, "Firma")
	if sig, err := decodeSignature(data.Signature); err == nil {
		imgName := fmt.Sprintf("signature_%d", time.Now().UnixNano())
		if pdf.RegisterImageOptionsReader(
			imgName,
			gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true},
			bytes.NewReader(sig),
		) != nil {
			pdf.ImageOptions(imgName, 10, pdf.GetY(), 64, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		}
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}

	pdfBytes := buffer.Bytes()
	if err := savePDF(pdfBytes, data.Email); err != nil {
		log.Printf("register: failed to persist pdf: %v", err)
	}

	return pdfBytes, nil
}

func labeledLine(pdf *gofpdf.Fpdf, tr func(string) string, label, value string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(50, 8, tr(label), "", 0, "", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(120, 8, tr(value), "", 1, "", false, 0, "")
}

func addSectionTitle(pdf *gofpdf.Fpdf, tr func(string) string, title string) {
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 250)
	pdf.CellFormat(0, 8, tr(title), "", 1, "L", true, 0, "")
	pdf.SetFont("Arial", "", 10)
}

func formatDate(value string) string {
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("02-01-2006")
	}
	return value
}

func formatMonth(value string) string {
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01", value); err == nil {
		month := strings.ToUpper(parsed.Format("January"))
		if translated, ok := meses[month]; ok {
			return translated
		}
		return month
	}
	return value
}

func decodeSignature(signature string) ([]byte, error) {
	raw := signature
	if _, after, ok := strings.Cut(signature, ","); ok {
		raw = after
	}
	raw = strings.TrimSpace(raw)
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func savePDF(data []byte, email string) error {
	dir := "generated_pdfs"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	clean := sanitizeFilename(email)
	name := fmt.Sprintf("%s_%s.pdf", clean, time.Now().Format("20060102"))
	path := filepath.Join(dir, name)
	return os.WriteFile(path, data, 0o644)
}

// Logo path constant
const logoPath = "opositalogo.png"

func sanitizeFilename(email string) string {
	var builder strings.Builder
	for _, r := range email {
		if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-", r) || r == '@' {
			builder.WriteRune(r)
			continue
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(builder.String(), "@", "_at_"), "__", "_")
}
