# AGENTS.md

## Cursor Cloud specific instructions

### Overview

RisegerDB Go Edition is a geographic spatial database using R-tree indexing. It is a pure Go project with zero external service dependencies (no Docker, databases, or caches required).

### Services

| Service | Command | Notes |
|---|---|---|
| **riseger-server** | `go run ./cmd/riseger-server --data ./data --port 12000` | Main HTTP server on port 12000 |
| **riseger-cli** | `go run ./cmd/riseger-cli --port 12000` | Interactive CLI (requires server running) |
| **riseger-geodata-gen** | `go run ./cmd/riseger-geodata-gen -o sample.geodata -db test_db -buildings 5` | Generates test `.geodata` files |

### Standard commands

- **Tests**: `go test ./...`
- **Lint**: `go vet ./...`
- **Build**: `go build -o <binary> ./cmd/<name>` (see README for all three targets)
- **Health check**: `curl http://localhost:12000/health`
- **Query**: `curl -X POST http://localhost:12000/query -H "Content-Type: application/json" -d '{"sql": "..."}'`

### Caveats

- The `PRELOAD` command is asynchronous — it returns immediately with `"status":"accepted"`. Allow ~1 second before querying imported data.
- The geodata generator creates a map named `china_mp` in the target database, not whatever map name you manually created. Query `china_mp` to see imported data.
- `GET DATABASES` may return `rowCount: 0` even when databases exist; use `GET MAPS` or `GET MODELS` scoped to a specific database to verify.
- The server's `--data` flag specifies the filesystem directory for persistence. Use a temp directory (e.g., `/tmp/riseger-data`) for ephemeral testing.
