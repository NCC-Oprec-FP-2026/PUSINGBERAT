//go:build integration

package ruleengine_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/watcher"
)

func TestLogLineCreatesAlert(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, integrationDSN())
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	logSourceRepo := repository.NewLogSourceRepo(db)
	eventRepo := repository.NewEventRepo(db)
	ruleRepo := repository.NewRuleRepo(db)
	alertRepo := repository.NewAlertRepo(db)

	if err := ruleengine.SeedRules(ctx, ruleRepo, "../../rules"); err != nil {
		t.Fatalf("seed rules: %v", err)
	}
	rules, err := ruleengine.LoadEnabledRulesFromDB(ctx, ruleRepo)
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}

	alertChan := make(chan domain.Alert, 10)
	alertService := service.NewAlertService(alertRepo)
	dispatcher := ruleengine.NewAlertDispatcher(alertChan, alertService)
	dispatcherCtx, stopDispatcher := context.WithCancel(ctx)
	defer stopDispatcher()
	go dispatcher.Run(dispatcherCtx)

	engine := ruleengine.NewEngine(rules)
	eventService := service.NewEventService(eventRepo, engine, alertChan)
	registry := watcher.NewRegistry(eventService)
	sourceService := service.NewLogSourceService(logSourceRepo, registry)

	file, err := os.CreateTemp("", "pusingberat-rule-*.log")
	if err != nil {
		t.Fatalf("create temp log: %v", err)
	}
	path := file.Name()
	_ = file.Close()
	defer os.Remove(path)

	source := &domain.LogSource{
		Name:     "integration-rule-" + time.Now().UTC().Format("20060102150405.000000000"),
		FilePath: path,
		LogType:  domain.LogSourceTypeSyslog,
		Status:   domain.LogSourceStatusActive,
	}
	if err := sourceService.Create(ctx, source); err != nil {
		t.Fatalf("create source: %v", err)
	}
	defer func() {
		registry.Stop(source.ID)
		_ = logSourceRepo.Delete(context.Background(), source.ID)
	}()

	line := fmt.Sprintf(
		"%s integration-host sshd[4321]: Failed password for invalid user root from 203.0.113.44 port 22 ssh2",
		time.Now().Format("Jan _2 15:04:05"),
	)
	if err := appendLine(path, line); err != nil {
		t.Fatalf("append log line: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		alerts, _, err := alertRepo.List(ctx, repository.AlertFilter{Page: 1, PerPage: 100})
		if err != nil {
			t.Fatalf("list alerts: %v", err)
		}
		for _, alert := range alerts {
			if alert.RawLine != nil && *alert.RawLine == line && alert.RuleName == "Failed Login" {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("alert row for test log line did not appear within 5 seconds")
}

func appendLine(path string, line string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintln(file, line)
	return err
}

func integrationDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		envOrDefault("DB_USER", "siem"),
		envOrDefault("DB_PASSWORD", "siem_password"),
		envOrDefault("DB_HOST", "postgres"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_NAME", "pusingberat"),
	)
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
