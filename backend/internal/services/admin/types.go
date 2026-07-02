package admin

import (
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/scoring"
)

type QuestionInput struct {
	ID            *uint   `json:"id,omitempty"`
	Name          *int    `json:"name,omitempty"`
	Label         *string `json:"label,omitempty"`
	CorrectOption string  `json:"correct_option"`
	IsActive      *bool   `json:"is_active,omitempty"`
	IsCancelled   *bool   `json:"is_cancelled,omitempty"`
	GroupPosition *int    `json:"group_position,omitempty"`
}

type QuestionGroupInput struct {
	ID              *uint    `json:"id,omitempty"`
	Name            string   `json:"name"`
	Position        int      `json:"position"`
	MaxScore        float64  `json:"max_score"`
	PointsPerWrong  float64  `json:"points_per_wrong"`
	MinPassingScore *float64 `json:"min_passing_score,omitempty"`
	Eliminatory     bool     `json:"eliminatory"`
	PassingPct      *float64 `json:"passing_pct,omitempty"`
}

type CreateExamRequest struct {
	Name                 string               `json:"name"`
	IsActive             bool                 `json:"is_active"`
	ShowScore            bool                 `json:"show_score"`
	ShowPercentile       bool                 `json:"show_percentile"`
	ShowScoreFull        bool                 `json:"show_score_full"`
	ValidatedTribunal    bool                 `json:"validated_tribunal"`
	SubtractsPoints      bool                 `json:"subtracts_points"`
	PenaltyValue         *float64             `json:"penalty_value,omitempty"`
	MaxScore             *float64             `json:"max_score,omitempty"`
	ScoringMode          string               `json:"scoring_mode,omitempty"`
	PointsPerCorrect     *float64             `json:"points_per_correct,omitempty"`
	PointsPerWrong       *float64             `json:"points_per_wrong,omitempty"`
	WrongBlockSize       *float64             `json:"wrong_block_size,omitempty"`
	SecondaryMaxScores   string               `json:"secondary_max_scores,omitempty"`
	PassingCriteriaType  string               `json:"passing_criteria_type"`
	PassingCriteriaValue *float64             `json:"passing_criteria_value,omitempty"`
	ExamWeight           float64              `json:"exam_weight"`
	MaxMerits            float64              `json:"max_merits"`
	DisplayExamWeight    *float64             `json:"display_exam_weight,omitempty"`
	SkipWeights          bool                 `json:"skip_weights"`
	RaffleEnabled        bool                 `json:"raffle_enabled"`
	RaffleTerms          string               `json:"raffle_terms"`
	Groups               []QuestionGroupInput `json:"groups"`
	Questions            []QuestionInput      `json:"questions"`
}

type EditExamRequest struct {
	Name                 *string              `json:"name,omitempty"`
	IsActive             *bool                `json:"is_active,omitempty"`
	ShowScore            *bool                `json:"show_score,omitempty"`
	ShowPercentile       *bool                `json:"show_percentile,omitempty"`
	ShowScoreFull        *bool                `json:"show_score_full,omitempty"`
	ValidatedTribunal    *bool                `json:"validated_tribunal,omitempty"`
	SubtractsPoints      *bool                `json:"subtracts_points,omitempty"`
	PenaltyValue         *float64             `json:"penalty_value,omitempty"`
	MaxScore             *float64             `json:"max_score,omitempty"`
	ScoringMode          *string              `json:"scoring_mode,omitempty"`
	PointsPerCorrect     *float64             `json:"points_per_correct,omitempty"`
	PointsPerWrong       *float64             `json:"points_per_wrong,omitempty"`
	WrongBlockSize       *float64             `json:"wrong_block_size,omitempty"`
	SecondaryMaxScores   *string              `json:"secondary_max_scores,omitempty"`
	PassingCriteriaType  *string              `json:"passing_criteria_type,omitempty"`
	PassingCriteriaValue *float64             `json:"passing_criteria_value,omitempty"`
	ExamWeight           *float64             `json:"exam_weight,omitempty"`
	MaxMerits            *float64             `json:"max_merits,omitempty"`
	DisplayExamWeight    *float64             `json:"display_exam_weight,omitempty"`
	ClearDisplayWeight   *bool                `json:"clear_display_weight,omitempty"`
	SkipWeights          *bool                `json:"skip_weights,omitempty"`
	RaffleEnabled        *bool                `json:"raffle_enabled,omitempty"`
	RaffleTerms          *string              `json:"raffle_terms,omitempty"`
	AssociatedExamIDs    *[]uint              `json:"associated_exam_ids,omitempty"`
	Groups               []QuestionGroupInput `json:"groups"`
	Questions            []QuestionInput      `json:"questions"`
}

// AdminSubmissionListItem is one row of the submissions list: the submission
// (with heavy answers_json stripped) plus its computed score breakdown, so the
// admin list can show per-group scores + correct/blank/wrong at a glance.
type AdminSubmissionListItem struct {
	models.UserExamSubmission
	Breakdown *SubmissionBreakdown `json:"breakdown,omitempty"`
}

type ListSubmissionsResult struct {
	Submissions               []AdminSubmissionListItem `json:"submissions"`
	TotalSubmissions          int64                     `json:"total_submissions,omitempty"`
	AverageScore              *float64                  `json:"average_score,omitempty"`
	AverageScoreOfficial      *float64                  `json:"average_score_official,omitempty"`
	GroupAverageScore         *float64                  `json:"group_average_score,omitempty"`
	GroupAverageScoreOfficial *float64                  `json:"group_average_score_official,omitempty"`
	GroupTotalSubmissions     *int64                    `json:"group_total_submissions,omitempty"`
	GroupExamNames            []string                  `json:"group_exam_names,omitempty"`
	StatsIncluded             bool                      `json:"stats_included"`
}

// SubmissionBreakdown is the admin-facing score breakdown for a single
// submission, mirroring what the student sees: general score, correct/blank/
// wrong counts, global pass, and per-group outcomes (empty for non-grouped
// exams). Percentile and merits stay on the submission itself.
type SubmissionBreakdown struct {
	Score            *float64               `json:"score"`
	CorrectAnswers   int                    `json:"correct_answers"`
	IncorrectAnswers int                    `json:"incorrect_answers"`
	NotAnswered      int                    `json:"not_answered"`
	TotalQuestions   int                    `json:"total_questions"`
	IsPassed         *bool                  `json:"is_passed"`
	Groups           []scoring.GroupOutcome `json:"groups,omitempty"`
}

// SubmissionDetailResponse is the GET /admin/results/{id} payload: the full
// submission (same top-level fields as before — answers are lazy-loaded here,
// omitted from the list endpoint) plus the score breakdown. Embedding the
// submission promotes its JSON fields to the top level, so existing clients
// keep working unchanged; breakdown is additive and omitted when unset (e.g.
// exam with no active questions). The list endpoint is intentionally unchanged.
type SubmissionDetailResponse struct {
	*models.UserExamSubmission
	Breakdown *SubmissionBreakdown `json:"breakdown,omitempty"`
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
