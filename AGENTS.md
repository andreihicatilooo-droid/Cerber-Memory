# AGENTS.md

## Cursor Cloud specific instructions

CERBER Memory is a Go (module `cerber-memory`, Go 1.25.x) hybrid pipeline + MVC service.
It ingests raw text, sends it to the Gemini API for structured extraction, and routes the
result into SQLite memory categories (core memory, notes, ideas, goals, tasks, documents,
resources). A web dashboard (`web/`) talks to a `--server` HTTP API, and there is an
optional Qdrant vector DB + GCS file storage.

### Build / lint / test
- Build library packages: `go build ./...`
- Lint: `go vet ./...`
- Test: `go test ./...`
- CGO is required (the `github.com/mattn/go-sqlite3` driver). `gcc` must be present and
  `CGO_ENABLED=1` (the Go default). The dependency refresh (`go mod download`) runs on startup.

### IMPORTANT: the application entrypoint is missing from the repository
- `README.md`, `Dockerfile`, and `docker-compose.yml` all reference `./cmd/cerber`, but the
  `cmd/` directory (and `internal/tools/`, `internal/mvc/views/`) is **not committed**, so
  there is **no `package main` / `func main`** anywhere in the tree.
- Consequence: `go build ./cmd/cerber` fails ("directory not found"), the `cerber` binary
  cannot be produced, and the web server / Docker image (`docker compose up`) cannot run as-is.
  Only the `internal/...` library packages build, vet, and test.
- To actually run the web dashboard or CLI, the missing `cmd/cerber/main.go` entrypoint must
  first be added (it must wire up `models.InitDB`, the `/api/*` handlers used by `web/app.js`,
  and serve the `web/` directory). This is a code gap, not an environment problem.

### Running the core pipeline
- The runtime ingest flow (`/api/process` → `pipeline.ParseTextWithLLM`) calls the Gemini API
  and needs a valid `GEMINI_API_KEY` (see `.env-dist`). Without it, only the deterministic
  routing/storage layer (`controllers.RouteItems` → `internal/mvc/models`, SQLite) can be
  exercised. See `internal/mvc/controllers/smoke_test.go` for a no-network example that
  initializes a temp SQLite DB and verifies items are routed into the correct categories.
- `internal/mvc/models.GetMindmapGraph` only includes ideas, tasks, notebooks, projects,
  documents, external_resources, files, and core_memory as nodes — notes and goals are
  intentionally not graphed.

### Config
- Copy `.env-dist` to `.env` (or set env vars). `GEMINI_API_KEY`, `CERBER_MASTER_KEY`
  (must be exactly 32 chars for AES-256), `QDRANT_HOST`, `QDRANT_PORT`, `PORT` are read via
  viper/`os.Getenv`. `run.sh` is an interactive Docker installer (prompts for keys — avoid in
  non-interactive/CI contexts).
