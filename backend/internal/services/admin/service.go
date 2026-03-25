package admin

import (
	"errors"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/repository"
)

var (
	ErrExamNameConflict       = errors.New("ya existe un examen con ese nombre")
	ErrActiveQuestions        = errors.New("el examen debe tener al menos una pregunta activa no anulada")
	ErrQuestionNotFound       = errors.New("la pregunta no pertenece al examen")
	ErrOfficialResultExists   = errors.New("ya existe un resultado oficial con ese DNI para este examen")
	ErrInvalidOfficialResult  = errors.New("datos de resultado oficial invalidos")
	ErrOfficialResultNotFound = errors.New("resultado oficial no encontrado")
	ErrExamNotFound           = errors.New("el examen no existe")
	ErrExamNoQuestions        = errors.New("el examen debe tener al menos una pregunta")
	ErrInvalidOption          = errors.New("opcion de respuesta no valida")
	ErrInvalidExamWeight      = errors.New("el peso del examen debe estar entre 0 y 1")
	ErrInvalidMaxMerits       = errors.New("el tope de méritos debe ser mayor que 0")
)

type Service struct {
	db             *gorm.DB
	examRepo       repository.ExamRepository
	submissionRepo repository.SubmissionRepository
	officialRepo   repository.OfficialResultRepository
	questionRepo   repository.QuestionRepository
}

func New(db *gorm.DB, examRepo repository.ExamRepository, subRepo repository.SubmissionRepository, offRepo repository.OfficialResultRepository, questionRepo repository.QuestionRepository) *Service {
	return &Service{
		db:             db,
		examRepo:       examRepo,
		submissionRepo: subRepo,
		officialRepo:   offRepo,
		questionRepo:   questionRepo,
	}
}
