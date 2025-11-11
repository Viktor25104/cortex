#!/usr/bin/env
# Cortex Frontend

Angular single-page application (v17+) serving the Cortex UI.

Build and run
- Local dev: `cd cortex-frontend && npm start`
- Production build: `cd cortex-frontend && npm run build`
- Docker: from repo root `docker build -f Dockerfile.frontend -t ghcr.io/your-org/cortex-frontend:latest .`

Notes
- Source currently lives in `cortex-frontend/` and will be physically moved into `frontend/` in the next refactor phase.
- Nginx config lives at repo root `nginx.conf` and is used by the Docker image.

