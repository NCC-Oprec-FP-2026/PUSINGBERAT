package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func TestAlertRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewAlertRepo(mock)

	ruleID := uuid.New()
	sourceID := uuid.New()
	alert := &domain.Alert{
		RuleID:       &ruleID,
		RuleName:     "High CPU",
		Severity:     "critical",
		Title:        "CPU usage > 90%",
		LogSourceID:  &sourceID,
		TriggeredAt:    time.Now(),
	}

	mock.ExpectQuery(`INSERT INTO alerts`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(uuid.New()))

	err = repo.Create(context.Background(), alert)
	if err != nil {
		t.Errorf("error was not expected while inserting: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAlertRepo_GetAlertsBySeverity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewAlertRepo(mock)

	mock.ExpectQuery(`SELECT severity::text, COUNT\(\*\) FROM alerts GROUP BY severity`).
		WillReturnRows(pgxmock.NewRows([]string{"severity", "count"}).
			AddRow("critical", int64(5)).
			AddRow("high", int64(10)))

	counts, err := repo.GetAlertsBySeverity(context.Background())
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if counts["critical"] != 5 {
		t.Errorf("expected 5 critical alerts, got %d", counts["critical"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
