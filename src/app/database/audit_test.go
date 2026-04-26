package database

import "testing"

func TestLogAuditAndGetRecent(t *testing.T) {
	d := OpenForTest(t)

	if err := d.LogAudit("create", "account", "acc-1", "person-1", "admin@x", "Created"); err != nil {
		t.Fatalf("LogAudit: %v", err)
	}
	if err := d.LogAudit("update", "account", "acc-1", "", "admin@x", "Edited"); err != nil {
		t.Fatalf("LogAudit: %v", err)
	}

	got, err := d.GetRecentAuditLog(10)
	if err != nil {
		t.Fatalf("GetRecentAuditLog: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d entries, want 2", len(got))
	}
}

func TestGetAuditLogForEntityIsolatesByEntity(t *testing.T) {
	d := OpenForTest(t)
	_ = d.LogAudit("create", "account", "acc-1", "", "admin", "")
	_ = d.LogAudit("create", "account", "acc-2", "", "admin", "")
	_ = d.LogAudit("update", "person", "p-1", "p-1", "admin", "")

	got, _ := d.GetAuditLogForEntity("account", "acc-1", 10)
	if len(got) != 1 || got[0].EntityID != "acc-1" {
		t.Errorf("isolation failed: %+v", got)
	}
}

func TestDashboardStats(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "", "", "")
	a, _ := d.CreateAccount("X", "dev", "", "", "", "", "")
	_, _ = d.CreateAssignment(p.ID, a.ID, "admin")

	stats, err := d.GetDashboardStats()
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	if stats["people"] != 1 {
		t.Errorf("people = %d, want 1", stats["people"])
	}
	if stats["accounts"] != 1 {
		t.Errorf("accounts = %d, want 1", stats["accounts"])
	}
	if stats["unassigned_accounts"] != 0 {
		t.Errorf("unassigned_accounts = %d, want 0", stats["unassigned_accounts"])
	}

	_, _ = d.CreateAccount("Lonely", "dev", "", "", "", "", "")
	stats, _ = d.GetDashboardStats()
	if stats["unassigned_accounts"] != 1 {
		t.Errorf("after adding lonely account: unassigned = %d, want 1", stats["unassigned_accounts"])
	}
}
