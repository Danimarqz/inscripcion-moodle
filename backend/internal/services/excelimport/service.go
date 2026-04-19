package excelimport

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"log"
	"regexp"
	"strconv"
	"strings"

	"sync"

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
	exam, err := s.repo.Find(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("exam %d not found: %w", examID, err)
	}
	useOfficialScores := exam.UseOfficialScores

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

	totalRows, imported, err := s.storeOfficialResults(ctx, examID, rows, replaceExisting, useOfficialScores)
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

func (s *Service) storeOfficialResults(ctx context.Context, examID uint, rows *excelize.Rows, replaceExisting bool, useOfficialScores bool) (totalRows int, imported int, err error) {
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if replaceExisting {
			if err := tx.Where("exam_id = ?", examID).Delete(&models.ExamOfficialResult{}).Error; err != nil {
				return err
			}
		}

		rowChan := make(chan []string, 100)
		resultChan := make(chan models.ExamOfficialResult, 100)
		errChan := make(chan error, 1)

		go func() {
			batchSize := 500
			batch := make([]models.ExamOfficialResult, 0, batchSize)

			for res := range resultChan {
				batch = append(batch, res)
				if len(batch) >= batchSize {
					if result := tx.Create(&batch); result.Error != nil {
						errChan <- result.Error
						return
					} else {
						imported += int(result.RowsAffected)
					}
					batch = batch[:0]
				}
			}
			if len(batch) > 0 {
				if result := tx.Create(&batch); result.Error != nil {
					errChan <- result.Error
					return
				} else {
					imported += int(result.RowsAffected)
				}
			}
			errChan <- nil
		}()

		numWorkers := 4
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		for range numWorkers {
			go func() {
				defer wg.Done()
				for rowData := range rowChan {
					if len(rowData) == 0 {
						continue
					}
					if res, ok := parseOfficialResultRow(examID, rowData, useOfficialScores); ok {
						resultChan <- res
					}
				}
			}()
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		rowIndex := 0
		for rows.Next() {
			row, rowErr := rows.Columns()
			if rowErr != nil {
				return rowErr
			}

			if rowIndex == 0 {
				isHeader := false
				if len(row) > 0 {
					firstCell := strings.TrimSpace(row[0])
					if firstCell != "" && isHeaderRegex.MatchString(firstCell) {
						isHeader = true
					}
				}
				if isHeader {
					rowIndex++
					continue
				}
			}
			totalRows++
			rowChan <- row
			rowIndex++
		}
		close(rowChan)

		if writeErr := <-errChan; writeErr != nil {
			return writeErr
		}

		return nil
	})

	return totalRows, imported, err
}

func cleanNameField(s string) string {
	return helpers.NormalizeName(s)
}

func parseOfficialResultRow(examID uint, row []string, useOfficialScores bool) (models.ExamOfficialResult, bool) {
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

	result := models.ExamOfficialResult{
		ExamID:     examID,
		DniMasked:  dni,
		Apellido1:  apellido1,
		Apellido2:  apellido2Ptr,
		Nombre:     nombre,
		ResultType: resultType,
	}

	if useOfficialScores {
		if len(row) >= 6 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(row[5]), 64); err == nil {
				result.Score = &v
			}
		}
		if len(row) >= 7 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(row[6]), 64); err == nil {
				result.Merits = &v
			}
		}
	}

	return result, true
}

func GenerateOfficialResultsTemplate(useOfficialScores bool) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheet := f.GetSheetName(0)
	headers := []string{"DNI", "Apellido1", "Apellido2", "Nombre", "Tipo"}
	if useOfficialScores {
		headers = append(headers, "Nota", "Meritos")
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write template: %w", err)
	}
	return buf, nil
}
