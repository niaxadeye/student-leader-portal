// Package httpserver — HTTP-сервер, envelope-ответы и middleware (SITE.md §20, §50).
package httpserver

import (
	"encoding/json"
	"net/http"
)

// Envelope успеха: {data, meta, request_id} (SITE.md §20).
type successEnvelope struct {
	Data      any    `json:"data"`
	Meta      any    `json:"meta,omitempty"`
	RequestID string `json:"request_id"`
}

// Envelope ошибки: {error:{code,message,details}, request_id}.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error     errorBody `json:"error"`
	RequestID string    `json:"request_id"`
}

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successEnvelope{
		Data:      data,
		Meta:      meta,
		RequestID: RequestIDFrom(r.Context()),
	})
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{
		Error:     errorBody{Code: code, Message: message, Details: details},
		RequestID: RequestIDFrom(r.Context()),
	})
}
