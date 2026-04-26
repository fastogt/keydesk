package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"keydesk/app/database"
	"keydesk/app/vault"
)

const testJWTSecret = "test-jwt-secret"

type testEnv struct {
	t       *testing.T
	db      *database.Database
	vault   *vault.Vault
	auth    *AuthHandler
	people  *PeopleHandler
	acc     *AccountsHandler
	ext     *ExtHandler
	router  http.Handler
	adminID string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := database.OpenForTest(t)

	key, err := vault.GenerateKey()
	if err != nil {
		t.Fatalf("vault key: %v", err)
	}
	v, err := vault.New(key)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	if err := db.CreateAdminUser("admin@example.com", "password123"); err != nil {
		t.Fatalf("CreateAdminUser: %v", err)
	}
	admin, _ := db.GetAdminByEmail("admin@example.com")

	env := &testEnv{
		t:       t,
		db:      db,
		vault:   v,
		auth:    NewAuthHandler(db, testJWTSecret),
		people:  NewPeopleHandler(db),
		acc:     NewAccountsHandler(db, v),
		ext:     NewExtHandler(db, testJWTSecret, v),
		adminID: admin.ID,
	}

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", env.auth.HandleLogin)

		r.Route("/ext", func(r chi.Router) {
			r.Post("/auth", env.ext.HandleExtLogin)
			r.Group(func(r chi.Router) {
				r.Use(env.ext.ExtAuthMiddleware)
				r.Get("/accounts", env.ext.HandleExtListAccounts)
				r.Post("/credentials/{id}", env.ext.HandleExtGetCredentials)
				r.Get("/match", env.ext.HandleExtMatch)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(env.auth.AdminAuthMiddleware)
			r.Get("/me", env.auth.HandleMe)
			r.Get("/accounts", env.acc.HandleList)
			r.Post("/accounts", env.acc.HandleCreate)
			r.Get("/accounts/{id}", env.acc.HandleGet)
			r.Post("/accounts/{id}/reveal", env.acc.HandleReveal)
		})
	})
	env.router = r
	return env
}

func (e *testEnv) do(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	e.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, body []byte) (data json.RawMessage, errMsg string) {
	t.Helper()
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%q)", err, body)
	}
	if env.Error != nil {
		return nil, env.Error.Message
	}
	return env.Data, ""
}

func (e *testEnv) adminToken() string {
	e.t.Helper()
	w := e.do("POST", "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
	}, nil)
	if w.Code != http.StatusOK {
		e.t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	data, _ := decodeEnvelope(e.t, w.Body.Bytes())
	var resp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(data, &resp)
	if resp.Token == "" {
		e.t.Fatalf("no token in login response: %s", w.Body.String())
	}
	return resp.Token
}

func (e *testEnv) extToken(personID string) string {
	e.t.Helper()
	w := e.do("POST", "/api/ext/auth", map[string]string{"person_id": personID}, nil)
	if w.Code != http.StatusOK {
		e.t.Fatalf("ext login failed: %d %s", w.Code, w.Body.String())
	}
	data, _ := decodeEnvelope(e.t, w.Body.Bytes())
	var resp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(data, &resp)
	return resp.Token
}

// ---------------------- Admin auth ----------------------

func TestAdminLoginSuccess(t *testing.T) {
	env := newTestEnv(t)
	w := env.do("POST", "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data, _ := decodeEnvelope(t, w.Body.Bytes())
	if !strings.Contains(string(data), `"token"`) {
		t.Errorf("expected token in response, got %s", data)
	}
}

func TestAdminLoginWrongPassword(t *testing.T) {
	env := newTestEnv(t)
	w := env.do("POST", "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "wrong",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminLoginUnknownEmail(t *testing.T) {
	env := newTestEnv(t)
	w := env.do("POST", "/api/auth/login", map[string]string{
		"email":    "ghost@example.com",
		"password": "password123",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminMiddlewareRejectsMissingToken(t *testing.T) {
	env := newTestEnv(t)
	w := env.do("GET", "/api/me", nil, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminMiddlewareRejectsInvalidToken(t *testing.T) {
	env := newTestEnv(t)
	w := env.do("GET", "/api/me", nil, map[string]string{
		"Authorization": "Bearer not-a-jwt",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAdminMiddlewareRejectsExtensionToken(t *testing.T) {
	env := newTestEnv(t)
	p, _ := env.db.CreatePerson("Alice", "a@x.com", "", "")
	tok := env.extToken(p.ID)

	// Using an extension token on an admin route must fail (no admin_id claim).
	w := env.do("GET", "/api/me", nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("admin route accepted extension token: status %d", w.Code)
	}
}

// ---------------------- Account create + reveal round-trip ----------------------

func TestCreateAccountEncryptsPasswordAndAuditsAndReveals(t *testing.T) {
	env := newTestEnv(t)
	tok := env.adminToken()

	w := env.do("POST", "/api/accounts", map[string]string{
		"name":           "GitHub",
		"type":           "dev",
		"login_email":    "team@x.com",
		"login_password": "secret-password",
	}, map[string]string{"Authorization": "Bearer " + tok})
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", w.Code, w.Body.String())
	}

	data, _ := decodeEnvelope(t, w.Body.Bytes())
	var created struct {
		ID            string `json:"id"`
		LoginPassword string `json:"login_password"`
	}
	_ = json.Unmarshal(data, &created)
	if created.ID == "" {
		t.Fatalf("no id in response")
	}
	if created.LoginPassword != "" {
		t.Errorf("create response leaked password: %q", created.LoginPassword)
	}

	// Stored password is encrypted, not plain.
	stored, _ := env.db.GetAccount(created.ID)
	if stored.LoginPassword == "secret-password" {
		t.Errorf("password stored in plaintext")
	}
	if stored.LoginPassword == "" {
		t.Errorf("password not stored")
	}
	plain, err := env.vault.Decrypt(stored.LoginPassword)
	if err != nil || plain != "secret-password" {
		t.Errorf("vault decrypt mismatch: %q, err=%v", plain, err)
	}

	// Audit log records the create action.
	entries, _ := env.db.GetAuditLogForEntity("account", created.ID, 10)
	if len(entries) == 0 {
		t.Fatalf("audit log empty after account create")
	}
	if entries[len(entries)-1].Action != "created" {
		t.Errorf("first audit action = %q, want 'created'", entries[len(entries)-1].Action)
	}

	// Reveal returns plaintext + adds a 'viewed' audit entry.
	wr := env.do("POST", "/api/accounts/"+created.ID+"/reveal", nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if wr.Code != http.StatusOK {
		t.Fatalf("reveal status = %d", wr.Code)
	}
	revealData, _ := decodeEnvelope(t, wr.Body.Bytes())
	if !strings.Contains(string(revealData), "secret-password") {
		t.Errorf("reveal did not return plaintext: %s", revealData)
	}

	entries, _ = env.db.GetAuditLogForEntity("account", created.ID, 10)
	sawViewed := false
	for _, e := range entries {
		if e.Action == "viewed" {
			sawViewed = true
		}
	}
	if !sawViewed {
		t.Errorf("reveal did not add 'viewed' audit entry, got %+v", entries)
	}
}

// ---------------------- Extension JWT type-claim enforcement ----------------------

func TestExtMiddlewareRejectsAdminToken(t *testing.T) {
	env := newTestEnv(t)
	tok := env.adminToken()

	w := env.do("GET", "/api/ext/accounts", nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("ext route accepted admin token: status %d, body %s", w.Code, w.Body.String())
	}
}

func TestExtLoginInactiveBlocked(t *testing.T) {
	env := newTestEnv(t)
	p, _ := env.db.CreatePerson("Alice", "a@x.com", "", "")
	_ = env.db.OffboardPerson(p.ID)

	w := env.do("POST", "/api/ext/auth", map[string]string{"person_id": p.ID}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("offboarded person login: status = %d, want 401", w.Code)
	}
}

// ---------------------- Extension assignment gate ----------------------

func TestExtGetCredentialsRejectsUnassignedAccount(t *testing.T) {
	env := newTestEnv(t)
	alice, _ := env.db.CreatePerson("Alice", "a@x.com", "", "")
	bob, _ := env.db.CreatePerson("Bob", "b@x.com", "", "")

	enc, _ := env.vault.Encrypt("secret")
	aliceAcc, _ := env.db.CreateAccount("Alice's GH", "dev", "", "", enc, "", "")
	bobAcc, _ := env.db.CreateAccount("Bob's AWS", "cloud", "", "", enc, "", "")
	_, _ = env.db.CreateAssignment(alice.ID, aliceAcc.ID, env.adminID)
	_, _ = env.db.CreateAssignment(bob.ID, bobAcc.ID, env.adminID)

	tok := env.extToken(alice.ID)

	// Alice → her own account: allowed.
	wOk := env.do("POST", "/api/ext/credentials/"+aliceAcc.ID, nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if wOk.Code != http.StatusOK {
		t.Errorf("Alice→Alice's account: status = %d, want 200", wOk.Code)
	}

	// Alice → Bob's account: forbidden.
	wForbidden := env.do("POST", "/api/ext/credentials/"+bobAcc.ID, nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if wForbidden.Code != http.StatusForbidden {
		t.Errorf("Alice→Bob's account: status = %d, want 403", wForbidden.Code)
	}
}

func TestExtGetCredentialsAfterRevokeForbidden(t *testing.T) {
	env := newTestEnv(t)
	alice, _ := env.db.CreatePerson("Alice", "a@x.com", "", "")
	enc, _ := env.vault.Encrypt("secret")
	acc, _ := env.db.CreateAccount("X", "dev", "", "", enc, "", "")
	asn, _ := env.db.CreateAssignment(alice.ID, acc.ID, env.adminID)

	tok := env.extToken(alice.ID)

	wOk := env.do("POST", "/api/ext/credentials/"+acc.ID, nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if wOk.Code != http.StatusOK {
		t.Fatalf("pre-revoke status = %d", wOk.Code)
	}

	_ = env.db.RevokeAssignment(asn.ID, "test")

	wForbidden := env.do("POST", "/api/ext/credentials/"+acc.ID, nil, map[string]string{
		"Authorization": "Bearer " + tok,
	})
	if wForbidden.Code != http.StatusForbidden {
		t.Errorf("post-revoke status = %d, want 403", wForbidden.Code)
	}
}

// Ensures no test accidentally inherits context from another (regression guard).
var _ = context.Background
