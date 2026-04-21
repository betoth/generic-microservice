package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/betoth/generic-microservice/internal/service"
	"github.com/google/uuid"
)

type Handler struct {
	entryService *service.EntryService
}

func New(entryService *service.EntryService) *Handler {
	return &Handler{entryService: entryService}
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) PostEntries(w http.ResponseWriter, r *http.Request) {
	var req createEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "400", "Bad Request")
		return
	}

	output, err := h.entryService.CreateEntry(r.Context(), service.CreateEntryInput{
		Date:    req.Date,
		Subject: req.Subject,
		Content: req.Content,
	})
	if err != nil {
		var bizErr *domain.BusinessError
		if errors.As(err, &bizErr) {
			writeError(w, http.StatusUnprocessableEntity, bizErr.Code, bizErr.Description)
			return
		}
		writeError(w, http.StatusInternalServerError, "500", "Internal Server Error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entryResponse{
		ID:        output.ID,
		CreatedAt: output.CreatedAt,
	})
}

func writeError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		ID:          uuid.New(),
		Code:        code,
		Description: description,
	})
}
