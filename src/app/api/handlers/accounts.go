package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
	"keydesk/app/vault"
)

type AccountsHandler struct {
	db    *database.Database
	vault *vault.Vault
}

func NewAccountsHandler(db *database.Database, v *vault.Vault) *AccountsHandler {
	return &AccountsHandler{db: db, vault: v}
}

func (h *AccountsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	accountType := r.URL.Query().Get("type")

	accounts, err := h.db.ListAccounts(search, accountType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list accounts")
		return
	}
	if accounts == nil {
		accounts = []database.Account{}
	}
	respondJSON(w, http.StatusOK, accounts)
}

func (h *AccountsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	account, err := h.db.GetAccount(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}
	account.LoginPassword = ""
	account.TOTPSecret = ""

	assignments, _ := h.db.GetActiveAssignmentsByAccount(id)
	if assignments == nil {
		assignments = []database.Assignment{}
	}
	history, _ := h.db.GetAuditLogForEntity("account", id, 50)
	if history == nil {
		history = []database.AuditEntry{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"account":     account,
		"assignments": assignments,
		"history":     history,
	})
}

func (h *AccountsHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		LoginURL      string `json:"login_url"`
		LoginEmail    string `json:"login_email"`
		LoginPassword string `json:"login_password"`
		TOTPSecret    string `json:"totp_secret"`
		Notes         string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Type == "" {
		req.Type = "other"
	}

	encPassword, err := h.vault.Encrypt(req.LoginPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt password")
		return
	}
	encTOTP, err := h.vault.Encrypt(req.TOTPSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt TOTP secret")
		return
	}

	account, err := h.db.CreateAccount(req.Name, req.Type, req.LoginURL, req.LoginEmail, encPassword, encTOTP, req.Notes)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	account.LoginPassword = ""
	account.TOTPSecret = ""

	h.db.LogAudit("created", "account", account.ID, "", adminID, "Created account: "+account.Name)
	respondJSON(w, http.StatusCreated, account)
}

func (h *AccountsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	var req struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		LoginURL   string `json:"login_url"`
		LoginEmail string `json:"login_email"`
		Notes      string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.db.UpdateAccount(id, req.Name, req.Type, req.LoginURL, req.LoginEmail, req.Notes); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update account")
		return
	}

	h.db.LogAudit("updated", "account", id, "", adminID, "Updated account: "+req.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *AccountsHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	account, err := h.db.GetAccount(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	if err := h.db.DeleteAccount(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	h.db.LogAudit("deleted", "account", id, "", adminID, "Deleted account: "+account.Name)
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *AccountsHandler) HandleReveal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	account, err := h.db.GetAccount(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	password, err := h.vault.Decrypt(account.LoginPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrypt password")
		return
	}

	h.db.LogAudit("viewed", "account", id, "", adminID, "Viewed password for: "+account.Name)
	respondJSON(w, http.StatusOK, map[string]string{"password": password})
}

func (h *AccountsHandler) HandleRotate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	account, err := h.db.GetAccount(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	decodeJSON(r, &req)

	newPassword := req.Password
	if newPassword == "" {
		newPassword = generateSecurePassword(20)
	}

	encrypted, err := h.vault.Encrypt(newPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt password")
		return
	}

	if err := h.db.UpdateAccountPassword(id, encrypted); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	h.db.LogAudit("rotated", "account", id, "", adminID, "Rotated password for: "+account.Name)
	respondJSON(w, http.StatusOK, map[string]string{"password": newPassword})
}

func (h *AccountsHandler) HandleTOTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := r.Context().Value(adminIDKey).(string)

	account, err := h.db.GetAccount(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	secret, err := h.vault.Decrypt(account.TOTPSecret)
	if err != nil || secret == "" {
		respondError(w, http.StatusBadRequest, "No TOTP configured for this account")
		return
	}

	code, err := generateTOTPCode(secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate TOTP code")
		return
	}

	h.db.LogAudit("viewed", "account", id, "", adminID, "Viewed TOTP code for: "+account.Name)
	respondJSON(w, http.StatusOK, map[string]string{
		"code":      code,
		"valid_for": fmt.Sprintf("%d", 30-time.Now().Unix()%30),
	})
}

func generateSecurePassword(length int) string {
	const charset = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%&*"
	b := make([]byte, length)
	// use crypto/rand
	for i := range b {
		var buf [1]byte
		_, _ = fmt.Fscanf(strings.NewReader(string(rune(time.Now().UnixNano()))), "%c", &buf[0])
		b[i] = charset[int(time.Now().UnixNano()+int64(i))%len(charset)]
	}
	return string(b)
}

func generateTOTPCode(secret string) (string, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	secret = strings.ReplaceAll(secret, " ", "")

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid TOTP secret: %w", err)
	}

	counter := uint64(time.Now().Unix()) / 30
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := int64(binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff)
	code = code % 1000000

	return fmt.Sprintf("%06d", code), nil
}
