# futureEnvironsBE

Go backend for the futureEnvirons IoT platform.

## Current status

- configuration via environment variables (optional `.env`)
- Gin HTTP server with health checks
- Swagger UI for login and logout
- PostgreSQL pool (`DATABASE_URL`)
- login / logout against the existing `identity` schema
- JWT access tokens
- request ID, recovery, and request logging middleware
- consistent JSON response envelope
- graceful shutdown

Identity tables and `identity.sp_get_login_details` already exist in PostgreSQL. This API does not recreate them.

## Run from VS Code

1. Copy `.env.example` to `.env` and set `DATABASE_URL` plus `JWT_SECRET`.
2. Install the recommended **Go** extension.
3. Run **Launch API** from the Run and Debug view (F5).
4. Open Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

From Swagger:

- Try **POST /api/v1/auth/login** with `admin@futureenvirons.com` / `Admin@123`
- Click **Authorize**, enter `Bearer <access_token>`
- Try **POST /api/v1/auth/logout**

You can also send the same calls from `tests/auth.http` with the REST Client extension.

## Run from the terminal

```bash
cp .env.example .env
# set DATABASE_URL and JWT_SECRET
go run ./cmd/api
```

Swagger UI: `http://localhost:8080/swagger/index.html`

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
