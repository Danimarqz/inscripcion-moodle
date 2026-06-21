package admin

import "github.com/inscripcion-moodle/go-backend/internal/models"

type QuestionInput struct {
	ID            *uint   `json:"id,omitempty"`
	Name          *int    `json:"name,omitempty"`
	Label         *string `json:"label,omitempty"`
	CorrectOption string  `json:"correct_option"`
	IsActive      *bool   `json:"is_active,omitempty"`
	IsCancelled   *bool   `json:"is_cancelled,omitempty"`
}

type CreateExamRequest struct {
	Name                 string          `json:"name"`
	IsActive             bool            `json:"is_active"`
	ShowScore            bool            `json:"show_score"`
	ShowPercentile       bool            `json:"show_percentile"`
	ShowScoreFull        bool            `json:"show_score_full"`
	ValidatedTribunal    bool            `json:"validated_tribunal"`
	SubtractsPoints      bool            `json:"subtracts_points"`
	PenaltyValue         *float64        `json:"penalty_value,omitempty"`
	MaxScore             *float64        `json:"max_score,omitempty"`
	ScoringMode          string          `json:"scoring_mode,omitempty"`
	PointsPerCorrect     *float64        `json:"points_per_correct,omitempty"`
	PointsPerWrong       *float64        `json:"points_per_wrong,omitempty"`
	SecondaryMaxScores   string          `json:"secondary_max_scores,omitempty"`
	PassingCriteriaType  string          `json:"passing_criteria_type"`
	PassingCriteriaValue *float64        `json:"passing_criteria_value,omitempty"`
	ExamWeight           float64         `json:"exam_weight"`
	MaxMerits            float64         `json:"max_merits"`
	DisplayExamWeight    *float64        `json:"display_exam_weight,omitempty"`
	SkipWeights          bool            `json:"skip_weights"`
	Questions            []QuestionInput `json:"questions"`
}

type EditExamRequest struct {
	Name                 *string         `json:"name,omitempty"`
	IsActive             *bool           `json:"is_active,omitempty"`
	ShowScore            *bool           `json:"show_score,omitempty"`
	ShowPercentile       *bool           `json:"show_percentile,omitempty"`
	ShowScoreFull        *bool           `json:"show_score_full,omitempty"`
	ValidatedTribunal    *bool           `json:"validated_tribunal,omitempty"`
	SubtractsPoints      *bool           `json:"subtracts_points,omitempty"`
	PenaltyValue         *float64        `json:"penalty_value,omitempty"`
	MaxScore             *float64        `json:"max_score,omitempty"`
	ScoringMode          *string         `json:"scoring_mode,omitempty"`
	PointsPerCorrect     *float64        `json:"points_per_correct,omitempty"`
	PointsPerWrong       *float64        `json:"points_per_wrong,omitempty"`
	SecondaryMaxScores   *string         `json:"secondary_max_scores,omitempty"`
	PassingCriteriaType  *string         `json:"passing_criteria_type,omitempty"`
	PassingCriteriaValue *float64        `json:"passing_criteria_value,omitempty"`
	ExamWeight           *float64        `json:"exam_weight,omitempty"`
	MaxMerits            *float64        `json:"max_merits,omitempty"`
	DisplayExamWeight    *float64        `json:"display_exam_weight,omitempty"`
	ClearDisplayWeight   *bool           `json:"clear_display_weight,omitempty"`
	SkipWeights          *bool           `json:"skip_weights,omitempty"`
	AssociatedExamIDs    *[]uint         `json:"associated_exam_ids,omitempty"`
	Questions            []QuestionInput `json:"questions"`
}

type ListSubmissionsResult struct {
	Submissions      []models.UserExamSubmission `json:"submissions"`
	TotalSubmissions int64                       `json:"total_submissions,omitempty"`
	AverageScore     *float64                    `json:"average_score,omitempty"`
	AverageScoreOfficial *float64                `json:"average_score_official,omitempty"`
	GroupTotalSubmissions *int64                 `json:"group_total_submissions,omitempty"`
	GroupExamNames   []string                    `json:"group_exam_names,omitempty"`
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
	Merits  *float64                `json:"merits,omitempty"`
}

type CreateOfficialResultRequest struct {
	DNI        string   `json:"dni"`
	Apellido1  string   `json:"apellido_1"`
	Apellido2  string   `json:"apellido_2,omitempty"`
	Nombre     string   `json:"nombre"`
	ResultType string   `json:"result_type"`
	Score      *float64 `json:"score,omitempty"`
	Merits     *float64 `json:"merits,omitempty"`
}

type EditOfficialResultRequest struct {
	DNI        *string  `json:"dni,omitempty"`
	Apellido1  *string  `json:"apellido_1,omitempty"`
	Apellido2  *string  `json:"apellido_2,omitempty"`
	Nombre     *string  `json:"nombre,omitempty"`
	ResultType *string  `json:"result_type,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Merits     *float64 `json:"merits,omitempty"`
}
