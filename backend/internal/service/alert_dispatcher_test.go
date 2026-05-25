package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type MockAlertRepo struct {
	CreateFunc              func(ctx context.Context, alert *domain.Alert) error
	GetByIDFunc             func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	ListFunc                func(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error)
	AcknowledgeFunc         func(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	GetAlertsBySeverityFunc func(ctx context.Context) (map[string]int64, error)
	DeleteFunc              func(ctx context.Context, id uuid.UUID) error
	MarkDiscordSentFunc     func(ctx context.Context, id uuid.UUID) error
}

func (m *MockAlertRepo) Create(ctx context.Context, alert *domain.Alert) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, alert)
	}
	return nil
}
func (m *MockAlertRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockAlertRepo) List(ctx context.Context, params repository.AlertListParams) ([]domain.Alert, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, 0, nil
}
func (m *MockAlertRepo) Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.AcknowledgeFunc != nil {
		return m.AcknowledgeFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockAlertRepo) GetAlertsBySeverity(ctx context.Context) (map[string]int64, error) {
	if m.GetAlertsBySeverityFunc != nil {
		return m.GetAlertsBySeverityFunc(ctx)
	}
	return nil, nil
}
func (m *MockAlertRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *MockAlertRepo) MarkDiscordSent(ctx context.Context, id uuid.UUID) error {
	if m.MarkDiscordSentFunc != nil {
		return m.MarkDiscordSentFunc(ctx, id)
	}
	return nil
}

func TestAlertDispatcher_Start(t *testing.T) {
	repo := &MockAlertRepo{
		CreateFunc: func(ctx context.Context, alert *domain.Alert) error {
			return nil
		},
	}
	
	alertChan := make(chan *domain.Alert, 5)
	discord := NewDiscordNotifier("")
	
	dispatcher := NewAlertDispatcher(repo, alertChan, nil, discord)
	
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher.Start(ctx)
	
	alertChan <- &domain.Alert{
		Title: "Test Alert",
	}
	
	time.Sleep(100 * time.Millisecond) // let goroutine process
	
	cancel()
	time.Sleep(100 * time.Millisecond) // let goroutine stop
}
