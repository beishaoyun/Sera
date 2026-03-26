package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	SSH      SSHConfig
	Vault    VaultConfig
	Milvus   MilvusConfig
	Temporal TemporalConfig
}

type ServerConfig struct {
	Port         string
	Mode         string // debug, release, test
	EnableSwagger bool
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret     string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
}

type SSHConfig struct {
	ConnectTimeout   time.Duration
	CommandTimeout   time.Duration
	MaxConnections   int
	KeepAlive        time.Duration
	SessionRecording bool
}

type VaultConfig struct {
	Address string
	Token   string
}

type MilvusConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	DBName     string
}

type TemporalConfig struct {
	HostPort string
	Namespace string
}

// Load 加载配置
func Load() (*Config, error) {
	// 从环境变量加载
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	config := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("SERVER_PORT"),
			Mode:         viper.GetString("SERVER_MODE"),
			EnableSwagger: viper.GetBool("ENABLE_SWAGGER"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("DB_HOST"),
			Port:            viper.GetString("DB_PORT"),
			User:            viper.GetString("DB_USER"),
			Password:        viper.GetString("DB_PASSWORD"),
			DBName:          viper.GetString("DB_NAME"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Auth: AuthConfig{
			JWTSecret:     viper.GetString("JWT_SECRET"),
			TokenExpiry:   viper.GetDuration("JWT_TOKEN_EXPIRY"),
			RefreshExpiry: viper.GetDuration("JWT_REFRESH_EXPIRY"),
		},
		SSH: SSHConfig{
			ConnectTimeout:   viper.GetDuration("SSH_CONNECT_TIMEOUT"),
			CommandTimeout:   viper.GetDuration("SSH_COMMAND_TIMEOUT"),
			MaxConnections:   viper.GetInt("SSH_MAX_CONNECTIONS"),
			KeepAlive:        viper.GetDuration("SSH_KEEP_ALIVE"),
			SessionRecording: viper.GetBool("SSH_SESSION_RECORDING"),
		},
		Vault: VaultConfig{
			Address: viper.GetString("VAULT_ADDRESS"),
			Token:   viper.GetString("VAULT_TOKEN"),
		},
		Milvus: MilvusConfig{
			Host:     viper.GetString("MILVUS_HOST"),
			Port:     viper.GetString("MILVUS_PORT"),
			Username: viper.GetString("MILVUS_USERNAME"),
			Password: viper.GetString("MILVUS_PASSWORD"),
			DBName:   viper.GetString("MILVUS_DB_NAME"),
		},
		Temporal: TemporalConfig{
			HostPort:  viper.GetString("TEMPORAL_HOST_PORT"),
			Namespace: viper.GetString("TEMPORAL_NAMESPACE"),
		},
	}

	return config, nil
}

func setDefaults() {
	// Server
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_MODE", "debug")
	viper.SetDefault("ENABLE_SWAGGER", true)

	// Database
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "servermind")
	viper.SetDefault("DB_PASSWORD", "servermind_dev")
	viper.SetDefault("DB_NAME", "servermind")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// Redis
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", 0)

	// Auth
	viper.SetDefault("JWT_SECRET", getEnvOrDefault("JWT_SECRET", "servermind_dev_secret_change_in_prod"))
	viper.SetDefault("JWT_TOKEN_EXPIRY", 24*time.Hour)
	viper.SetDefault("JWT_REFRESH_EXPIRY", 7*24*time.Hour)

	// SSH
	viper.SetDefault("SSH_CONNECT_TIMEOUT", 30*time.Second)
	viper.SetDefault("SSH_COMMAND_TIMEOUT", 300*time.Second)
	viper.SetDefault("SSH_MAX_CONNECTIONS", 100)
	viper.SetDefault("SSH_KEEP_ALIVE", 60*time.Second)
	viper.SetDefault("SSH_SESSION_RECORDING", true)

	// Milvus
	viper.SetDefault("MILVUS_HOST", "localhost")
	viper.SetDefault("MILVUS_PORT", "19530")

	// Temporal
	viper.SetDefault("TEMPORAL_HOST_PORT", "localhost:7233")
	viper.SetDefault("TEMPORAL_NAMESPACE", "default")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DSN 返回数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// Address 返回 Redis 地址
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Address 返回 Milvus 地址
func (c *MilvusConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
