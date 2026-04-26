package database

import "testing"

func TestCreateAndGetPerson(t *testing.T) {
	d := OpenForTest(t)

	p, err := d.CreatePerson("Alice", "alice@x.com", "eng", "n/a")
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	if p.ID == "" {
		t.Errorf("expected ID assigned")
	}
	if p.Status != "active" {
		t.Errorf("default status = %q, want active", p.Status)
	}

	got, err := d.GetPerson(p.ID)
	if err != nil {
		t.Fatalf("GetPerson: %v", err)
	}
	if got.Name != "Alice" || got.Email != "alice@x.com" || got.Department != "eng" {
		t.Errorf("got %+v", got)
	}
}

func TestListPeopleFiltersAndSearches(t *testing.T) {
	d := OpenForTest(t)
	_, _ = d.CreatePerson("Alice", "alice@x.com", "eng", "")
	_, _ = d.CreatePerson("Bob", "bob@x.com", "ops", "")
	c, _ := d.CreatePerson("Carol", "carol@x.com", "eng", "")
	_ = d.OffboardPerson(c.ID)

	all, _ := d.ListPeople("", "")
	if len(all) != 3 {
		t.Errorf("list all: got %d, want 3", len(all))
	}

	active, _ := d.ListPeople("", "active")
	if len(active) != 2 {
		t.Errorf("active: got %d, want 2", len(active))
	}

	search, _ := d.ListPeople("alice", "")
	if len(search) != 1 || search[0].Name != "Alice" {
		t.Errorf("search 'alice': got %+v", search)
	}
}

func TestUpdatePerson(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "alice@x.com", "eng", "")

	if err := d.UpdatePerson(p.ID, "Alice B", "alice.b@x.com", "platform", "promo"); err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}
	got, _ := d.GetPerson(p.ID)
	if got.Name != "Alice B" || got.Department != "platform" || got.Notes != "promo" {
		t.Errorf("update not applied: %+v", got)
	}
}

func TestOffboardPerson(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "a@x.com", "", "")

	if err := d.OffboardPerson(p.ID); err != nil {
		t.Fatalf("OffboardPerson: %v", err)
	}
	got, _ := d.GetPerson(p.ID)
	if got.Status != "offboarded" {
		t.Errorf("status = %q, want offboarded", got.Status)
	}
}

func TestDeletePersonCascadesAssignments(t *testing.T) {
	d := OpenForTest(t)
	p, _ := d.CreatePerson("Alice", "a@x.com", "", "")
	a, _ := d.CreateAccount("LinkedIn", "social", "https://x", "u@x", "pw", "", "")
	_, _ = d.CreateAssignment(p.ID, a.ID, "admin")

	if err := d.DeletePerson(p.ID); err != nil {
		t.Fatalf("DeletePerson: %v", err)
	}
	assigns, _ := d.GetActiveAssignmentsByAccount(a.ID)
	if len(assigns) != 0 {
		t.Errorf("expected ON DELETE CASCADE to remove assignments, got %d", len(assigns))
	}
}

func TestGetPersonNotFound(t *testing.T) {
	d := OpenForTest(t)
	if _, err := d.GetPerson("nope"); err == nil {
		t.Errorf("expected error for missing person")
	}
}
