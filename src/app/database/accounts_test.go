package database

import "testing"

func TestCreateAndGetAccount(t *testing.T) {
	d := OpenForTest(t)

	a, err := d.CreateAccount("GitHub", "dev", "https://github.com", "u@x.com", "ENC_PASS", "ENC_TOTP", "team")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if a.ID == "" {
		t.Errorf("expected ID")
	}

	got, err := d.GetAccount(a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.LoginPassword != "ENC_PASS" || got.TOTPSecret != "ENC_TOTP" {
		t.Errorf("encrypted fields not stored verbatim: %+v", got)
	}
}

func TestListAccountsFiltersByTypeAndSearch(t *testing.T) {
	d := OpenForTest(t)
	_, _ = d.CreateAccount("LinkedIn", "social", "", "", "", "", "")
	_, _ = d.CreateAccount("AWS", "cloud", "", "", "", "", "")
	_, _ = d.CreateAccount("GitHub", "dev", "", "gh@x.com", "", "", "")

	all, _ := d.ListAccounts("", "")
	if len(all) != 3 {
		t.Errorf("got %d, want 3", len(all))
	}

	cloud, _ := d.ListAccounts("", "cloud")
	if len(cloud) != 1 || cloud[0].Name != "AWS" {
		t.Errorf("filter by type: %+v", cloud)
	}

	search, _ := d.ListAccounts("github", "")
	if len(search) != 1 || search[0].Name != "GitHub" {
		t.Errorf("search by name: %+v", search)
	}

	emailSearch, _ := d.ListAccounts("gh@", "")
	if len(emailSearch) != 1 {
		t.Errorf("search by email: %+v", emailSearch)
	}
}

func TestListAccountsExcludesSecretFields(t *testing.T) {
	d := OpenForTest(t)
	_, _ = d.CreateAccount("X", "dev", "", "", "SECRET", "TOTP", "")
	list, _ := d.ListAccounts("", "")
	if len(list) != 1 {
		t.Fatalf("got %d", len(list))
	}
	if list[0].LoginPassword != "" || list[0].TOTPSecret != "" {
		t.Errorf("ListAccounts must not return secret fields, got %+v", list[0])
	}
}

func TestUpdateAccount(t *testing.T) {
	d := OpenForTest(t)
	a, _ := d.CreateAccount("X", "dev", "", "", "PW", "T", "")
	if err := d.UpdateAccount(a.ID, "X2", "social", "https://x", "u@x", "n"); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	got, _ := d.GetAccount(a.ID)
	if got.Name != "X2" || got.Type != "social" {
		t.Errorf("update not applied: %+v", got)
	}
	if got.LoginPassword != "PW" || got.TOTPSecret != "T" {
		t.Errorf("UpdateAccount must not touch secret fields, got %+v", got)
	}
}

func TestUpdateAccountPasswordAndTOTP(t *testing.T) {
	d := OpenForTest(t)
	a, _ := d.CreateAccount("X", "dev", "", "", "OLD", "OLD_TOTP", "")

	_ = d.UpdateAccountPassword(a.ID, "NEW")
	got, _ := d.GetAccount(a.ID)
	if got.LoginPassword != "NEW" {
		t.Errorf("password not updated: %q", got.LoginPassword)
	}

	_ = d.UpdateAccountTOTP(a.ID, "NEW_TOTP")
	got, _ = d.GetAccount(a.ID)
	if got.TOTPSecret != "NEW_TOTP" {
		t.Errorf("totp not updated: %q", got.TOTPSecret)
	}
}

func TestDeleteAccountCascadesAssignments(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "a@x.com", "", "")
	a, _ := d.CreateAccount("X", "dev", "", "", "", "", "")
	_, _ = d.CreateAssignment(p.ID, a.ID, "admin")

	if err := d.DeleteAccount(a.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	assigns, _ := d.GetActiveAssignmentsByPerson(p.ID)
	if len(assigns) != 0 {
		t.Errorf("expected cascade, got %d assignments", len(assigns))
	}
}

func TestGetAccountLoginURL(t *testing.T) {
	d := OpenForTest(t)
	a, _ := d.CreateAccount("X", "dev", "https://example.com", "", "", "", "")
	got, err := d.GetAccountLoginURL(a.ID)
	if err != nil {
		t.Fatalf("GetAccountLoginURL: %v", err)
	}
	if got != "https://example.com" {
		t.Errorf("got %q, want https://example.com", got)
	}
}

func TestAccountPeopleCount(t *testing.T) {
	d := OpenForTest(t)
	a, _ := d.CreateAccount("X", "dev", "", "", "", "", "")
	p1, _ := d.CreatePerson("A", "a@x", "", "")
	p2, _ := d.CreatePerson("B", "b@x", "", "")
	_, _ = d.CreateAssignment(p1.ID, a.ID, "admin")
	asn, _ := d.CreateAssignment(p2.ID, a.ID, "admin")

	got, _ := d.GetAccount(a.ID)
	if got.PeopleCount != 2 {
		t.Errorf("PeopleCount = %d, want 2", got.PeopleCount)
	}

	_ = d.RevokeAssignment(asn.ID, "left")
	got, _ = d.GetAccount(a.ID)
	if got.PeopleCount != 1 {
		t.Errorf("after revoke PeopleCount = %d, want 1", got.PeopleCount)
	}
}
