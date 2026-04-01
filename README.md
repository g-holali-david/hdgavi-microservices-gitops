# Microservices + GitOps Pipeline

[![CI](https://github.com/g-holali-david/hdgavi-microservices-gitops/actions/workflows/ci.yml/badge.svg)](https://github.com/g-holali-david/hdgavi-microservices-gitops/actions/workflows/ci.yml)

Full DevOps lifecycle demo: from code to automated deployment on Kubernetes via GitOps, with observability.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  API Gateway │────▶│  Auth Svc   │     │  Worker Svc │
│  (Go)        │     │  (Go + JWT) │     │  (Python)   │
└──────┬───────┘     └─────────────┘     └──────┬──────┘
       │                                        │
       └────────────────────────────────────────┘
                        │
                 ┌──────▼──────┐     ┌──────────┐
                 │  PostgreSQL │     │  Redis   │
                 │             │     │  (Queue) │
                 └─────────────┘     └──────────┘
```

## Services

| Service | Language | Port | Description |
|---------|----------|------|-------------|
| **api-gateway** | Go | 8080 | Reverse proxy, JWT validation, CORS, rate limiting |
| **auth-service** | Go | 8081 | JWT token issuance (login, verify, refresh) |
| **worker-service** | Python | 8082 | Async task processing via Redis queue |

## Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.22 (Gateway + Auth), Python 3.12 / FastAPI (Worker) |
| **Queue** | Redis 7 |
| **Database** | PostgreSQL 16 |
| **Containers** | Docker multi-stage, Distroless (Go), slim (Python) |
| **Orchestration** | Kubernetes (Kind/Minikube) |
| **Packaging** | Helm Charts (1 per service) |
| **CI** | GitHub Actions (test → scan → push GHCR) |
| **CD** | ArgoCD (GitOps, auto-sync) |
| **Progressive** | Argo Rollouts (canary 10% → 50% → 100%) |
| **Monitoring** | Prometheus + Grafana |

## Quick Start (Docker Compose)

```bash
# Start all services
docker compose up -d

# Test the API
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password"}'

# Use the token
TOKEN="<access_token from response>"
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"echo","payload":{"msg":"hello"}}'

# Check Prometheus: http://localhost:9090
# Check Grafana:    http://localhost:3000 (admin/admin)
```

## Project Structure

```
.
├── services/
│   ├── api-gateway/          # Go reverse proxy
│   ├── auth-service/         # Go JWT auth
│   └── worker-service/       # Python async worker
├── deploy/
│   ├── helm/                 # Helm charts (1 per service)
│   │   ├── api-gateway/
│   │   ├── auth-service/
│   │   └── worker-service/
│   ├── argocd/               # ArgoCD App-of-Apps + Rollouts
│   ├── gitops-config/        # Per-env value overrides (dev/prod)
│   └── docker/               # Prometheus + Grafana config
├── .github/workflows/
│   └── ci.yml                # Build, test, scan, push per service
├── docker-compose.yml        # Local dev environment
└── README.md
```

## CI/CD Pipeline

```
Code Push
  │
  ├── Detect changed services (paths-filter)
  │
  ├── Per service (parallel):
  │   ├── Run tests (Go test / pytest)
  │   ├── Lint (golangci-lint / ruff)
  │   ├── Build Docker image
  │   ├── Trivy security scan
  │   └── Push to GHCR
  │
  └── Update GitOps config repo
        └── ArgoCD auto-syncs to cluster
```

## Kubernetes Deployment

```bash
# 1. Create cluster
kind create cluster --name microservices

# 2. Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 3. Apply App of Apps
kubectl apply -f deploy/argocd/app-of-apps.yaml

# 4. ArgoCD syncs all services automatically
```

## Observability

- **Prometheus** scrapes `/metrics` from all services (RED metrics)
- **Grafana** dashboards: request rate, error rate, duration (P50/P95/P99)
- **Structured logging**: JSON format from all services

## API Endpoints

### Public
- `POST /api/v1/login` — Get JWT tokens
- `POST /api/v1/refresh` — Refresh access token

### Protected (requires Bearer token)
- `POST /api/v1/tasks` — Create a task
- `GET /api/v1/tasks` — List tasks
- `GET /api/v1/tasks/{id}` — Get task by ID

## License

MIT
