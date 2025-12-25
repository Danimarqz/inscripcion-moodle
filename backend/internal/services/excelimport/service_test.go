package excelimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
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

func TestImportOfficialResultsExcel(t *testing.T) {
	exam := &models.Exam{ID: 123, Name: "Oficial", IsActive: true}

	excelPath := writeSimpleExcel(t, [][]string{
		{"12345678A", "GARCIA", "LOPEZ", "MARIA"},
		{"87654321B", "RODRIGUEZ", "PEREZ", "JUAN"},
	})
	defer func() {
		_ = os.Remove(excelPath)
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ExamOfficialResult{}))

	f, err := os.Open(excelPath)
	require.NoError(t, err)
	defer f.Close()

	svc := &Service{repo: &stubExamRepository{exam: exam}, db: db}
	result, err := svc.ImportOfficialResultsExcel(context.Background(), exam.ID, f, true)
	require.NoError(t, err)
	require.Equal(t, exam.ID, result.ExamID)
	require.Equal(t, 2, result.TotalRows)
	require.Equal(t, 2, result.ImportedResults)

	var results []models.ExamOfficialResult
	require.NoError(t, db.Find(&results).Error)
	require.Len(t, results, 2)
	require.Equal(t, "12345678A", results[0].DniMasked)
	require.Equal(t, "GARCIA", results[0].Apellido1)
	require.NotNil(t, results[0].Apellido2)
	require.Equal(t, "LOPEZ", *results[0].Apellido2)
	require.Equal(t, "MARIA", results[0].Nombre)
}

func TestImportOfficialResultsExcel_WithHeader(t *testing.T) {
	exam := &models.Exam{ID: 123, Name: "Oficial", IsActive: true}

	excelPath := writeSimpleExcel(t, [][]string{
		{"DNI", "PRIMER APELLIDO", "SEGUNDO APELLIDO", "NOMBRE"},
		{"12345678A", "GARCIA", "LOPEZ", "MARIA"},
		{"87654321B", "RODRIGUEZ", "PEREZ", "JUAN"},
	})
	defer func() {
		_ = os.Remove(excelPath)
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ExamOfficialResult{}))

	f, err := os.Open(excelPath)
	require.NoError(t, err)
	defer f.Close()

	svc := &Service{repo: &stubExamRepository{exam: exam}, db: db}
	result, err := svc.ImportOfficialResultsExcel(context.Background(), exam.ID, f, true)
	require.NoError(t, err)
	require.Equal(t, exam.ID, result.ExamID)
	require.Equal(t, 2, result.TotalRows)
	require.Equal(t, 2, result.ImportedResults)

	var results []models.ExamOfficialResult
	require.NoError(t, db.Find(&results).Error)
	require.Len(t, results, 2)
	require.Equal(t, "12345678A", results[0].DniMasked)
	require.Equal(t, "GARCIA", results[0].Apellido1)
	require.NotNil(t, results[0].Apellido2)
	require.Equal(t, "LOPEZ", *results[0].Apellido2)
	require.Equal(t, "MARIA", results[0].Nombre)
}

func writeSimpleExcel(t *testing.T, data [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	sheetName := "Sheet1"
	err := f.SetSheetName("Sheet1", sheetName)
	require.NoError(t, err)

	for i, row := range data {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		require.NoError(t, err)
		err = f.SetSheetRow(sheetName, cell, &row)
		require.NoError(t, err)
	}

	dir, err := os.MkdirTemp("", "excel-test")
	require.NoError(t, err)
	filePath := filepath.Join(dir, "test.xlsx")

	err = f.SaveAs(filePath)
	require.NoError(t, err)

	return filePath
}
