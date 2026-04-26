package database

import "testing"

func TestCreateAndGetAdmin(t *testing.T) {
	d := OpenForTest(t)

	if err := d.CreateAdminUser("admin@example.com", "hunter2"); err != nil {
		t.Fatalf("CreateAdminUser: %v", err)
	}

	a, err := d.GetAdminByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("GetAdminByEmail: %v", err)
	}
	if a.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", a.Email, "admin@example.com")
	}
	if a.PasswordHash == "" || a.PasswordHash == "hunter2" {
		t.Errorf("password must be hashed, got %q", a.PasswordHash)
	}

	got, err := d.GetAdminByID(a.ID)
	if err != nil || got.ID != a.ID {
		t.Errorf("GetAdminByID returned %v, %v", got, err)
	}
}

func TestVerifyAdminPassword(t *testing.T) {
	d := OpenForTest(t)
	_ = d.CreateAdminUser("a@b.c", "correct-password")
	a, _ := d.GetAdminByEmail("a@b.c")

	if !d.VerifyAdminPassword(a, "correct-password") {
		t.Errorf("VerifyAdminPassword should accept correct password")
	}
	if d.VerifyAdminPassword(a, "wrong") {
		t.Errorf("VerifyAdminPassword should reject wrong password")
	}
}

func TestCreateAdminUpsertsOnConflict(t *testing.T) {
	d := OpenForTest(t)
	_ = d.CreateAdminUser("a@b.c", "first")
	first, _ := d.GetAdminByEmail("a@b.c")

	if err := d.CreateAdminUser("a@b.c", "second"); err != nil {
		t.Fatalf("second CreateAdminUser: %v", err)
	}
	second, _ := d.GetAdminByEmail("a@b.c")

	if second.PasswordHash == first.PasswordHash {
		t.Errorf("password hash should change after re-create")
	}
	if !d.VerifyAdminPassword(second, "second") {
		t.Errorf("new password should verify")
	}
}

func TestUpdateAdminPassword(t *testing.T) {
	d := OpenForTest(t)
	_ = d.CreateAdminUser("a@b.c", "old")
	a, _ := d.GetAdminByEmail("a@b.c")

	if err := d.UpdateAdminPassword(a.ID, "new"); err != nil {
		t.Fatalf("UpdateAdminPassword: %v", err)
	}
	updated, _ := d.GetAdminByID(a.ID)
	if !d.VerifyAdminPassword(updated, "new") {
		t.Errorf("new password should verify")
	}
	if d.VerifyAdminPassword(updated, "old") {
		t.Errorf("old password should no longer verify")
	}
}

func TestUpdateAdminProfile(t *testing.T) {
	d := OpenForTest(t)
	_ = d.CreateAdminUser("old@x.com", "pw")
	a, _ := d.GetAdminByEmail("old@x.com")

	if err := d.UpdateAdminProfile(a.ID, "Alice", "alice@x.com"); err != nil {
		t.Fatalf("UpdateAdminProfile: %v", err)
	}
	got, _ := d.GetAdminByID(a.ID)
	if got.Name != "Alice" || got.Email != "alice@x.com" {
		t.Errorf("profile not updated: %+v", got)
	}
}
