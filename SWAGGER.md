# Smart Swagger Handler - Development Workflow

## Quick Start

### Development (Instant Updates! ⚡)

```bash
# 1. Edit proto and regenerate
cd omnipos-proto
vim user/v1/user.proto
buf generate

# 2. That's it! Gateway auto-detects changes
cd ../omnipos-gateway
go run cmd/http/main.go
# Open http://localhost:8081/swagger-ui
```

**No copying needed!** The gateway automatically serves swagger specs from `../omnipos-proto/proto` in development.

---

### Production Build

```bash
# Before building for production, sync swagger specs
cd omnipos-gateway
./scripts/sync-swagger.sh

# Build binary (specs are embedded)
go build -o bin/gateway cmd/http/main.go

# Or build Docker image
docker build -t omnipos-gateway .
```

---

## How It Works

The swagger handler is **smart**:

- ✅ **Development**: Detects `../omnipos-proto/proto` directory and serves specs directly (live reload!)
- ✅ **Production**: Falls back to embedded specs when proto directory doesn't exist

**Auto-detection:**

```
Gateway startup checks:
├─ Does ../omnipos-proto/proto exist?
│  ├─ YES → Development mode (serve from proto directory)
│  └─ NO  → Production mode (serve from embedded specs)
```

---

## Files

- `internal/swagger/handler.go` - Smart handler with dev/prod detection
- `scripts/sync-swagger.sh` - Sync script for production build
- `internal/swagger/specs/` - Embedded specs directory (gitignored)

---

## Swagger Spec URLs

**Development mode:**

- Swagger UI: `http://localhost:8081/swagger-ui`
- User spec: `http://localhost:8081/openapi/user/v1/user.swagger.json`

**Production mode:**

- Swagger UI: `http://localhost:8081/swagger-ui`
- User spec: `http://localhost:8081/openapi/user.swagger.json`

The URLs differ because development serves the full proto directory structure.

---

## Adding New Services

When you add more services (product, order, etc.):

1. Generate proto: `cd omnipos-proto && buf generate`
2. Gateway automatically detects new swagger files!
3. Update Swagger UI to show multiple specs (multi-spec support coming soon)

---

## Troubleshooting

**Swagger UI shows old API?**

- Development: Re-run `buf generate` in omnipos-proto
- Production: Run `./scripts/sync-swagger.sh` and rebuild

**"No swagger files found" error?**

- Make sure you ran `buf generate` in omnipos-proto first
- Check that `../omnipos-proto/proto` exists relative to gateway

**Want to force production mode in development?**

- Temporarily rename `../omnipos-proto` to test embedded mode
