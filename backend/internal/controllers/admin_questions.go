package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/inscripcion-moodle/go-backend/internal/models"
)

type patchQuestionRequest struct {
	FeedbackVideoKey *string `json:"feedback_video_key"`
}

func (h *AdminController) getQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid question id", http.StatusBadRequest)
		return
	}
	var q models.Question
	if err := h.db.First(&q, id).Error; err != nil {
		http.Error(w, "question not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":                 q.ID,
		"feedback_video_key": q.FeedbackVideoKey,
	})
}

func (h *AdminController) patchQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid question id", http.StatusBadRequest)
		return
	}

	var req patchQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FeedbackVideoKey != nil {
		k := *req.FeedbackVideoKey
		if strings.Contains(k, "..") || strings.Contains(k, "//") {
			http.Error(w, "invalid feedback_video_key", http.StatusBadRequest)
			return
		}
		if k == "" {
			req.FeedbackVideoKey = nil
		}
	}

	var q models.Question
	if err := h.db.First(&q, id).Error; err != nil {
		http.Error(w, "question not found", http.StatusNotFound)
		return
	}

	if err := h.db.Model(&q).Update("feedback_video_key", req.FeedbackVideoKey).Error; err != nil {
		http.Error(w, "failed to update question", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":                 q.ID,
		"feedback_video_key": req.FeedbackVideoKey,
	})
}
