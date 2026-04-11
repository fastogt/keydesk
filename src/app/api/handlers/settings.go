package handlers

import (
	"net/http"

	"keydesk/app/database"
)

type SettingsHandler struct {
	db *database.Database
}

func NewSettingsHandler(db *database.Database) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)
	admin, err := h.db.GetAdminByID(adminID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Admin not found")
		return
	}
	respondJSON(w, http.StatusOK, admin)
}

func (h *SettingsHandler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.db.UpdateAdminProfile(adminID, req.Name, req.Email); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *SettingsHandler) HandleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	admin, err := h.db.GetAdminByID(adminID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Admin not found")
		return
	}

	if !h.db.VerifyAdminPassword(admin, req.CurrentPassword) {
		respondError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	if err := h.db.UpdateAdminPassword(adminID, req.NewPassword); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	h.db.LogAudit("updated", "admin", adminID, "", adminID, "Changed password")
	respondJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (h *SettingsHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "CSV import not yet implemented")
}

func (h *SettingsHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "CSV export not yet implemented")
}
