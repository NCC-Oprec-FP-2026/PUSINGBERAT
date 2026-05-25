// Package config reads application configuration from environment variables.
// All configuration is loaded once at startup and propagated via dependency
// injection. Never use global variables; always pass *Config through constructors.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds every runtime setting for the PUSINGBERAT backend.
// Fields map 1-to-1 with the environment variables documented in section 4.3.
type Config struct {
	// Database (PostgreSQL via pgx)
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// HTTP server
	ServerPort int

	// External integrations
	DiscordWebhookURL string

	// Rule engine
	RulesDir string

	// Observability
	LogLevel string
}

// DSN returns a pgx-compatible connection string built from the DB fields.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword,
	)
}

// ServerAddress returns the ":PORT" string expected by gin.Engine.Run().
func (c *Config) ServerAddress() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

// Load reads all required environment variables and returns a populated Config.
// An error is returned if any required field is missing or unparseable.
func Load() (*Config, error) {
	loader := &envLoader{}

	cfg := &Config{
		// Required — application cannot start without these.
		DBHost:     loader.require("DB_HOST"),
		DBName:     loader.require("DB_NAME"),
		DBUser:     loader.require("DB_USER"),
		DBPassword: loader.require("DB_PASSWORD"),

		// Required integers.
		DBPort:     loader.requireInt("DB_PORT"),
		ServerPort: loader.requireInt("SERVER_PORT"),

		// Optional with sensible defaults.
		DiscordWebhookURL: loader.optional("DISCORD_WEBHOOK_URL", ""),
		RulesDir:          loader.optional("RULES_DIR", "./rules"),
		LogLevel:          loader.optional("LOG_LEVEL", "info"),
	}

	cfg.DBPort = loader.defaultInt("DB_PORT", cfg.DBPort, 5432)
	cfg.ServerPort = loader.defaultInt("SERVER_PORT", cfg.ServerPort, 8080)

	if len(loader.missing) > 0 {
		return nil, fmt.Errorf(
			"config: missing or invalid required environment variables: %s",
			strings.Join(loader.missing, ", "),
		)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

type envLoader struct {
	missing []string
}

func (l *envLoader) require(key string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		l.missing = append(l.missing, key)
	}
	return v
}

func (l *envLoader) optional(key, fallback string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func (l *envLoader) requireInt(key string) int {
	raw := l.require(key)
	if raw == "" {
		return 0
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		l.missing = append(l.missing, key+" (must be an integer, got: "+raw+")")
		return 0
	}
	return v
}

func (l *envLoader) defaultInt(key string, current, fallback int) int {
	if current != 0 || os.Getenv(key) != "" {
		return current
	}

	l.missing = filterOut(l.missing, key)
	return fallback
}

// validate runs semantic checks on the loaded config values.
func (c *Config) validate() error {
	if c.DBPort < 1 || c.DBPort > 65535 {
		return fmt.Errorf("config: DB_PORT %d is out of valid range (1-65535)", c.DBPort)
	}
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("config: SERVER_PORT %d is out of valid range (1-65535)", c.ServerPort)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.LogLevel)] {
		return errors.New("config: LOG_LEVEL must be one of: debug, info, warn, error")
	}

	return nil
}

// filterOut removes a specific string from a slice (used to withdraw a
// "missing" entry when we decide to fall back to a default instead).
func filterOut(ss []string, target string) []string {
	out := ss[:0]
	for _, s := range ss {
		if s != target {
			out = append(out, s)
		}
	}
	return out
}
