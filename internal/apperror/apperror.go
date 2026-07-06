package apperror

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Detail  any `json:",omitempty"`
}

func (e AppError) Error() string {
	return e.Message
}

func ConflictWithDetail(msg string, detail any) AppError {
	return AppError{Code: http.StatusConflict, Message: msg, Detail: detail}
}

func NotFound(message string) AppError {
	return AppError{Code: http.StatusNotFound, Message: message}
}

func Forbidden(message string) AppError {
	return AppError{Code: http.StatusForbidden, Message: message}
}

func Unprocessable(message string) AppError {
	return AppError{Code: http.StatusUnprocessableEntity, Message: message}
}

func Conflict(message string) AppError {
	return AppError{Code: http.StatusConflict, Message: message}
}

func Unauthorized(message string) AppError {
	return AppError{Code: http.StatusUnauthorized, Message: message}
}

func Internal(message string) AppError {
	return AppError{Code: http.StatusInternalServerError, Message: message}
}

// Wrap adiciona contexto a um AppError preservando o Code original.
func Wrap(err AppError, context string) AppError {
	return AppError{
		Code:    err.Code,
		Message: fmt.Sprintf("%s: %s", context, err.Message),
	}
}

// IsAppError verifica se um erro é do tipo AppError.
func IsAppError(err error) (AppError, bool) {
	ae, ok := err.(AppError)
	return ae, ok
}
