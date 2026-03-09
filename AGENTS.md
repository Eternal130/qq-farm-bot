# AGENTS.md — qq-farm-bot

## Project Overview

Go 1.24 + Vue 3 automation bot for QQ/WeChat farm mini-program. Communicates with game server via WebSocket + Protocol Buffers. Single-binary deployment with embedded frontend.

**Tech stack**: Go / Gin / SQLite / WebSocket / Protobuf / JWT | Vue 3 / TypeScript / Element Plus / Vite / Pinia

## Build & Run Commands

```bash
# Install all dependencies (frontend npm + Go modules)
make deps

# Full build (frontend → embedded into Go binary)
make all

# Individual builds
make frontend          # cd web && npm run build (outputs to cmd/server/dist/)
make backend           # go build -o qq-farm-bot ./cmd/server/
make clean             # rm build artifacts

# Development (run separately in two terminals)
go run ./cmd/server/           # backend on :18080
cd web && npm run dev          # frontend dev server on :3000 (proxies /api → :8080)

# Type-check frontend only
cd web && npx vue-tsc --noEmit
```

**No test suite exists.** There are zero `*_test.go` files and no frontend tests. No linter or formatter configuration files are present.

## Project Structure

```
cmd/
  server/main.go              # Entry point, embeds frontend via //go:embed
  gen-crop-yield/main.go      # Offline tool for crop yield data generation
internal/
  api/                        # Gin HTTP handlers + WebSocket log streaming
    router.go                 # Route setup, CORS, SPA fallback, embedded frontend serving
    account.go                # Account CRUD endpoints
    bot.go                    # Bot start/stop control
    dashboard.go              # Dashboard statistics
    log.go                    # Log API + WebSocket push
  auth/                       # JWT auth (handler.go, jwt.go, middleware.go)
  bot/                        # Core bot logic
    manager.go                # Multi-account bot lifecycle manager
    instance.go               # Single bot instance (connect, reconnect, watchdog)
    network.go                # WebSocket connection + Protobuf message codec
    farm.go                   # Farm automation (harvest, plant, fertilize, weed, water)
    friend.go                 # Friend farm operations (steal, help)
    fertilizer.go             # Fertilizer purchase/use system
    warehouse.go              # Crop selling
    task.go                   # In-game task claiming
    qqlogin.go                # QQ QR code login flow
    gameconfig.go             # Static game data loading (crops, levels, items)
    landcache.go              # Land state cache + level-up estimation
    logger.go                 # Structured logger with SQLite storage + WebSocket broadcast
  config/config.go            # JSON config loading with defaults
  model/account.go            # Data models: Account, BotStatus, LogEntry, OpRecord
  store/db.go                 # SQLite storage layer with migrations
proto/                        # .proto definitions, one subdir per domain (plantpb/, friendpb/, etc.)
itempb/, mallpb/              # Generated Go protobuf code (top-level, some kept outside proto/)
gameConfig/                   # Static JSON game data (Plant.json, RoleLevel.json, ItemInfo.json)
web/                          # Vue 3 frontend (see Frontend section below)
data/                         # Runtime SQLite database (gitignored)
config.json                   # Server config (listen addr, JWT secret, DB path, game server URL)
```

## Go Code Style

### Imports

Three groups separated by blank lines: stdlib → third-party → internal.

```go
import (
    "fmt"
    "sync"

    "github.com/gin-gonic/gin"

    "qq-farm-bot/internal/config"
    "qq-farm-bot/internal/model"
    "qq-farm-bot/internal/store"
)
```

Within the internal group, proto imports may form a sub-group:

```go
    "qq-farm-bot/internal/model"

    "qq-farm-bot/proto/plantpb"
    "qq-farm-bot/proto/shoppb"
```

### Naming

- **Packages**: lowercase single word (`api`, `auth`, `bot`, `config`, `model`, `store`)
- **Exported types**: PascalCase (`Manager`, `FarmWorker`, `ServerError`)
- **Unexported types**: camelCase (`landStatus`, `loginReq`)
- **Constants**: PascalCase for exported (`OpHarvest`), camelCase for unexported (`normalFertilizerID`)
- **Protobuf packages**: `{domain}pb` naming (`plantpb`, `friendpb`, `corepb`)

### Error Handling

- Early return on error — no else branches after error checks
- `fmt.Errorf("context: %w", err)` for error wrapping in store/infra code
- Simple `return err` passthrough in business logic
- Custom `ServerError` type in `bot/network.go` for game server errors
- Logger methods (`Warnf`, `Errorf`) for non-fatal operational errors — no panics
- `fmt.Printf` + `os.Exit(1)` for fatal startup errors in `main.go`

```go
if err != nil {
    f.logger.Warnf("巡田", "检查失败: %v", err)
    return
}
```

### Struct Tags

JSON tags use `snake_case`. Use `omitempty` for optional fields. Use `json:"-"` for internal-only fields.

```go
type Account struct {
    ID        int64  `json:"id"`
    UserID    int64  `json:"user_id"`
    Name      string `json:"name"`
    Platform  string `json:"platform"`
    AutoStart bool   `json:"auto_start"`
}
```

Gin binding tags for request validation:

```go
type registerReq struct {
    Username string `json:"username" binding:"required,min=3,max=32"`
    Password string `json:"password" binding:"required,min=6"`
}
```

### Comments

- Exported types/functions: `// TypeName does X.` (godoc style)
- Inline comments for non-obvious logic, in English or Chinese
- Section separators with `// ---` in larger files (e.g., `network.go`)

### Concurrency

- `sync.RWMutex` for shared state (Manager.mu, Logger.mu)
- `context.Context` for cancellation (bot instance lifecycle)
- Channel-based pub/sub for log broadcasting
- `defer m.mu.Unlock()` / `defer m.mu.RUnlock()` immediately after lock

### API Pattern (Gin)

Handlers are registered via `Register*Routes(r *gin.RouterGroup, ...)` functions. JSON responses use `gin.H{}`. Auth via JWT middleware setting `userID`/`username`/`isAdmin` in context.

```go
func RegisterAccountRoutes(r *gin.RouterGroup, s *store.Store, mgr *bot.Manager, cfg *config.Config) {
    r.GET("/accounts", func(c *gin.Context) {
        userID := c.GetInt64("userID")
        // ...
        c.JSON(http.StatusOK, result)
    })
}
```

Error responses: `c.JSON(http.StatusXxx, gin.H{"error": "message"})`.

### Protobuf Usage

- `.proto` files in `proto/{domain}pb/` with `option go_package = "qq-farm-bot/proto/{domain}pb"`
- Generated Go code lives alongside `.proto` files
- `proto.Marshal` / `proto.Unmarshal` for serialization
- Messages sent via `net.SendRequest("gamepb.{pkg}.{Service}", "Method", body)`

## Frontend Code Style (web/)

### Stack

Vue 3 + Composition API (`<script setup lang="ts">`) / TypeScript (strict) / Element Plus / Vite / Pinia / Axios

### File Organization

```
web/src/
  api/index.ts         # Axios instance, interceptors, all API functions + TypeScript interfaces
  stores/auth.ts       # Pinia store (setup function pattern)
  router/index.ts      # Vue Router with auth guards
  views/*View.vue      # Page components (PascalCase + "View" suffix)
  layouts/MainLayout.vue
  components/          # Reusable components
  data/                # Static data files
```

### Conventions

- **Components**: `<script setup lang="ts">` — no Options API
- **Path alias**: `@/` maps to `src/` (configured in tsconfig + vite)
- **API layer**: Single `api/index.ts` exports typed API objects (`authApi`, `accountApi`, `dashboardApi`)
- **State**: Pinia setup stores with `defineStore('name', () => { ... })`
- **Routing**: Lazy-loaded routes via `() => import('@/views/XxxView.vue')`
- **UI**: Element Plus components with Chinese locale (`zh-cn`)
- **Styling**: Scoped `<style scoped>` per component, CSS custom properties
- **Error handling**: `try/catch` with `getErrorMessage(error, fallback)` helper + `ElMessage.error()`

### TypeScript

- Strict mode enabled (`strict: true`, `noUnusedLocals`, `noUnusedParameters`)
- Interfaces defined in `api/index.ts` matching backend JSON shape (snake_case field names)
- Type imports: `import type { X } from '...'`

### Frontend Build

Vite builds to `../cmd/server/dist/` which the Go binary embeds via `//go:embed all:dist`. In production, the Go server serves the SPA with a fallback to `index.html` for client-side routing.

## Key Files to Read First

When onboarding, read in this order:
1. `cmd/server/main.go` — entry point, dependency wiring
2. `internal/bot/instance.go` — bot lifecycle (start → connect → run loops → reconnect)
3. `internal/bot/network.go` — WebSocket + protobuf message layer
4. `internal/bot/farm.go` — core farm automation logic
5. `internal/api/router.go` — HTTP routing + frontend serving
6. `web/src/api/index.ts` — all API types and endpoints

## Important Notes

- **No tests exist** — add `*_test.go` files alongside the code they test
- **No linter config** — use `go vet` and `gofmt` as baseline
- **Log messages are in Chinese** — match existing style for consistency
- **config.json contains secrets** — never commit real credentials
- **Protobuf code is checked in** — regenerate with `protoc` if `.proto` files change
- **Frontend is embedded** — must run `make frontend` before `make backend` for production builds
