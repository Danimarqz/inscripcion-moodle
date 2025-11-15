package pdfimport

import (
	"context"
	"os"
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubExamRepository struct {
	exam *models.Exam
	err  error
}

func (s *stubExamRepository) Find(_ context.Context, id uint) (*models.Exam, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.exam != nil && s.exam.ID == id {
		return s.exam, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func TestImportOfficialResultsPDF(t *testing.T) {
	exam := &models.Exam{ID: 123, Name: "Oficial", IsActive: true}

	pdfPath := writeSimplePDF(t, "Resultados oficiales")
	defer func() {
		_ = os.Remove(pdfPath)
	}()

	svc := &Service{repo: &stubExamRepository{exam: exam}}
	result, err := svc.ImportOfficialResultsPDF(context.Background(), exam.ID, pdfPath, true)
	require.NoError(t, err)
	require.Equal(t, exam.ID, result.ExamID)
	require.True(t, result.PageCount >= 1, "expected at least one page")
	require.Contains(t, result.TextPreview, "Resultados oficiales")
}

func writeSimplePDF(t *testing.T, text string) string {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, text)

	out, err := os.CreateTemp("", "test-results-*.pdf")
	require.NoError(t, err)
	defer func() {
		_ = out.Close()
	}()

	err = pdf.Output(out)
	require.NoError(t, err)

	return out.Name()
}
