package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
	"keydesk/app/vault"
)

type CredentialsHandler struct {
	db    *database.Database
	vault *vault.Vault
}

func NewCredentialsHandler(db *database.Database, v *vault.Vault) *CredentialsHandler {
	return &CredentialsHandler{db: db, vault: v}
}

func (h *CredentialsHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		ServiceID   string  `json:"service_id"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Provider    string  `json:"provider"`
		KeyValue    string  `json:"key_value"`
		SecretValue string  `json:"secret_value"`
		ExpiresAt   *string `json:"expires_at"`
		WhereUsed   string  `json:"where_used"`
		Notes       string  `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.ServiceID == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "service_id and name are required")
		return
	}
	if req.Type == "" {
		req.Type = "api_key"
	}
	if req.Provider == "" {
		req.Provider = "custom"
	}

	encKey, err := h.vault.Encrypt(req.KeyValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt key")
		return
	}
	encSecret, err := h.vault.Encrypt(req.SecretValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt secret")
		return
	}

	cred, err := h.db.CreateCredential(req.ServiceID, req.Name, req.Type, req.Provider, encKey, encSecret, req.ExpiresAt, req.WhereUsed, req.Notes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create credential")
		return
	}
	cred.KeyValue = ""
	cred.SecretValue = ""

	h.db.LogAudit("created", "credential", cred.ID, "", adminID, "Created credential: "+cred.Name)
	respondJSON(w, http.StatusCreated, cred)
}

func (h *CredentialsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name      string  `json:"name"`
		Type      string  `json:"type"`
		Provider  string  `json:"provider"`
		ExpiresAt *string `json:"expires_at"`
		WhereUsed string  `json:"where_used"`
		Notes     string  `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.db.UpdateCredential(id, req.Name, req.Type, req.Provider, req.ExpiresAt, req.WhereUsed, req.Notes); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update credential")
		return
	}

	h.db.LogAudit("updated", "credential", id, "", adminID, "Updated credential: "+req.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *CredentialsHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	cred, err := h.db.GetCredential(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Credential not found")
		return
	}

	if err := h.db.DeleteCredential(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete credential")
		return
	}

	h.db.LogAudit("deleted", "credential", id, "", adminID, "Deleted credential: "+cred.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *CredentialsHandler) HandleReveal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	cred, err := h.db.GetCredential(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Credential not found")
		return
	}

	keyValue, err := h.vault.Decrypt(cred.KeyValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrypt key")
		return
	}
	secretValue, err := h.vault.Decrypt(cred.SecretValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrypt secret")
		return
	}

	h.db.LogAudit("viewed", "credential", id, "", adminID, "Viewed credential: "+cred.Name)
	respondJSON(w, http.StatusOK, map[string]string{
		"key_value":    keyValue,
		"secret_value": secretValue,
	})
}

func (h *CredentialsHandler) HandleRotate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	cred, err := h.db.GetCredential(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Credential not found")
		return
	}

	var req struct {
		KeyValue    string `json:"key_value"`
		SecretValue string `json:"secret_value"`
	}
	decodeJSON(r, &req)

	encKey, err := h.vault.Encrypt(req.KeyValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt key")
		return
	}
	encSecret, err := h.vault.Encrypt(req.SecretValue)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt secret")
		return
	}

	if err := h.db.UpdateCredentialValue(id, encKey, encSecret); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update credential")
		return
	}

	h.db.LogAudit("rotated", "credential", id, "", adminID, "Rotated credential: "+cred.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "rotated"})
}
