package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
)

type AssignmentsHandler struct {
	db *database.Database
}

func NewAssignmentsHandler(db *database.Database) *AssignmentsHandler {
	return &AssignmentsHandler{db: db}
}

func (h *AssignmentsHandler) HandleAssign(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		PersonID  string `json:"person_id"`
		AccountID string `json:"account_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.PersonID == "" || req.AccountID == "" {
		respondError(w, http.StatusBadRequest, "person_id and account_id are required")
		return
	}

	person, err := h.db.GetPerson(req.PersonID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Person not found")
		return
	}

	account, err := h.db.GetAccount(req.AccountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	assignment, err := h.db.CreateAssignment(req.PersonID, req.AccountID, adminID)
	if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	h.db.LogAudit("assigned", "account", req.AccountID, req.PersonID, adminID, "Gave "+account.Name+" to "+person.Name)
	h.db.LogAudit("assigned", "person", req.PersonID, req.PersonID, adminID, "Received access to "+account.Name)

	respondJSON(w, http.StatusCreated, assignment)
}

func (h *AssignmentsHandler) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	if err := h.db.RevokeAssignment(id, "manual"); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke assignment")
		return
	}

	h.db.LogAudit("revoked", "assignment", id, "", adminID, "Revoked account access")
	respondJSON(w, http.StatusOK, map[string]string{"message": "revoked"})
}
