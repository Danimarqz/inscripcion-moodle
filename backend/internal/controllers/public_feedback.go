package controllers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/inscripcion-moodle/go-backend/internal/services/feedback"
)

// GetFeedbackPlaylist serves the CloudFront-signed HLS playlist for a question.
// It only parses/normalizes the request and maps service errors to HTTP status
// codes; all business logic lives in the feedback service.
func (h *PublicController) GetFeedbackPlaylist(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseUint(chi.URLParam(r, "question_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid question_id", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	email := strings.ToLower(strings.TrimSpace(q.Get("email")))
	dni := examservice.NormalizeDNI(q.Get("dni"))
	examIDStr := q.Get("exam_id")
	if email == "" || dni == "" || examIDStr == "" {
		http.Error(w, "email, dni and exam_id are required", http.StatusBadRequest)
		return
	}
	examID, err := strconv.ParseUint(examIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid exam_id", http.StatusBadRequest)
		return
	}

	playlist, err := h.feedback.SignedPlaylist(r.Context(), uint(questionID), email, dni, uint(examID))
	if err != nil {
		switch {
		case errors.Is(err, feedback.ErrNotConfigured):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case errors.Is(err, feedback.ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(err, feedback.ErrNotFound):
			http.Error(w, feedback.ErrNotFound.Error(), http.StatusNotFound)
		case errors.Is(err, feedback.ErrUpstream):
			log.Printf("feedback playlist upstream error for question %d: %v", questionID, err)
			http.Error(w, "failed to fetch playlist", http.StatusBadGateway)
		default:
			log.Printf("feedback playlist error for question %d: %v", questionID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(playlist)
}
