package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"user-service/internal/domain"
	"user-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AuthHandler struct {
	useCase usecase.AuthUseCase
}

func NewAuthHandler(useCase usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/me", h.GetProfile)
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	userID, err := h.useCase.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidEmailFormat), errors.Is(err, usecase.ErrPasswordTooShort):
			writeJSONError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrUserAlreadyExists):
			writeJSONError(w, http.StatusConflict, "user with this email already exists")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id": userID.String(),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	token, err := h.useCase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeJSONError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": token,
	})
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-Id")
	if userIDStr == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing X-User-Id header")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user id format")
		return
	}

	user, err := h.useCase.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeJSONError(w, http.StatusNotFound, "user not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := UserProfileResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("02.01.2006 15:04"),
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
