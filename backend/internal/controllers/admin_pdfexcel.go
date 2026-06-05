package controllers

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/constants"
)

const (
	// pdfExcelMaxBody is the hard cap on the entire multipart request body,
	// enforced via http.MaxBytesReader so an oversized upload cannot exhaust
	// disk by spilling to /tmp. Covers several PDFs per batch.
	pdfExcelMaxBody = 128 << 20 // 128 MiB
	// pdfExcelMaxMemory is how much of the form is buffered in RAM before the
	// rest spills to (capped) temp files during parsing.
	pdfExcelMaxMemory = 16 << 20 // 16 MiB
)

// filenameSafe matches the characters allowed in a sanitized S3 object key.
var filenameSafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// pdfMagic is the leading byte signature of a PDF file: "%PDF".
var pdfMagic = []byte{0x25, 0x50, 0x44, 0x46}

// examPrefix derives the server-side S3 job prefix from a validated exam id.
// The prefix is NEVER taken from client input.
func examPrefix(examID uint) string {
	return "exam-" + strconv.FormatUint(uint64(examID), 10)
}

// sanitizePDFFilename reduces a client-supplied filename to a safe S3 key
// component: basename only, restricted charset, forced .pdf extension.
func sanitizePDFFilename(name string) string {
	base := filepath.Base(name)
	base = strings.TrimSpace(base)
	// Drop any existing extension; we force .pdf below.
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = filenameSafe.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		base = "documento"
	}
	return base + ".pdf"
}

// uploadPDFExcel: POST /admin/exams/{exam_id}/results/pdf-excel/upload
// Accepts multiple PDFs under form field "files", clears the existing job
// prefix in S3, validates each is a real PDF, sanitizes filenames, and uploads
// them to <prefix>/input/<filename>.
func (h *AdminController) uploadPDFExcel(w http.ResponseWriter, r *http.Request) {
	if h.pdfExcel == nil {
		http.Error(w, "pdf-excel converter unavailable", http.StatusServiceUnavailable)
		return
	}

	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}
	prefix := examPrefix(examID)

	// Cap the whole request body before parsing so an oversized upload cannot
	// fill disk via temp-file spillover. ParseMultipartForm returns an error
	// once the cap is exceeded.
	r.Body = http.MaxBytesReader(w, r.Body, pdfExcelMaxBody)
	if err := r.ParseMultipartForm(pdfExcelMaxMemory); err != nil {
		http.Error(w, "upload too large or malformed", http.StatusRequestEntityTooLarge)
		return
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File["files"]) == 0 {
		http.Error(w, "missing pdf files", http.StatusBadRequest)
		return
	}
	headers := r.MultipartForm.File["files"]

	// Clear stale objects so a fresh batch is not merged with old PDFs.
	if err := h.pdfExcel.ClearJob(r.Context(), prefix); err != nil {
		log.Printf("pdfexcel: ClearJob failed exam_id=%d: %v", examID, err)
		http.Error(w, "failed to prepare upload", http.StatusInternalServerError)
		return
	}

	uploaded := make([]string, 0, len(headers))
	for _, fh := range headers {
		file, err := fh.Open()
		if err != nil {
			http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
			return
		}

		magic := make([]byte, 4)
		if _, err := io.ReadFull(file, magic); err != nil || !bytes.Equal(magic, pdfMagic) {
			_ = file.Close()
			http.Error(w, "invalid file: only pdf files are accepted", http.StatusBadRequest)
			return
		}

		filename := sanitizePDFFilename(fh.Filename)
		full := io.MultiReader(bytes.NewReader(magic), file)
		if err := h.pdfExcel.UploadPDF(r.Context(), prefix, filename, full); err != nil {
			_ = file.Close()
			log.Printf("pdfexcel: UploadPDF failed exam_id=%d file=%s: %v", examID, filename, err)
			http.Error(w, "failed to upload pdf", http.StatusInternalServerError)
			return
		}
		_ = file.Close()
		uploaded = append(uploaded, filename)
	}

	log.Printf("AUDIT: uploadPDFExcel: exam_id=%d count=%d", examID, len(uploaded))
	writeJSON(w, map[string]any{
		"files": uploaded,
		"count": len(uploaded),
	})
}

// processPDFExcel: POST /admin/exams/{exam_id}/results/pdf-excel/process
// Invokes the converter Lambda (idempotently) and returns its parsed result.
func (h *AdminController) processPDFExcel(w http.ResponseWriter, r *http.Request) {
	if h.pdfExcel == nil {
		http.Error(w, "pdf-excel converter unavailable", http.StatusServiceUnavailable)
		return
	}

	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}
	prefix := examPrefix(examID)

	result, err := h.pdfExcel.Process(r.Context(), prefix)
	if err != nil {
		log.Printf("pdfexcel: Process failed exam_id=%d: %v", examID, err)
		http.Error(w, "failed to convert pdfs", http.StatusBadGateway)
		return
	}

	// Coalesce nil slices to [] so the JSON never sends null for arrays the
	// frontend iterates over.
	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	pdfsProcessed := result.PDFsProcessed
	if pdfsProcessed == nil {
		pdfsProcessed = []string{}
	}

	log.Printf("AUDIT: processPDFExcel: exam_id=%d total_records=%d already_generated=%v", examID, result.TotalRecords, result.AlreadyGenerated)
	writeJSON(w, map[string]any{
		"total_records":     result.TotalRecords,
		"expected_count":    result.ExpectedCount,
		"warnings":          warnings,
		"pdfs_processed":    pdfsProcessed,
		"ready":             true,
		"already_generated": result.AlreadyGenerated,
	})
}

// downloadPDFExcel: GET /admin/exams/{exam_id}/results/pdf-excel/download
// Streams the generated XLSX. The S3 key/URL is never exposed to the client.
func (h *AdminController) downloadPDFExcel(w http.ResponseWriter, r *http.Request) {
	if h.pdfExcel == nil {
		http.Error(w, "pdf-excel converter unavailable", http.StatusServiceUnavailable)
		return
	}

	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}
	prefix := examPrefix(examID)

	body, size, err := h.pdfExcel.Download(r.Context(), prefix)
	if err != nil {
		log.Printf("pdfexcel: Download failed exam_id=%d: %v", examID, err)
		http.Error(w, "converted file not available", http.StatusNotFound)
		return
	}
	defer func() { _ = body.Close() }()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename="+prefix+".xlsx")
	if size != nil && *size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(*size, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}
