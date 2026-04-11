package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"keydesk/app/database"
	"keydesk/app/vault"
)

type ExtHandler struct {
	db        *database.Database
	jwtSecret string
	vault     *vault.Vault
}

func NewExtHandler(db *database.Database, jwtSecret string, v *vault.Vault) *ExtHandler {
	return &ExtHandler{db: db, jwtSecret: jwtSecret, vault: v}
}

func (h *ExtHandler) HandleExtLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PersonID string `json:"person_id"`
		PIN      string `json:"pin"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	person, err := h.db.GetPerson(req.PersonID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if person.Status != "active" {
		respondError(w, http.StatusUnauthorized, "Account is not active")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"person_id": person.ID,
		"type":      "extension",
		"exp":       time.Now().Add(8 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token":  tokenString,
		"person": person,
	})
}

func (h *ExtHandler) ExtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			return []byte(h.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			respondError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		tokenType, _ := claims["type"].(string)
		if tokenType != "extension" {
			respondError(w, http.StatusUnauthorized, "Invalid token type")
			return
		}

		personID, ok := claims["person_id"].(string)
		if !ok {
			respondError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), personIDKey, personID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *ExtHandler) HandleExtListAccounts(w http.ResponseWriter, r *http.Request) {
	personID := r.Context().Value(personIDKey).(string)

	assignments, err := h.db.GetActiveAssignmentsByPerson(personID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list accounts")
		return
	}
	if assignments == nil {
		assignments = []database.Assignment{}
	}

	type extAccount struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		LoginURL string `json:"login_url"`
	}

	var accounts []extAccount
	for _, a := range assignments {
		acc, err := h.db.GetAccount(a.AccountID)
		if err != nil {
			continue
		}
		accounts = append(accounts, extAccount{
			ID:       acc.ID,
			Name:     acc.Name,
			Type:     acc.Type,
			LoginURL: acc.LoginURL,
		})
	}

	if accounts == nil {
		accounts = []extAccount{}
	}
	respondJSON(w, http.StatusOK, accounts)
}

func (h *ExtHandler) HandleExtGetCredentials(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	personID := r.Context().Value(personIDKey).(string)

	assignments, err := h.db.GetActiveAssignmentsByPerson(personID)
	if err != nil {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	hasAccess := false
	for _, a := range assignments {
		if a.AccountID == accountID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	account, err := h.db.GetAccount(accountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	password, err := h.vault.Decrypt(account.LoginPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrypt")
		return
	}

	h.db.LogAudit("ext_accessed", "account", accountID, personID, personID, "Extension accessed credentials")

	respondJSON(w, http.StatusOK, map[string]string{
		"login_email": account.LoginEmail,
		"password":    password,
		"login_url":   account.LoginURL,
	})
}

func (h *ExtHandler) HandleExtMatch(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	personID := r.Context().Value(personIDKey).(string)

	if url == "" {
		respondError(w, http.StatusBadRequest, "url parameter required")
		return
	}

	assignments, err := h.db.GetActiveAssignmentsByPerson(personID)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]any{"match": false})
		return
	}

	for _, a := range assignments {
		loginURL, err := h.db.GetAccountLoginURL(a.AccountID)
		if err != nil {
			continue
		}
		if loginURL != "" && strings.Contains(url, loginURL) {
			respondJSON(w, http.StatusOK, map[string]any{
				"match":      true,
				"account_id": a.AccountID,
			})
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{"match": false})
}

func (h *ExtHandler) HandleExtGetTOTP(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	personID := r.Context().Value(personIDKey).(string)

	assignments, err := h.db.GetActiveAssignmentsByPerson(personID)
	if err != nil {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	hasAccess := false
	for _, a := range assignments {
		if a.AccountID == accountID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	account, err := h.db.GetAccount(accountID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	secret, err := h.vault.Decrypt(account.TOTPSecret)
	if err != nil || secret == "" {
		respondError(w, http.StatusBadRequest, "No TOTP configured")
		return
	}

	code, err := generateTOTPCode(secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate TOTP")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"code": code})
}

func (h *ExtHandler) HandleExtAudit(w http.ResponseWriter, r *http.Request) {
	personID := r.Context().Value(personIDKey).(string)

	var req struct {
		Action    string `json:"action"`
		AccountID string `json:"account_id"`
		Details   string `json:"details"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.db.LogAudit("ext_"+req.Action, "account", req.AccountID, personID, personID, req.Details)
	respondJSON(w, http.StatusOK, map[string]string{"message": "logged"})
}
