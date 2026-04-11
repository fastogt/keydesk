package database

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Environment     string `json:"environment"`
	OwnerID         string `json:"owner_id"`
	OwnerName       string `json:"owner_name,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	CredentialCount int    `json:"credential_count,omitempty"`
	ExpiringCount   int    `json:"expiring_count,omitempty"`
	ExpiredCount    int    `json:"expired_count,omitempty"`
}

func (d *Database) ListServices(search string) ([]Service, error) {
	query := `SELECT s.id, s.name, s.description, s.environment, COALESCE(s.owner_id, ''), COALESCE(p.name, '') as owner_name,
		s.created_at, s.updated_at,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id) as credential_count,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id AND expires_at IS NOT NULL AND expires_at != '' AND expires_at > datetime('now') AND expires_at <= datetime('now', '+30 days')) as expiring_count,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id AND expires_at IS NOT NULL AND expires_at != '' AND expires_at <= datetime('now')) as expired_count
		FROM services s
		LEFT JOIN people p ON s.owner_id = p.id
		WHERE 1=1`
	args := []any{}

	if search != "" {
		query += " AND (s.name LIKE ? OR s.description LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s)
	}
	query += " ORDER BY s.name ASC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Environment, &s.OwnerID, &s.OwnerName,
			&s.CreatedAt, &s.UpdatedAt, &s.CredentialCount, &s.ExpiringCount, &s.ExpiredCount); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, s)
	}
	return services, nil
}

func (d *Database) GetService(id string) (*Service, error) {
	var s Service
	var ownerID sql.NullString
	err := d.db.QueryRow(
		`SELECT s.id, s.name, s.description, s.environment, s.owner_id, COALESCE(p.name, '') as owner_name,
		s.created_at, s.updated_at,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id) as credential_count,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id AND expires_at IS NOT NULL AND expires_at != '' AND expires_at > datetime('now') AND expires_at <= datetime('now', '+30 days')) as expiring_count,
		(SELECT COUNT(*) FROM credentials WHERE service_id = s.id AND expires_at IS NOT NULL AND expires_at != '' AND expires_at <= datetime('now')) as expired_count
		FROM services s
		LEFT JOIN people p ON s.owner_id = p.id
		WHERE s.id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.Description, &s.Environment, &ownerID, &s.OwnerName,
		&s.CreatedAt, &s.UpdatedAt, &s.CredentialCount, &s.ExpiringCount, &s.ExpiredCount)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}
	if ownerID.Valid {
		s.OwnerID = ownerID.String
	}
	return &s, nil
}

func (d *Database) CreateService(name, description, environment, ownerID string) (*Service, error) {
	now := nowUTC()
	s := &Service{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Environment: environment,
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var ownerVal any
	if ownerID != "" {
		ownerVal = ownerID
	}

	_, err := d.db.Exec(
		`INSERT INTO services (id, name, description, environment, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Description, s.Environment, ownerVal, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	return s, nil
}

func (d *Database) UpdateService(id, name, description, environment, ownerID string) error {
	var ownerVal any
	if ownerID != "" {
		ownerVal = ownerID
	}
	_, err := d.db.Exec(
		`UPDATE services SET name = ?, description = ?, environment = ?, owner_id = ?, updated_at = ? WHERE id = ?`,
		name, description, environment, ownerVal, nowUTC(), id,
	)
	return err
}

func (d *Database) UpdateServiceOwner(id, ownerID string) error {
	var ownerVal any
	if ownerID != "" {
		ownerVal = ownerID
	}
	_, err := d.db.Exec(
		`UPDATE services SET owner_id = ?, updated_at = ? WHERE id = ?`,
		ownerVal, nowUTC(), id,
	)
	return err
}

func (d *Database) DeleteService(id string) error {
	_, err := d.db.Exec(`DELETE FROM services WHERE id = ?`, id)
	return err
}

func (d *Database) GetServicesByOwner(ownerID string) ([]Service, error) {
	rows, err := d.db.Query(
		`SELECT id, name, description, environment, COALESCE(owner_id, ''), created_at, updated_at FROM services WHERE owner_id = ?`,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list services by owner: %w", err)
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Environment, &s.OwnerID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, s)
	}
	return services, nil
}
