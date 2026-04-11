package database

import (
	"fmt"

	"github.com/google/uuid"
)

type Account struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	LoginURL      string `json:"login_url"`
	LoginEmail    string `json:"login_email"`
	LoginPassword string `json:"login_password,omitempty"`
	TOTPSecret    string `json:"totp_secret,omitempty"`
	Notes         string `json:"notes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	PeopleCount   int    `json:"people_count,omitempty"`
}

func (d *Database) ListAccounts(search, accountType string) ([]Account, error) {
	query := `SELECT a.id, a.name, a.type, a.login_url, a.login_email, a.notes, a.created_at, a.updated_at,
		(SELECT COUNT(*) FROM assignments WHERE account_id = a.id AND revoked_at IS NULL) as people_count
		FROM accounts a WHERE 1=1`
	args := []any{}

	if accountType != "" {
		query += " AND a.type = ?"
		args = append(args, accountType)
	}
	if search != "" {
		query += " AND (a.name LIKE ? OR a.login_email LIKE ? OR a.notes LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s)
	}
	query += " ORDER BY a.name ASC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.LoginURL, &a.LoginEmail, &a.Notes, &a.CreatedAt, &a.UpdatedAt, &a.PeopleCount); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (d *Database) GetAccount(id string) (*Account, error) {
	var a Account
	err := d.db.QueryRow(
		`SELECT a.id, a.name, a.type, a.login_url, a.login_email, a.login_password, a.totp_secret, a.notes, a.created_at, a.updated_at,
		(SELECT COUNT(*) FROM assignments WHERE account_id = a.id AND revoked_at IS NULL) as people_count
		FROM accounts a WHERE a.id = ?`, id,
	).Scan(&a.ID, &a.Name, &a.Type, &a.LoginURL, &a.LoginEmail, &a.LoginPassword, &a.TOTPSecret, &a.Notes, &a.CreatedAt, &a.UpdatedAt, &a.PeopleCount)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}
	return &a, nil
}

func (d *Database) CreateAccount(name, accountType, loginURL, loginEmail, loginPassword, totpSecret, notes string) (*Account, error) {
	now := nowUTC()
	a := &Account{
		ID:            uuid.New().String(),
		Name:          name,
		Type:          accountType,
		LoginURL:      loginURL,
		LoginEmail:    loginEmail,
		LoginPassword: loginPassword,
		TOTPSecret:    totpSecret,
		Notes:         notes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := d.db.Exec(
		`INSERT INTO accounts (id, name, type, login_url, login_email, login_password, totp_secret, notes, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.Type, a.LoginURL, a.LoginEmail, a.LoginPassword, a.TOTPSecret, a.Notes, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}
	return a, nil
}

func (d *Database) UpdateAccount(id, name, accountType, loginURL, loginEmail, notes string) error {
	_, err := d.db.Exec(
		`UPDATE accounts SET name = ?, type = ?, login_url = ?, login_email = ?, notes = ?, updated_at = ? WHERE id = ?`,
		name, accountType, loginURL, loginEmail, notes, nowUTC(), id,
	)
	return err
}

func (d *Database) UpdateAccountPassword(id, password string) error {
	_, err := d.db.Exec(
		`UPDATE accounts SET login_password = ?, updated_at = ? WHERE id = ?`,
		password, nowUTC(), id,
	)
	return err
}

func (d *Database) UpdateAccountTOTP(id, totpSecret string) error {
	_, err := d.db.Exec(
		`UPDATE accounts SET totp_secret = ?, updated_at = ? WHERE id = ?`,
		totpSecret, nowUTC(), id,
	)
	return err
}

func (d *Database) DeleteAccount(id string) error {
	_, err := d.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func (d *Database) GetAccountLoginURL(id string) (string, error) {
	var url string
	err := d.db.QueryRow(`SELECT login_url FROM accounts WHERE id = ?`, id).Scan(&url)
	return url, err
}
