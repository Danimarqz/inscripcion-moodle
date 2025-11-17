package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	registerservice "github.com/inscripcion-moodle/go-backend/internal/services/register"
)

type RegisterHandler struct {
	service *registerservice.Service
}

func NewRegisterHandler(cfg *config.Config) *RegisterHandler {
	return &RegisterHandler{
		service: registerservice.New(cfg),
	}
}

func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var data registerservice.Data
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(data.Website) != "" {
		http.Error(w, "spam detected", http.StatusBadRequest)
		return
	}

	result, err := h.service.Register(r.Context(), data)
	if err != nil {
		http.Error(w, "failed to register submission", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
