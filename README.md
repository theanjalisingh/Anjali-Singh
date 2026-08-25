# futureEnvironsBE

Go backend for the futureEnvirons IoT platform.

## Current status

- configuration via environment variables
- Gin HTTP server with health checks
- PostgreSQL pool (`DATABASE_URL`)
- login / logout against the existing `identity` schema
- JWT access tokens
- request ID, recovery, and request logging middleware
- consistent JSON response envelope
- graceful shutdown

Identity tables and `identity.sp_get_login_details` already exist in PostgreSQL. This API does not recreate them.

## Run

```bash
cp .env.example .env
# set DATABASE_URL and JWT_SECRET
go run ./cmd/api
```

### Login

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email_id":"admin@futureenvirons.com","password":"Admin@123"}'
```

Password hashes are verified in Go with bcrypt. The stored procedure only returns `password_hash`.

### Logout

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>"
```

Logout currently validates the access token. Server-side invalidation will be added when `identity.refresh_session` exists. Clients should discard the token locally.

## Test

```bash
go test ./...
```
