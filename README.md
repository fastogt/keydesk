# KeyDesk

**Self-hosted corporate credential manager. Employees use company accounts without seeing passwords.**

Share company credentials on onboarding. Revoke everything in one click on offboarding. Chrome extension lets employees login to company accounts — LinkedIn, Gmail, Reddit, AWS, Stripe — without ever seeing a password.

## The Problem

Your company has shared accounts — Gmail, LinkedIn, Reddit, AWS, Stripe.
You track them in a spreadsheet. Someone gets fired. You forget to change 3 passwords.
Ex-employee still has your company LinkedIn.

## The Solution

KeyDesk is a self-hosted credential desk for your company.
Add accounts. Assign to employees. They login via Chrome extension — no passwords visible.
Someone leaves? One click — all access revoked, all passwords rotated.

## Features

- **Credential vault** — encrypted storage (AES-256-GCM) for passwords, API keys, tokens, OAuth credentials, TOTP seeds
- **Give / take access** — assign company accounts to employees, track who has what
- **Chrome extension** — employees login to company accounts without seeing passwords
- **One-click offboarding** — revoke all access, rotate all passwords, reassign services
- **Service credentials** — track API keys with expiry dates, get warnings before they expire
- **TOTP auto-fill** — extension handles 2FA codes automatically
- **Full audit log** — who accessed what, when, given by whom
- **Single binary** — one `.deb` package, one systemd service, SQLite database
- **No Docker required** — standard Linux daemon, installs like nginx

## Quick Start

```bash
# Download latest release
wget https://github.com/fastogt/keydesk/releases/latest/download/keydesk-1.0.0.1-amd64.deb

# Install
sudo dpkg -i keydesk-1.0.0.1-amd64.deb

# Edit config (set jwt_secret and vault_master_key)
sudo nano /etc/keydesk.conf

# Start
sudo systemctl start keydesk
sudo systemctl enable keydesk

# Create admin user
sudo keydesk create-admin --email admin@company.com --password changeme

# Open browser
# http://localhost:6690
```

## Chrome Extension

Install from [Chrome Web Store](#) or load unpacked from the `extension/` directory.

1. Employee installs the extension
2. Enters KeyDesk server URL and their Person ID
3. Extension shows their assigned accounts
4. Click **Open** — logged in automatically, password never visible

On managed corporate laptops with DevTools disabled, employees physically cannot extract passwords.

## How It Works

```
Admin adds company accounts (LinkedIn, Gmail, AWS, Stripe...)
     ↓
Admin assigns accounts to employees
     ↓
Employee opens Chrome → extension shows their accounts
     ↓
Employee clicks [Open] → logged in, password never visible
     ↓
Employee fired → admin clicks [Offboard] → done
     ↓
All passwords rotated, remaining users notified
```

## Why Not...

| Tool | Problem |
|------|---------|
| **Spreadsheet** | No security, no tracking, forget to revoke |
| **Bitwarden / 1Password** | No assignment tracking, no offboarding automation, employee sees all passwords |
| **CyberArk** | $200k+/year, 6 months to deploy, needs 8-10 Windows servers |
| **KeyDesk** | Free, self-hosted, 5-minute install, employees never see passwords |

## Tech Stack

- **Backend:** Go, Chi, SQLite, AES-256-GCM encryption
- **Frontend:** TypeScript, esbuild, custom CSS
- **Extension:** Chrome Manifest V3
- **Packaging:** `.deb` / `.rpm` via nfpm, systemd service
- **Dependencies:** gofastogt, logrus, jwt-go

## Configuration

```yaml
# /etc/keydesk.conf
settings:
  host: 127.0.0.1:6690
  log_path: ~/keydesk.log
  log_level: INFO
  database: /var/lib/keydesk/keydesk.db
  jwt_secret: "YOUR_SECRET_HERE"
  vault_master_key: "YOUR_32_BYTE_HEX_KEY"
```

Generate a vault master key:
```bash
openssl rand -hex 32
```

## Building from Source

```bash
# Prerequisites: Go 1.25+, Node.js 20+, npm

# Clone
git clone https://github.com/fastogt/keydesk.git
cd keydesk

# Development setup
make dev-setup

# Build
make build

# Run locally
./build/bin/keydesk --config config/keydesk.conf --no-pid-file

# Build .deb package
make package-deb-amd64
```

## API

All responses follow the `{"data": {...}}` / `{"error": {"code": N, "message": "..."}}` envelope.

### Admin API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | Admin login (email + password → JWT) |
| GET | `/api/dashboard` | Stats, warnings, recent activity |
| GET/POST | `/api/people` | List / create people |
| GET/PUT/DELETE | `/api/people/:id` | Get / update / delete person |
| POST | `/api/people/:id/offboard` | One-click offboarding |
| GET/POST | `/api/accounts` | List / create accounts |
| GET/PUT/DELETE | `/api/accounts/:id` | Get / update / delete account |
| POST | `/api/accounts/:id/reveal` | Decrypt and return password |
| POST | `/api/accounts/:id/rotate` | Generate new password |
| GET/POST | `/api/services` | List / create services |
| POST | `/api/credentials` | Add credential to service |
| POST | `/api/credentials/:id/reveal` | Decrypt credential value |
| POST | `/api/assignments` | Give account to person |
| DELETE | `/api/assignments/:id` | Take account back |

### Extension API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/ext/auth` | Extension login (person_id → JWT) |
| GET | `/api/ext/accounts` | List assigned accounts |
| POST | `/api/ext/credentials/:id` | Get credentials for auto-fill |
| GET | `/api/ext/match?url=` | Check if URL matches an account |
| POST | `/api/ext/totp/:id` | Get current TOTP code |

## License

Apache 2.0

## Contributing

Issues and PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

Built by [FastoCloud](https://github.com/fastogt)
