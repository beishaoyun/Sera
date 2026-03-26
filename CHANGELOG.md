# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0.0] - 2026-03-26

### Added - Phase 1 Core Infrastructure

#### Multi-Agent System
- Complete Agent interfaces in `internal/agent/agents.go`
  - RequirementParser with GitHub API integration
  - CodeAnalyzer with LLM-based code analysis
  - DeploymentExecutor with SSH and command sandbox
  - Troubleshooter with RAG knowledge base support
- Agent helper functions in `internal/agent/helpers.go`
  - GitHub URL parsing
  - Repository metadata fetching
  - File tree analysis
  - LLM prompt builders for all Agents

#### Command Security Sandbox
- New package `internal/sandbox/command_sandbox.go`
  - Command allowlist (20+ commands, 100+ subcommands)
  - Dangerous pattern detection (15+ patterns)
  - Severity levels (critical, high, medium, low)
  - Special command checks for rm, curl/wget, chmod

#### NATS JetStream Integration
- New package `internal/nats/client.go`
  - JetStream stream management
  - Consumer management with AckPolicy
  - Dead Letter Queue (DLQ) support
  - Event bus abstraction
  - Retry with exponential backoff

#### Temporal Workflow Engine
- New package `internal/temporal/`
  - `workflow.go` - Deployment workflow definitions
  - `state.go` - State machine with 10 states and validated transitions
  - `activity.go` - Activity implementations for all Agents

#### SSH Client with Sandbox Integration
- Updated `internal/ssh/client.go`
  - Command validation before SSH execution
  - Connection pooling with sandbox
- New `internal/ssh/factory.go`
  - SSHClientFactory for simple configurations
  - DatabaseSSHClientFactory for database-backed configurations

#### Error Handling
- New package `pkg/errors/errors.go`
  - Unified AppError wrapper
  - 20+ error codes across categories
  - Severity levels (debug, info, warning, error, critical)
  - Fluent API for error construction
  - Helper functions (IsRetryable, IsNotFound, CommandBlocked, etc.)

#### LLM Client
- Updated `internal/llm/client.go`
  - Anthropic Claude API support
  - OpenAI API support
  - JSON schema-constrained generation
  - Automatic retry with exponential backoff

#### GitHub API Client
- New package `internal/github/client.go`
  - Repository metadata fetching
  - File tree retrieval (Git Trees API)
  - README content fetching
  - Rate limit monitoring
  - SSH and HTTPS URL parsing

#### Integration Testing
- New package `internal/integration/`
  - Testcontainers helpers for PostgreSQL, Redis, NATS, Temporal
  - Test environment setup
  - Assertion helpers

#### Database Migrations
- New migration `000002_add_deployment_tables.up.sql`
  - `deployment_state_histories` - State change tracking
  - `ssh_recordings` - asciinema session recordings
  - `cloud_credentials` - Vault-integrated credentials
  - `deployment_templates` - Template marketplace
  - `natps_dlq_logs` - NATS Dead Letter Queue
  - `scaling_policies` - Auto-scaling policies
- Deployment table extensions (current_state, state_history, idempotency_key)

#### Docker Compose
- Added NATS JetStream service (--jetstream, --mem_size=1gb)
- Added HashiCorp Vault service (dev mode)

#### Documentation
- New `docs/IMPLEMENTATION_PHASE1.md` with full architecture docs
- Updated `PROJECT_STATUS.md` with Phase 1 completion status
- Updated `TODOS.md` with Phase 1 completed items

### Changed
- Renamed project from "ServerMind" to "Sera"
- SSH client now integrates command sandbox for security

### Technical Details
- State machine with 10 states: PENDING, ENV_PREPARING, CODE_FETCHING,
  DEPENDENCY_INSTALLING, BUILDING, CONFIGURING, DEPLOYING, VERIFYING,
  COMPLETED, FAILED, ROLLING_BACK
- Command sandbox patterns detect: rm -rf /, mkfs, curl|bash, chmod 777, etc.
- NATS DLQ automatically retries failed messages up to 3 times

## [0.1.0.0] - 2026-03-26

### Added
- Initial ServerMind platform release
- Multi-Agent system (RequirementParser, CodeAnalyzer, DeploymentExecutor, Troubleshooter)
- User management with JWT authentication
- Server management with SSH connection pooling
- RAG knowledge base integration with Milvus
- Temporal workflow engine for deployments
- Next.js 14 frontend with TypeScript
- PostgreSQL database layer with pgx
- Redis caching support
- GitHub Actions CI/CD workflows
- Docker Compose for local development
- Unit tests for auth, config, and models packages
