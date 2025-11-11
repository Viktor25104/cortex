#!/usr/bin/env
# Cortex Backend

Go API + workers providing scan orchestration.

Build and run
- Local: `go build -o cortex . && ./cortex --server`
- Docker: from repo root `docker build -f Dockerfile.backend -t ghcr.io/your-org/cortex-backend:latest .`

Env
- `CORTEX_API_KEY` (required)
- `REDIS_ADDR` (default `localhost:6379` or in k8s via ConfigMap)

Notes
- Will be moved under `backend/` with a root `go.work` in the next refactor phase to avoid import rewrites.
- Health endpoint expected at `/healthz` for probes (configure in API if missing).
- The binary expects `./nmap-service-probes` in working directory (packaged into Docker image in `/app/nmap-service-probes`).
