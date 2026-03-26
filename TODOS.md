# TODOS

## Phase 1: Core Infrastructure (COMPLETED 2026-03-26)

### Database & Migrations
- [x] **Priority:** P0
  Create database migration for 6 new tables (deployment_state_histories, ssh_recordings, cloud_credentials, deployment_templates, natps_dlq_logs, scaling_policies)

- [x] **Priority:** P0
  Add Deployment table extensions (current_state, state_history, idempotency_key)

### Multi-Agent System
- [x] **Priority:** P0
  Implement RequirementParser Agent with prompt templates

- [x] **Priority:** P0
  Implement CodeAnalyzer Agent with code scanning logic

- [x] **Priority:** P0
  Implement DeploymentExecutor Agent with SSH command execution and sandbox

- [x] **Priority:** P1
  Implement Troubleshooter Agent with RAG-based diagnosis

### Command Security Sandbox
- [x] **Priority:** P0
  Implement command allowlist (20+ commands)

- [x] **Priority:** P0
  Implement dangerous pattern detection (15+ patterns)

- [x] **Priority:** P0
  Integrate sandbox with SSH client

### NATS JetStream
- [x] **Priority:** P0
  Implement NATS JetStream client

- [x] **Priority:** P0
  Implement Dead Letter Queue (DLQ)

- [x] **Priority:** P1
  Add NATS service to docker-compose.yml

### Temporal Workflow
- [x] **Priority:** P0
  Implement deployment workflow definitions

- [x] **Priority:** P0
  Implement state machine with validated transitions

- [x] **Priority:** P0
  Implement Activity handlers

### Error Handling
- [x] **Priority:** P0
  Implement unified error wrapper (AppError)

- [x] **Priority:** P0
  Define error codes and severity levels

### SSH Integration
- [x] **Priority:** P0
  Integrate command sandbox with SSH client

- [x] **Priority:** P0
  Implement SSH connection factory

## Phase 2: Remaining Work

### Backend

#### LLM Integration
- [ ] **Priority:** P0
  Implement LLM client (Claude/OpenAI)

- [ ] **Priority:** P0
  Complete RequirementParser LLM prompts and schema

- [ ] **Priority:** P0
  Complete CodeAnalyzer LLM prompts and schema

- [ ] **Priority:** P0
  Complete Troubleshooter LLM prompts and schema

#### GitHub API
- [ ] **Priority:** P0
  Implement GitHub API client for repo analysis

- [ ] **Priority:** P0
  Implement rate limiting and error handling

#### RAG Knowledge Base
- [ ] **Priority:** P1
  Set up Milvus vector database integration

- [ ] **Priority:** P1
  Implement knowledge retrieval and storage

#### Vault Integration
- [ ] **Priority:** P1
  Implement HashiCorp Vault integration for SSH keys

- [ ] **Priority:** P1
  Dynamic certificate generation

#### Real-time Features
- [ ] **Priority:** P1
  WebSocket streaming for deployment progress

- [ ] **Priority:** P1
  asciinema recording integration

- [ ] **Priority:** P1
  MinIO storage for recordings

### Frontend

#### Authentication
- [ ] **Priority:** P0
  Complete login/register pages with form validation

- [ ] **Priority:** P0
  Implement JWT token storage and refresh logic

#### Dashboard
- [ ] **Priority:** P0
  Server management dashboard (add, remove, status)

- [ ] **Priority:** P0
  Deployment monitoring UI with real-time status

#### AI Chat Interface
- [ ] **Priority:** P1
  Chat interface for AI troubleshooting sessions

### Infrastructure

#### Testing
- [ ] **Priority:** P0
  Write integration tests with Testcontainers

- [ ] **Priority:** P0
  Write end-to-end deployment tests

- [ ] **Priority:** P1
  Set up frontend testing with Vitest

#### Documentation
- [ ] **Priority:** P1
  Complete API documentation with OpenAPI/Swagger

- [ ] **Priority:** P1
  Write production deployment guide

---

## Previously Completed

- [x] **Priority:** P0
  Set up Git repository with initial commit

- [x] **Priority:** P0
  Create project infrastructure (VERSION, CHANGELOG, .gitignore, Makefile)

- [x] **Priority:** P0
  Set up GitHub Actions CI/CD pipelines (Go tests, frontend tests, Docker build)

- [x] **Priority:** P0
  Write unit tests for auth package (JWT generation, verification, refresh)

- [x] **Priority:** P0
  Write unit tests for config package (loading, defaults, DSN generation)

- [x] **Priority:** P0
  Write unit tests for models package (User, Server, Deployment, Project)

- [x] **Priority:** P1
  Create database migration scripts (users, servers, projects, deployments, knowledge_cases, audit_logs)
