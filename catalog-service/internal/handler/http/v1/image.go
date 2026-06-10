package v1

import (
	"catalog-service/internal/usecase"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ImageHandler struct {
	useCase usecase.ImageUseCase
}

func NewImageHandler(useCase usecase.ImageUseCase) *ImageHandler {
	return &ImageHandler{
		useCase: useCase,
	}
}

func (h *ImageHandler) Register(r chi.Router) {
	r.Post("/upload", h.Upload)
}

type UploadResponse struct {
	URL string `json:"url"`
}

func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "файл слишком большой или неверный формат")
		return
	}

	//Достаем файл по ключу "file" (именно так инпут должен называться на фронтенде)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "не найден файл в поле 'file'")
		return
	}
	defer file.Close()

	url, err := h.useCase.Upload(r.Context(), file, header.Size, header.Filename)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidFileType) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "ошибка сохранения файла")
		return
	}

	writeJSON(w, http.StatusCreated, UploadResponse{URL: url})
}
