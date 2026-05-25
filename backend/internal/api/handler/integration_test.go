package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/api/handler"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/service"
)

// ---------------------------------------------------------------------------
// Mock Repositories
// ---------------------------------------------------------------------------
type mockLogSourceRepo struct{}

func (m *mockLogSourceRepo) Create(ctx context.Context, ls *domain.LogSource) error {
	ls.ID = uuid.New()
	return nil
}
func (m *mockLogSourceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.LogSource, error) {
	return nil, nil
}
func (m *mockLogSourceRepo) List(ctx context.Context) ([]domain.LogSource, error) { return nil, nil }
func (m *mockLogSourceRepo) Update(ctx context.Context, ls *domain.LogSource) error { return nil }
func (m *mockLogSourceRepo) Delete(ctx context.Context, id uuid.UUID) error         { return nil }

type mockRuleRepo struct{}

func (m *mockRuleRepo) Create(ctx context.Context, rule *domain.Rule) error {
	rule.ID = uuid.New()
	return nil
}
func (m *mockRuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Rule, error) {
	return nil, nil
}
func (m *mockRuleRepo) GetByName(ctx context.Context, name string) (*domain.Rule, error) {
	return nil, nil
}
func (m *mockRuleRepo) List(ctx context.Context) ([]domain.Rule, error)        { return nil, nil }
func (m *mockRuleRepo) ListEnabled(ctx context.Context) ([]domain.Rule, error) { return nil, nil }
func (m *mockRuleRepo) Update(ctx context.Context, rule *domain.Rule) error    { return nil }
func (m *mockRuleRepo) Delete(ctx context.Context, id uuid.UUID) error         { return nil }

// ---------------------------------------------------------------------------
// Test Setup
// ---------------------------------------------------------------------------
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	lsRepo := &mockLogSourceRepo{}
	ruleRepo := &mockRuleRepo{}

	lsSvc := service.NewLogSourceService(lsRepo)
	ruleSvc := service.NewRuleService(ruleRepo)

	deps := api.RouterDeps{
		LogSource: handler.NewLogSourceHandler(lsSvc),
		Rule:      handler.NewRuleHandler(ruleSvc),
	}

	return api.NewRouter(deps)
}

// ---------------------------------------------------------------------------
// Malformed Input Tests
// ---------------------------------------------------------------------------
func TestAPI_MalformedInput(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		method         string
		path           string
		payload        string
		expectedStatus int
	}{
		// ---------------------------------------------------------
		// Log Source Validation
		// ---------------------------------------------------------
		{
			name:           "LogSource - Missing Name",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `{"file_path": "/var/log/syslog", "log_type": "syslog"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "LogSource - Missing FilePath",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `{"name": "test", "log_type": "syslog"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "LogSource - Invalid LogType enum",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `{"name": "test", "file_path": "/var/log/syslog", "log_type": "invalid_type"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "LogSource - Invalid JSON Syntax",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `{"name": "test", "file_path": "/var/log/syslog"`, // missing closing brace
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "LogSource - Path Traversal Payload",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `{"name": "test", "file_path": "/var/log/../../etc/shadow", "log_type": "generic"}`,
			expectedStatus: http.StatusBadRequest,
		},
		// ---------------------------------------------------------
		// Rule Validation
		// ---------------------------------------------------------
		{
			name:           "Rule - Missing Name",
			method:         "POST",
			path:           "/api/v1/rules",
			payload:        `{"yaml_content": "some content", "severity": "high"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Rule - Missing YAMLContent",
			method:         "POST",
			path:           "/api/v1/rules",
			payload:        `{"name": "test rule", "severity": "high"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Rule - Invalid Severity Enum",
			method:         "POST",
			path:           "/api/v1/rules",
			payload:        `{"name": "test rule", "yaml_content": "content", "severity": "nuclear"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Rule - Invalid JSON Syntax",
			method:         "POST",
			path:           "/api/v1/rules",
			payload:        `{"name": "test"`, // missing closing brace
			expectedStatus: http.StatusBadRequest,
		},
		// ---------------------------------------------------------
		// Oversized Payloads & Extreme Malformations
		// ---------------------------------------------------------
		{
			name:           "Oversized 10MB string mapping to struct",
			method:         "POST",
			path:           "/api/v1/sources",
			payload:        `"` + strings.Repeat("A", 10*1024*1024) + `"`, // 10MB JSON string where object is expected
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Oversized 10MB nested array",
			method:         "POST",
			path:           "/api/v1/rules",
			payload:        strings.Repeat("[", 1000) + strings.Repeat("]", 1000), // Huge nested array where object expected
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.payload))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 1. Assert NO 500s. We expect 400 Bad Request.
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			// 2. Assert clean standardized JSON Error Envelope
			var resp handler.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse JSON error envelope: %v. Body was: %s", err, w.Body.String())
			}

			if resp.Error == "" || resp.Message == "" {
				t.Errorf("error envelope missing required fields: %+v", resp)
			}
		})
	}
}
