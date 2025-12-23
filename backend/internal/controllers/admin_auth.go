package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

func (h *AdminController) createAdmin(w http.ResponseWriter, r *http.Request) {
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var existing int64
	if err := h.db.Model(&models.AdminUser{}).Count(&existing).Error; err != nil {
		http.Error(w, "failed to verify existing admin", http.StatusInternalServerError)
		return
	}
	if existing > 0 {
		http.Error(w, "administrador ya existe", http.StatusForbidden)
		return
	}

	hash, err := h.auth.HashPassword(payload.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	admin := models.AdminUser{
		Username:     payload.Username,
		PasswordHash: hash,
	}
	if err := h.db.Create(&admin).Error; err != nil {
		http.Error(w, "failed to create admin user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"message":"Administrador creado con exito"}`))
}

func (h *AdminController) login(w http.ResponseWriter, r *http.Request) {
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var admin models.AdminUser
	if err := h.db.Where("username = ?", payload.Username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, genericError{Message: "credenciales invalidas"})
			return
		}
		http.Error(w, "failed to lookup admin", http.StatusInternalServerError)
		return
	}

	if !h.auth.VerifyPassword(admin.PasswordHash, payload.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, genericError{Message: "credenciales invalidas"})
		return
	}

	token, err := h.auth.CreateToken(admin.Username)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, tokenResponse{AccessToken: token})
}

func (h *AdminController) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		username, err := h.auth.ParseToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		var admin models.AdminUser
		if err := h.db.Where("username = ?", username).First(&admin).Error; err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), adminContextKey, &admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *AdminController) checkToken(w http.ResponseWriter, r *http.Request) {
	admin := adminFromContext(r.Context())
	if admin == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{
		"detail": "Token valido",
		"user":   admin.Username,
	})
}
