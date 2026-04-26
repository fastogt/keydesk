# CLAUDE.md

Project-specific instructions for KeyDesk. Read these before making changes.

## Project Goal

Self-hosted corporate credential manager. Employees use company accounts (LinkedIn, Gmail, Reddit, AWS, Stripe, etc.) without seeing passwords via Chrome extension. One-click offboarding revokes all access.

**Phase 1 priority: traffic and adoption, not revenue.** No paid tiers, no license keys.

## Architecture

```
src/cmd/keydesk.go          ← daemon entry (flags, PID, signals, SIGHUP reload)
src/app/app.go              ← Initialize / Run / Stop / DeInitialize lifecycle
src/app/api/server.go       ← chi router, middleware, static file serving
src/app/api/handlers/       ← HTTP handlers (one file per resource)
src/app/database/           ← SQLite operations (one file per table)
src/app/vault/vault.go      ← AES-256-GCM encryption
src/frontend/               ← TypeScript + esbuild
src/install/public/         ← HTML + CSS + compiled JS
extension/                  ← Chrome Manifest V3 extension
packaging/                  ← systemd service + postinst
```

## Stack — DO NOT CHANGE

- **Go 1.25** + chi v5 router
- **SQLite** via `mattn/go-sqlite3` (foreign keys ON)
- **gofastogt** for HTTP response wrappers (`NewOkResponse`, `NewErrorResponse`, `ErrorJson`)
- **logrus** for logging
- **JWT** via `golang-jwt/jwt/v5` (HS256)
- **bcrypt** for admin passwords (`golang.org/x/crypto/bcrypt`)
- **AES-256-GCM** for credential vault
- **TypeScript + esbuild** for frontend (NO React, NO npm framework)
- **Custom CSS** (NO Tailwind, NO Bootstrap)
- **Chrome Manifest V3** for extension
- **nfpm + systemd** for packaging (NOT Docker)

## Critical Rules

### 1. Never Store Plain Passwords/Secrets

```go
// WRONG
db.CreateAccount(name, ..., req.LoginPassword, ...)

// RIGHT
encrypted, err := h.vault.Encrypt(req.LoginPassword)
if err != nil { ... }
db.CreateAccount(name, ..., encrypted, ...)
```

What MUST be encrypted via `vault.Encrypt`:
- `accounts.login_password`
- `accounts.totp_secret`
- `credentials.key_value`
- `credentials.secret_value`

What is NOT encrypted (plain in SQLite):
- Names, emails, URLs, types, departments, descriptions
- Audit log entries
- Admin password hashes (bcrypt, separate from vault)

### 2. Always Audit Log Write Actions

Every `Create*`, `Update*`, `Delete*`, `Rotate*`, `Reveal*`, `Offboard*` handler must call:

```go
h.db.LogAudit(action, entityType, entityID, personID, performedBy, details)
```

Examples in [src/app/api/handlers/people.go](src/app/api/handlers/people.go), [src/app/api/handlers/accounts.go](src/app/api/handlers/accounts.go).

### 3. Offboarding Does NOT Rotate Passwords

The whole point: **employees never see passwords through the extension**. So when offboarding:
- Revoke assignments (set `revoked_at = NOW`)
- Mark person status = "offboarded"
- Optionally reassign service ownership
- **DO NOT rotate passwords** — other users still need them and the offboarded employee never knew them

Password rotation is a separate manual action (button on Account detail page).

### 4. HTTP Responses Use gofastogt Envelope

```go
// Success
respondJSON(w, http.StatusOK, data)
// → {"data": {...}}

// Error
respondError(w, http.StatusBadRequest, "message")
// → {"error": {"code": 400, "message": "..."}}
```

Helpers in [src/app/api/handlers/utils.go](src/app/api/handlers/utils.go) wrap `gofastogt.NewOkResponse` / `gofastogt.NewErrorResponse` / `gofastogt.ErrorJson`.

Frontend unwraps via `apiCall` in [src/frontend/core/api.ts](src/frontend/core/api.ts).

### 5. Two Separate JWT Auth Flows

- **Admin JWT** (web UI) — `auth.go` middleware, claim `admin_id`, 24h expiry
- **Extension JWT** (employee Chrome extension) — `ext.go` middleware, claim `person_id` + `type: "extension"`, 8h expiry

Never mix tokens — extension routes check `claims["type"] == "extension"`.

### 6. Extension Routes Always Verify Person → Account Assignment

In every `ext.go` handler that returns or uses credentials:

```go
assignments, _ := h.db.GetActiveAssignmentsByPerson(personID)
hasAccess := false
for _, a := range assignments {
    if a.AccountID == accountID { hasAccess = true; break }
}
if !hasAccess {
    respondError(w, http.StatusForbidden, "Access denied")
    return
}
```

Never trust the `account_id` from the request alone.

### 7. Database Patterns

- All IDs are `uuid.New().String()`
- Timestamps stored as `TEXT` in RFC3339 UTC (`nowUTC()` helper)
- Foreign keys with `ON DELETE CASCADE` where deletion should propagate, `ON DELETE SET NULL` for soft links (e.g., `services.owner_id`)
- Always paginate or filter — never `SELECT * FROM table` without limit on user-facing endpoints

### 8. Frontend Conventions

- Each page = one TypeScript file in `src/frontend/` + one HTML in `src/install/public/`
- Shared logic in `src/frontend/core/` (`api.ts`, `storage.ts`, `sidebar.ts`, `utils.ts`, `types.ts`)
- Always `esc()` user-supplied strings before injecting into innerHTML
- Use `esbuild` config in `build.js` — add new entry point when adding a new page
- TypeScript strict mode, no `any` except for API responses (`type` from server is dynamic)

### 9. Extension Code

- Manifest V3 (service worker, not background page)
- Background service worker handles all API calls (centralized auth)
- Content script only talks to background via `chrome.runtime.sendMessage`
- Never store passwords in `chrome.storage` — fetch from server, fill form, discard
- Auto-detect: content script polls `/api/ext/match` on page load, shows banner if URL matches an assigned account

### 10. Build & Packaging

```bash
make build              # local build
make build-linux-amd64  # cross-compile
make package-deb        # create .deb
make package-rpm        # create .rpm
make package-all        # both

make frontend-build     # rebuild frontend only
make frontend-watch     # watch mode for development
make frontend-check     # TypeScript type check
make fmt                # gofmt
make vet                # go vet
make test               # go test
```

Version is generated into `src/app/version/version.go` by `scripts/generate_version.sh`. Never edit `version.go` directly — it's regenerated on every build.

## Chrome Extension Threat Model

The extension hides passwords from employees ONLY when:
1. Corporate laptop with **managed Chrome** (Group Policy / MDM)
2. **DevTools disabled** via `DeveloperToolsAvailability=2` policy
3. **Extension force-installed** (employee can't uninstall)
4. **Password manager save disabled** via Chrome policy

On unmanaged personal devices, a determined user can still extract the password via DevTools. This is a known limitation. Document it — don't pretend otherwise.

## Common Pitfalls to Avoid

- **Don't add Docker.** Stack is `.deb` + systemd, like nginx.
- **Don't add React/Tailwind.** Plain TypeScript + custom CSS.
- **Don't add Google OAuth login.** MVP uses local email/password (admin) and person ID (extension).
- **Don't add background expiry checker / Slack alerts.** Not in MVP. Dashboard query is enough.
- **Don't add multi-admin RBAC.** Single admin tier in MVP.
- **Don't replace `gofastogt` response wrapping.** All FastoCloud Go projects use it.
- **Don't commit `src/go.sum`.** It's in `.gitignore`.
- **Don't use Co-Authored-By in git commits.** User's global rule.
- **Don't push or commit unless explicitly asked.**

## When Adding a New Resource

If you add a new entity (e.g., "tags", "groups"):

1. Create migration in [src/app/database/database.go](src/app/database/database.go) `migrate()` function
2. Create CRUD file `src/app/database/<resource>.go`
3. Create handler file `src/app/api/handlers/<resource>.go`
4. Wire routes in [src/app/api/server.go](src/app/api/server.go) `Routes()`
5. Audit log every write action
6. Add TypeScript page in `src/frontend/<resource>.ts`
7. Add HTML page in `src/install/public/<resource>.html`
8. Add esbuild entry in [src/frontend/build.js](src/frontend/build.js)
9. Add navigation link in [src/frontend/core/sidebar.ts](src/frontend/core/sidebar.ts)
10. Add type in [src/frontend/core/types.ts](src/frontend/core/types.ts)

## Reference

- Schema: 7 tables (admins, people, accounts, services, credentials, assignments, audit_log)
- 9 web pages (login, dashboard, people list/detail, accounts list/detail, services list/detail, settings)
- API base: `/api/*` (admin) and `/api/ext/*` (extension)
- Default config: `/etc/keydesk.conf`, port `6690`, data in `/var/lib/keydesk/`
- Repository: https://github.com/fastogt/keydesk
- License: Apache 2.0
