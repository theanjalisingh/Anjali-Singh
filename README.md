# futureEnvironsBE

Go backend for the futureEnvirons IoT platform.

## Current status

Foundation only:

- configuration via environment variables
- Gin HTTP server
- `/health` and `/api/v1/health`
- request ID, recovery, and request logging middleware
- consistent JSON response envelope
- graceful shutdown

## Run

```bash
cp .env.example .env
go run ./cmd/api
```

Then:

```bash
curl http://localhost:8080/health
```

## Test

```bash
go test ./...
```
