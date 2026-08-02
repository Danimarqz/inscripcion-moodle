package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"time"
)

func buildOfficialResultsOrder(orderBy, orderDir string) string {
	order := strings.ToLower(strings.TrimSpace(orderBy))
	dir := strings.ToUpper(strings.TrimSpace(orderDir))
	if dir != "ASC" && dir != "DESC" {
		dir = ""
	}

	switch order {
	case "dni":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.dni_masked %s", dir)
	case "nombre":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.nombre %s", dir)
	case "apellidos":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.apellido_1 %s, exam_official_result.apellido_2 %s", dir, dir)
	case "usuario":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_user.name %s, exam_user.surname %s", dir, dir)
	case "creado", "created_at":
		if dir == "" {
			dir = "DESC"
		}
		return fmt.Sprintf("exam_official_result.created_at %s", dir)
	default:
		if dir == "" {
			dir = "DESC"
		}
		return fmt.Sprintf("exam_official_result.created_at %s", dir)
	}
}

func (s *Service) ListOfficialResults(examID uint, limit, offset int, resultType, search string, hasUser *bool, orderBy, orderDir string) (*OfficialResultsList, error) {
	orderClause := buildOfficialResultsOrder(orderBy, orderDir)
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	results, total, err := s.officialRepo.List(queryCtx, s.db, examID, resultType, search, hasUser, offset, limit, orderClause)
	if err != nil {
		return nil, err
	}

	return &OfficialResultsList{
		Results: results,
		Total:   total,
	}, nil
}

func (s *Service) CreateOfficialResult(examID uint, req CreateOfficialResultRequest) (*models.ExamOfficialResult, error) {
	newResult := models.ExamOfficialResult{
		ExamID:    examID,
		DniMasked: req.DNI,
		Apellido1: req.Apellido1,
		Nombre:    req.Nombre,
	}
	if req.Apellido2 != "" {
		newResult.Apellido2 = &req.Apellido2
	}

	newResult.Normalize()
	if !newResult.IsValid() {
		return nil, ErrInvalidOfficialResult
	}

	if err := s.db.First(&models.Exam{}, examID).Error; err != nil {
		return nil, err
	}

	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	existing, err := s.officialRepo.FindByExamAndDNI(queryCtx, s.db, examID, newResult.DniMasked)
	if err == nil && existing != nil {
		return nil, ErrOfficialResultExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var user models.ExamUser
	if err := s.db.Where("UPPER(dni) = ?", newResult.DniMasked).First(&user).Error; err == nil {
		newResult.UserID = &user.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if req.ResultType != "" {
		newResult.ResultType = req.ResultType
	}
	newResult.Score = req.Score
	newResult.Merits = req.Merits

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	if err := s.officialRepo.Create(queryCtx, s.db, &newResult); err != nil {
		return nil, err
	}

	return &newResult, nil
}

func (s *Service) UpdateOfficialResult(id uint, req EditOfficialResultRequest) (*models.ExamOfficialResult, error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	result, err := s.officialRepo.FindByID(queryCtx, s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOfficialResultNotFound
		}
		return nil, err
	}

	if req.Apellido1 != nil {
		result.Apellido1 = *req.Apellido1
	}
	if req.Apellido2 != nil {
		result.Apellido2 = req.Apellido2
	}
	if req.Nombre != nil {
		result.Nombre = *req.Nombre
	}
	if req.ResultType != nil {
		result.ResultType = *req.ResultType
	}
	if req.Score != nil {
		result.Score = req.Score
	}
	if req.Merits != nil {
		result.Merits = req.Merits
	}

	result.Normalize()
	if !result.IsValid() {
		return nil, ErrInvalidOfficialResult
	}

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	if err := s.officialRepo.Update(queryCtx, s.db, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) DeleteOfficialResult(id uint) error {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	_, err := s.officialRepo.FindByID(queryCtx, s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOfficialResultNotFound
		}
		return err
	}

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	return s.officialRepo.Delete(queryCtx, s.db, id)
}
