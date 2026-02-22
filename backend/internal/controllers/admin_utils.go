package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/inscripcion-moodle/go-backend/internal/models"
)

func adminFromContext(ctx context.Context) *models.AdminUser {
	val, _ := ctx.Value(adminContextKey).(*models.AdminUser)
	return val
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func parseLimitParam(raw string) int {
	if raw == "" {
		return defaultSubmissionsLimit
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultSubmissionsLimit
	}
	if value > maxSubmissionsLimit {
		return maxSubmissionsLimit
	}
	return value
}

func parseOffsetParam(raw string) int {
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func parseOptionalBool(raw string) *bool {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(raw)))
	if err != nil {
		return nil
	}
	return &value
}

func parseBoolParam(raw string, def bool) bool {
	if raw == "" {
		return def
	}
	value, err := strconv.ParseBool(strings.ToLower(raw))
	if err != nil {
		return def
	}
	return value
}

func sanitizeOrderBy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "score":
		return "score"
	case "name":
		return "name"
	case "surname":
		return "surname"
	case "submitted_at", "time":
		return "submitted_at"
	default:
		return "submitted_at"
	}
}

func sanitizeOrderDir(raw string) string {
	dir := strings.ToLower(strings.TrimSpace(raw))
	if dir == "asc" {
		return "asc"
	}
	return "desc"
}

func sanitizeOfficialResultsOrderBy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dni":
		return "dni"
	case "nombre":
		return "nombre"
	case "apellidos":
		return "apellidos"
	case "usuario":
		return "usuario"
	case "creado", "created_at":
		return "creado"
	default:
		return "apellidos"
	}
}

func (h *AdminController) invalidateExamCaches(examID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	_ = h.cache.Del(ctx, examsCacheKey)
	_ = h.cache.Del(ctx, fmt.Sprintf("%s:%d", questionsCachePrefix, examID))
}

func parseUintURLParam(r *http.Request, param string) (uint, error) {
	valStr := chi.URLParam(r, param)
	if valStr == "" {
		return 0, fmt.Errorf("param %s is required", param)
	}
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", param)
	}
	return uint(val), nil
}

func parseUintQueryParam(r *http.Request, param string) (uint, error) {
	valStr := r.URL.Query().Get(param)
	if valStr == "" {
		return 0, fmt.Errorf("query param %s is required", param)
	}
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", param)
	}
	return uint(val), nil
}
