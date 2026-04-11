package handlers

import (
	"net/http"

	"keydesk/app/database"
)

type DashboardHandler struct {
	db *database.Database
}

func NewDashboardHandler(db *database.Database) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetDashboardStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	recentActivity, _ := h.db.GetRecentAuditLog(20)
	if recentActivity == nil {
		recentActivity = []database.AuditEntry{}
	}

	expiringCreds, _ := h.db.GetExpiringCredentials(30)
	if expiringCreds == nil {
		expiringCreds = []database.Credential{}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"stats":                stats,
		"recent_activity":      recentActivity,
		"expiring_credentials": expiringCreds,
	})
}
