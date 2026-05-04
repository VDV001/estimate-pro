package errors

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func BadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, "CONFLICT", message)
}

func TooManyRequests(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", message)
}

func PayloadTooLarge(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", message)
}

func InternalError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
