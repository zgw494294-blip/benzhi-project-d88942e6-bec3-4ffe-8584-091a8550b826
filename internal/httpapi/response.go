package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"termpack/internal/domain"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code, message := "internal_error", "服务暂时无法处理请求"
	var rule *domain.RuleError
	if errors.As(err, &rule) {
		code, message = rule.Code, rule.Message
		switch rule.Code {
		case "not_found":
			status = http.StatusNotFound
		case "version_conflict", "invalid_transition", "invalid_state", "released_immutable", "review_incomplete", "open_findings", "finding_closed", "duplicate_term":
			status = http.StatusConflict
		default:
			status = http.StatusUnprocessableEntity
		}
	}
	var details any
	if rule != nil {
		details = rule.Details
	}
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message, Details: details}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, errorEnvelope{Error: apiError{Code: "content_type_required", Message: "请求必须使用 application/json"}})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "invalid_json", Message: "JSON 请求格式无效"}})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "invalid_json", Message: "请求只能包含一个 JSON 对象"}})
		return false
	}
	return true
}
