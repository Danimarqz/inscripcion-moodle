package admin

import "github.com/inscripcion-moodle/go-backend/internal/models"

type QuestionInput struct {
	ID            *uint  `json:"id,omitempty"`
	Name          *int   `json:"name,omitempty"`
	CorrectOption string `json:"correct_option"`
	IsActive      *bool  `json:"is_active,omitempty"`
	IsCancelled   *bool  `json:"is_cancelled,omitempty"`
}

type CreateExamRequest struct {
	Name              string          `json:"name"`
	IsActive          bool            `json:"is_active"`
	ShowScore         bool            `json:"show_score"`
	ShowPercentile    bool            `json:"show_percentile"`
	ShowScoreFull     bool            `json:"show_score_full"`
	ValidatedTribunal bool            `json:"validated_tribunal"`
	Questions         []QuestionInput `json:"questions"`
}

type EditExamRequest struct {
	Name              *string         `json:"name,omitempty"`
	IsActive          *bool           `json:"is_active,omitempty"`
	ShowScore         *bool           `json:"show_score,omitempty"`
	ShowPercentile    *bool           `json:"show_percentile,omitempty"`
	ShowScoreFull     *bool           `json:"show_score_full,omitempty"`
	ValidatedTribunal *bool           `json:"validated_tribunal,omitempty"`
	Questions         []QuestionInput `json:"questions"`
}

type ListSubmissionsResult struct {
	Submissions      []models.UserExamSubmission `json:"submissions"`
	TotalSubmissions int64                       `json:"total_submissions,omitempty"`
	AverageScore     *float64                    `json:"average_score,omitempty"`
	StatsIncluded    bool                        `json:"stats_included"`
}

type OfficialResultsList struct {
	Results []models.ExamOfficialResult `json:"results"`
	Total   int64                       `json:"total"`
}

type SubmissionAnswerInput struct {
	QuestionID uint   `json:"question_id"`
	Answer     string `json:"answer"`
}

type SubmissionUpdateRequest struct {
	Name    string                  `json:"name"`
	Surname string                  `json:"surname"`
	Email   string                  `json:"email"`
	DNI     string                  `json:"dni"`
	Answers []SubmissionAnswerInput `json:"answers"`
}

type CreateOfficialResultRequest struct {
	DNI       string `json:"dni"`
	Apellido1 string `json:"apellido_1"`
	Apellido2 string `json:"apellido_2,omitempty"`
	Nombre    string `json:"nombre"`
}
