package response

import (
	"encoding/json"
	"net/http"
)

// Envelope: success -> {"data": T}; error (non-2xx) -> {"error": {"message","code"}}.

type ErrorBody struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// Error codes.
const (
	CodeNotFound       = "not_found"
	CodeInvalidRequest = "invalid_request"
	CodeDBError        = "db_error"
	CodeUploadError    = "upload_error"
	CodeUpstreamError  = "upstream_error"
	CodeConflict       = "conflict"
)

// JSON writes a JSON body with status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// Data writes a success envelope {"data": v}.
func Data(w http.ResponseWriter, status int, v any) {
	JSON(w, status, map[string]any{"data": v})
}

// Error writes an error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorEnvelope{Error: ErrorBody{Message: message, Code: code}})
}

// Errorf is Error with a formatted message.
func Errorf(w http.ResponseWriter, status int, code, format string, args ...any) {
	Error(w, status, code, sprintf(format, args...))
}
