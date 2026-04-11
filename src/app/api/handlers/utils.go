package handlers

import (
	"encoding/json"
	"net/http"

	"gitlab.com/fastogt/gofastogt/gofastogt"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	response, err := json.Marshal(gofastogt.NewOkResponse(data))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, status int, message string) {
	errJson := gofastogt.ErrorJson{Code: status, Message: message}
	response, err := json.Marshal(gofastogt.NewErrorResponse(errJson))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(response)
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type contextKey string

const adminIDKey contextKey = "admin_id"
const personIDKey contextKey = "person_id"
