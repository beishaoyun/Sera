package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_MODE", "test")
	os.Setenv("DB_HOST", "test-host")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "test-user")
	os.Setenv("DB_PASSWORD", "test-password")
	os.Setenv("DB_NAME", "test-db")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("REDIS_HOST", "test-redis")
	os.Setenv("REDIS_PORT", "6380")

	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_MODE")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Expected server port '9090', got '%s'", cfg.Server.Port)
	}
	if cfg.Server.Mode != "test" {
		t.Errorf("Expected server mode 'test', got '%s'", cfg.Server.Mode)
	}
	if cfg.Database.Host != "test-host" {
		t.Errorf("Expected database host 'test-host', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != "5433" {
		t.Errorf("Expected database port '5433', got '%s'", cfg.Database.Port)
	}
	if cfg.Auth.JWTSecret != "test-jwt-secret" {
		t.Errorf("Expected JWT secret 'test-jwt-secret', got '%s'", cfg.Auth.JWTSecret)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Clear environment variables
	envVars := []string{
		"SERVER_PORT", "SERVER_MODE", "ENABLE_SWAGGER",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"JWT_SECRET", "JWT_TOKEN_EXPIRY", "JWT_REFRESH_EXPIRY",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"SSH_CONNECT_TIMEOUT", "SSH_COMMAND_TIMEOUT", "SSH_MAX_CONNECTIONS",
		"MILVUS_HOST", "MILVUS_PORT", "TEMPORAL_HOST_PORT",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("Expected default server port '8080', got '%s'", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Expected default server mode 'debug', got '%s'", cfg.Server.Mode)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default database host 'localhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != "5432" {
		t.Errorf("Expected default database port '5432', got '%s'", cfg.Database.Port)
	}
}

func TestDatabaseConfigDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
	}

	dsn := cfg.DSN()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable TimeZone=UTC"

	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestRedisConfigAddress(t *testing.T) {
	cfg := RedisConfig{
		Host: "localhost",
		Port: "6379",
	}

	address := cfg.Address()
	expected := "localhost:6379"

	if address != expected {
		t.Errorf("Expected address '%s', got '%s'", expected, address)
	}
}

func TestMilvusConfigAddress(t *testing.T) {
	cfg := MilvusConfig{
		Host: "localhost",
		Port: "19530",
	}

	address := cfg.Address()
	expected := "localhost:19530"

	if address != expected {
		t.Errorf("Expected address '%s', got '%s'", expected, address)
	}
}

func TestSSHConfigDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.SSH.MaxConnections != 100 {
		t.Errorf("Expected default SSH max connections 100, got %d", cfg.SSH.MaxConnections)
	}
	if cfg.SSH.SessionRecording != true {
		t.Errorf("Expected default SSH session recording to be true")
	}
}

func TestTemporalConfigDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Temporal.HostPort != "localhost:7233" {
		t.Errorf("Expected default Temporal host 'localhost:7233', got '%s'", cfg.Temporal.HostPort)
	}
	if cfg.Temporal.Namespace != "default" {
		t.Errorf("Expected default Temporal namespace 'default', got '%s'", cfg.Temporal.Namespace)
	}
}
