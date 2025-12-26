package controllers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
	"github.com/inscripcion-moodle/go-backend/internal/services/email"
)

type sendSubmissionEmailsRequest struct {
	ExamID       uint                          `json:"exam_id"`
	Subject      string                        `json:"subject"`
	Body         string                        `json:"body"`
	Recipients   []string                      `json:"recipients"`
	Attachments  []adminEmailAttachmentRequest `json:"attachments"`
	Search       string                        `json:"search"`
	OrderBy      string                        `json:"order_by"`
	OrderDir     string                        `json:"order_dir"`
	MoodleSynced *bool                         `json:"moodle_synced"`
}

type adminEmailAttachmentRequest struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

func (h *AdminController) listSubmissions(w http.ResponseWriter, r *http.Request) {
	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		http.Error(w, "exam_id required", http.StatusBadRequest)
		return
	}
	examID, err := strconv.ParseUint(examIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"))
	offset := parseOffsetParam(r.URL.Query().Get("offset"))
	includeStats := parseBoolParam(r.URL.Query().Get("first_load"), true)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	orderBy := sanitizeOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	moodleSynced := parseOptionalBool(r.URL.Query().Get("moodle_synced"))
	result, err := h.service.ListSubmissions(uint(examID), limit, offset, includeStats, search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submissions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *AdminController) downloadSubmissionEmails(w http.ResponseWriter, r *http.Request) {
	examID, search, orderBy, orderDir, moodleSynced, err := parseSubmissionEmailFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(uint(examID), search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("emails_exam_%d.txt", examID)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := w.Write([]byte(strings.Join(emails, "\n"))); err != nil {
		log.Printf("failed to write emails response: %v", err)
	}
}

func (h *AdminController) listSubmissionEmailsJSON(w http.ResponseWriter, r *http.Request) {
	examID, search, orderBy, orderDir, moodleSynced, err := parseSubmissionEmailFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(uint(examID), search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}

	writeJSON(w, emails)
}

func (h *AdminController) sendSubmissionEmails(w http.ResponseWriter, r *http.Request) {
	var payload sendSubmissionEmailsRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if payload.ExamID == 0 {
		http.Error(w, "exam_id is required", http.StatusBadRequest)
		return
	}
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		http.Error(w, "email body is required", http.StatusBadRequest)
		return
	}
	if len(payload.Recipients) == 0 {
		http.Error(w, "at least one recipient is required", http.StatusBadRequest)
		return
	}

	attachments, err := decodeAdminEmailAttachments(payload.Attachments)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid attachments: %v", err), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(payload.ExamID, payload.Search, payload.OrderBy, payload.OrderDir, payload.MoodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}
	allowed := make(map[string]struct{}, len(emails))
	for _, candidate := range emails {
		allowed[strings.ToLower(strings.TrimSpace(candidate))] = struct{}{}
	}

	var (
		validRecipients []string
		invalid         []string
		seen            = make(map[string]struct{}, len(payload.Recipients))
	)
	for _, raw := range payload.Recipients {
		email := strings.TrimSpace(raw)
		if email == "" {
			continue
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		if _, ok := allowed[lower]; !ok {
			invalid = append(invalid, email)
			continue
		}
		validRecipients = append(validRecipients, email)
	}

	if len(invalid) > 0 {
		http.Error(w, fmt.Sprintf("los siguientes correos no pertenecen al listado filtrado: %s", strings.Join(invalid, ", ")), http.StatusBadRequest)
		return
	}
	if len(validRecipients) == 0 {
		http.Error(w, "no valid recipients to send email", http.StatusBadRequest)
		return
	}

	subject := strings.TrimSpace(payload.Subject)
	if subject == "" {
		subject = "Comunicado oficial"
	}

	toAddress := strings.TrimSpace(h.cfg.AdminEmail)
	if toAddress == "" {
		toAddress = strings.TrimSpace(h.cfg.SMTPUser)
	}
	if toAddress == "" {
		http.Error(w, "admin email not configured", http.StatusInternalServerError)
		return
	}
	if err := email.SendEmail(h.cfg, []string{toAddress}, subject, body, attachments, validRecipients); err != nil {
		log.Printf("failed to send admin emails: %v", err)
		http.Error(w, "failed to send emails", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]int{"sent": len(validRecipients)})
}

func (h *AdminController) updateSubmission(w http.ResponseWriter, r *http.Request) {
	submissionIDStr := chi.URLParam(r, "submission_id")
	submissionID, err := strconv.ParseUint(submissionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid submission id", http.StatusBadRequest)
		return
	}
	var payload admin.SubmissionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.service.UpdateSubmission(uint(submissionID), payload)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Intento no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update submission", http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func (h *AdminController) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID, err := strconv.ParseUint(chi.URLParam(r, "submission_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid submission id", http.StatusBadRequest)
		return
	}
	examID, err := h.service.DeleteSubmission(uint(submissionID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Intento no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete submission", http.StatusInternalServerError)
		return
	}
	h.invalidateExamCaches(examID)
	writeJSON(w, map[string]string{"detail": "Intento eliminado correctamente"})
}

func parseSubmissionEmailFilters(r *http.Request) (uint, string, string, string, *bool, error) {
	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		return 0, "", "", "", nil, fmt.Errorf("exam_id required")
	}
	examID, err := strconv.ParseUint(examIDStr, 10, 64)
	if err != nil {
		return 0, "", "", "", nil, fmt.Errorf("invalid exam id")
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	orderBy := sanitizeOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	moodleSynced := parseOptionalBool(r.URL.Query().Get("moodle_synced"))
	return uint(examID), search, orderBy, orderDir, moodleSynced, nil
}

func decodeAdminEmailAttachments(items []adminEmailAttachmentRequest) ([]email.Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	attachments := make([]email.Attachment, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Filename) == "" {
			return nil, fmt.Errorf("attachment filename is required")
		}
		content, err := decodeBase64Payload(item.Content)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, email.Attachment{
			Filename:    strings.TrimSpace(item.Filename),
			Content:     content,
			ContentType: strings.TrimSpace(item.ContentType),
		})
	}
	return attachments, nil
}

func decodeBase64Payload(raw string) ([]byte, error) {
	if raw == "" {
		return nil, fmt.Errorf("attachment content is empty")
	}
	if idx := strings.Index(raw, ","); idx != -1 && strings.HasPrefix(raw[:idx], "data:") {
		raw = raw[idx+1:]
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
