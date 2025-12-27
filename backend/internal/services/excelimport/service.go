package excelimport

import (
	"context"
	"fmt"
	"io"

	"log"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
)

var isHeaderRegex = regexp.MustCompile(`^[a-zA-Z\s]+$`)

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

func (s *Service) ImportOfficialResultsExcel(ctx context.Context, examID uint, r io.Reader, replaceExisting bool) (*models.ExcelImportResult, error) {
	if _, err := s.repo.Find(ctx, examID); err != nil {
		return nil, fmt.Errorf("exam %d not found: %w", examID, err)
	}

	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel reader: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close excel file: %v", err)
		}
	}()

	// Assuming the data is on the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows iterator: %w", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("excel import not configured with database")
	}

	totalRows, imported, err := s.storeOfficialResults(ctx, examID, rows, replaceExisting)
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

func (s *Service) storeOfficialResults(ctx context.Context, examID uint, rows *excelize.Rows, replaceExisting bool) (totalRows int, imported int, err error) {
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

		// Handle first row (header detection)
		if rows.Next() {
			firstRow, err := rows.Columns()
			if err != nil {
				return err
			}
			
			isHeader := false
			if len(firstRow) > 0 {
				firstCell := strings.TrimSpace(firstRow[0])
				// Check if it looks like a header (text only)
				if firstCell != "" && isHeaderRegex.MatchString(firstCell) {
					isHeader = true
				}
			}

			if !isHeader {
				totalRows++
				if res, ok := parseOfficialResultRow(examID, firstRow); ok {
					batch = append(batch, res)
				}
			}
		}

		for rows.Next() {
			row, err := rows.Columns()
			if err != nil {
				return err
			}
			if len(row) == 0 {
				continue
			}
			totalRows++ // We count data rows

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

func cleanNameField(s string) string {
	return helpers.NormalizeName(s)
}

func parseOfficialResultRow(examID uint, row []string) (models.ExamOfficialResult, bool) {
	if len(row) < 4 {
		return models.ExamOfficialResult{}, false
	}

	dni := strings.TrimSpace(row[0])
	apellido1 := cleanNameField(row[1])
	apellido2 := cleanNameField(row[2])
	nombre := cleanNameField(row[3])
	
	resultType := "General"
	if len(row) >= 5 {
		val := strings.TrimSpace(row[4])
		// Normalize or validate if needed. For now, take as is if valid, or default.
		// Allowed types: General, Promoción interna, Discapacidad, Otros
		// We can do a simple check or just cleaner upper/case.
		// Let's Capitalize first letter.
		if val != "" {
			resultType = val
		}
	}

	if dni == "" || apellido1 == "" || nombre == "" {
		return models.ExamOfficialResult{}, false
	}

	var apellido2Ptr *string
	if apellido2 != "" {
		apellido2Ptr = &apellido2
	}

	return models.ExamOfficialResult{
		ExamID:     examID,
		DniMasked:  dni,
		Apellido1:  apellido1,
		Apellido2:  apellido2Ptr,
		Nombre:     nombre,
		ResultType: resultType,
	}, true
}
