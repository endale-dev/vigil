# Vigil backend

Vigil is a self-hosted uptime monitor. The backend checks HTTP, TCP, and TLS targets, stores monitor history and incidents in MongoDB, and sends configured email or Telegram alerts. It also serves the Vue application from the same process in production.

## Architecture

`main.go` loads configuration and wires the application dependencies. HTTP handlers call services through interfaces; services use repository interfaces; MongoDB repositories implement persistence. The scheduler runs checks alongside the HTTP server. Platform packages contain integrations such as MongoDB, probes, Google OAuth, email, Telegram, clocks, and security helpers.

## Requirements

- Go 1.26 or newer
- Node.js 24 and npm
- MongoDB 7 or a compatible MongoDB deployment
- GNU Make for the commands below

Docker and Docker Compose are optional for container deployment.

## Local development

From the repository root, create the local Go module files and install frontend dependencies:

```sh
make backend-module
npm --prefix frontend ci
```

Start MongoDB, then set the backend environment in the same terminal before starting it. For PowerShell:

```powershell
$env:APP_ENV = "development"
$env:APP_BASE_URL = "http://localhost:8080"
$env:MONGO_URI = "mongodb://localhost:27017"
$env:MONGO_DB = "vigil"
make run
```

The backend listens on `http://localhost:8080`. In a second terminal, start the Vue development server:

```sh
npm --prefix frontend run dev
```

Open `http://localhost:3000`. Vite forwards `/api` requests to the backend. The backend loads the optional `.env` file in its working directory (`backend/.env` when using `make run` or `go -C backend run .`). Process environment variables take precedence, including explicitly empty values. Restart the backend after changing the file. Docker Compose also reads `backend/.env` as described below.

For a production-style local build, run `make build`; the command builds the frontend, copies it into `backend/dist`, and creates `backend/bin/vigil` (or `vigil.exe` on Windows). The production server serves that embedded frontend from the backend origin.

## Configuration

Copy `backend/.env.example` to `backend/.env` and fill in your local settings, including `TELEGRAM_BOT_TOKEN` to enable Telegram. Alternatively, export variables in your shell. The local file supports `KEY=value`, single or double quotes, comments, and optional `export` prefixes; shell expansion and multiline values are not supported. It is ignored by Git. After restarting the backend, reload the app, start a chat with your bot, and save your numeric chat ID under Alerts.

| Variable | Required | Default or purpose |
| --- | --- | --- |
| `APP_ENV` | No | `development`; use `production` for deployment. |
| `APP_ADDR` | No | `:8080`, the HTTP listen address. |
| `APP_BASE_URL` | Yes | Public origin, such as `http://localhost:8080`; production requires HTTPS. |
| `MONGO_URI` | Yes | MongoDB connection URI. |
| `MONGO_DB` | No | `vigil`, database name. |
| `COOKIE_SECURE` | No | Defaults to false in development and true in production. Set explicitly only when needed. |
| `SESSION_TTL` | No | `720h`, session lifetime. |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | No | Set both to enable Google sign-in. The callback is `{APP_BASE_URL}/api/auth/google/callback`. |
| `TELEGRAM_BOT_TOKEN` | No | Enables Telegram notification settings and alerts. |
| `SMTP_HOST` | Production | SMTP server; required in production. Leave blank in development to log email delivery instead. |
| `SMTP_PORT` | When SMTP is enabled | `587` when omitted. |
| `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` | When SMTP is enabled | SMTP credentials and sender address. |
| `CHECK_WORKERS` | No | `50`, maximum concurrent check workers. |
| `SCHEDULER_TICK` | No | `1s`, scheduler interval. |
| `FAILURE_THRESHOLD` | No | `2`, consecutive failures before an incident is opened. |
| `MAX_MONITORS_PER_USER` | No | `25`. |
| `CHECK_RETENTION` | No | `720h`, check history retention. |
| `ALLOW_PRIVATE_TARGETS` | No | `false`; controls checks to private network targets. |
| `TRUST_PROXY_HEADERS` | No | `false`; enable only behind a trusted reverse proxy. |

## Development commands

Run `make test` for Go race-enabled tests, `make web` for frontend type checking and build, `make build` for the executable, and `make check` for the complete local verification suite. `make docker` builds the container image as `vigil:local` and requires a running Docker engine.

## Docker deployment

Create `backend/.env` with production values. Set `APP_ENV=production`, `APP_BASE_URL` to the HTTPS origin users will visit, and provide real SMTP settings. Set Google credentials if Google sign-in is desired. Do not use placeholder secrets in a deployed environment.

From the repository root:

```sh
docker compose -f backend/docker-compose.yml up --build -d
```

Compose starts MongoDB with a persistent named volume, waits for its health check, and points Vigil at that service. Vigil binds to `127.0.0.1:8080`; terminate HTTPS at a trusted reverse proxy and configure `TRUST_PROXY_HEADERS=true` only when that proxy controls the forwarded headers. Back up the MongoDB volume as part of deployment operations.

## Security notes

- Keep `.env` files and credentials out of Git. Rotate credentials if they have been exposed.
- Use HTTPS in production so session cookies can be secure.
- Keep `ALLOW_PRIVATE_TARGETS=false` unless private target monitoring is a deliberate deployment requirement.
- Keep the service behind a trusted network boundary and configure proxy header trust only for a controlled reverse proxy.
- MongoDB has no authentication configured in the included Compose file; restrict database access to the Compose network and add deployment-specific database security when needed.

## Current limitations

- Account name can be changed; email and sign-in method are read-only.
- Email delivery in development is logged unless SMTP is configured.
- The application does not terminate TLS itself; production HTTPS requires a reverse proxy or another TLS terminator.
- The included Compose setup is a single-instance deployment starter, not a highly available or automated backup system.
