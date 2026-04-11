package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"keydesk/app/database"
)

type AuthHandler struct {
	db        *database.Database
	jwtSecret string
}

func NewAuthHandler(db *database.Database, jwtSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret}
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	admin, err := h.db.GetAdminByEmail(req.Email)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !h.db.VerifyAdminPassword(admin, req.Password) {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": admin.ID,
		"email":    admin.Email,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token": tokenString,
		"admin": admin,
	})
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value(adminIDKey).(string)
	admin, err := h.db.GetAdminByID(adminID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Admin not found")
		return
	}
	respondJSON(w, http.StatusOK, admin)
}

func (h *AuthHandler) AdminAuthMiddleware(next http.Handler) http.Handler {
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

		adminID, ok := claims["admin_id"].(string)
		if !ok {
			respondError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), adminIDKey, adminID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
