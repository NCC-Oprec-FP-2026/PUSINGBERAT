package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockLogSourceRepo struct {
	createFn    func(ctx context.Context, ls *domain.LogSource) error
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*domain.LogSource, error)
	listFn      func(ctx context.Context) ([]domain.LogSource, error)
	updateFn    func(ctx context.Context, ls *domain.LogSource) error
	deleteFn    func(ctx context.Context, id uuid.UUID) error
}

func (m *mockLogSourceRepo) Create(ctx context.Context, ls *domain.LogSource) error {
	if m.createFn != nil {
		return m.createFn(ctx, ls)
	}
	return nil
}
func (m *mockLogSourceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &domain.LogSource{ID: id, Name: "test", FilePath: "/var/log/test.log"}, nil
}
func (m *mockLogSourceRepo) List(ctx context.Context) ([]domain.LogSource, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []domain.LogSource{}, nil
}
func (m *mockLogSourceRepo) Update(ctx context.Context, ls *domain.LogSource) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, ls)
	}
	return nil
}
func (m *mockLogSourceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockWatcherRegistry struct {
	addFn    func(source *domain.LogSource) error
	removeFn func(sourceID uuid.UUID)
}

func (m *mockWatcherRegistry) AddWatcher(source *domain.LogSource) error {
	if m.addFn != nil {
		return m.addFn(source)
	}
	return nil
}
func (m *mockWatcherRegistry) RemoveWatcher(sourceID uuid.UUID) {
	if m.removeFn != nil {
		m.removeFn(sourceID)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtrLS(s string) *string { return &s }

func validLogSource() *domain.LogSource {
	return &domain.LogSource{
		Name:     "nginx access",
		FilePath: "/var/log/nginx/access.log",
		LogType:  "nginx",
		Status:   domain.LogSourceStatusActive,
	}
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestLogSourceService_Create_Happy(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := validLogSource()
	if err := svc.Create(context.Background(), ls); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogSourceService_Create_DefaultsApplied(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{
		Name:     "syslog",
		FilePath: "/var/log/syslog",
	}
	if err := svc.Create(context.Background(), ls); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ls.Status != domain.LogSourceStatusActive {
		t.Errorf("expected status %q, got %q", domain.LogSourceStatusActive, ls.Status)
	}
	if ls.LogType != "generic" {
		t.Errorf("expected log_type %q, got %q", "generic", ls.LogType)
	}
}

func TestLogSourceService_Create_EmptyName(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{FilePath: "/var/log/test.log"}
	err := svc.Create(context.Background(), ls)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogSourceService_Create_EmptyFilePath(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{Name: "test"}
	err := svc.Create(context.Background(), ls)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogSourceService_Create_InvalidStatus(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{
		Name:     "test",
		FilePath: "/var/log/test.log",
		Status:   "unknown-status",
	}
	err := svc.Create(context.Background(), ls)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogSourceService_Create_InvalidLogType(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{
		Name:     "test",
		FilePath: "/var/log/test.log",
		LogType:  "unsupported-type",
	}
	err := svc.Create(context.Background(), ls)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogSourceService_Create_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	svc := NewLogSourceService(&mockLogSourceRepo{
		createFn: func(_ context.Context, _ *domain.LogSource) error { return repoErr },
	})
	ls := validLogSource()
	err := svc.Create(context.Background(), ls)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogSourceService_Create_WatcherError_DoesNotFail(t *testing.T) {
	// Watcher error should be logged but NOT propagate as a service error.
	svc := NewLogSourceService(&mockLogSourceRepo{})
	svc.SetRegistry(&mockWatcherRegistry{
		addFn: func(_ *domain.LogSource) error { return errors.New("watcher fail") },
	})
	ls := validLogSource()
	if err := svc.Create(context.Background(), ls); err != nil {
		t.Fatalf("watcher error should not propagate, got: %v", err)
	}
}

func TestLogSourceService_Create_WithRegistry_InactiveSource_NoWatcher(t *testing.T) {
	watcherCalled := false
	svc := NewLogSourceService(&mockLogSourceRepo{})
	svc.SetRegistry(&mockWatcherRegistry{
		addFn: func(_ *domain.LogSource) error {
			watcherCalled = true
			return nil
		},
	})
	ls := &domain.LogSource{
		Name:     "test",
		FilePath: "/var/log/test.log",
		Status:   domain.LogSourceStatusInactive,
	}
	if err := svc.Create(context.Background(), ls); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watcherCalled {
		t.Error("watcher should not be called for inactive source")
	}
}

// ---------------------------------------------------------------------------
// GetByID tests
// ---------------------------------------------------------------------------

func TestLogSourceService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, got uuid.UUID) (*domain.LogSource, error) {
			if got != id {
				return nil, domain.ErrNotFound
			}
			return &domain.LogSource{ID: id, Name: "found"}, nil
		},
	})
	ls, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ls.ID != id {
		t.Errorf("expected id %v, got %v", id, ls.ID)
	}
}

func TestLogSourceService_GetByID_NotFound(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.LogSource, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestLogSourceService_List_ReturnsSources(t *testing.T) {
	expected := []domain.LogSource{{Name: "a"}, {Name: "b"}}
	svc := NewLogSourceService(&mockLogSourceRepo{
		listFn: func(_ context.Context) ([]domain.LogSource, error) { return expected, nil },
	})
	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Errorf("expected %d sources, got %d", len(expected), len(got))
	}
}

func TestLogSourceService_List_RepoError(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{
		listFn: func(_ context.Context) ([]domain.LogSource, error) {
			return nil, errors.New("db error")
		},
	})
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestLogSourceService_Delete_Happy(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogSourceService_Delete_RemovesWatcher(t *testing.T) {
	id := uuid.New()
	removedID := uuid.Nil
	svc := NewLogSourceService(&mockLogSourceRepo{})
	svc.SetRegistry(&mockWatcherRegistry{
		removeFn: func(sourceID uuid.UUID) { removedID = sourceID },
	})
	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removedID != id {
		t.Errorf("expected watcher removed for %v, got %v", id, removedID)
	}
}

func TestLogSourceService_Delete_RepoError(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return errors.New("db error") },
	})
	err := svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdateMetadata tests
// ---------------------------------------------------------------------------

func TestLogSourceService_UpdateMetadata_UpdatesName(t *testing.T) {
	id := uuid.New()
	updated := false
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, got uuid.UUID) (*domain.LogSource, error) {
			return &domain.LogSource{ID: got, Name: "old-name", FilePath: "/var/log/t.log"}, nil
		},
		updateFn: func(_ context.Context, ls *domain.LogSource) error {
			updated = true
			if ls.Name != "new-name" {
				return errors.New("name not updated")
			}
			return nil
		},
	})
	newName := "new-name"
	ls, err := svc.UpdateMetadata(context.Background(), id, &newName, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("repo.Update was not called")
	}
	if ls.Name != "new-name" {
		t.Errorf("expected name %q, got %q", "new-name", ls.Name)
	}
}

func TestLogSourceService_UpdateMetadata_NoOp(t *testing.T) {
	updateCalled := false
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.LogSource, error) {
			return &domain.LogSource{ID: id, Name: "existing"}, nil
		},
		updateFn: func(_ context.Context, _ *domain.LogSource) error {
			updateCalled = true
			return nil
		},
	})
	// nil name and nil description → no-op
	_, err := svc.UpdateMetadata(context.Background(), uuid.New(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalled {
		t.Error("repo.Update should not be called for no-op")
	}
}

func TestLogSourceService_UpdateMetadata_EmptyNameError(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.LogSource, error) {
			return &domain.LogSource{ID: id, Name: "existing"}, nil
		},
	})
	emptyName := "   "
	_, err := svc.UpdateMetadata(context.Background(), uuid.New(), &emptyName, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogSourceService_UpdateMetadata_NotFound(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.LogSource, error) {
			return nil, domain.ErrNotFound
		},
	})
	name := "new"
	_, err := svc.UpdateMetadata(context.Background(), uuid.New(), &name, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogSourceService_UpdateMetadata_UpdatesDescription(t *testing.T) {
	id := uuid.New()
	svc := NewLogSourceService(&mockLogSourceRepo{
		getByIDFn: func(_ context.Context, got uuid.UUID) (*domain.LogSource, error) {
			return &domain.LogSource{ID: got, Name: "source"}, nil
		},
	})
	desc := "new description"
	ls, err := svc.UpdateMetadata(context.Background(), id, nil, &desc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ls.Description == nil || *ls.Description != desc {
		t.Errorf("description not updated correctly")
	}
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestLogSourceService_Update_Happy(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := validLogSource()
	if err := svc.Update(context.Background(), ls); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogSourceService_Update_ValidationError(t *testing.T) {
	svc := NewLogSourceService(&mockLogSourceRepo{})
	ls := &domain.LogSource{} // no name, no file path
	err := svc.Update(context.Background(), ls)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateFilePath tests
// ---------------------------------------------------------------------------

func TestValidateFilePath_AllowedPaths(t *testing.T) {
	cases := []string{
		"/var/log/syslog",
		"/var/log/nginx/access.log",
		"/host/logs/app.log",
		"/tmp/test.log",
	}
	for _, path := range cases {
		if err := ValidateFilePath(path); err != nil {
			t.Errorf("expected %q to be allowed, got: %v", path, err)
		}
	}
}

func TestValidateFilePath_RelativePath(t *testing.T) {
	err := ValidateFilePath("var/log/syslog")
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for relative path, got %v", err)
	}
}

func TestValidateFilePath_PathTraversal(t *testing.T) {
	traversals := []string{
		"/var/log/../../etc/shadow",
		"/var/log/../../../etc/passwd",
	}
	for _, path := range traversals {
		err := ValidateFilePath(path)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("expected ErrValidation for traversal %q, got %v", path, err)
		}
	}
}

func TestValidateFilePath_PartialPrefixAttack(t *testing.T) {
	// "/var/logfake/evil" starts with "/var/log" but is NOT under it
	err := ValidateFilePath("/var/logfake/evil")
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for partial prefix attack, got %v", err)
	}
}

func TestValidateFilePath_DisallowedRoot(t *testing.T) {
	err := ValidateFilePath("/etc/shadow")
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for /etc/shadow, got %v", err)
	}
}

// unused var guard
var _ = strPtrLS
