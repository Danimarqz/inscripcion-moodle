package admin

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
