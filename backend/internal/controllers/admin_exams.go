package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
)

func (h *AdminController) listExams(w http.ResponseWriter, r *http.Request) {
	exams, err := h.service.ListExams()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, genericError{Message: "failed to list exams"})
		return
	}
	writeJSON(w, exams)
}

func (h *AdminController) createExam(w http.ResponseWriter, r *http.Request) {
	var req admin.CreateExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid request"})
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
	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid exam id"})
		return
	}
	var req admin.EditExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid request"})
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
	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid exam id"})
		return
	}
	if err := h.service.DeleteExam(uint(examID)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, genericError{Message: "failed to delete exam"})
		return
	}
	h.invalidateExamCaches(uint(examID))
	writeJSON(w, map[string]string{"detail": "Examen eliminado correctamente"})
}

func (h *AdminController) getExam(w http.ResponseWriter, r *http.Request) {
	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid exam id"})
		return
	}
	exam, err := h.service.GetExam(uint(examID))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, genericError{Message: "exam not found"})
		return
	}
	writeJSON(w, exam)
}

func (h *AdminController) getExamQuestions(w http.ResponseWriter, r *http.Request) {
	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, genericError{Message: "invalid exam id"})
		return
	}
	questions, err := h.service.GetExamQuestions(uint(examID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, genericError{Message: "failed to load questions"})
		return
	}
	writeJSON(w, questions)
}

func (h *AdminController) handleAdminError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	switch {
	case errors.Is(err, admin.ErrExamNameConflict),
		errors.Is(err, admin.ErrQuestionNotFound),
		errors.Is(err, admin.ErrActiveQuestions),
		errors.Is(err, admin.ErrInvalidExamWeight),
		errors.Is(err, admin.ErrInvalidMaxMerits),
		errors.Is(err, admin.ErrInvalidDisplayWeight),
		errors.Is(err, admin.ErrInvalidQuestionNumbers):
		writeJSON(w, genericError{Message: err.Error()})
	case strings.Contains(err.Error(), "uk_question_exam_name"):
		// Safety net: service-layer validateQuestionNumbers should have
		// caught this, but the DB unique constraint is our last line of
		// defence against duplicate (exam_id, name) pairs.
		writeJSON(w, genericError{Message: admin.ErrInvalidQuestionNumbers.Error()})
	default:
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, genericError{Message: "operation failed"})
	}
}
