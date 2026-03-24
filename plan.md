# Training & Judge Center — Implementation Plan

**Created**: 2026-02-15

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Backend API | Go (Hexagonal Architecture) | Main API server |
| Judge Workers | Go | Microservicio separado para evaluar submissions |
| Relational DB | PostgreSQL 16 | Users, Groups, Problems, Contests, Submissions, Materials |
| NoSQL DB | MongoDB | Contest standings (collections per contest) |
| Message Queue | Redis Streams | Communication between Backend and Judge Workers |
| Object Storage | MinIO (dev) / S3-compatible (prod) | Source code files, test cases, checkers, validators |
| Frontend | React + Node.js | Web UI (separate repo/project) |
| Orchestration | Kubernetes (Minikube for dev) | Container orchestration |
| Containerization | Docker | Container images |

---

## Architecture Overview

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────▶│   Backend    │────▶│   Judge      │
│   React/Node │     │   Go (Hex)   │     │   Workers    │
└──────────────┘     └──────┬───────┘     └──────▲───────┘
                           │                     │
                    ┌──────▼───────┐     ┌───────┴──────┐
                    │  PostgreSQL  │     │    Redis     │
                    │  (relacional)│     │   Streams    │
                    └──────────────┘     └──────────────┘
                    ┌──────────────┐     ┌──────────────┐
                    │   MongoDB    │     │    MinIO     │
                    │  (standings) │     │  (archivos)  │
                    └──────────────┘     └──────────────┘
```

### Hexagonal Architecture (Backend)

```
              ┌─────────────────────────────────────────┐
              │              DOMAIN (core)               │
              │  Entities, Value Objects, Ports (ifaces) │
              └────────────────┬────────────────────────┘
                               │
              ┌────────────────▼────────────────────────┐
              │            APPLICATION                   │
              │  Use Cases (orchestrate domain logic)    │
              └────────────────┬────────────────────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         ▼                     ▼                     ▼
   ┌───────────┐      ┌──────────────┐      ┌──────────────┐
   │  HTTP API │      │  PostgreSQL  │      │   MongoDB    │
   │ (handler) │      │  (adapter)   │      │  (adapter)   │
   └───────────┘      └──────────────┘      └──────────────┘
   Driving Adapter     Driven Adapter        Driven Adapter
```

**Rule**: `domain/` NEVER imports from `platform/`. Ports (interfaces) live in `domain/`, adapters (implementations) live in `platform/`. Dependency injection happens in `cmd/api/main.go`.

---

## Backend Project Structure

```
training-and-judge-center-backend/
├── cmd/
│   └── api/
│       └── main.go                  # Entry point, DI wiring
├── internal/
│   ├── domain/                      # CORE: Pure business logic
│   │   ├── user/
│   │   │   ├── user.go              # Entity
│   │   │   ├── repository.go        # Port (interface)
│   │   │   └── service.go           # Domain service
│   │   ├── group/
│   │   ├── problem/
│   │   ├── contest/
│   │   ├── submission/
│   │   └── material/
│   ├── application/                 # USE CASES (orchestrate domain)
│   │   ├── user/
│   │   │   └── create_user.go
│   │   ├── group/
│   │   ├── problem/
│   │   ├── contest/
│   │   ├── submission/
│   │   └── material/
│   ├── platform/                    # ADAPTERS: Concrete implementations
│   │   ├── postgres/                # Relational DB adapter
│   │   │   ├── user_repository.go
│   │   │   └── migrations/
│   │   ├── mongo/                   # NoSQL adapter (standings)
│   │   │   └── standing_repository.go
│   │   ├── storage/                 # Object Storage adapter (MinIO/S3)
│   │   │   └── file_storage.go
│   │   ├── queue/                   # Message Queue adapter (Redis)
│   │   │   └── publisher.go
│   │   └── email/                   # Email sender adapter
│   │       └── sender.go
│   └── server/                      # HTTP adapter (driving)
│       ├── router.go
│       ├── middleware/
│       │   ├── auth.go
│       │   └── ratelimit.go
│       └── handler/
│           ├── health_handler.go
│           ├── user_handler.go
│           └── ...
├── pkg/                             # Shared utilities
│   ├── apperror/                    # Standardized app errors
│   └── validator/                   # Common validators
├── deployments/
│   └── k8s/                         # Kubernetes manifests (future)
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml               # Local development
```

---

## Implementation Phases

### Phase 0: Project Skeleton ✅ (current)

- [x] Initialize Go module
- [x] Create hexagonal folder structure
- [x] Basic HTTP server with `/ping` endpoint
- [ ] Dockerfile + docker-compose.yml
- [ ] CI linting (golangci-lint)

### Phase 1: Users Management

The base module that everything depends on.

**Order**:
1. **Create User** (`POST /users`) — Entity, Repository interface, PostgreSQL adapter, handler
2. **Authentication** (JWT) — Login endpoint, auth middleware, token generation/validation
3. **Get User** (`GET /users/me`, `GET /users/{nickname}`) — Profile retrieval
4. **Update User** (`PUT /users`) — Profile updates
5. **Update Email** (`POST /users/email-change/*`) — Email change with verification
6. **Update Password** (`PUT /users/password`) — Password change
7. **Recover Password** (`POST /password/*`) — Recovery flow with verification code
8. **Admin Update User** (`PUT /admin/users/{id}`) — Admin capabilities
9. **Self Deactivate** (`POST /users/deactivation/*`) — Self-deactivation flow
10. **Admin Deactivate** (`POST /admin/users/{id}/deactivate`) — Admin deactivation

**Key decisions**:
- Password hashing: bcrypt
- JWT library: `golang-jwt/jwt`
- DB driver: `jackc/pgx`
- Migration tool: `pressly/goose` or `golang-migrate/migrate`

### Phase 2: Group Management

**Order**:
1. **Create Group** — With Global Group as DB seed/migration
2. **Update Group** — Metadata and policy changes
3. **Delete Group** — With cascade handling
4. **Join Group** — OPEN, REQUEST, INVITE modes
5. **Manage Join Requests** — Approve/reject
6. **Invite to Group** — JWT-based invitations
7. **Manage Members** — Add/remove, role changes

**Key decisions**:
- Global Group: Created via DB migration seed
- Invitation tokens: JWT with 3-day TTL
- Admin: Implicit permissions (no GroupMember record)

### Phase 3: Problem Management

**Order**:
1. **Create Problem** — Minimal creation + ICPC ZIP import
2. **Update Problem** — Metadata + file uploads (test cases, solutions, checker, validator)
3. **Change Visibility** — Publish/unpublish with full validation pipeline
4. **Rejudge Submissions** — When judging components change
5. **View Problem** — Single + list with filters
6. **Problem Statistics** — Acceptance rate, verdict distribution
7. **Delete Problem** — Preserving submissions

**Key decisions**:
- File storage: MinIO (S3-compatible) for dev, real S3/GCS for prod
- Slug: User-provided, immutable, 3-70 chars
- Publication: Full validation (compile checker, run solutions against test cases)

### Phase 4: Contest Management

**Order**:
1. **Create Contest** — With initial problems
2. **Update Contest** — Details, problems, times, lock/unlock
3. **Delete Contest** — Cascade handling
4. **Register to Contest** — Individual + team registration
5. **View Contest** — Details, problem list, listing
6. **View Standings** — ICPC-style with freeze + SSE real-time
7. **View Submissions** — List with visibility rules and freeze

**Key decisions**:
- Status: Computed from time (not stored)
- Standings: MongoDB with atomic operations
- SSE: Server-Sent Events for real-time standings
- Freeze: Optional, configurable minutes before endTime

### Phase 5: Submission Management

**Order**:
1. **Submit Solution** (outside contest) — `POST /api/problems/{slug}/submissions`
2. **Submit Solution** (in contest) — `POST /api/groups/{groupId}/contests/{contestId}/problems/{slug}/submissions`
3. **Queue integration** — Publish to Redis Streams with priority

**Key decisions**:
- File hash: SHA256 for duplicate detection
- Rate limit: 1 second between same user+problem
- Storage paths: `{problemId}/{userId}/general/{submissionId}.{ext}` or `{problemId}/{userId}/{contestId}/{submissionId}.{ext}`

### Phase 6: Material Management

**Order**:
1. **Create Material** — Markdown content with tags
2. **Update Material** — Content and metadata
3. **Change Visibility** — DRAFT/PUBLISHED
4. **Delete Material** — Hard delete

Most independent module — can be done at any point.

### Phase 7: Judge System (Separate Microservice)

**Order**:
1. **Dummy Judge** — Consumer that reads from queue, sleeps, returns random verdict
2. **Docker Executor** — Real execution in isolated containers
3. **Compilation** — g++ (C++20), javac (Java 17), pypy3 (Python 3.10)
4. **Output Comparison** — Default comparison + custom checker
5. **Standing Updates** — MongoDB atomic updates when applicable
6. **Error Handling** — Retry logic for transient errors

**Key decisions**:
- Container runtime: Docker with gVisor (runsc)
- Security: Non-root, read-only filesystem, no network, 1 process
- Queue priority: 4 levels (contest > postcompetition > practice > bulk rejudge)

### Phase 8: Kubernetes Deployment

**Order**:
1. **Dockerize** everything (Backend API + Judge Worker)
2. **Minikube/kind** setup locally
3. **K8s manifests**: Deployments, Services, ConfigMaps, Secrets
4. **StatefulSets** for PostgreSQL, MongoDB, Redis (or use managed services)
5. **Ingress** for Backend API
6. **HPA** for Judge Workers (scale by queue depth)
7. **NetworkPolicies** (isolate judge containers)
8. **PersistentVolumeClaims** for databases

**K8s component mapping**:

| Component | K8s Resource | Scaling |
|-----------|-------------|---------|
| Backend API | Deployment + Service + Ingress | HPA by CPU |
| Judge Workers | Deployment | HPA by queue depth |
| PostgreSQL | StatefulSet + PVC | Single replica (or managed) |
| MongoDB | StatefulSet + PVC | Single replica (or managed) |
| Redis | StatefulSet | Single replica |
| MinIO | StatefulSet + PVC | Single replica |

---

## Key Go Libraries

| Purpose | Library |
|---------|---------|
| HTTP Router | `go-chi/chi` or `gorilla/mux` |
| PostgreSQL | `jackc/pgx/v5` |
| MongoDB | `go.mongodb.org/mongo-driver` |
| Redis | `redis/go-redis/v9` |
| JWT | `golang-jwt/jwt/v5` |
| Validation | `go-playground/validator/v10` |
| Logging | `log/slog` (stdlib) |
| Config | `spf13/viper` or env vars |
| Migrations | `pressly/goose/v3` |
| Testing | `stretchr/testify` |
| MinIO/S3 | `minio/minio-go/v7` |
| Docker | `docker/docker/client` (judge worker) |
| UUID | `google/uuid` |
| Bcrypt | `golang.org/x/crypto/bcrypt` |

---

## Database Schema Summary

### PostgreSQL (Relational)

Tables: `users`, `groups`, `group_members`, `problems`, `contests`, `contest_problems`, `submissions`, `materials`, `group_invitations`, `join_requests`, `password_recovery_requests`, `email_change_requests`, `deactivation_requests`, `deactivation_audit_logs`, `recovery_rate_limits`, `password_update_attempts`

Full schema: see `model/diagram.md`

### MongoDB (NoSQL)

Collections (per contest):
- `contest_{contestId}_standings` — Active standings during contest
- `contest_{contestId}_standings_final` — Frozen snapshot after contest ends

---

## Development Workflow

```
1. Write domain code (entities, ports) — NO external dependencies
2. Write use case (application layer) — Depends only on domain
3. Write adapter (platform layer) — Implements domain ports
4. Write handler (server layer) — Calls use cases
5. Wire everything in main.go — Dependency injection
6. Write tests at each layer
7. Run locally with docker-compose
8. Deploy to k8s when ready
```

---

*This plan is a living document. Update as implementation progresses.*
