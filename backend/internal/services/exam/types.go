package exam

type AnswerSubmission struct {
	QuestionID uint   `json:"question_id"`
	Answer     string `json:"answer"`
}

type SubmitExamRequest struct {
	Email            string             `json:"email"`
	DNI              string             `json:"dni"`
	Name             string             `json:"name"`
	Surname          string             `json:"surname"`
	ExamID           uint               `json:"exam_id"`
	Answers          []AnswerSubmission `json:"answers"`
	AcceptsMarketing bool               `json:"accepts_marketing"`
	ResultType       string             `json:"result_type"`
}

type SubmissionCheckRequest struct {
	DNI    string `json:"dni"`
	Email  string `json:"email"`
	ExamID uint   `json:"exam_id"`
}

type OfficialResultMatchRequest struct {
	ExamID  uint   `json:"exam_id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	DNI     string `json:"dni"`
}

type OfficialResultMatchResponse struct {
	Match bool `json:"match"`
}

type SubmissionPayload struct {
	Message          string         `json:"message"`
	Score            *float64       `json:"score"`
	Percentile       *float64       `json:"percentile"`
	Position         *int           `json:"position"`
	TotalSubmissions *int           `json:"total_submissions"`
	CorrectAnswers   *int           `json:"correct_answers"`
	TotalQuestions   *int           `json:"total_questions"`
	AnswersReview    []AnswerReview `json:"answers_review,omitempty"`
}

type AnswerReview struct {
	QuestionID     uint    `json:"question_id"`
	QuestionLabel  *int    `json:"question_label"`
	SelectedOption *string `json:"selected_option"`
	CorrectOption  *string `json:"correct_option"`
	IsCorrect      bool    `json:"is_correct"`
}

type ScoreBreakdown struct {
	Score          float64
	CorrectAnswers int
	TotalQuestions int
}

type QuestionStub struct {
	ID          uint `json:"id"`
	Name        int  `json:"name"`
	IsActive    bool `json:"is_active"`
	IsCancelled bool `json:"is_cancelled"`
}
