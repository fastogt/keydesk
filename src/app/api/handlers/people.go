package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
	"keydesk/app/vault"
)

type PeopleHandler struct {
	db    *database.Database
	vault *vault.Vault
}

func NewPeopleHandler(db *database.Database, v *vault.Vault) *PeopleHandler {
	return &PeopleHandler{db: db, vault: v}
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
		RotatePasswords   bool              `json:"rotate_passwords"`
		ServiceOwners     map[string]string `json:"service_owners"`
		RotateCredentials bool              `json:"rotate_credentials"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	revokedAccountIDs, err := h.db.RevokeAllAssignmentsForPerson(id, "offboarded")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke assignments")
		return
	}

	newPasswords := map[string]string{}
	if req.RotatePasswords {
		for _, accountID := range revokedAccountIDs {
			newPass := generatePassword(20)
			encrypted, err := h.vault.Encrypt(newPass)
			if err != nil {
				continue
			}
			h.db.UpdateAccountPassword(accountID, encrypted)
			newPasswords[accountID] = newPass
			h.db.LogAudit("rotated", "account", accountID, id, adminID, "Password rotated during offboarding of "+person.Name)
		}
	}

	for serviceID, newOwnerID := range req.ServiceOwners {
		h.db.UpdateServiceOwner(serviceID, newOwnerID)
		h.db.LogAudit("reassigned", "service", serviceID, id, adminID, "Service ownership reassigned during offboarding of "+person.Name)
	}

	h.db.OffboardPerson(id)
	h.db.LogAudit("offboarded", "person", id, id, adminID, "Offboarded person: "+person.Name+". Revoked "+string(rune(len(revokedAccountIDs)+'0'))+" accounts.")

	respondJSON(w, http.StatusOK, map[string]any{
		"message":       "offboarded",
		"revoked_count": len(revokedAccountIDs),
		"new_passwords": newPasswords,
	})
}

func generatePassword(length int) string {
	const charset = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%&*"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	// use crypto/rand for real randomness
	return string(b)
}
