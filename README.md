# go-template

Production-ready Go REST API template using Gin + GORM with JWT authentication, structured logging, and clean architecture.

## Stack

| Layer | Technology |
|-------|-----------|
| Framework | [Gin](https://github.com/gin-gonic/gin) |
| ORM | [GORM](https://gorm.io) + `gorm.io/driver/postgres` |
| Database | PostgreSQL 16 |
| Cache / Session | Redis 7 |
| Authentication | [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) |
| Logging | `log/slog` (Go 1.21+) |
| Validation | [`go-playground/validator/v10`](https://github.com/go-playground/validator) |
| Rate Limiting | `golang.org/x/time/rate` (per-IP token bucket) |
| Config | `joho/godotenv` |
| CLI | `spf13/cobra` |

## Prerequisites

- Go 1.22+
- PostgreSQL 16 (or [NeonDB](https://neon.tech) for cloud)
- Redis 7 *(only required when `JWT_MODE=stateful`)*
- Docker + Docker Compose *(optional)*

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/Ganasa18/go-template.git
cd go-template
cp .env.example .env
```

Edit `.env` and fill in the required values:

```bash
# Generate JWT secrets
openssl rand -base64 64   # paste into JWT_ACCESS_SECRET
openssl rand -base64 64   # paste into JWT_REFRESH_SECRET

# Set your database credentials
DB_HOST=localhost
DB_PASSWORD=your_password
```

### 2. Run database migration

```bash
psql -U postgres -d go_template_dev -f scripts/migrations/0001_create_users_up.sql
```

### 3. Start the server

```bash
go run main.go http
```

Server starts on `http://localhost:8899`.

---

## Docker Compose

Runs the full stack (app + PostgreSQL + Redis) with a single command:

```bash
# Copy docker env file (already pre-filled for local docker use)
cp .env.docker .env.docker   # already exists

docker compose up --build
```

The migration in `scripts/migrations/` runs automatically on first PostgreSQL container start via `docker-entrypoint-initdb.d`.

---

## Environment Variables

See [`.env.example`](.env.example) for the full list. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | `development` or `production` |
| `HTTP_SERVER_PORT` | `8899` | HTTP listen port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_SSL` | `false` | Set `true` for NeonDB / cloud Postgres |
| `JWT_MODE` | `stateful` | `stateful` (access+refresh, Redis) or `stateless` (access only) |
| `JWT_ACCESS_SECRET` | — | **Required.** Generate: `openssl rand -base64 64` |
| `JWT_REFRESH_SECRET` | — | **Required in stateful mode.** |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `LOG_FORMAT` | `text` | `text` for dev, `json` for production |
| `RATE_LIMIT_RPS` | `20` | Requests per second per IP |

---

## API Endpoints

### Health

```
GET /health
```

### Auth

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/api/v1/auth/register` | Register new user | — |
| `POST` | `/api/v1/auth/login` | Login, returns token(s) | — |
| `POST` | `/api/v1/auth/refresh` | Rotate refresh token | — *(stateful only)* |
| `POST` | `/api/v1/auth/logout` | Revoke current token | Bearer |

> `/refresh` and `/logout` are only registered when `JWT_MODE=stateful`.

### Request / Response Shape

**Register**
```json
POST /api/v1/auth/register
{
  "username": "john",
  "email": "john@example.com",
  "password": "secret123"
}
```

**Login**
```json
POST /api/v1/auth/login
{
  "email": "john@example.com",
  "password": "secret123"
}
```

**Success response envelope:**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": 200,
  "message": "success",
  "data": { ... }
}
```

**Validation error (422):**
```json
{
  "request_id": "...",
  "status": 422,
  "message": "validation failed",
  "data": {
    "errors": [
      { "field": "email", "message": "email is required" },
      { "field": "password", "message": "password must be at least 8 characters" }
    ]
  }
}
```

---

## JWT Modes

### `stateful` (default)
- Issues **access token** (15m) + **refresh token** (168h)
- Refresh token stored in Redis: `refresh:{jti}` → userID
- Token rotation on `/refresh` (old key deleted, new pair issued)
- Revocation via `/logout` (sets `revoked:{jti}` key in Redis)
- Requires Redis

### `stateless`
- Issues **access token only**
- No Redis required
- `/refresh` and `/logout` routes are not registered
- Tokens cannot be revoked before expiry

Switch mode with `JWT_MODE=stateless` in `.env`.

---

## Project Structure

```
go-template/
├── cmd/                    # Cobra CLI commands
│   ├── http.go             # `http` subcommand — starts server
│   └── boot.go             # Dependency injection / wiring
├── config/                 # Config loading + DB/Redis/Logger init
├── http/server/
│   ├── server.go           # HTTP server lifecycle (Start/Shutdown)
│   └── router.go           # Route registration + Handlers struct
├── internal/
│   ├── auth/               # Auth domain
│   │   ├── handler/        # HTTP handlers (Register, Login, Refresh, Logout)
│   │   ├── middleware/      # JWT middleware
│   │   ├── models/         # User model, Token types, Request DTOs
│   │   ├── repository/     # DB queries
│   │   └── service/        # stateful.go / stateless.go / register.go
│   └── base/
│       ├── app/            # Context (request_id) + CostumeResponse envelope
│       └── handler/        # RunAction wrapper (logging, panic recovery)
├── pkg/
│   ├── apperror/           # Typed application errors (AppError)
│   ├── logger/             # slog wrapper (WithRequestID, FromContext)
│   ├── middleware/         # CORS, Security headers, Rate limit, Recovery
│   └── validator/          # go-playground/validator wrapper → []FieldError
├── scripts/migrations/     # SQL migration files
├── ci/
│   ├── Dockerfile          # Multi-stage build
│   ├── startup.sh          # Secret injection (Kubernetes secret-volume)
│   └── env.conf            # Production env template with placeholders
├── .env.example            # Safe template — commit this, not .env
└── docker-compose.yml
```

---

## Adding a New Domain

Three files to touch:

1. **Create your handler/service/repository** under `internal/{domain}/`

2. **Register in `http/server/router.go`** — add to `Handlers` struct and wire routes:
   ```go
   type Handlers struct {
       Base          *baseHandler.BaseHTTPHandler
       Auth          *authHandler.HTTPHandler
       Authenticator authService.Authenticator
       // Add here:
       Product *productHandler.HTTPHandler
   }
   ```
   ```go
   // In setupRoutes():
   protected.GET("/products", h.Base.RunAction(h.Product.GetAll))
   ```

3. **Wire in `cmd/boot.go`**:
   ```go
   productRepo := productRepository.New(db)
   productSvc  := productService.New(productRepo)
   productH    := productHandler.New(productSvc)

   return server.New(cfg, &server.Handlers{
       Base:          baseH,
       Auth:          authH,
       Authenticator: authSvc,
       Product:       productH,
   })
   ```

`server.go` never needs to change.

---

## Middleware Stack

Applied globally in this order:

1. **Recovery** — catches panics, logs with slog, returns 500
2. **Security** — sets `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Cache-Control: no-store`
3. **CORS** — explicit origin allowlist from `CORS_ALLOWED_ORIGINS` (no wildcard)
4. **Rate Limit** — per-IP token bucket, idle cleanup every 3 minutes
5. **MaxBodySize** — rejects bodies over `MAX_BODY_BYTES` (default 4 MB)

---

## Production Deployment (Kubernetes)

The `ci/startup.sh` script handles secret injection at container start:

1. Copies `temp/env.conf` (baked into image) to working directory
2. If `/opt/secret-volume/` exists, replaces placeholder tokens with secret file contents
3. If `.env` already exists (e.g. docker-compose mount), skips overwrite
4. Starts the binary

Secret file naming convention matches tokens in `ci/env.conf`:
```
/opt/secret-volume/db_password     → replaces {{DB_PASSWORD}}
/opt/secret-volume/jwt_access_secret → replaces {{JWT_ACCESS_SECRET}}
```

**Checklist before production:**
- [ ] `APP_ENV=production`, `APP_DEBUG=false`
- [ ] `LOG_FORMAT=json`
- [ ] `DB_SSL=true` (if using cloud Postgres / NeonDB)
- [ ] `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` set to strong random values
- [ ] `CORS_ALLOWED_ORIGINS` set to your actual frontend domain(s)
- [ ] `DEV_SHOW_ROUTE=false`
- [ ] Redis password set if Redis is exposed

---

## Makefile

```bash
make run       # go run main.go http
make build     # go build -o bin/app ./...
make test      # go test ./...
make docker    # docker compose up --build
```

---

## License

MIT
