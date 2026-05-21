package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/repository"
)

type LogSourceRepository interface {
	Create(ctx context.Context, source *domain.LogSource) error
	GetByID(ctx context.Context, id string) (*domain.LogSource, error)
	GetByFilePath(ctx context.Context, filePath string) (*domain.LogSource, error)
	List(ctx context.Context, filter repository.LogSourceFilter) ([]domain.LogSource, int64, error)
	Update(ctx context.Context, source *domain.LogSource) error
	UpdateStatus(ctx context.Context, id string, status domain.LogSourceStatus) error
	Delete(ctx context.Context, id string) error
}

type LogSourceService struct {
	repo LogSourceRepository
}

func NewLogSourceService(repo LogSourceRepository) *LogSourceService {
	return &LogSourceService{repo: repo}
}

func (s *LogSourceService) Create(ctx context.Context, source *domain.LogSource) error {
	normalizeLogSource(source)
	if err := validateLogSource(source); err != nil {
		return err
	}
	return s.repo.Create(ctx, source)
}

func (s *LogSourceService) GetByID(ctx context.Context, id string) (*domain.LogSource, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *LogSourceService) List(ctx context.Context, filter repository.LogSourceFilter) ([]domain.LogSource, int64, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	return s.repo.List(ctx, filter)
}

func (s *LogSourceService) Update(ctx context.Context, source *domain.LogSource) error {
	if source == nil {
		return fmt.Errorf("%w: source is required", ErrValidation)
	}
	if strings.TrimSpace(source.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	normalizeLogSource(source)
	if err := validateLogSource(source); err != nil {
		return err
	}
	return s.repo.Update(ctx, source)
}

func (s *LogSourceService) UpdateStatus(ctx context.Context, id string, status domain.LogSourceStatus) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	if status == "" {
		return fmt.Errorf("%w: status is required", ErrValidation)
	}
	return s.repo.UpdateStatus(ctx, strings.TrimSpace(id), status)
}

func (s *LogSourceService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func normalizeLogSource(source *domain.LogSource) {
	if source == nil {
		return
	}
	source.Name = strings.TrimSpace(source.Name)
	source.FilePath = strings.TrimSpace(source.FilePath)

	if source.LogType == "" {
		source.LogType = domain.LogSourceTypeGeneric
	}
	if source.Status == "" {
		source.Status = domain.LogSourceStatusActive
	}
}

func validateLogSource(source *domain.LogSource) error {
	if source == nil {
		return fmt.Errorf("%w: source is required", ErrValidation)
	}
	if source.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if source.FilePath == "" {
		return fmt.Errorf("%w: file_path is required", ErrValidation)
	}

	switch source.LogType {
	case domain.LogSourceTypeGeneric, domain.LogSourceTypeSyslog, domain.LogSourceTypeNginx:
	default:
		return fmt.Errorf("%w: log_type must be one of generic, syslog, nginx", ErrValidation)
	}

	switch source.Status {
	case domain.LogSourceStatusActive, domain.LogSourceStatusInactive, domain.LogSourceStatusError:
	default:
		return fmt.Errorf("%w: status must be one of active, inactive, error", ErrValidation)
	}

	return nil
}
