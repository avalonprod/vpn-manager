package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPageSize = 25
	maxPageSize     = 200
)

type errorResponse struct {
	Error string `json:"error"`
}

// page — обёртка постраничного ответа.
type page[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if body == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// decodeJSON читает тело запроса строго: неизвестные поля отвергаются, чтобы
// опечатка в клиенте не проходила молча.
func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

// pagination разбирает limit/offset с безопасными значениями по умолчанию.
func pagination(r *http.Request) (limit, offset int) {
	query := r.URL.Query()

	limit = defaultPageSize
	if raw := query.Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = min(parsed, maxPageSize)
		}
	}

	if raw := query.Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}

	return limit, offset
}

// daysParam разбирает окно аналитики в днях (по умолчанию 30, максимум 365).
func daysParam(r *http.Request) int {
	days := 30

	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			days = min(parsed, 365)
		}
	}

	return days
}

func userIDParam(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

// errNotFound помечает ошибки, которые должны стать 404 вместо 500.
var errNotFound = errors.New("not found")
