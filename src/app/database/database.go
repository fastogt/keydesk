package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func Open(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (d *Database) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS admins (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS people (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL DEFAULT '',
		department TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_people_status ON people(status);

	CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'other',
		login_url TEXT NOT NULL DEFAULT '',
		login_email TEXT NOT NULL DEFAULT '',
		login_password TEXT NOT NULL DEFAULT '',
		totp_secret TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS services (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		environment TEXT NOT NULL DEFAULT 'production',
		owner_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (owner_id) REFERENCES people(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		service_id TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'api_key',
		provider TEXT NOT NULL DEFAULT 'custom',
		key_value TEXT NOT NULL DEFAULT '',
		secret_value TEXT NOT NULL DEFAULT '',
		expires_at TEXT,
		last_rotated_at TEXT,
		where_used TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_credentials_service ON credentials(service_id);
	CREATE INDEX IF NOT EXISTS idx_credentials_expires ON credentials(expires_at);

	CREATE TABLE IF NOT EXISTS assignments (
		id TEXT PRIMARY KEY,
		person_id TEXT NOT NULL,
		account_id TEXT NOT NULL,
		assigned_by TEXT NOT NULL DEFAULT '',
		assigned_at TEXT NOT NULL,
		revoked_at TEXT,
		revoked_reason TEXT,
		FOREIGN KEY (person_id) REFERENCES people(id) ON DELETE CASCADE,
		FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_assignments_person ON assignments(person_id);
	CREATE INDEX IF NOT EXISTS idx_assignments_account ON assignments(account_id);
	CREATE INDEX IF NOT EXISTS idx_assignments_active ON assignments(revoked_at);

	CREATE TABLE IF NOT EXISTS audit_log (
		id TEXT PRIMARY KEY,
		action TEXT NOT NULL,
		entity_type TEXT NOT NULL DEFAULT '',
		entity_id TEXT NOT NULL DEFAULT '',
		person_id TEXT,
		performed_by TEXT NOT NULL DEFAULT '',
		details TEXT NOT NULL DEFAULT '',
		timestamp TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);
	`

	_, err := d.db.Exec(schema)
	return err
}
