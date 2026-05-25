package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/ruleengine"
)

// ---------------------------------------------------------------------------
// Mock RuleRepository
// ---------------------------------------------------------------------------

type mockRuleRepo struct {
	createFn      func(ctx context.Context, rule *domain.Rule) error
	getByIDFn     func(ctx context.Context, id uuid.UUID) (*domain.Rule, error)
	getByNameFn   func(ctx context.Context, name string) (*domain.Rule, error)
	listFn        func(ctx context.Context) ([]domain.Rule, error)
	listEnabledFn func(ctx context.Context) ([]domain.Rule, error)
	updateFn      func(ctx context.Context, rule *domain.Rule) error
	deleteFn      func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRuleRepo) Create(ctx context.Context, rule *domain.Rule) error {
	if m.createFn != nil {
		return m.createFn(ctx, rule)
	}
	return nil
}
func (m *mockRuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.Rule{ID: id, Name: "test", YAMLContent: minimalYAML(), Severity: domain.SeverityMedium}, nil
}
func (m *mockRuleRepo) GetByName(ctx context.Context, name string) (*domain.Rule, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return nil, domain.ErrNotFound
}
func (m *mockRuleRepo) List(ctx context.Context) ([]domain.Rule, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []domain.Rule{}, nil
}
func (m *mockRuleRepo) ListEnabled(ctx context.Context) ([]domain.Rule, error) {
	if m.listEnabledFn != nil {
		return m.listEnabledFn(ctx)
	}
	return []domain.Rule{}, nil
}
func (m *mockRuleRepo) Update(ctx context.Context, rule *domain.Rule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, rule)
	}
	return nil
}
func (m *mockRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// minimalYAML returns the smallest valid rule YAML.
func minimalYAML() string {
	return `
id: test-rule-001
name: Test Rule
enabled: true
severity: medium
conditions:
  - field: message
    operator: contains
    value: "error"
alert:
  title: "Test Alert"
`
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRuleService_Create_Happy(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{
		Name:        "Test Rule",
		YAMLContent: minimalYAML(),
		Severity:    domain.SeverityMedium,
	}
	if err := svc.Create(context.Background(), rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleService_Create_DefaultsSeverity(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{
		Name:        "Test Rule",
		YAMLContent: minimalYAML(),
	}
	if err := svc.Create(context.Background(), rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Severity != domain.SeverityMedium {
		t.Errorf("expected default severity %q, got %q", domain.SeverityMedium, rule.Severity)
	}
}

func TestRuleService_Create_EmptyName(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{YAMLContent: minimalYAML()}
	err := svc.Create(context.Background(), rule)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestRuleService_Create_EmptyYAML(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{Name: "Test Rule"}
	err := svc.Create(context.Background(), rule)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestRuleService_Create_InvalidSeverity(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{
		Name:        "Test Rule",
		YAMLContent: minimalYAML(),
		Severity:    "super-critical",
	}
	err := svc.Create(context.Background(), rule)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestRuleService_Create_RepoError(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{
		createFn: func(_ context.Context, _ *domain.Rule) error { return errors.New("db error") },
	})
	rule := &domain.Rule{Name: "Test Rule", YAMLContent: minimalYAML()}
	err := svc.Create(context.Background(), rule)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRuleService_Create_TriggersHotReload(t *testing.T) {
	reloadCalled := false
	repo := &mockRuleRepo{
		listEnabledFn: func(_ context.Context) ([]domain.Rule, error) {
			reloadCalled = true
			return []domain.Rule{}, nil
		},
	}
	loader := ruleengine.NewRuleLoader()
	svc := NewRuleService(repo)
	svc.SetLoader(loader)

	rule := &domain.Rule{Name: "Test Rule", YAMLContent: minimalYAML()}
	if err := svc.Create(context.Background(), rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reloadCalled {
		t.Error("expected hot-reload (ListEnabled) to be called")
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestRuleService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, got uuid.UUID) (*domain.Rule, error) {
			return &domain.Rule{ID: got, Name: "found"}, nil
		},
	})
	r, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID != id {
		t.Errorf("expected id %v, got %v", id, r.ID)
	}
}

func TestRuleService_GetByID_NotFound(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Rule, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// List / ListEnabled
// ---------------------------------------------------------------------------

func TestRuleService_List_ReturnsRules(t *testing.T) {
	expected := []domain.Rule{{Name: "a"}, {Name: "b"}}
	svc := NewRuleService(&mockRuleRepo{
		listFn: func(_ context.Context) ([]domain.Rule, error) { return expected, nil },
	})
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Errorf("expected %d rules, got %d", len(expected), len(got))
	}
}

func TestRuleService_ListEnabled_ReturnsEnabledRules(t *testing.T) {
	expected := []domain.Rule{{Name: "enabled-rule", Enabled: true}}
	svc := NewRuleService(&mockRuleRepo{
		listEnabledFn: func(_ context.Context) ([]domain.Rule, error) { return expected, nil },
	})
	got, err := svc.ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 enabled rule, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestRuleService_Update_Happy(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{Name: "Updated", YAMLContent: minimalYAML(), Severity: domain.SeverityHigh}
	if err := svc.Update(context.Background(), rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleService_Update_ValidationError(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{} // empty name and yaml
	err := svc.Update(context.Background(), rule)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Toggle
// ---------------------------------------------------------------------------

func TestRuleService_Toggle_EnablesDisabledRule(t *testing.T) {
	id := uuid.New()
	rule := &domain.Rule{
		ID: id, Name: "test", YAMLContent: minimalYAML(),
		Severity: domain.SeverityMedium, Enabled: false,
	}
	updatedEnabled := false
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Rule, error) {
			return rule, nil
		},
		updateFn: func(_ context.Context, r *domain.Rule) error {
			updatedEnabled = r.Enabled
			return nil
		},
	})
	result, err := svc.Toggle(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Enabled {
		t.Error("expected rule to be enabled after toggle")
	}
	if !updatedEnabled {
		t.Error("repo.Update was not called with enabled=true")
	}
}

func TestRuleService_Toggle_DisablesEnabledRule(t *testing.T) {
	id := uuid.New()
	rule := &domain.Rule{
		ID: id, Name: "test", YAMLContent: minimalYAML(),
		Severity: domain.SeverityMedium, Enabled: true,
	}
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Rule, error) {
			return rule, nil
		},
	})
	result, err := svc.Toggle(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Enabled {
		t.Error("expected rule to be disabled after toggle")
	}
}

func TestRuleService_Toggle_NotFound(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Rule, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.Toggle(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRuleService_Toggle_UpdateError(t *testing.T) {
	id := uuid.New()
	rule := &domain.Rule{ID: id, Name: "test", YAMLContent: minimalYAML(), Severity: domain.SeverityMedium}
	svc := NewRuleService(&mockRuleRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Rule, error) { return rule, nil },
		updateFn:  func(_ context.Context, _ *domain.Rule) error { return errors.New("db error") },
	})
	_, err := svc.Toggle(context.Background(), id)
	if err == nil {
		t.Fatal("expected error on update failure, got nil")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRuleService_Delete_Happy(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleService_Delete_RepoError(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return errors.New("db error") },
	})
	err := svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRuleService_Delete_TriggersHotReload(t *testing.T) {
	reloadCalled := false
	repo := &mockRuleRepo{
		listEnabledFn: func(_ context.Context) ([]domain.Rule, error) {
			reloadCalled = true
			return []domain.Rule{}, nil
		},
	}
	loader := ruleengine.NewRuleLoader()
	svc := NewRuleService(repo)
	svc.SetLoader(loader)

	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reloadCalled {
		t.Error("expected hot-reload (ListEnabled) to be called after delete")
	}
}

// ---------------------------------------------------------------------------
// SetLoader / Validate helpers
// ---------------------------------------------------------------------------

func TestRuleService_SetLoader_NoLoader_NoHotReload(t *testing.T) {
	// Without calling SetLoader, Create should not panic or error on reload.
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{Name: "Test Rule", YAMLContent: minimalYAML()}
	if err := svc.Create(context.Background(), rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleService_Validate_WhitespaceName(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	rule := &domain.Rule{Name: "   ", YAMLContent: minimalYAML()}
	err := svc.Create(context.Background(), rule)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for whitespace-only name, got %v", err)
	}
}

func TestRuleService_SeedFromDirectory(t *testing.T) {
	svc := NewRuleService(&mockRuleRepo{})
	// Assuming there are rules in backend/migrations, but wait, the tests run in internal/service.
	// We can create a temp directory and write some YAML files.
	dir, err := os.MkdirTemp("", "rule_seed")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = os.WriteFile(filepath.Join(dir, "rule1.yaml"), []byte(minimalYAML()), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.SeedFromDirectory(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
