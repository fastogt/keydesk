package database

import (
	"fmt"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	PersonID    string `json:"person_id,omitempty"`
	PerformedBy string `json:"performed_by"`
	Details     string `json:"details"`
	Timestamp   string `json:"timestamp"`
}

func (d *Database) LogAudit(action, entityType, entityID, personID, performedBy, details string) error {
	_, err := d.db.Exec(
		`INSERT INTO audit_log (id, action, entity_type, entity_id, person_id, performed_by, details, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), action, entityType, entityID, personID, performedBy, details, nowUTC(),
	)
	return err
}

func (d *Database) GetRecentAuditLog(limit int) ([]AuditEntry, error) {
	rows, err := d.db.Query(
		`SELECT id, action, entity_type, entity_id, COALESCE(person_id, ''), performed_by, details, timestamp
		 FROM audit_log ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.EntityType, &e.EntityID, &e.PersonID, &e.PerformedBy, &e.Details, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *Database) GetAuditLogForEntity(entityType, entityID string, limit int) ([]AuditEntry, error) {
	rows, err := d.db.Query(
		`SELECT id, action, entity_type, entity_id, COALESCE(person_id, ''), performed_by, details, timestamp
		 FROM audit_log WHERE entity_type = ? AND entity_id = ? ORDER BY timestamp DESC LIMIT ?`,
		entityType, entityID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.EntityType, &e.EntityID, &e.PersonID, &e.PerformedBy, &e.Details, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *Database) GetDashboardStats() (map[string]int, error) {
	stats := map[string]int{}

	var people, accounts, services, credentials, expiring, expired, unassigned int
	d.db.QueryRow(`SELECT COUNT(*) FROM people WHERE status = 'active'`).Scan(&people)
	d.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&accounts)
	d.db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&services)
	d.db.QueryRow(`SELECT COUNT(*) FROM credentials`).Scan(&credentials)
	d.db.QueryRow(`SELECT COUNT(*) FROM credentials WHERE expires_at IS NOT NULL AND expires_at != '' AND expires_at <= datetime('now', '+30 days') AND expires_at > datetime('now')`).Scan(&expiring)
	d.db.QueryRow(`SELECT COUNT(*) FROM credentials WHERE expires_at IS NOT NULL AND expires_at != '' AND expires_at <= datetime('now')`).Scan(&expired)
	d.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE id NOT IN (SELECT account_id FROM assignments WHERE revoked_at IS NULL)`).Scan(&unassigned)

	stats["people"] = people
	stats["accounts"] = accounts
	stats["services"] = services
	stats["credentials"] = credentials
	stats["expiring_credentials"] = expiring
	stats["expired_credentials"] = expired
	stats["unassigned_accounts"] = unassigned

	return stats, nil
}
