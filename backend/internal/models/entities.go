package models

import "time"

type Exam struct {
	ID                uint                 `gorm:"column:id;primaryKey" json:"id"`
	Name              string               `gorm:"column:name" json:"name"`
	IsActive          bool                 `gorm:"column:is_active" json:"is_active"`
	ShowScore         bool                 `gorm:"column:show_score" json:"show_score"`
	ShowPercentile    bool                 `gorm:"column:show_percentile" json:"show_percentile"`
	ShowScoreFull     bool                 `gorm:"column:show_score_full" json:"show_score_full"`
	ValidatedTribunal bool                 `gorm:"column:validated_tribunal" json:"validated_tribunal"`
	Questions         []Question           `gorm:"foreignKey:ExamID" json:"questions"`
	Submissions       []UserExamSubmission `gorm:"foreignKey:ExamID" json:"submissions"`
}

func (Exam) TableName() string {
	return "exam"
}

type Question struct {
	ID            uint   `gorm:"column:id;primaryKey" json:"id"`
	ExamID        uint   `gorm:"column:exam_id;index:idx_question_exam_id" json:"exam_id"`
	Name          int    `gorm:"column:name" json:"name"`
	CorrectOption string `gorm:"column:correct_option" json:"correct_option"`
	IsActive      bool   `gorm:"column:is_active" json:"is_active"`
	IsCancelled   bool   `gorm:"column:is_cancelled" json:"is_cancelled"`
}

func (Question) TableName() string {
	return "question"
}

type ExamUser struct {
	ID               uint                 `gorm:"column:id;primaryKey" json:"id"`
	Name             string               `gorm:"column:name" json:"name"`
	Surname          string               `gorm:"column:surname" json:"surname"`
	Email            string               `gorm:"column:email" json:"email"`
	DNI              string               `gorm:"column:dni" json:"dni"`
	AcceptsMarketing bool                 `gorm:"column:accepts_marketing" json:"accepts_marketing"`
	CreatedAt        time.Time            `gorm:"column:created_at" json:"created_at"`
	Submissions      []UserExamSubmission `gorm:"foreignKey:UserID"`
}

func (ExamUser) TableName() string {
	return "exam_user"
}

type UserExamSubmission struct {
	ID          uint         `gorm:"column:id;primaryKey" json:"id"`
	UserID      uint         `gorm:"column:user_id" json:"user_id"`
	ExamID      uint         `gorm:"column:exam_id;index:idx_user_exam_submission_exam_id" json:"exam_id"`
	Score       *float64     `gorm:"column:score" json:"score"`
	Percentile  *float64     `gorm:"column:percentile" json:"percentile"`
	SubmittedAt time.Time    `gorm:"column:submitted_at;autoCreateTime;index:idx_user_exam_submission_submitted_at" json:"submitted_at"`
	User        ExamUser     `gorm:"foreignKey:UserID" json:"user"`
	Exam        Exam         `gorm:"foreignKey:ExamID" json:"exam"`
	Answers     []UserAnswer `gorm:"foreignKey:SubmissionID" json:"answers"`
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
	ID        uint      `gorm:"column:id;primaryKey" json:"id"`
	ExamID    uint      `gorm:"column:exam_id" json:"exam_id"`
	UserID    *uint     `gorm:"column:user_id" json:"user_id,omitempty"`
	DniMasked string    `gorm:"column:dni_masked" json:"dni_masked"`
	Apellido1 string    `gorm:"column:apellido_1" json:"apellido_1"`
	Apellido2 *string   `gorm:"column:apellido_2" json:"apellido_2,omitempty"`
	Nombre    string    `gorm:"column:nombre" json:"nombre"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	Exam      Exam      `gorm:"foreignKey:ExamID"`
	User      *ExamUser `gorm:"foreignKey:UserID"`
}

func (ExamOfficialResult) TableName() string {
	return "exam_official_result"
}
