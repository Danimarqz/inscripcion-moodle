package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
)

func (h *AdminController) createOfficialResult(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}

	var payload admin.CreateOfficialResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.CreateOfficialResult(uint(examID), payload)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			http.Error(w, "exam not found", http.StatusNotFound)
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
		http.Error(w, "invalid exam id", http.StatusBadRequest)
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

	tmp, err := os.CreateTemp("", "official_results_*.xlsx")
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()
	defer func() {
		_ = tmp.Close()
	}()

	if _, err := io.Copy(tmp, file); err != nil {
		http.Error(w, "failed to store excel file", http.StatusInternalServerError)
		return
	}

	replace := parseBoolParam(r.URL.Query().Get("replace_existing"), true)
	report, err := h.excelImport.ImportOfficialResultsExcel(r.Context(), uint(examID), tmp.Name(), replace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report)
}

func (h *AdminController) listOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"))
	offset := parseOffsetParam(r.URL.Query().Get("offset"))
	orderBy := sanitizeOfficialResultsOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	results, err := h.service.ListOfficialResults(uint(examID), limit, offset, orderBy, orderDir)
	if err != nil {
		http.Error(w, "failed to load official results", http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}
