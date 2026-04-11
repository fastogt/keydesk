package database

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Assignment struct {
	ID            string  `json:"id"`
	PersonID      string  `json:"person_id"`
	PersonName    string  `json:"person_name,omitempty"`
	AccountID     string  `json:"account_id"`
	AccountName   string  `json:"account_name,omitempty"`
	AssignedBy    string  `json:"assigned_by"`
	AssignedAt    string  `json:"assigned_at"`
	RevokedAt     *string `json:"revoked_at"`
	RevokedReason *string `json:"revoked_reason"`
}

func (d *Database) GetActiveAssignmentsByPerson(personID string) ([]Assignment, error) {
	rows, err := d.db.Query(
		`SELECT a.id, a.person_id, a.account_id, acc.name as account_name, a.assigned_by, a.assigned_at
		 FROM assignments a
		 JOIN accounts acc ON a.account_id = acc.id
		 WHERE a.person_id = ? AND a.revoked_at IS NULL
		 ORDER BY a.assigned_at DESC`, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.PersonID, &a.AccountID, &a.AccountName, &a.AssignedBy, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func (d *Database) GetActiveAssignmentsByAccount(accountID string) ([]Assignment, error) {
	rows, err := d.db.Query(
		`SELECT a.id, a.person_id, p.name as person_name, a.account_id, a.assigned_by, a.assigned_at
		 FROM assignments a
		 JOIN people p ON a.person_id = p.id
		 WHERE a.account_id = ? AND a.revoked_at IS NULL
		 ORDER BY a.assigned_at DESC`, accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.PersonID, &a.PersonName, &a.AccountID, &a.AssignedBy, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func (d *Database) CreateAssignment(personID, accountID, assignedBy string) (*Assignment, error) {
	var existing int
	err := d.db.QueryRow(
		`SELECT COUNT(*) FROM assignments WHERE person_id = ? AND account_id = ? AND revoked_at IS NULL`,
		personID, accountID,
	).Scan(&existing)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing assignment: %w", err)
	}
	if existing > 0 {
		return nil, fmt.Errorf("person already has access to this account")
	}

	now := nowUTC()
	a := &Assignment{
		ID:         uuid.New().String(),
		PersonID:   personID,
		AccountID:  accountID,
		AssignedBy: assignedBy,
		AssignedAt: now,
	}

	_, err = d.db.Exec(
		`INSERT INTO assignments (id, person_id, account_id, assigned_by, assigned_at) VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.PersonID, a.AccountID, a.AssignedBy, a.AssignedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}
	return a, nil
}

func (d *Database) RevokeAssignment(id, reason string) error {
	now := nowUTC()
	_, err := d.db.Exec(
		`UPDATE assignments SET revoked_at = ?, revoked_reason = ? WHERE id = ? AND revoked_at IS NULL`,
		now, reason, id,
	)
	return err
}

func (d *Database) RevokeAllAssignmentsForPerson(personID, reason string) ([]string, error) {
	rows, err := d.db.Query(
		`SELECT account_id FROM assignments WHERE person_id = ? AND revoked_at IS NULL`, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active assignments: %w", err)
	}
	defer rows.Close()

	var accountIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan account id: %w", err)
		}
		accountIDs = append(accountIDs, id)
	}

	now := nowUTC()
	_, err = d.db.Exec(
		`UPDATE assignments SET revoked_at = ?, revoked_reason = ? WHERE person_id = ? AND revoked_at IS NULL`,
		now, reason, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke assignments: %w", err)
	}

	return accountIDs, nil
}

func (d *Database) GetUnassignedAccountsForPerson(personID string) ([]Account, error) {
	rows, err := d.db.Query(
		`SELECT a.id, a.name, a.type, a.login_url, a.login_email, a.notes, a.created_at, a.updated_at
		 FROM accounts a
		 WHERE a.id NOT IN (SELECT account_id FROM assignments WHERE person_id = ? AND revoked_at IS NULL)
		 ORDER BY a.name ASC`, personID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list unassigned accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.LoginURL, &a.LoginEmail, &a.Notes, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (d *Database) GetAssignmentHistory(entityType, entityID string) ([]Assignment, error) {
	var query string
	switch entityType {
	case "person":
		query = `SELECT a.id, a.person_id, '' as person_name, a.account_id, acc.name as account_name, a.assigned_by, a.assigned_at, a.revoked_at, a.revoked_reason
			FROM assignments a JOIN accounts acc ON a.account_id = acc.id WHERE a.person_id = ? ORDER BY a.assigned_at DESC`
	case "account":
		query = `SELECT a.id, a.person_id, p.name as person_name, a.account_id, '' as account_name, a.assigned_by, a.assigned_at, a.revoked_at, a.revoked_reason
			FROM assignments a JOIN people p ON a.person_id = p.id WHERE a.account_id = ? ORDER BY a.assigned_at DESC`
	default:
		return nil, fmt.Errorf("invalid entity type: %s", entityType)
	}

	rows, err := d.db.Query(query, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment history: %w", err)
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		var revokedAt, revokedReason sql.NullString
		if err := rows.Scan(&a.ID, &a.PersonID, &a.PersonName, &a.AccountID, &a.AccountName, &a.AssignedBy, &a.AssignedAt, &revokedAt, &revokedReason); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		if revokedAt.Valid {
			a.RevokedAt = &revokedAt.String
		}
		if revokedReason.Valid {
			a.RevokedReason = &revokedReason.String
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}
