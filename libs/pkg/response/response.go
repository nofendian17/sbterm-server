// Package response standardizes REST API responses into a single envelope
// with consistent, machine-readable error semantics.
package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
)

const (
	CodeBadRequest      = "BAD_REQUEST"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeValidation      = "VALIDATION_ERROR"
	CodeInternalError   = "INTERNAL_ERROR"
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
	CodeUpstreamError   = "UPSTREAM_ERROR"
)

type Envelope struct {
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
	Data    any        `json:"data,omitempty"`
	Meta    *MetaBody  `json:"meta,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

type MetaBody struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func OK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Success: true, Data: data})
}

func Created(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusCreated, Envelope{Success: true, Data: data})
}

// Paginated sends a 200 OK response with data and pagination metadata.
func Paginated(w http.ResponseWriter, data any, meta *MetaBody) {
	WriteJSON(w, http.StatusOK, Envelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Success(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, Envelope{Success: true, Data: data})
}

func Message(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, Envelope{Success: true, Message: message})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, Envelope{
		Success: false,
		Message: message,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func ValidationError(w http.ResponseWriter, message string, details map[string]string) {
	WriteJSON(w, http.StatusUnprocessableEntity, Envelope{
		Success: false,
		Message: message,
		Error: &ErrorBody{
			Code:    CodeValidation,
			Message: message,
			Details: details,
		},
	})
}

// Upstream maps one upstream failure onto the envelope and returns true when
// it wrote a response. Status 400 (or any status in alsoNoData) becomes 422
// VALIDATION_ERROR with message; 429 becomes 429 TOO_MANY_REQUESTS honoring
// retryAfter (RFC 8574, seconds or HTTP-date, passed through verbatim);
// anything else becomes 500 with fallback.
func Upstream(w http.ResponseWriter, upstreamStatus int, retryAfter, message, fallback string, alsoNoData ...int) bool {
	switch {
	case upstreamStatus == http.StatusBadRequest || slices.Contains(alsoNoData, upstreamStatus):
		Error(w, http.StatusUnprocessableEntity, CodeValidation, message)
	case upstreamStatus == http.StatusTooManyRequests:
		if retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		Error(w, http.StatusTooManyRequests, CodeTooManyRequests, "upstream rate limited; slow down and retry")
	default:
		Error(w, http.StatusInternalServerError, CodeInternalError, fallback)
	}
	return true
}
