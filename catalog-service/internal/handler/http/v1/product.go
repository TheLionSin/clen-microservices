package v1

import (
	"catalog-service/internal/domain"
	"catalog-service/internal/usecase"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProductHandler struct {
	useCase usecase.ProductUseCase
}

func NewProductHandler(u usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{
		useCase: u,
	}
}

func (h *ProductHandler) Register(r chi.Router) {
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}", h.GetByID)
	r.Get("/", h.List)
}

type CreateRequest struct {
	CategoryID  string   `json:"category_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Stock       int      `json:"stock"`
	ImageURLs   []string `json:"image_urls"`
}

type UpdateRequest struct {
	ID          string   `json:"id"`
	CategoryID  string   `json:"category_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Stock       int      `json:"stock"`
	ImageURLs   []string `json:"image_urls"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid category_id format")
		return
	}

	input := usecase.CreateProductInput{
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		ImageURLs:   req.ImageURLs,
	}

	id, err := h.useCase.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid uuid format")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid category_id format")
		return
	}

	input := usecase.UpdateProductInput{
		ID:          productID,
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		ImageURLs:   req.ImageURLs,
	}

	if err := h.useCase.Update(r.Context(), input); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			writeJSONError(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, usecase.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid uuid format")
		return
	}
	if err := h.useCase.Delete(r.Context(), productID); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			writeJSONError(w, http.StatusNotFound, "product not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid uuid format")
		return
	}

	product, err := h.useCase.GetByID(r.Context(), productID)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	products, err := h.useCase.List(r.Context(), limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
