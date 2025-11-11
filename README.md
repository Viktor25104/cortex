# Cortex v6.2 → v8.0 Design Transplant & Infra

This repo hosts the Cortex backend (Go) and frontend (Angular). We are migrating to a clean mono‑layout with:

- `frontend/`: Angular SPA (currently source in `cortex-frontend/`, move planned)
- `backend/`: Go API (currently in repo root folders, move planned)
- `infra/`: Kubernetes manifests and Terraform IaC
- Root Dockerfiles and Nginx config

Quick Start
- Backend (local): `go build -o cortex . && ./cortex --server`
- Frontend (local): `cd cortex-frontend && npm start`
- Docker builds:
  - Frontend: `docker build -f Dockerfile.frontend -t ghcr.io/your-org/cortex-frontend:latest .`
  - Backend: `docker build -f Dockerfile.backend -t ghcr.io/your-org/cortex-backend:latest .`

Kubernetes (manifests)
- Apply namespace, config, secrets, deploys, services, ingress:
  - `kubectl apply -f infra/k8s/namespace.yaml`
  - `kubectl apply -f infra/k8s/configmap.yaml`
  - `kubectl apply -f infra/k8s/secret-example.yaml` (replace token)
  - `kubectl apply -f infra/k8s/deployment-frontend.yaml -f infra/k8s/service-frontend.yaml`
  - `kubectl apply -f infra/k8s/deployment-backend.yaml -f infra/k8s/service-backend.yaml`
  - `kubectl apply -f infra/k8s/networkpolicy-namespace-default-deny.yaml -f infra/k8s/networkpolicy-backend.yaml`
  - `kubectl apply -f infra/k8s/ingress.yaml` (set host + cert-manager)

Terraform IaC (deploy into existing cluster)
- `cd infra/terraform`
- Set variables via `-var` or `terraform.tfvars` (at minimum `api_key` and optionally `host`, images)
- `terraform init && terraform apply`

Security Baseline
- Namespace isolation and default deny NetworkPolicy
- Backend ingress allowed only from frontend and ingress-nginx
- Backend egress minimal (DNS + Redis)
- Containers run as nonroot, read-only FS
- Ingress via ingress-nginx + TLS (cert‑manager)

Notes
- We will introduce a root `go.work` and physically move sources into `backend/` and `frontend/` in the next step without rewriting imports.
