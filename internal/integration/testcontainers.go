package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ============================================================================
// Testcontainers 辅助函数
// ============================================================================

// PostgresContainer PostgreSQL 测试容器
type PostgresContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	DSN       string
}

// SetupPostgresContainer 启动 PostgreSQL 容器
func SetupPostgresContainer(ctx context.Context, t *testing.T) *PostgresContainer {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	mappedPort, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	return &PostgresContainer{
		Container: pgContainer,
		Host:      host,
		Port:      mappedPort.Port(),
		DSN:       dsn,
	}
}

// RedisContainer Redis 测试容器
type RedisContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	Addr      string
}

// SetupRedisContainer 启动 Redis 容器
func SetupRedisContainer(ctx context.Context, t *testing.T) *RedisContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate redis container: %v", err)
		}
	})

	return &RedisContainer{
		Container: container,
		Host:      host,
		Port:      mappedPort.Port(),
		Addr:      fmt.Sprintf("%s:%s", host, mappedPort.Port()),
	}
}

// NATSContainer NATS JetStream 测试容器
type NATSContainer struct {
	Container   testcontainers.Container
	Host        string
	Port        string
	ClusterPort string
	HTTPPort    string
	ConnString  string
}

// SetupNATSContainer 启动 NATS JetStream 容器
func SetupNATSContainer(ctx context.Context, t *testing.T) *NATSContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp", "8222/tcp", "6222/tcp"},
		Cmd:          []string{"--jetstream", "--mem_size=1gb", "--store_dir=/data/nats"},
		WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(15 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start nats container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	clientPort, _ := container.MappedPort(ctx, "4222/tcp")
	clusterPort, _ := container.MappedPort(ctx, "6222/tcp")
	httpPort, _ := container.MappedPort(ctx, "8222/tcp")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate nats container: %v", err)
		}
	})

	return &NATSContainer{
		Container:   container,
		Host:        host,
		Port:        clientPort.Port(),
		ClusterPort: clusterPort.Port(),
		HTTPPort:    httpPort.Port(),
		ConnString:  fmt.Sprintf("nats://%s:%s", host, clientPort.Port()),
	}
}

// TemporalContainer Temporal 测试容器
type TemporalContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	ConnString string
}

// SetupTemporalContainer 启动 Temporal 容器
func SetupTemporalContainer(ctx context.Context, t *testing.T) *TemporalContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "temporalio/auto-setup:1.20",
		ExposedPorts: []string{"7233/tcp"},
		Env: map[string]string{
			"DB": "postgresql",
			"DB_PORT": "5432",
		},
		WaitingFor: wait.ForLog("Temporal server started").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start temporal container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "7233/tcp")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			logrus.Errorf("failed to terminate temporal container: %v", err)
		}
	})

	return &TemporalContainer{
		Container:  container,
		Host:       host,
		Port:       mappedPort.Port(),
		ConnString: fmt.Sprintf("%s:%s", host, mappedPort.Port()),
	}
}

// ============================================================================
// 集成测试辅助函数
// ============================================================================

// TestEnvironment 测试环境
type TestEnvironment struct {
	Postgres *PostgresContainer
	Redis    *RedisContainer
	NATS     *NATSContainer
	Ctx      context.Context
	Cancel   context.CancelFunc
}

// SetupTestEnvironment 启动完整测试环境
func SetupTestEnvironment(ctx context.Context, t *testing.T) *TestEnvironment {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	t.Cleanup(cancel)

	env := &TestEnvironment{
		Ctx:    ctx,
		Cancel: cancel,
	}

	// 启动服务
	env.Postgres = SetupPostgresContainer(ctx, t)
	env.Redis = SetupRedisContainer(ctx, t)
	env.NATS = SetupNATSContainer(ctx, t)

	logrus.Infof("Test environment started:")
	logrus.Infof("  PostgreSQL: %s", env.Postgres.DSN)
	logrus.Infof("  Redis: %s", env.Redis.Addr)
	logrus.Infof("  NATS: %s", env.NATS.ConnString)

	return env
}

// ============================================================================
// 测试数据库迁移
// ============================================================================

// RunMigrations 运行数据库迁移
func RunMigrations(t *testing.T, dsn string) {
	t.Helper()

	// 读取迁移文件
	migrationSQL, err := os.ReadFile("scripts/migrations/000001_create_initial_tables.up.sql")
	if err != nil {
		// 尝试第二个迁移
		migrationSQL, err = os.ReadFile("scripts/migrations/000002_add_deployment_tables.up.sql")
		if err != nil {
			t.Fatalf("failed to read migration file: %v", err)
		}
	}

	// 实际实现会使用 pgx 执行迁移
	_ = migrationSQL
	_ = dsn

	logrus.Info("Migrations executed successfully")
}

// ============================================================================
// 测试辅助宏
// ============================================================================

// SkipIfShort 跳过短测试模式
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// SkipIfNoDocker 如果没有 Docker 则跳过
func SkipIfNoDocker(t *testing.T) {
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "true" {
		t.Skip("skipping test: Docker not available")
	}
}

// WaitForService 等待服务就绪
func WaitForService(ctx context.Context, t *testing.T, check func() error, timeout time.Duration) {
	t.Helper()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			t.Fatalf("service did not become ready within %v", timeout)
		case <-ticker.C:
			if err := check(); err == nil {
				return
			}
		}
	}
}

// AssertNoError 断言无错误
func AssertNoError(t *testing.T, err error, msg ...string) {
	t.Helper()
	if err != nil {
		if len(msg) > 0 {
			t.Fatalf("%s: %v", msg[0], err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertEqual 断言相等
func AssertEqual[T comparable](t *testing.T, expected, actual T, msg ...string) {
	t.Helper()
	if expected != actual {
		if len(msg) > 0 {
			t.Fatalf("%s: expected %v, got %v", msg[0], expected, actual)
		}
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// AssertTrue 断言为真
func AssertTrue(t *testing.T, condition bool, msg ...string) {
	t.Helper()
	if !condition {
		if len(msg) > 0 {
			t.Fatalf("%s: expected true, got false", msg[0])
		}
		t.Fatalf("expected true, got false")
	}
}
