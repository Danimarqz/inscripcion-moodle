package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
)

func (h *AdminController) listExams(w http.ResponseWriter, r *http.Request) {
	exams, err := h.service.ListExams()
	if err != nil {
		http.Error(w, "failed to list exams", http.StatusInternalServerError)
		return
	}
	writeJSON(w, exams)
}

func (h *AdminController) createExam(w http.ResponseWriter, r *http.Request) {
	var req admin.CreateExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	exam, err := h.service.CreateExam(req)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}
	h.invalidateExamCaches(exam.ID)
	writeJSON(w, exam)
}

func (h *AdminController) updateExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	var req admin.EditExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	exam, err := h.service.UpdateExam(uint(examID), req)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}
	h.invalidateExamCaches(exam.ID)
	writeJSON(w, exam)
}

func (h *AdminController) deleteExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteExam(uint(examID)); err != nil {
		http.Error(w, "failed to delete exam", http.StatusInternalServerError)
		return
	}
	h.invalidateExamCaches(uint(examID))
	writeJSON(w, map[string]string{"detail": "Examen eliminado correctamente"})
}

func (h *AdminController) getExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	exam, err := h.service.GetExam(uint(examID))
	if err != nil {
		http.Error(w, "exam not found", http.StatusNotFound)
		return
	}
	writeJSON(w, exam)
}

func (h *AdminController) handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admin.ErrExamNameConflict), errors.Is(err, admin.ErrQuestionNotFound):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, admin.ErrActiveQuestions):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "operation failed", http.StatusInternalServerError)
	}
}
