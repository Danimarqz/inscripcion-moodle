package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/constants"
	"github.com/inscripcion-moodle/go-backend/internal/models"
)

func (h *AdminController) createAdmin(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, constants.UsernameAndPassword, http.StatusBadRequest)
		return
	}

	var existing int64
	if err := h.db.Model(&models.AdminUser{}).Count(&existing).Error; err != nil {
		http.Error(w, constants.FailedToVerifyAdmin, http.StatusInternalServerError)
		return
	}
	if existing > 0 {
		http.Error(w, constants.AdminAlreadyExists, http.StatusForbidden)
		return
	}

	hash, err := h.auth.HashPassword(payload.Password)
	if err != nil {
		http.Error(w, constants.FailedToHashPassword, http.StatusInternalServerError)
		return
	}

	admin := models.AdminUser{
		Username:     payload.Username,
		PasswordHash: hash,
	}
	if err := h.db.Create(&admin).Error; err != nil {
		http.Error(w, constants.FailedToCreateAdmin, http.StatusInternalServerError)
		return
	}

	h.cache.Del(r.Context(), "admin_not_found:"+payload.Username)

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(constants.AdminCreated))
}

func (h *AdminController) login(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, constants.UsernameAndPassword, http.StatusBadRequest)
		return
	}

	attemptHash := sha256.Sum256([]byte(payload.Username + ":" + payload.Password))
	cacheKey := fmt.Sprintf("admin_auth_failed:%x", attemptHash)
	userNotFoundKey := "admin_not_found:" + payload.Username

	if _, ok := h.cache.Get(r.Context(), cacheKey); ok {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, genericError{Message: constants.InvalidCredentials})
		return
	}
	if _, ok := h.cache.Get(r.Context(), userNotFoundKey); ok {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, genericError{Message: constants.InvalidCredentials})
		return
	}

	var admin models.AdminUser
	if err := h.db.Where("username = ?", payload.Username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.cache.Set(r.Context(), userNotFoundKey, []byte("1"), 5*time.Minute)
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, genericError{Message: constants.InvalidCredentials})
			return
		}
		http.Error(w, constants.FailedToLookupAdmin, http.StatusInternalServerError)
		return
	}

	if !h.auth.VerifyPassword(admin.PasswordHash, payload.Password) {
		h.cache.Set(r.Context(), cacheKey, []byte("1"), 5*time.Minute)
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, genericError{Message: constants.InvalidCredentials})
		return
	}

	token, err := h.auth.CreateToken(admin.Username)
	if err != nil {
		http.Error(w, constants.FailedToCreateToken, http.StatusInternalServerError)
		return
	}

	writeJSON(w, tokenResponse{AccessToken: token})
}

func (h *AdminController) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, constants.MissingAuthHeader, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, constants.InvalidAuthHeader, http.StatusUnauthorized)
			return
		}

		username, err := h.auth.ParseToken(parts[1])
		if err != nil {
			http.Error(w, constants.InvalidToken, http.StatusUnauthorized)
			return
		}

		var admin models.AdminUser
		cacheKey := "admin_auth_user:" + username
		if cachedAdmin, ok := h.cache.Get(r.Context(), cacheKey); ok {
			if err := json.Unmarshal(cachedAdmin, &admin); err == nil {
				ctx := context.WithValue(r.Context(), adminContextKey, &admin)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if err := h.db.Where("username = ?", username).First(&admin).Error; err != nil {
			http.Error(w, constants.Unauthorized, http.StatusUnauthorized)
			return
		}

		if adminBytes, err := json.Marshal(admin); err == nil {
			h.cache.Set(r.Context(), cacheKey, adminBytes, 30*24*time.Hour)
		}

		ctx := context.WithValue(r.Context(), adminContextKey, &admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *AdminController) checkToken(w http.ResponseWriter, r *http.Request) {
	admin := adminFromContext(r.Context())
	if admin == nil {
		http.Error(w, constants.Unauthorized, http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{
		"detail": "Token valido",
		"user":   admin.Username,
	})
}
