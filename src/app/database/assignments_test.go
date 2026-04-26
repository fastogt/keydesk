package database

import "testing"

func setupPersonAndAccount(t *testing.T, d *Database) (*Person, *Account) {
	t.Helper()
	p, err := d.CreatePerson("Alice", "a@x.com", "", "")
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	a, err := d.CreateAccount("X", "dev", "", "", "", "", "")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	return p, a
}

func TestCreateAssignmentAndQuery(t *testing.T) {
	d := OpenForTest(t)
	p, a := setupPersonAndAccount(t, d)

	asn, err := d.CreateAssignment(p.ID, a.ID, "admin@x")
	if err != nil {
		t.Fatalf("CreateAssignment: %v", err)
	}
	if asn.ID == "" {
		t.Errorf("expected ID")
	}

	byPerson, _ := d.GetActiveAssignmentsByPerson(p.ID)
	if len(byPerson) != 1 || byPerson[0].AccountID != a.ID || byPerson[0].AccountName != "X" {
		t.Errorf("byPerson = %+v", byPerson)
	}

	byAccount, _ := d.GetActiveAssignmentsByAccount(a.ID)
	if len(byAccount) != 1 || byAccount[0].PersonID != p.ID || byAccount[0].PersonName != "Alice" {
		t.Errorf("byAccount = %+v", byAccount)
	}
}

func TestCreateAssignmentRejectsDuplicate(t *testing.T) {
	d := OpenForTest(t)
	p, a := setupPersonAndAccount(t, d)

	if _, err := d.CreateAssignment(p.ID, a.ID, "admin"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := d.CreateAssignment(p.ID, a.ID, "admin"); err == nil {
		t.Errorf("duplicate active assignment should be rejected")
	}
}

func TestCreateAssignmentAfterRevokeAllowed(t *testing.T) {
	d := OpenForTest(t)
	p, a := setupPersonAndAccount(t, d)

	asn, _ := d.CreateAssignment(p.ID, a.ID, "admin")
	_ = d.RevokeAssignment(asn.ID, "left")

	if _, err := d.CreateAssignment(p.ID, a.ID, "admin"); err != nil {
		t.Errorf("re-assigning after revoke should be allowed, got %v", err)
	}
}

func TestRevokeAssignmentSetsRevokedAt(t *testing.T) {
	d := OpenForTest(t)
	p, a := setupPersonAndAccount(t, d)
	asn, _ := d.CreateAssignment(p.ID, a.ID, "admin")

	if err := d.RevokeAssignment(asn.ID, "offboard"); err != nil {
		t.Fatalf("RevokeAssignment: %v", err)
	}

	active, _ := d.GetActiveAssignmentsByPerson(p.ID)
	if len(active) != 0 {
		t.Errorf("expected no active assignments after revoke, got %d", len(active))
	}

	hist, _ := d.GetAssignmentHistory("person", p.ID)
	if len(hist) != 1 || hist[0].RevokedAt == nil || hist[0].RevokedReason == nil {
		t.Errorf("history should retain revoke metadata: %+v", hist)
	}
	if *hist[0].RevokedReason != "offboard" {
		t.Errorf("reason = %q, want offboard", *hist[0].RevokedReason)
	}
}

func TestRevokeAllAssignmentsForPerson(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "a@x.com", "", "")
	a1, _ := d.CreateAccount("A1", "dev", "", "", "", "", "")
	a2, _ := d.CreateAccount("A2", "dev", "", "", "", "", "")
	_, _ = d.CreateAssignment(p.ID, a1.ID, "admin")
	_, _ = d.CreateAssignment(p.ID, a2.ID, "admin")

	ids, err := d.RevokeAllAssignmentsForPerson(p.ID, "offboard")
	if err != nil {
		t.Fatalf("RevokeAllAssignmentsForPerson: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("returned %d account ids, want 2", len(ids))
	}

	active, _ := d.GetActiveAssignmentsByPerson(p.ID)
	if len(active) != 0 {
		t.Errorf("expected zero active after revoke-all, got %d", len(active))
	}
}

func TestGetUnassignedAccountsForPerson(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "a@x.com", "", "")
	a1, _ := d.CreateAccount("Assigned", "dev", "", "", "", "", "")
	_, _ = d.CreateAccount("Free1", "dev", "", "", "", "", "")
	_, _ = d.CreateAccount("Free2", "dev", "", "", "", "", "")
	_, _ = d.CreateAssignment(p.ID, a1.ID, "admin")

	un, err := d.GetUnassignedAccountsForPerson(p.ID)
	if err != nil {
		t.Fatalf("GetUnassignedAccountsForPerson: %v", err)
	}
	if len(un) != 2 {
		t.Errorf("got %d unassigned, want 2", len(un))
	}
	for _, acc := range un {
		if acc.ID == a1.ID {
			t.Errorf("assigned account should not be in unassigned list")
		}
	}
}

func TestGetAssignmentHistoryInvalidEntityType(t *testing.T) {
	d := OpenForTest(t)
	if _, err := d.GetAssignmentHistory("bogus", "x"); err == nil {
		t.Errorf("expected error for invalid entity type")
	}
}
