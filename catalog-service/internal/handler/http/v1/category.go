package v1

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	useCase usecase.CategoryUseCase
}

func NewCategoryHandler(useCase usecase.CategoryUseCase) *CategoryHandler {
	return &CategoryHandler{useCase: useCase}
}

func (h *CategoryHandler) Register(r chi.Router) {
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.List)
}

type CreateCategoryRequest struct {
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
}

type UpdateCategoryRequest struct {
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	var parsedParentID *uuid.UUID
	//Двойная проверка. 1.От паники если пришлют без ParentID
	//                  2.Защита от пустой строки parent_id : ""
	if req.ParentID != nil && *req.ParentID != "" {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent_id format")
			return
		}
		parsedParentID = &id
	}

	input := usecase.CreateCategoryInput{
		ParentID: parsedParentID,
		Name:     req.Name,
	}

	id, err := h.useCase.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	categoryID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid uuid format")
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	var parsedParentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent_id format")
			return
		}
		parsedParentID = &id
	}

	input := usecase.UpdateCategoryInput{
		ID:       categoryID,
		ParentID: parsedParentID,
		Name:     req.Name,
	}

	if err := h.useCase.Update(r.Context(), input); err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			writeJSONError(w, http.StatusNotFound, "category not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	categoryID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid uuid format")
		return
	}

	if err := h.useCase.Delete(r.Context(), categoryID); err != nil {
		if errors.Is(err, domain.ErrCategoryNotFound) {
			writeJSONError(w, http.StatusNotFound, "category not found")
			return
		}
		if errors.Is(err, domain.ErrCategoryInUse) {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.useCase.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}
