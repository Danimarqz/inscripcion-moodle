package pdfimport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	lpdf "github.com/ledongthuc/pdf"
)

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

type PDFImportResult struct {
	ExamID          uint   `json:"exam_id"`
	PageCount       int    `json:"page_count"`
	ReplaceExisting bool   `json:"replace_existing"`
	TotalRows       int    `json:"total_rows"`
	ImportedResults int    `json:"imported_results"`
	CreatedUsers    int    `json:"created_users"`
	UpdatedUsers    int    `json:"updated_users"`
}

func (s *Service) ImportOfficialResultsPDF(ctx context.Context, examID uint, pdfPath string, replaceExisting bool) (*PDFImportResult, error) {
	_, err := s.repo.Find(ctx, examID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("exam %d not found: %w", examID, err)
		}
		return nil, err
	}

	conf := model.NewDefaultConfiguration()
	if err := api.ValidateFile(pdfPath, conf); err != nil {
		return nil, fmt.Errorf("validate pdf: %w", err)
	}

	ctxFile, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read pdf context: %w", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("pdf import not configured with database")
	}

	linesText, _ := extractImportantLines(pdfPath)
	totalRows, imported, err := s.storeOfficialResults(ctx, examID, linesText, replaceExisting)
	if err != nil {
		return nil, err
	}

	return &PDFImportResult{
		ExamID:          examID,
		PageCount:       ctxFile.PageCount,
		ReplaceExisting: replaceExisting,
		TotalRows:       totalRows,
		ImportedResults: imported,
		CreatedUsers:    0,
		UpdatedUsers:    0,
	}, nil
}

func extractImportantLines(pdfPath string) (string, error) {
	f, reader, err := lpdf.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(textReader)
	// Increase the buffer to tolerate long lines.
	const maxLine = 5 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxLine)

	var entries []string
	var entry strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "###") {
			// Save current entry before starting a new one.
			if entry.Len() > 0 {
				entries = append(entries, entry.String())
				entry.Reset()
			}
			entry.WriteString(line)
			continue
		}

		// Skip obvious page headers.
		headerLine := strings.HasPrefix(line, "ANEXO") ||
			strings.Contains(line, "LISTADOS") ||
			strings.Contains(line, "ASPIRANTES") ||
			len(line) > 120
		if headerLine {
			continue
		}

		if entry.Len() > 0 && line != "" {
			entry.WriteString(" ")
			entry.WriteString(line)
		}
	}

	if entry.Len() > 0 {
		entries = append(entries, entry.String())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", nil
	}

	return strings.Join(entries, "\n"), nil
}

func parseLines(linesText string) []string {
	raw := strings.Split(linesText, "\n")
	var out []string
	for _, l := range raw {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func parseOfficialResultLine(examID uint, line string) (models.ExamOfficialResult, bool) {
	trimmed := strings.TrimSpace(line)
	parts := strings.Fields(trimmed)
	if len(parts) < 2 {
		return models.ExamOfficialResult{}, false
	}

	dniMasked := parts[0]
	nameParts := parts[1:]

	var apellido1 string
	var apellido2 *string
	var nombre string

	switch len(nameParts) {
	case 1:
		apellido1 = ""
		nombre = nameParts[0]
	case 2:
		apellido1 = nameParts[0]
		nombre = nameParts[1]
	default:
		apellido1 = nameParts[0]
		apellido2Val := nameParts[1]
		apellido2 = &apellido2Val
		nameTokens := nameParts[2:]
		if len(nameTokens) > 3 {
			nameTokens = nameTokens[:3]
		}
		nombre = strings.Join(nameTokens, " ")
	}

	return models.ExamOfficialResult{
		ExamID:    examID,
		DniMasked: dniMasked,
		Apellido1: apellido1,
		Apellido2: apellido2,
		Nombre:    nombre,
	}, true
}

func (s *Service) storeOfficialResults(ctx context.Context, examID uint, linesText string, replaceExisting bool) (totalRows int, imported int, err error) {
	lines := parseLines(linesText)
	totalRows = len(lines)
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
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}
			imported += len(batch)
			batch = batch[:0]
			return nil
		}

		for _, line := range lines {
			if res, ok := parseOfficialResultLine(examID, line); ok {
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
