# Architecture — Microservices + GitOps Pipeline

## Vue d'ensemble

Ce projet illustre le cycle DevOps complet : du code au déploiement automatisé sur Kubernetes via GitOps, avec observabilité intégrée.

## Diagramme d'architecture

```
                         ┌──────────────┐
              Internet ──▶│   Ingress    │
                         │   (Nginx)    │
                         └──────┬───────┘
                                │
                         ┌──────▼───────┐
                         │  API Gateway │ :8080
                         │     (Go)     │
                         │  - Routing   │
                         │  - JWT check │
                         │  - CORS      │
                         │  - Metrics   │
                         └──┬───────┬───┘
                            │       │
                  ┌─────────▼┐    ┌─▼──────────┐
                  │  Auth    │    │  Worker    │
                  │  Service │    │  Service   │
                  │  (Go)    │    │  (Python)  │
                  │  :8081   │    │  :8082     │
                  │          │    │            │
                  │ - Login  │    │ - Task API │
                  │ - Verify │    │ - Queue    │
                  │ - Refresh│    │ - Process  │
                  └──────────┘    └─────┬──────┘
                                        │
                 ┌──────────┐    ┌──────▼──────┐
                 │PostgreSQL│    │    Redis    │
                 │  :5432   │    │   :6379     │
                 └──────────┘    └─────────────┘

       ┌─────────────────────────────────────────┐
       │              Observabilité               │
       │  Prometheus :9090  ←── scrape /metrics   │
       │  Grafana    :3000  ←── dashboards        │
       └─────────────────────────────────────────┘
```

## Services

### API Gateway (Go)

**Rôle** : Point d'entrée unique. Reverse proxy vers les services backend.

| Fonctionnalité | Détail |
|----------------|--------|
| Routing | Préfixe `/api/v1/` → dispatch vers auth ou worker |
| Auth middleware | Appelle `auth-service/verify` pour valider le JWT |
| CORS | Headers Access-Control-Allow-* |
| Metrics | Prometheus `http_requests_total`, `http_request_duration_seconds` |
| Health check | `GET /health`, `GET /ready` |

### Auth Service (Go)

**Rôle** : Gestion des tokens JWT (émission, validation, rafraîchissement).

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/login` | POST | Authentifie et retourne access + refresh tokens |
| `/verify` | GET | Valide un token (appelé par le gateway) |
| `/refresh` | POST | Émet un nouveau access token depuis un refresh token |

**Tokens** :
- Access token : 15 min, signé HS256
- Refresh token : 7 jours
- Claims : subject, role, issuer, iat, exp

### Worker Service (Python/FastAPI)

**Rôle** : Traitement asynchrone de tâches via Redis queue.

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/tasks` | POST | Crée et enqueue une tâche |
| `/tasks` | GET | Liste les tâches (filtrage par status) |
| `/tasks/{id}` | GET | Détail d'une tâche |

**Worker loop** : poll Redis avec `BRPOP`, traite les tâches, met à jour le status.

**Task types** : `echo`, `compute`, `notify` (extensible).

## Déploiement

### Docker Compose (local)

7 services : gateway, auth, worker, redis, postgres, prometheus, grafana.

### Kubernetes (production)

```
ArgoCD (App of Apps)
  ├── api-gateway     (Helm chart)
  ├── auth-service    (Helm chart)
  └── worker-service  (Helm chart)
```

**Helm Charts** incluent :
- Deployment (runAsNonRoot, resource limits, probes)
- Service (ClusterIP)
- HPA (autoscaling CPU)
- PDB (PodDisruptionBudget)
- NetworkPolicy (least-access)
- ServiceMonitor (Prometheus)

### GitOps Flow

```
Developer push
  → GitHub Actions CI (test, build, scan, push image)
    → Update image tag in gitops-config/
      → ArgoCD détecte le changement
        → Auto-sync vers le cluster K8s
```

### Progressive Delivery (Argo Rollouts)

Canary strategy pour l'API Gateway :
1. 10% du trafic → nouvelle version (2 min)
2. 50% du trafic (2 min)
3. 100% rollout

## CI Pipeline

Le workflow CI est **per-service** (ne rebuild que ce qui a changé) :

1. `dorny/paths-filter` détecte les services modifiés
2. Par service (parallèle) :
   - Tests (Go test / pytest)
   - Build image Docker
   - Trivy scan (vulnérabilités)
   - Push vers GHCR (GitHub Container Registry)
3. Mise à jour des tags dans la config GitOps

## Sécurité

- **Images Distroless** (Go) / slim (Python) — surface d'attaque minimale
- **Non-root** dans tous les conteneurs
- **JWT** pour l'authentification inter-services
- **NetworkPolicy** : chaque service n'accepte que le trafic du gateway
- **Trivy scan** sur chaque build
- **Secrets Kubernetes** pour le JWT_SECRET (pas en dur)

## Métriques exposées

Tous les services exposent `/metrics` (Prometheus format) :

| Métrique | Type | Labels |
|----------|------|--------|
| `http_requests_total` | Counter | method, path, status |
| `http_request_duration_seconds` | Histogram | method, path |
| `worker_tasks_processed_total` | Counter | status |
| `worker_task_duration_seconds` | Histogram | — |
