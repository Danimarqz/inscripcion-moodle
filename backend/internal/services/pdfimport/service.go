package pdfimport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

type Service struct {
	repo ExamRepository
}

func New(db *gorm.DB) *Service {
	return &Service{
		repo: &gormExamRepository{db: db},
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
	TextPreview     string `json:"text_preview,omitempty"`
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

	outDir, err := os.MkdirTemp("", "pdfcpu-import-")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(outDir)
	}()

	if err := api.ExtractContentFile(pdfPath, outDir, nil, conf); err != nil {
		return &PDFImportResult{
			ExamID:          examID,
			PageCount:       ctxFile.PageCount,
			ReplaceExisting: replaceExisting,
		}, nil
	}

	files, err := collectExtractedText(outDir)
	if err != nil {
		return nil, err
	}

	return &PDFImportResult{
		ExamID:          examID,
		PageCount:       ctxFile.PageCount,
		ReplaceExisting: replaceExisting,
		TextPreview:     joinTextSnippets(files, 800),
	}, nil
}

func collectExtractedText(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_Content_page_*.txt"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, nil
	}
	return files, nil
}

func joinTextSnippets(files []string, limit int) string {
	if len(files) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n---\n")
		}
		if builder.Len()+len(text) > limit {
			builder.WriteString(text[:limit-builder.Len()])
			break
		}
		builder.WriteString(text)
	}
	return builder.String()
}
