package controllers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/constants"
	registerservice "github.com/inscripcion-moodle/go-backend/internal/services/register"
)

type RegisterController struct {
	service *registerservice.Service
}

func NewRegisterController(cfg *config.Config) *RegisterController {
	return &RegisterController{
		service: registerservice.New(cfg),
	}
}

func (h *RegisterController) Register(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	var data registerservice.Data
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(data.Website) != "" {
		http.Error(w, constants.SpamDetected, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(data.Email) == "" || strings.TrimSpace(data.DNI) == "" || strings.TrimSpace(data.Name) == "" || strings.TrimSpace(data.Surname) == "" {
		http.Error(w, constants.AllFieldsRequired, http.StatusBadRequest)
		return
	}

	result, err := h.service.Register(r.Context(), data)
	if err != nil {
		http.Error(w, constants.FailedToRegister, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
