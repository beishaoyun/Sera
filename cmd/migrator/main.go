package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// 数据库迁移脚本
func main() {
	logrus.Info("Starting database migration...")

	// 从环境变量获取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "servermind")
	dbPassword := getEnv("DB_PASSWORD", "servermind_dev")
	dbName := getEnv("DB_NAME", "servermind")

	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logrus.Fatalf("Failed to ping database: %v", err)
	}

	logrus.Info("Database connection established")

	// 执行迁移
	migrations := []string{
		createUsersTable,
		createServersTable,
		createProjectsTable,
		createDeploymentsTable,
		createKnowledgeCasesTable,
		createAuditLogsTable,
		createIndexes,
	}

	for i, migration := range migrations {
		logrus.Infof("Running migration %d...", i+1)
		if _, err := pool.Exec(ctx, migration); err != nil {
			logrus.Fatalf("Migration %d failed: %v", i+1, err)
		}
	}

	logrus.Info("Database migration completed successfully!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar VARCHAR(512),
    tier VARCHAR(50) NOT NULL DEFAULT 'free',
    max_servers INTEGER NOT NULL DEFAULT 3,
    max_deployments INTEGER NOT NULL DEFAULT 10,
    max_concurrent INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);
`

const createServersTable = `
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    username VARCHAR(255) NOT NULL,
    credential_id VARCHAR(255) NOT NULL,
    os VARCHAR(100),
    os_version VARCHAR(100),
    kernel VARCHAR(100),
    cpu_cores INTEGER,
    memory_gb INTEGER,
    disk_gb INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    last_seen TIMESTAMP WITH TIME ZONE,
    tags TEXT[] DEFAULT '{}',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_server_name UNIQUE (user_id, name)
);
`

const createProjectsTable = `
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    repo_url VARCHAR(512) NOT NULL,
    repo_owner VARCHAR(255) NOT NULL,
    repo_name VARCHAR(255) NOT NULL,
    default_branch VARCHAR(255) DEFAULT 'main',
    language VARCHAR(100),
    framework VARCHAR(100),
    deploy_type VARCHAR(50) NOT NULL DEFAULT 'docker',
    has_dockerfile BOOLEAN DEFAULT FALSE,
    has_docker_compose BOOLEAN DEFAULT FALSE,
    min_cpu_cores DOUBLE PRECISION DEFAULT 0.5,
    min_memory_mb INTEGER DEFAULT 512,
    min_disk_gb INTEGER DEFAULT 1,
    exposed_ports INTEGER[] DEFAULT '{}',
    env_vars JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`

const createDeploymentsTable = `
CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    project_name VARCHAR(255) NOT NULL,
    repo_url VARCHAR(512) NOT NULL,
    branch VARCHAR(255) DEFAULT 'main',
    commit_hash VARCHAR(64),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    state VARCHAR(100),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration BIGINT, -- milliseconds
    result JSONB,
    error_message TEXT,
    workflow_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deployments_user_id ON deployments(user_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at DESC);
`

const createKnowledgeCasesTable = `
CREATE TABLE IF NOT EXISTS knowledge_cases (
    id UUID PRIMARY KEY,
    case_type VARCHAR(50) NOT NULL,
    os VARCHAR(100) NOT NULL,
    os_version VARCHAR(100) NOT NULL,
    tech_stack VARCHAR(255) NOT NULL,
    runtime VARCHAR(255),
    error_type VARCHAR(255),
    error_log TEXT,
    root_cause TEXT,
    solution TEXT,
    commands TEXT[] DEFAULT '{}',
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    quality_score DOUBLE PRECISION DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_case_type ON knowledge_cases(case_type);
CREATE INDEX IF NOT EXISTS idx_knowledge_os ON knowledge_cases(os, os_version);
CREATE INDEX IF NOT EXISTS idx_knowledge_tech_stack ON knowledge_cases(tech_stack);
`

const createAuditLogsTable = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
`

const createIndexes = `
-- 添加 updated_at 触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_cases_updated_at BEFORE UPDATE ON knowledge_cases
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
`
