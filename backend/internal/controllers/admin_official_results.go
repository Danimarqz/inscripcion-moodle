package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/constants"
	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
)

func (h *AdminController) createOfficialResult(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}

	var payload admin.CreateOfficialResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.service.CreateOfficialResult(uint(examID), payload)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			http.Error(w, constants.ExamNotFound, http.StatusNotFound)
		case errors.Is(err, admin.ErrInvalidOfficialResult):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, admin.ErrOfficialResultExists):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to create official result", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, result)
}

func (h *AdminController) importOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
		http.Error(w, "failed to parse multipart data", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing excel file", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	replace := parseBoolParam(r.URL.Query().Get("replace_existing"), true)
	report, err := h.excelImport.ImportOfficialResultsExcel(r.Context(), uint(examID), file, replace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report)
}

func (h *AdminController) listOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"))
	offset := parseOffsetParam(r.URL.Query().Get("offset"))
	orderBy := sanitizeOfficialResultsOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	resultType := r.URL.Query().Get("type")
	search := r.URL.Query().Get("search")
	var hasUser *bool
	if hasUserStr := r.URL.Query().Get("has_user"); hasUserStr != "" {
		val := parseBoolParam(hasUserStr, false)
		hasUser = &val
	}

	results, err := h.service.ListOfficialResults(uint(examID), limit, offset, resultType, search, hasUser, orderBy, orderDir)
	if err != nil {
		http.Error(w, "failed to load official results", http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

func (h *AdminController) updateOfficialResult(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	var payload admin.EditOfficialResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	result, err := h.service.UpdateOfficialResult(uint(id), payload)
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrOfficialResultNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, admin.ErrInvalidOfficialResult):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to update official result", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, result)
}

func (h *AdminController) deleteOfficialResult(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteOfficialResult(uint(id)); err != nil {
		if errors.Is(err, admin.ErrOfficialResultNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete official result", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
