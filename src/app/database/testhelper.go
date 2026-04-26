package database

import (
	"path/filepath"
	"testing"
)

func OpenForTest(t *testing.T) *Database {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenForTest: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
