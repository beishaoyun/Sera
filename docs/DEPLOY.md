# Sera Platform - Deployment Guide

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Node.js 20+ (for frontend)
- asciinema (optional, for SSH recording)

### One-Command Deployment

```bash
# Development environment
./scripts/deploy.sh dev

# Production environment
./scripts/deploy.sh prod

# Stop all services
./scripts/deploy.sh stop

# Check status
./scripts/deploy.sh status
```

## Manual Deployment

### Step 1: Start Infrastructure Services

```bash
docker compose up -d postgres redis milvus-standalone neo4j clickhouse temporal nats vault etcd minio
```

Wait for all services to be healthy (about 2-3 minutes).

### Step 2: Run Database Migrations

```bash
go run cmd/migrator/main.go
```

### Step 3: Configure Environment

```bash
cp .env.example .env
# Edit .env and set your API keys
```

### Step 4: Start API Server

```bash
# Development mode (with hot reload)
go run cmd/server/main.go

# Or build and run
go build -o bin/api ./cmd/server
./bin/api
```

### Step 5: Start Worker

```bash
# In a separate terminal
go run cmd/worker/main.go
```

### Step 6: Build and Start Frontend

```bash
cd frontend
npm install
npm run dev
```

## Service Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| API Server | http://localhost:8080 | - |
| Frontend | http://localhost:3000 | - |
| PostgreSQL | localhost:5432 | servermind / servermind_dev |
| Redis | localhost:6379 | password: servermind_dev |
| Milvus | localhost:19530 | root / Milvus |
| Neo4j | localhost:7474 | neo4j / servermind_dev |
| ClickHouse | localhost:8123 | servermind / servermind_dev |
| Temporal UI | localhost:8233 | - |
| NATS | localhost:4222 | - |
| Vault | localhost:8200 | token: servermind_dev_token |
| MinIO Console | localhost:9001 | minioadmin / minioadmin |

## API Endpoints

### Authentication

```bash
# Register
POST /api/v1/auth/register
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "User Name"
}

# Login
POST /api/v1/auth/login
{
  "email": "user@example.com",
  "password": "securepassword"
}

# Refresh Token
POST /api/v1/auth/refresh
{
  "refresh_token": "your-refresh-token"
}
```

### Servers

```bash
# Create Server
POST /api/v1/servers
Authorization: Bearer <token>
{
  "name": "Production Server",
  "host": "192.168.1.100",
  "port": 22,
  "username": "root",
  "password": "server-password"  # Or use ssh_key
}

# List Servers
GET /api/v1/servers
Authorization: Bearer <token>

# Get Server
GET /api/v1/servers/:id
Authorization: Bearer <token>

# Test Connection
POST /api/v1/servers/:id/test
Authorization: Bearer <token>
```

### Deployments

```bash
# Create Deployment
POST /api/v1/deployments
Authorization: Bearer <token>
{
  "repo_url": "https://github.com/user/repo",
  "server_id": "uuid-here",
  "branch": "main"
}

# Get Deployment Status
GET /api/v1/deployments/:id
Authorization: Bearer <token>

# Get Deployment Progress
GET /api/v1/deployments/:id/progress
Authorization: Bearer <token>

# Cancel Deployment
POST /api/v1/deployments/:id/cancel
Authorization: Bearer <token>
```

### WebSocket

```bash
# Connect to deployment progress
ws://localhost:8080/ws/deployments/:id
```

## Environment Variables

See `.env.example` for all available options.

Key variables:
- `LLM_API_KEY`: Your Anthropic/OpenAI API key
- `GITHUB_TOKEN`: GitHub personal access token (optional, for private repos)
- `JWT_SECRET`: Change this in production!

## Production Deployment

### Docker Compose (Recommended)

```bash
# Build images
docker compose build

# Start all services
docker compose up -d

# View logs
docker compose logs -f api worker

# Stop all services
docker compose down
```

### Kubernetes (Coming Soon)

Helm charts and Kubernetes manifests will be available in a future release.

## Troubleshooting

### Database Connection Failed

```bash
# Check if PostgreSQL is running
docker compose ps postgres

# View logs
docker compose logs postgres
```

### Milvus Connection Failed

```bash
# Milvus takes longer to start, wait 2-3 minutes
docker compose logs milvus-standalone
```

### LLM API Errors

```bash
# Check your API key
echo $LLM_API_KEY

# Test connection
curl -H "Authorization: Bearer $LLM_API_KEY" https://api.anthropic.com/v1/messages
```

### WebSocket Connection Failed

```bash
# Check if API server is running
curl http://localhost:8080/health

# Check CORS settings in .env
# CORS_ALLOWED_ORIGINS=http://localhost:3000
```

## Backup and Restore

### Backup Database

```bash
docker compose exec postgres pg_dump -U servermind servermind > backup.sql
```

### Restore Database

```bash
docker compose exec -T postgres psql -U servermind servermind < backup.sql
```

### Backup Milvus

Follow Milvus documentation for backup/restore procedures.

## Monitoring

### Health Checks

```bash
# API health
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready
```

### Temporal UI

Open http://localhost:8233 to view workflow executions.

### NATS Monitoring

```bash
# Server info
curl http://localhost:8222/varz

# Connection info
curl http://localhost:8222/connz
```

## Security Considerations

1. **Change all default passwords** in `.env` before deploying to production
2. **Use HTTPS** in production (configure in reverse proxy)
3. **Rotate JWT_SECRET** for each deployment
4. **Enable Vault** for SSH key management
5. **Configure firewall** rules for all services
6. **Enable audit logging** for compliance

## Performance Tuning

### Database

```env
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=25
```

### Redis

```env
REDIS_MAX_CONNECTIONS=100
```

### SSH

```env
SSH_MAX_CONNECTIONS=200
SSH_CONNECT_TIMEOUT=15s
```

## Updates

```bash
# Pull latest changes
git pull

# Stop services
./scripts/deploy.sh stop

# Start fresh
./scripts/deploy.sh prod
```

## Support

- GitHub Issues: https://github.com/beishaoyun/Sera/issues
- Documentation: https://github.com/beishaoyun/Sera/docs/
