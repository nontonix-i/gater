# Gater

Multi-provider file upload gateway with a single-binary web UI. Upload files or remote URLs to multiple cloud storage and video hosting services simultaneously.

## Features

- **12 providers** — Gofile, Streamtape, Doodstream, Rapidgator, RPMShare, VikingFiles, TurboviPlay, Vidoza, LuluStream, SeekStreaming, Abyss, AnonMP4
- **Dual upload mode** — direct file upload or remote URL fetch per provider capability
- **Real-time progress** — SSE endpoint streams upload progress per provider
- **Web UI** — React + Vite + shadcn/ui + Tailwind CSS v4, embedded in the Go binary
- **Auth** — email/password registration + API key authentication
- **Keepalive scheduler** — periodically visits completed upload URLs to prevent auto-deletion
- **Credential isolation** — per-user provider credentials with global seed fallback

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+ (only for frontend development)
- PostgreSQL

### 1. Clone & configure

```bash
git clone https://github.com/nontonix-i/gater.git
cd gater
cp .env.example .env
```

Edit `.env` with your database URL and provider credentials.

### 2. Seed credentials

```bash
go run ./cmd/seed/main.go
```

This creates a default user and stores provider credentials from `SEED_*` env vars into the database.

### 3. Build frontend (optional — pre-built dist included)

```bash
cd web && npm install && npm run build && cd ..
```

The production build is already at `cmd/server/web/dist/` and embedded via `//go:embed`. Skip this step unless modifying the UI.

### 4. Run

```bash
go run ./cmd/server/main.go
```

Open **http://localhost:8080**

### Build binary

```bash
make build
# or
go build -o gater ./cmd/server/main.go
```

## Configuration

### `config.yaml`

| Key | Default | Description |
|-----|---------|-------------|
| `server.port` | `8080` | HTTP listen port |
| `auth.enabled` | `true` | Require authentication |
| `upload.temp_dir` | `/tmp/gater` | Temp file storage |
| `upload.max_file_size` | `754974720` | Max upload size (bytes) |
| `keepalive.enabled` | `true` | Enable keepalive scheduler |
| `keepalive.check_every` | `1440` | Run interval (minutes) |
| `keepalive.visit_older` | `30` | Visit files older than N days |
| `keepalive.request_limit` | `50` | Max URLs per run |

### `.env`

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `SEED_API_KEY` | Default user API key |
| `SEED_<PROVIDER>` | Provider credentials (`key=val,key2=val2`) |

Provider credentials go to the database via `cmd/seed/main.go`. The server reads credentials from the database at runtime.

## API

All endpoints are under `/api/v1`. Authenticate via `X-API-Key` header or `api_key` query parameter.

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register with email, password, name |
| POST | `/auth/login` | Login, returns API key as token |
| GET | `/auth/me` | Current user info |
| POST | `/auth/regenerate-key` | Generate new API key |
| POST | `/auth/credential` | Save an auth credential |
| GET | `/auth/credentials` | List saved auth credentials |

### Uploads

| Method | Path | Description |
|--------|------|-------------|
| POST | `/upload` | Multipart file upload |
| POST | `/upload/url` | Upload from remote URL |
| GET | `/task/{id}` | Task detail with provider results |
| GET | `/task/{id}/progress` | SSE real-time progress stream |
| GET | `/tasks` | List tasks (paginated) |

### Providers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/providers` | List all providers |
| GET | `/providers/{name}` | Provider detail |
| GET | `/providers/{name}/credentials` | Get credential fields with mask |
| PUT | `/providers/{name}/credentials` | Save per-user credentials |

### Settings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/settings` | Get default providers & API key |
| PUT | `/settings` | Update default providers |

## Architecture

```
                   ┌──────────────┐
                   │   Browser    │
                   │  (React SPA) │
                   └──────┬───────┘
                          │
                   ┌──────▼───────┐
                   │  chi Router  │
                   │  :8080       │
                   └──────┬───────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
       ┌──────▼────┐ ┌───▼───┐ ┌────▼────┐
       │ Auth      │ │ Task  │ │ SSE     │
       │ Middleware │ │Worker │ │ Progress│
       └───────────┘ └───┬───┘ └─────────┘
                         │
              ┌──────────▼──────────┐
              │     Provider.X      │
              │  (12 implementations)│
              └─────────────────────┘
```

- **chi router** — all routes under `/api/v1/`, SPA fallback for non-API paths
- **Auth middleware** — authenticates via `X-API-Key` header or `api_key` query
- **Task worker** — goroutine per provider, concurrent uploads, progress callback
- **SSE** — `GET /task/{id}/progress` polls DB every 1s until all providers finish
- **Keepalive** — separate goroutine, visits completed URLs to prevent auto-deletion

## Frontend

Built with React + Vite + TypeScript + shadcn/ui (base-nova) + Tailwind CSS v4.

### Pages

| Route | Page |
|-------|------|
| `/` | Dashboard — quick upload, provider stats, recent tasks |
| `/tasks` | Task list |
| `/tasks/:id` | Task detail with per-provider results |
| `/settings` | Provider credentials, default providers, API key |
| `/docs` | API documentation |
| `/login` | Login / Register |

> [!NOTE]
> Frontend source is in `web/`. The production build goes to `cmd/server/web/dist/` and is embedded in the Go binary with `//go:embed`. The server serves both the API and the SPA on the same port.

## Providers

| Provider | Type | Upload | Remote URL | API |
|----------|------|--------|-------------|-----|
| Abyss | storage | ✓ | | ✓ |
| AnonMP4 | video | ✓ | ✓ | ✓ |
| Doodstream | video | ✓ | ✓ | ✓ |
| Gofile | storage | ✓ | | ✓ |
| LuluStream | video | ✓ | ✓ | ✓ |
| Rapidgator | storage | ✓ | | ✓ |
| RPMShare | video | ✓ | ✓ | ✓ |
| SeekStreaming | video | ✓ | ✓ | ✓ |
| Streamtape | video | ✓ | ✓ | ✓ |
| TurboviPlay | video | ✓ | | ✓ |
| Vidoza | video | ✓ | ✓ | ✓ |
| VikingFiles | storage | ✓ | ✓ | ✓ |

## License

MIT
