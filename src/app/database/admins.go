package database

import (
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (d *Database) CreateAdminUser(email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := nowUTC()
	_, err = d.db.Exec(
		`INSERT INTO admins (id, email, name, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(email) DO UPDATE SET password_hash = excluded.password_hash, updated_at = excluded.updated_at`,
		uuid.New().String(), email, "Admin", string(hash), now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	return nil
}

func (d *Database) GetAdminByEmail(email string) (*Admin, error) {
	var a Admin
	err := d.db.QueryRow(
		`SELECT id, email, name, password_hash, created_at, updated_at FROM admins WHERE email = ?`,
		email,
	).Scan(&a.ID, &a.Email, &a.Name, &a.PasswordHash, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("admin not found: %w", err)
	}
	return &a, nil
}

func (d *Database) GetAdminByID(id string) (*Admin, error) {
	var a Admin
	err := d.db.QueryRow(
		`SELECT id, email, name, password_hash, created_at, updated_at FROM admins WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Email, &a.Name, &a.PasswordHash, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("admin not found: %w", err)
	}
	return &a, nil
}

func (d *Database) UpdateAdminProfile(id, name, email string) error {
	_, err := d.db.Exec(
		`UPDATE admins SET name = ?, email = ?, updated_at = ? WHERE id = ?`,
		name, email, nowUTC(), id,
	)
	return err
}

func (d *Database) UpdateAdminPassword(id, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	_, err = d.db.Exec(
		`UPDATE admins SET password_hash = ?, updated_at = ? WHERE id = ?`,
		string(hash), nowUTC(), id,
	)
	return err
}

func (d *Database) VerifyAdminPassword(admin *Admin, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) == nil
}
