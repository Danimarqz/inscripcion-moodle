package public

import "github.com/google/uuid"

type QuestionStub struct {
	ID    uuid.UUID `json:"id"`
	Order int       `json:"order"`
}
