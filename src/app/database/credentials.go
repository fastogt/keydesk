package database

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Credential struct {
	ID            string  `json:"id"`
	ServiceID     string  `json:"service_id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Provider      string  `json:"provider"`
	KeyValue      string  `json:"key_value,omitempty"`
	SecretValue   string  `json:"secret_value,omitempty"`
	ExpiresAt     *string `json:"expires_at"`
	LastRotatedAt *string `json:"last_rotated_at"`
	WhereUsed     string  `json:"where_used"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func (d *Database) ListCredentialsByService(serviceID string) ([]Credential, error) {
	rows, err := d.db.Query(
		`SELECT id, service_id, name, type, provider, expires_at, last_rotated_at, where_used, notes, created_at, updated_at
		 FROM credentials WHERE service_id = ? ORDER BY name ASC`, serviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		var expiresAt, lastRotatedAt sql.NullString
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.Name, &c.Type, &c.Provider, &expiresAt, &lastRotatedAt, &c.WhereUsed, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.String
		}
		if lastRotatedAt.Valid {
			c.LastRotatedAt = &lastRotatedAt.String
		}
		creds = append(creds, c)
	}
	return creds, nil
}

func (d *Database) GetCredential(id string) (*Credential, error) {
	var c Credential
	var expiresAt, lastRotatedAt sql.NullString
	err := d.db.QueryRow(
		`SELECT id, service_id, name, type, provider, key_value, secret_value, expires_at, last_rotated_at, where_used, notes, created_at, updated_at
		 FROM credentials WHERE id = ?`, id,
	).Scan(&c.ID, &c.ServiceID, &c.Name, &c.Type, &c.Provider, &c.KeyValue, &c.SecretValue, &expiresAt, &lastRotatedAt, &c.WhereUsed, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.String
	}
	if lastRotatedAt.Valid {
		c.LastRotatedAt = &lastRotatedAt.String
	}
	return &c, nil
}

func (d *Database) CreateCredential(serviceID, name, credType, provider, keyValue, secretValue string, expiresAt *string, whereUsed, notes string) (*Credential, error) {
	now := nowUTC()
	c := &Credential{
		ID:        uuid.New().String(),
		ServiceID: serviceID,
		Name:      name,
		Type:      credType,
		Provider:  provider,
		KeyValue:  keyValue,
		ExpiresAt: expiresAt,
		WhereUsed: whereUsed,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := d.db.Exec(
		`INSERT INTO credentials (id, service_id, name, type, provider, key_value, secret_value, expires_at, where_used, notes, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ServiceID, c.Name, c.Type, c.Provider, keyValue, secretValue, expiresAt, c.WhereUsed, c.Notes, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}
	return c, nil
}

func (d *Database) UpdateCredential(id, name, credType, provider string, expiresAt *string, whereUsed, notes string) error {
	_, err := d.db.Exec(
		`UPDATE credentials SET name = ?, type = ?, provider = ?, expires_at = ?, where_used = ?, notes = ?, updated_at = ? WHERE id = ?`,
		name, credType, provider, expiresAt, whereUsed, notes, nowUTC(), id,
	)
	return err
}

func (d *Database) UpdateCredentialValue(id, keyValue, secretValue string) error {
	_, err := d.db.Exec(
		`UPDATE credentials SET key_value = ?, secret_value = ?, last_rotated_at = ?, updated_at = ? WHERE id = ?`,
		keyValue, secretValue, nowUTC(), nowUTC(), id,
	)
	return err
}

func (d *Database) DeleteCredential(id string) error {
	_, err := d.db.Exec(`DELETE FROM credentials WHERE id = ?`, id)
	return err
}

func (d *Database) GetExpiringCredentials(withinDays int) ([]Credential, error) {
	rows, err := d.db.Query(
		`SELECT c.id, c.service_id, c.name, c.type, c.provider, c.expires_at, c.last_rotated_at, c.where_used, c.notes, c.created_at, c.updated_at
		 FROM credentials c
		 WHERE c.expires_at IS NOT NULL AND c.expires_at != ''
		 AND c.expires_at <= datetime('now', '+' || ? || ' days')
		 ORDER BY c.expires_at ASC`, withinDays,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring credentials: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		var expiresAt, lastRotatedAt sql.NullString
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.Name, &c.Type, &c.Provider, &expiresAt, &lastRotatedAt, &c.WhereUsed, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.String
		}
		if lastRotatedAt.Valid {
			c.LastRotatedAt = &lastRotatedAt.String
		}
		creds = append(creds, c)
	}
	return creds, nil
}

func (d *Database) GetCredentialsByServiceOwner(ownerID string) ([]Credential, error) {
	rows, err := d.db.Query(
		`SELECT c.id, c.service_id, c.name, c.type, c.provider, c.key_value, c.secret_value, c.expires_at, c.last_rotated_at, c.where_used, c.notes, c.created_at, c.updated_at
		 FROM credentials c
		 JOIN services s ON c.service_id = s.id
		 WHERE s.owner_id = ?`, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials by owner: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		var expiresAt, lastRotatedAt sql.NullString
		if err := rows.Scan(&c.ID, &c.ServiceID, &c.Name, &c.Type, &c.Provider, &c.KeyValue, &c.SecretValue, &expiresAt, &lastRotatedAt, &c.WhereUsed, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.String
		}
		if lastRotatedAt.Valid {
			c.LastRotatedAt = &lastRotatedAt.String
		}
		creds = append(creds, c)
	}
	return creds, nil
}
