package excelimport

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

var isHeaderRegex = regexp.MustCompile(`^[a-zA-Z\s]+$`)
var cleanNameRegex = regexp.MustCompile(`[^a-zA-ZáéíóúÁÉÍÓÚñÑ\s]`)

type Service struct {
	repo ExamRepository
	db   *gorm.DB
}

func New(db *gorm.DB) *Service {
	return &Service{
		repo: &gormExamRepository{db: db},
		db:   db,
	}
}

type ExamRepository interface {
	Find(ctx context.Context, id uint) (*models.Exam, error)
}

type gormExamRepository struct {
	db *gorm.DB
}

func (r *gormExamRepository) Find(ctx context.Context, id uint) (*models.Exam, error) {
	var exam models.Exam
	if err := r.db.WithContext(ctx).First(&exam, id).Error; err != nil {
		return nil, err
	}
	return &exam, nil
}

func (s *Service) ImportOfficialResultsExcel(ctx context.Context, examID uint, excelPath string, replaceExisting bool) (*models.ExcelImportResult, error) {
	if _, err := s.repo.Find(ctx, examID); err != nil {
		return nil, fmt.Errorf("exam %d not found: %w", examID, err)
	}

	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	// Assuming the data is on the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet: %w", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("excel import not configured with database")
	}

	var dataRows [][]string
	if len(rows) > 0 && hasHeader(rows[0]) {
		dataRows = rows[1:]
	} else {
		dataRows = rows
	}

	totalRows, imported, err := s.storeOfficialResults(ctx, examID, dataRows, replaceExisting)
	if err != nil {
		return nil, err
	}

	return &models.ExcelImportResult{
		ExamID:          examID,
		ReplaceExisting: replaceExisting,
		TotalRows:       totalRows,
		ImportedResults: imported,
	}, nil
}

func hasHeader(row []string) bool {
	if len(row) == 0 {
		return false
	}
	firstCell := strings.TrimSpace(row[0])
	if firstCell == "" {
		return false
	}
	return isHeaderRegex.MatchString(firstCell)
}

func (s *Service) storeOfficialResults(ctx context.Context, examID uint, rows [][]string, replaceExisting bool) (totalRows int, imported int, err error) {
	totalRows = len(rows)
	if totalRows == 0 {
		return 0, 0, nil
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if replaceExisting {
			if err := tx.Where("exam_id = ?", examID).Delete(&models.ExamOfficialResult{}).Error; err != nil {
				return err
			}
		}

		batchSize := 500
		batch := make([]models.ExamOfficialResult, 0, batchSize)

		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			if result := tx.Create(&batch); result.Error != nil {
				return result.Error
			} else {
				imported += int(result.RowsAffected)
			}
			batch = batch[:0]
			return nil
		}

		for _, row := range rows {
			if res, ok := parseOfficialResultRow(examID, row); ok {
				batch = append(batch, res)
				if len(batch) >= batchSize {
					if err := flush(); err != nil {
						return err
					}
				}
			}
		}

		return flush()
	})

	return totalRows, imported, err
}

func parseOfficialResultRow(examID uint, row []string) (models.ExamOfficialResult, bool) {
	if len(row) < 4 {
		return models.ExamOfficialResult{}, false
	}

	dni := strings.TrimSpace(row[0])
	apellido1 := strings.TrimSpace(row[1])
	apellido2 := strings.TrimSpace(row[2])
	nombre := strings.TrimSpace(row[3])

	nombre = strings.TrimSpace(cleanNameRegex.ReplaceAllString(nombre, ""))
	apellido1 = strings.TrimSpace(cleanNameRegex.ReplaceAllString(apellido1, ""))
	apellido2 = strings.TrimSpace(cleanNameRegex.ReplaceAllString(apellido2, ""))

	if dni == "" || apellido1 == "" || nombre == "" {
		return models.ExamOfficialResult{}, false
	}

	var apellido2Ptr *string
	if apellido2 != "" {
		apellido2Ptr = &apellido2
	}

	return models.ExamOfficialResult{
		ExamID:    examID,
		DniMasked: dni,
		Apellido1: apellido1,
		Apellido2: apellido2Ptr,
		Nombre:    nombre,
	}, true
}
