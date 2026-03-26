package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/servermind/aixm/internal/config"
	"github.com/servermind/aixm/pkg/models"
	"github.com/google/uuid"
)

// Database 数据库连接封装
type Database struct {
	Pool *pgxpool.Pool
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	connString := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close 关闭数据库连接
func (db *Database) Close() {
	db.Pool.Close()
}

// UserRepository 用户数据访问
type UserRepository struct {
	db *Database
}

func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, tier, created_at, updated_at,
		                   max_servers, max_deployments, max_concurrent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		user.ID, user.Email, user.Password, user.Name, user.Tier,
		user.CreatedAt, user.UpdatedAt,
		user.MaxServers, user.MaxDeployments, user.MaxConcurrent,
	)

	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, name, avatar, tier, created_at, updated_at,
		       max_servers, max_deployments, max_concurrent
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Tier,
		&user.CreatedAt, &user.UpdatedAt,
		&user.MaxServers, &user.MaxDeployments, &user.MaxConcurrent,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, avatar, tier, created_at, updated_at,
		       max_servers, max_deployments, max_concurrent
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.Avatar, &user.Tier,
		&user.CreatedAt, &user.UpdatedAt,
		&user.MaxServers, &user.MaxDeployments, &user.MaxConcurrent,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ServerRepository 服务器数据访问
type ServerRepository struct {
	db *Database
}

func NewServerRepository(db *Database) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Create(ctx context.Context, server *models.Server) error {
	query := `
		INSERT INTO servers (id, user_id, name, host, port, username, credential_id,
		                     os, os_version, kernel, cpu_cores, memory_gb, disk_gb,
		                     status, tags, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		server.ID, server.UserID, server.Name, server.Host, server.Port, server.Username,
		server.CredentialID,
		server.OS, server.OSVersion, server.Kernel, server.CPUCores, server.MemoryGB, server.DiskGB,
		server.Status, server.Tags, server.Notes, server.CreatedAt, server.UpdatedAt,
	)

	return err
}

func (r *ServerRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	query := `
		SELECT id, user_id, name, host, port, username, credential_id,
		       os, os_version, kernel, cpu_cores, memory_gb, disk_gb,
		       status, last_seen, tags, notes, created_at, updated_at
		FROM servers
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.Server
	for rows.Next() {
		server := &models.Server{}
		err := rows.Scan(
			&server.ID, &server.UserID, &server.Name, &server.Host, &server.Port, &server.Username,
			&server.CredentialID,
			&server.OS, &server.OSVersion, &server.Kernel, &server.CPUCores, &server.MemoryGB, &server.DiskGB,
			&server.Status, &server.LastSeen, &server.Tags, &server.Notes, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

func (r *ServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	query := `
		SELECT id, user_id, name, host, port, username, credential_id,
		       os, os_version, kernel, cpu_cores, memory_gb, disk_gb,
		       status, last_seen, tags, notes, created_at, updated_at
		FROM servers
		WHERE id = $1
	`

	server := &models.Server{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&server.ID, &server.UserID, &server.Name, &server.Host, &server.Port, &server.Username,
		&server.CredentialID,
		&server.OS, &server.OSVersion, &server.Kernel, &server.CPUCores, &server.MemoryGB, &server.DiskGB,
		&server.Status, &server.LastSeen, &server.Tags, &server.Notes, &server.CreatedAt, &server.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return server, nil
}

func (r *ServerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastSeen time.Time) error {
	query := `
		UPDATE servers
		SET status = $2, last_seen = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, id, status, lastSeen, time.Now())
	return err
}

// DeploymentRepository 部署记录数据访问
type DeploymentRepository struct {
	db *Database
}

func NewDeploymentRepository(db *Database) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

func (r *DeploymentRepository) Create(ctx context.Context, deployment *models.Deployment) error {
	query := `
		INSERT INTO deployments (id, user_id, server_id, project_id, project_name, repo_url,
		                         branch, commit_hash, status, state, workflow_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		deployment.ID, deployment.UserID, deployment.ServerID, deployment.ProjectID,
		deployment.ProjectName, deployment.RepoURL, deployment.Branch, deployment.CommitHash,
		deployment.Status, deployment.State, deployment.WorkflowID,
		deployment.CreatedAt, deployment.UpdatedAt,
	)

	return err
}

func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, state string) error {
	query := `
		UPDATE deployments
		SET status = $2, state = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, id, status, state, time.Now())
	return err
}

func (r *DeploymentRepository) Complete(ctx context.Context, id uuid.UUID, result *models.DeployResult, errorMessage string) error {
	now := time.Now()
	duration := now.Sub(now) // 实际应该是 started_at 到现在的时长

	query := `
		UPDATE deployments
		SET status = $2, result = $3, error_message = $4, completed_at = $5, duration = $6, updated_at = $7
		WHERE id = $8
	`

	_, err := r.db.Pool.Exec(ctx, query,
		statusFromResult(result), result, errorMessage, &now, int64(duration.Milliseconds()), now, id,
	)

	return err
}

func statusFromResult(result *models.DeployResult) string {
	if result != nil && result.Success {
		return "completed"
	}
	return "failed"
}

func (r *DeploymentRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Deployment, error) {
	query := `
		SELECT id, user_id, server_id, project_id, project_name, repo_url,
		       branch, commit_hash, status, state, started_at, completed_at,
		       duration, result, error_message, workflow_id, created_at, updated_at
		FROM deployments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []*models.Deployment
	for rows.Next() {
		d := &models.Deployment{}
		err := rows.Scan(
			&d.ID, &d.UserID, &d.ServerID, &d.ProjectID, &d.ProjectName, &d.RepoURL,
			&d.Branch, &d.CommitHash, &d.Status, &d.State, &d.StartedAt, &d.CompletedAt,
			&d.Duration, &d.Result, &d.ErrorMessage, &d.WorkflowID, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

func (r *DeploymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Deployment, error) {
	query := `
		SELECT id, user_id, server_id, project_id, project_name, repo_url,
		       branch, commit_hash, status, state, started_at, completed_at,
		       duration, result, error_message, workflow_id, created_at, updated_at
		FROM deployments
		WHERE id = $1
	`

	d := &models.Deployment{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.ServerID, &d.ProjectID, &d.ProjectName, &d.RepoURL,
		&d.Branch, &d.CommitHash, &d.Status, &d.State, &d.StartedAt, &d.CompletedAt,
		&d.Duration, &d.Result, &d.ErrorMessage, &d.WorkflowID, &d.CreatedAt, &d.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return d, nil
}
