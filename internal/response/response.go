package response

import (
	"encoding/json"
	"net/http"

	"omie-sync-api/internal/apperror"
)

type body struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type bodyWithMeta struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Meta    any    `json:"meta"`
}

type errorBody struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, data any) {
	write(w, http.StatusOK, body{Success: true, Message: "OK", Data: data})
}

func OKPaginated(w http.ResponseWriter, data any, meta Meta) {
	write(w, http.StatusOK, bodyWithMeta{Success: true, Message: "OK", Data: data, Meta: meta})
}

func Created(w http.ResponseWriter, data any) {
	write(w, http.StatusCreated, body{Success: true, Message: "criado", Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, message string, err error) {
	b := errorBody{Success: false, Message: message}
	if err != nil {
		b.Error = err.Error()
	}
	write(w, status, b)
}

func Unauthorized(w http.ResponseWriter, message string) {
	write(w, http.StatusUnauthorized, errorBody{Success: false, Message: message})
}

func Forbidden(w http.ResponseWriter, message string) {
	write(w, http.StatusForbidden, errorBody{Success: false, Message: message})
}

func NotFound(w http.ResponseWriter, message string) {
	write(w, http.StatusNotFound, errorBody{Success: false, Message: message})
}

// FromAppError converte um AppError no status HTTP correto.
func FromAppError(w http.ResponseWriter, err error) {
	if ae, ok := apperror.IsAppError(err); ok {
		write(w, ae.Code, body{Success: false, Message: ae.Message, Data: ae.Detail})
		return
	}
	Error(w, http.StatusInternalServerError, "erro interno", err)
}
