package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
)

type PeopleHandler struct {
	db *database.Database
}

func NewPeopleHandler(db *database.Database) *PeopleHandler {
	return &PeopleHandler{db: db}
}

func (h *PeopleHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	people, err := h.db.ListPeople(search, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list people")
		return
	}
	if people == nil {
		people = []database.Person{}
	}
	respondJSON(w, http.StatusOK, people)
}

func (h *PeopleHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	person, err := h.db.GetPerson(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Person not found")
		return
	}

	assignments, _ := h.db.GetActiveAssignmentsByPerson(id)
	if assignments == nil {
		assignments = []database.Assignment{}
	}
	services, _ := h.db.GetServicesByOwner(id)
	if services == nil {
		services = []database.Service{}
	}
	history, _ := h.db.GetAuditLogForEntity("person", id, 50)
	if history == nil {
		history = []database.AuditEntry{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"person":      person,
		"assignments": assignments,
		"services":    services,
		"history":     history,
	})
}

func (h *PeopleHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Department string `json:"department"`
		Notes      string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	person, err := h.db.CreatePerson(req.Name, req.Email, req.Department, req.Notes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create person")
		return
	}

	h.db.LogAudit("created", "person", person.ID, "", adminID, "Created person: "+person.Name)
	respondJSON(w, http.StatusCreated, person)
}

func (h *PeopleHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Department string `json:"department"`
		Notes      string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.db.UpdatePerson(id, req.Name, req.Email, req.Department, req.Notes); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update person")
		return
	}

	h.db.LogAudit("updated", "person", id, "", adminID, "Updated person: "+req.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *PeopleHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	person, err := h.db.GetPerson(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Person not found")
		return
	}

	if err := h.db.DeletePerson(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete person")
		return
	}

	h.db.LogAudit("deleted", "person", id, "", adminID, "Deleted person: "+person.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *PeopleHandler) HandleOffboard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	person, err := h.db.GetPerson(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Person not found")
		return
	}

	var req struct {
		ServiceOwners map[string]string `json:"service_owners"`
	}
	decodeJSON(r, &req)

	revokedAccountIDs, err := h.db.RevokeAllAssignmentsForPerson(id, "offboarded")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke assignments")
		return
	}

	for serviceID, newOwnerID := range req.ServiceOwners {
		h.db.UpdateServiceOwner(serviceID, newOwnerID)
		h.db.LogAudit("reassigned", "service", serviceID, id, adminID, "Service ownership reassigned during offboarding of "+person.Name)
	}

	h.db.OffboardPerson(id)
	h.db.LogAudit("offboarded", "person", id, id, adminID, fmt.Sprintf("Offboarded %s. Revoked %d accounts.", person.Name, len(revokedAccountIDs)))

	respondJSON(w, http.StatusOK, map[string]any{
		"message":       "offboarded",
		"revoked_count": len(revokedAccountIDs),
	})
}
