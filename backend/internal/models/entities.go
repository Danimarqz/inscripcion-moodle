package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AnswersJSON maps question IDs to answer strings.
// Stored as JSON in the database: {"123": "A", "124": "B"}
type AnswersJSON map[uint]string

func (a AnswersJSON) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (a *AnswersJSON) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("AnswersJSON.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(data, a)
}

type Exam struct {
	ID                   uint                 `gorm:"column:id;primaryKey" json:"id"`
	Name                 string               `gorm:"column:name" json:"name"`
	Slug                 string               `gorm:"-" json:"slug"`
	IsActive             bool                 `gorm:"column:is_active" json:"is_active"`
	ShowScore            bool                 `gorm:"column:show_score" json:"show_score"`
	ShowPercentile       bool                 `gorm:"column:show_percentile" json:"show_percentile"`
	ShowScoreFull        bool                 `gorm:"column:show_score_full" json:"show_score_full"`
	ValidatedTribunal    bool                 `gorm:"column:validated_tribunal" json:"validated_tribunal"`
	SubtractsPoints      bool                 `gorm:"column:subtracts_points" json:"subtracts_points"`
	PenaltyValue         *float64             `gorm:"column:penalty_value" json:"penalty_value"`
	MaxScore             *float64             `gorm:"column:max_score;default:100.0" json:"max_score"`
	ScoringMode          string               `gorm:"column:scoring_mode;default:'legacy'" json:"scoring_mode"`
	PointsPerCorrect     *float64             `gorm:"column:points_per_correct" json:"points_per_correct"`
	PointsPerWrong       *float64             `gorm:"column:points_per_wrong" json:"points_per_wrong"`
	SecondaryMaxScores   string               `gorm:"column:secondary_max_scores" json:"secondary_max_scores"`
	PassingCriteriaType  string               `gorm:"column:passing_criteria_type;default:'disabled'" json:"passing_criteria_type"`
	PassingCriteriaValue *float64             `gorm:"column:passing_criteria_value" json:"passing_criteria_value"`
	PassingThreshold     *float64             `gorm:"column:passing_threshold" json:"passing_threshold"`
	ExamWeight           float64              `gorm:"column:exam_weight;default:0.5" json:"exam_weight"`
	MaxMerits            float64              `gorm:"column:max_merits;default:100" json:"max_merits"`
	DisplayExamWeight    *float64             `gorm:"column:display_exam_weight" json:"display_exam_weight"`
	SkipWeights          bool                 `gorm:"column:skip_weights;default:false" json:"skip_weights"`
	PercentileGroup      *uint                `gorm:"column:percentile_group" json:"percentile_group"`
	RaffleEnabled        bool                 `gorm:"column:raffle_enabled;default:false" json:"raffle_enabled"`
	RaffleTerms          string               `gorm:"column:raffle_terms" json:"raffle_terms"`
	AssociatedExamIDs    []uint               `gorm:"-" json:"associated_exam_ids"` // other exams sharing PercentileGroup; populated on read
	Questions            []Question           `gorm:"foreignKey:ExamID" json:"questions"`
	Groups               []QuestionGroup      `gorm:"foreignKey:ExamID" json:"groups"`
	Submissions          []UserExamSubmission `gorm:"foreignKey:ExamID" json:"-"`
}

func (Exam) TableName() string {
	return "exam"
}

type Question struct {
	ID               uint    `gorm:"column:id;primaryKey" json:"id"`
	ExamID           uint    `gorm:"column:exam_id;index:idx_question_exam_id" json:"exam_id"`
	GroupID          *uint   `gorm:"column:group_id" json:"group_id,omitempty"`
	Name             int     `gorm:"column:name" json:"name"`
	Label            *string `gorm:"column:label" json:"label,omitempty"`
	CorrectOption    string  `gorm:"column:correct_option" json:"correct_option"`
	IsActive         bool    `gorm:"column:is_active" json:"is_active"`
	IsCancelled      bool    `gorm:"column:is_cancelled" json:"is_cancelled"`
	FeedbackVideoKey *string `gorm:"column:feedback_video_key" json:"-"`
}

func (Question) TableName() string {
	return "question"
}

type QuestionGroup struct {
	ID              uint     `gorm:"column:id;primaryKey" json:"id"`
	ExamID          uint     `gorm:"column:exam_id" json:"exam_id"`
	Name            string   `gorm:"column:name" json:"name"`
	Position        int      `gorm:"column:position" json:"position"`
	MaxScore        float64  `gorm:"column:max_score" json:"max_score"`
	PointsPerWrong  float64  `gorm:"column:points_per_wrong" json:"points_per_wrong"`
	MinPassingScore *float64 `gorm:"column:min_passing_score" json:"min_passing_score"`
	Eliminatory     bool     `gorm:"column:eliminatory" json:"eliminatory"`
}

func (QuestionGroup) TableName() string {
	return "question_group"
}

type ExamUser struct {
	ID               uint                 `gorm:"column:id;primaryKey" json:"id"`
	Name             string               `gorm:"column:name" json:"name"`
	Surname          string               `gorm:"column:surname" json:"surname"`
	Email            string               `gorm:"column:email;index:idx_exam_user_email" json:"email"`
	DNI              string               `gorm:"column:dni;index:idx_exam_user_dni" json:"dni"`
	MoodleID         *int                 `gorm:"column:moodle_id" json:"moodle_id,omitempty"`
	AcceptsMarketing bool                 `gorm:"column:accepts_marketing" json:"accepts_marketing"`
	CreatedAt        time.Time            `gorm:"column:created_at" json:"created_at"`
	Submissions      []UserExamSubmission `gorm:"foreignKey:UserID" json:"-"`
}

func (ExamUser) TableName() string {
	return "exam_user"
}

type UserExamSubmission struct {
	ID                 uint         `gorm:"column:id;primaryKey" json:"id"`
	UserID             uint         `gorm:"column:user_id;index:idx_user_exam_submission_user_id" json:"user_id"`
	ExamID             uint         `gorm:"column:exam_id;index:idx_user_exam_submission_exam_id" json:"exam_id"`
	Score              *float64     `gorm:"column:score" json:"score"`
	Merits             *float64     `gorm:"column:merits" json:"merits"`
	Percentile         *float64     `gorm:"column:percentile" json:"percentile"`
	SelectedResultType string       `gorm:"column:selected_result_type;default:'General'" json:"selected_result_type"`
	AnswersData        *AnswersJSON `gorm:"column:answers_json;type:json" json:"answers_data"`
	SubmittedAt        time.Time    `gorm:"column:submitted_at;autoCreateTime;index:idx_user_exam_submission_submitted_at" json:"submitted_at"`
	User               ExamUser     `gorm:"foreignKey:UserID" json:"user"`
	// Exam/Answers are loaded for scoring logic, never returned to clients —
	// json:"-" keeps the (often empty) association objects out of list payloads.
	Exam    Exam         `gorm:"foreignKey:ExamID" json:"-"`
	Answers []UserAnswer `gorm:"foreignKey:SubmissionID" json:"-"`
}

func (UserExamSubmission) TableName() string {
	return "user_exam_submission"
}

type UserAnswer struct {
	ID           uint   `gorm:"column:id;primaryKey" json:"id"`
	SubmissionID uint   `gorm:"column:submission_id;index:idx_user_answer_submission" json:"submission_id"`
	QuestionID   uint   `gorm:"column:question_id;index:idx_user_answer_question" json:"question_id"`
	Answer       string `gorm:"column:answer" json:"answer"`
}

func (UserAnswer) TableName() string {
	return "user_answer"
}

type AdminUser struct {
	ID           uint   `gorm:"column:id;primaryKey" json:"id"`
	Username     string `gorm:"column:username;unique" json:"username"`
	PasswordHash string `gorm:"column:password_hash" json:"-"`
}

func (AdminUser) TableName() string {
	return "admin_user"
}

type ExamOfficialResult struct {
	ID         uint      `gorm:"column:id;primaryKey" json:"id"`
	ExamID     uint      `gorm:"column:exam_id;index:idx_official_result_exam_dni" json:"exam_id"`
	UserID     *uint     `gorm:"column:user_id" json:"user_id,omitempty"`
	DniMasked  string    `gorm:"column:dni_masked;index:idx_official_result_exam_dni" json:"dni_masked"`
	Apellido1  string    `gorm:"column:apellido_1" json:"apellido_1"`
	Apellido2  *string   `gorm:"column:apellido_2" json:"apellido_2,omitempty"`
	Nombre     string    `gorm:"column:nombre" json:"nombre"`
	ResultType string    `gorm:"column:result_type;default:'General'" json:"result_type"`
	Score      *float64  `gorm:"column:score" json:"score,omitempty"`
	Merits     *float64  `gorm:"column:merits" json:"merits,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	Exam       Exam      `gorm:"foreignKey:ExamID" json:"exam"`
	User       *ExamUser `gorm:"foreignKey:UserID" json:"user"`
}

func (ExamOfficialResult) TableName() string {
	return "exam_official_result"
}

func (e *ExamOfficialResult) Normalize() {
	e.DniMasked = strings.ToUpper(strings.TrimSpace(e.DniMasked))
	e.Apellido1 = strings.ToUpper(strings.TrimSpace(e.Apellido1))
	if e.Apellido2 != nil {
		val := strings.ToUpper(strings.TrimSpace(*e.Apellido2))
		if val == "" {
			e.Apellido2 = nil
		} else {
			e.Apellido2 = &val
		}
	}
	e.Nombre = strings.ToUpper(strings.TrimSpace(e.Nombre))
}

func (e *ExamOfficialResult) IsValid() bool {
	return e.DniMasked != "" && e.Apellido1 != "" && e.Nombre != ""
}

type ExcelImportResult struct {
	ExamID          uint `json:"exam_id"`
	ReplaceExisting bool `json:"replace_existing"`
	TotalRows       int  `json:"total_rows"`
	ImportedResults int  `json:"imported_results"`
}
