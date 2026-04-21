package handler

import (
	"time"

	"github.com/google/uuid"
)

type createEntryRequest struct {
	Date    time.Time `json:"date"`
	Subject string    `json:"subject"`
	Content string    `json:"content"`
}

type entryResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type errorResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
}
