package database

import (
	"fmt"

	"github.com/google/uuid"
)

type Person struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Department   string `json:"department"`
	Notes        string `json:"notes"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	AccountCount int    `json:"account_count,omitempty"`
	ServiceCount int    `json:"service_count,omitempty"`
}

func (d *Database) ListPeople(search, status string) ([]Person, error) {
	query := `SELECT p.id, p.name, p.email, p.department, p.notes, p.status, p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM assignments WHERE person_id = p.id AND revoked_at IS NULL) as account_count,
		(SELECT COUNT(*) FROM services WHERE owner_id = p.id) as service_count
		FROM people p WHERE 1=1`
	args := []any{}

	if status != "" {
		query += " AND p.status = ?"
		args = append(args, status)
	}
	if search != "" {
		query += " AND (p.name LIKE ? OR p.email LIKE ? OR p.department LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s)
	}
	query += " ORDER BY p.status ASC, p.name ASC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list people: %w", err)
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Email, &p.Department, &p.Notes, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.AccountCount, &p.ServiceCount); err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}
		people = append(people, p)
	}
	return people, nil
}

func (d *Database) GetPerson(id string) (*Person, error) {
	var p Person
	err := d.db.QueryRow(
		`SELECT p.id, p.name, p.email, p.department, p.notes, p.status, p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM assignments WHERE person_id = p.id AND revoked_at IS NULL) as account_count,
		(SELECT COUNT(*) FROM services WHERE owner_id = p.id) as service_count
		FROM people p WHERE p.id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Email, &p.Department, &p.Notes, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.AccountCount, &p.ServiceCount)
	if err != nil {
		return nil, fmt.Errorf("person not found: %w", err)
	}
	return &p, nil
}

func (d *Database) CreatePerson(name, email, department, notes string) (*Person, error) {
	now := nowUTC()
	p := &Person{
		ID:         uuid.New().String(),
		Name:       name,
		Email:      email,
		Department: department,
		Notes:      notes,
		Status:     "active",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err := d.db.Exec(
		`INSERT INTO people (id, name, email, department, notes, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Email, p.Department, p.Notes, p.Status, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create person: %w", err)
	}
	return p, nil
}

func (d *Database) UpdatePerson(id, name, email, department, notes string) error {
	_, err := d.db.Exec(
		`UPDATE people SET name = ?, email = ?, department = ?, notes = ?, updated_at = ? WHERE id = ?`,
		name, email, department, notes, nowUTC(), id,
	)
	return err
}

func (d *Database) DeletePerson(id string) error {
	_, err := d.db.Exec(`DELETE FROM people WHERE id = ?`, id)
	return err
}

func (d *Database) OffboardPerson(id string) error {
	_, err := d.db.Exec(
		`UPDATE people SET status = 'offboarded', updated_at = ? WHERE id = ?`,
		nowUTC(), id,
	)
	return err
}
