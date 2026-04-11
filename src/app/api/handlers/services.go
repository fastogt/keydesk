package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
)

type ServicesHandler struct {
	db *database.Database
}

func NewServicesHandler(db *database.Database) *ServicesHandler {
	return &ServicesHandler{db: db}
}

func (h *ServicesHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	services, err := h.db.ListServices(search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list services")
		return
	}
	if services == nil {
		services = []database.Service{}
	}
	respondJSON(w, http.StatusOK, services)
}

func (h *ServicesHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	service, err := h.db.GetService(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Service not found")
		return
	}

	credentials, _ := h.db.ListCredentialsByService(id)
	if credentials == nil {
		credentials = []database.Credential{}
	}
	history, _ := h.db.GetAuditLogForEntity("service", id, 50)
	if history == nil {
		history = []database.AuditEntry{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"service":     service,
		"credentials": credentials,
		"history":     history,
	})
}

func (h *ServicesHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Environment string `json:"environment"`
		OwnerID     string `json:"owner_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Environment == "" {
		req.Environment = "production"
	}

	service, err := h.db.CreateService(req.Name, req.Description, req.Environment, req.OwnerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create service")
		return
	}

	h.db.LogAudit("created", "service", service.ID, "", adminID, "Created service: "+service.Name)
	respondJSON(w, http.StatusCreated, service)
}

func (h *ServicesHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Environment string `json:"environment"`
		OwnerID     string `json:"owner_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.db.UpdateService(id, req.Name, req.Description, req.Environment, req.OwnerID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update service")
		return
	}

	h.db.LogAudit("updated", "service", id, "", adminID, "Updated service: "+req.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *ServicesHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	service, err := h.db.GetService(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Service not found")
		return
	}

	if err := h.db.DeleteService(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete service")
		return
	}

	h.db.LogAudit("deleted", "service", id, "", adminID, "Deleted service: "+service.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
