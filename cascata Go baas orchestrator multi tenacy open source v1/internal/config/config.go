package config

import (
	"log/slog"
	"os"
)

// Config represents the orchestrator's global configuration.
type Config struct {
	Port         string
	DatabaseURL  string
	DflyURL      string
	Environment  string
	McpEnabled   bool
	LogLevel     string
	SocketDir    string
	VaultAddr       string
	VaultToken      string
	SystemJWTSecret string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	StoragePath     string
}

// Load fetches configurations from environment variables and returns a Config object.
func Load() *Config {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DB_URL"),
		DflyURL:    os.Getenv("DFLY_URL"),
		Environment: getEnv("GO_ENV", "development"),
		McpEnabled:  getEnv("MCP_ENABLED", "true") == "true",
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		SocketDir:   getEnv("SOCKET_DIR", "/tmp/cascata_sockets"),
		VaultAddr:       getEnv("VAULT_ADDR", "http://cascata-vault:8200"),
		VaultToken:      getEnv("VAULT_TOKEN", "cascata_root_token"),
		SystemJWTSecret: getEnv("SYSTEM_JWT_SECRET", "cascata_system_default_secret_32_chars"),
		SMTPHost:        getEnv("SMTP_HOST", "localhost"),
		SMTPPort:        getEnv("SMTP_PORT", "1025"),
		SMTPUser:        getEnv("SMTP_USER", ""),
		SMTPPass:        getEnv("SMTP_PASS", ""),
		SMTPFrom:        getEnv("SMTP_FROM", "cascata@example.com"),
		StoragePath:     getEnv("STORAGE_PATH", "/tmp/cascata_storage"),
	}

	// Fail-fast validation
	if cfg.DatabaseURL == "" {
		slog.Error("FATAL: DB_URL not defined. Cascata requires a metadata database to operate.")
		os.Exit(1)
	}

	return cfg
}

// getEnv returns the environment variable value or a fallback.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
