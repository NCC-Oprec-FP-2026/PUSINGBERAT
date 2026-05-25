package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDB(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/pusingberat?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("Skipping test: Failed to connect to test db: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping test: Failed to ping test db: %v", err)
	}
	return pool
}

func seedTestData(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	// 1. Create a dummy log source
	sourceID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO log_sources (id, name, file_path, log_type, status)
		VALUES ($1, 'perf_test_source', '/tmp/perf.log', 'generic', 'active')
		ON CONFLICT (file_path) DO NOTHING
	`, sourceID)
	if err != nil {
		t.Fatalf("Failed to insert log source: %v", err)
	}

	// Make sure we have the source ID if it already existed
	err = pool.QueryRow(ctx, "SELECT id FROM log_sources WHERE file_path = '/tmp/perf.log'").Scan(&sourceID)
	if err != nil {
		t.Fatalf("Failed to retrieve log source id: %v", err)
	}

	// 2. Seed 50,000 Events
	var eventCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE log_source_id = $1", sourceID).Scan(&eventCount)
	if eventCount < 50000 {
		t.Log("Seeding 50,000 events... This might take a moment.")
		
		var rows [][]interface{}
		baseTime := time.Now().Add(-48 * time.Hour) // Spread over last 48 hours
		
		for i := 0; i < 50000; i++ {
			eventTime := baseTime.Add(time.Duration(i) * time.Second)
			rows = append(rows, []interface{}{
				sourceID, "raw_line_dummy", "dummy message", "localhost", "testproc", 1234, "info", eventTime, []byte(`{}`),
			})
		}
		
		_, err = pool.CopyFrom(ctx, pgx.Identifier{"events"}, 
			[]string{"log_source_id", "raw_line", "message", "hostname", "process", "pid", "log_level", "event_time", "extra"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			t.Fatalf("Failed to bulk insert events: %v", err)
		}
	}

	// 3. Seed 1,000 Alerts
	var alertCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE log_source_id = $1", sourceID).Scan(&alertCount)
	if alertCount < 1000 {
		t.Log("Seeding 1,000 alerts...")
		var rows [][]interface{}
		for i := 0; i < 1000; i++ {
			rows = append(rows, []interface{}{
				"test_rule", sourceID, "high", "Test Alert", time.Now(), false,
			})
		}
		_, err = pool.CopyFrom(ctx, pgx.Identifier{"alerts"},
			[]string{"rule_name", "log_source_id", "severity", "title", "triggered_at", "acknowledged"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			t.Fatalf("Failed to bulk insert alerts: %v", err)
		}
	}
	
	// Ensure ANALYZE runs so postgres updates its query planner statistics
	_, _ = pool.Exec(ctx, "ANALYZE events")
	_, _ = pool.Exec(ctx, "ANALYZE alerts")
}

// Plan represents a Postgres EXPLAIN JSON node
type Plan struct {
	NodeType     string  `json:"Node Type"`
	RelationName string  `json:"Relation Name,omitempty"`
	IndexName    string  `json:"Index Name,omitempty"`
	Plans        []Plan  `json:"Plans,omitempty"`
}

func checkIndexUsage(t *testing.T, plans []Plan, tableName string, expectedIndex string) {
	var seqScanFound bool
	var indexFound bool

	var walk func(p Plan)
	walk = func(p Plan) {
		if p.NodeType == "Seq Scan" && p.RelationName == tableName {
			seqScanFound = true
		}
		if (p.NodeType == "Index Scan" || p.NodeType == "Index Only Scan" || p.NodeType == "Bitmap Index Scan") && p.IndexName == expectedIndex {
			indexFound = true
		}
		for _, child := range p.Plans {
			walk(child)
		}
	}

	for _, p := range plans {
		walk(p)
	}

	if seqScanFound {
		t.Errorf("FAIL: Sequential Scan detected on table '%s'!", tableName)
	} else {
		t.Logf("SUCCESS: No Sequential Scan on table '%s'.", tableName)
	}

	if !indexFound {
		t.Errorf("FAIL: Expected index '%s' was NOT used! PostgreSQL planner might have chosen a different strategy.", expectedIndex)
	} else {
		t.Logf("SUCCESS: Index '%s' was successfully utilized.", expectedIndex)
	}
}

func parseExplain(t *testing.T, pool *pgxpool.Pool, query string) []Plan {
	ctx := context.Background()
	
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, FORMAT JSON) %s", query)
	
	var explainOutputBytes []byte
	var explainOutputString string
	
	// Depending on driver behavior, it might return a string or []byte
	var result interface{}
	err := pool.QueryRow(ctx, explainQuery).Scan(&result)
	if err != nil {
		t.Fatalf("Failed to run EXPLAIN: %v", err)
	}

	switch v := result.(type) {
	case string:
		explainOutputBytes = []byte(v)
	case []byte:
		explainOutputBytes = v
	default:
		// Attempt to JSON marshal if it's already parsed map
		b, err := json.Marshal(v)
		if err == nil {
			explainOutputBytes = b
		} else {
			explainOutputString = fmt.Sprintf("%v", v)
			explainOutputBytes = []byte(explainOutputString)
		}
	}

	var parsed []struct {
		Plan Plan `json:"Plan"`
	}
	
	if err := json.Unmarshal(explainOutputBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse EXPLAIN JSON: %v. Output: %s", err, string(explainOutputBytes))
	}
	
	if len(parsed) == 0 {
		t.Fatalf("Parsed EXPLAIN JSON is empty")
	}
	
	return []Plan{parsed[0].Plan}
}

func TestPerformance_DashboardOverviewQuery(t *testing.T) {
	pool := getTestDB(t)
	defer pool.Close()
	
	seedTestData(t, pool)

	query := `SELECT COUNT(*) FROM events WHERE event_time > NOW() - INTERVAL '24 hours'`
	
	plans := parseExplain(t, pool, query)
	checkIndexUsage(t, plans, "events", "idx_events_event_time")
}

func TestPerformance_TimelineQuery(t *testing.T) {
	pool := getTestDB(t)
	defer pool.Close()
	
	seedTestData(t, pool)

	query := `
		SELECT date_trunc('hour', event_time) AS hour, COUNT(*) AS count 
		FROM events 
		WHERE event_time > NOW() - INTERVAL '24 hours' 
		GROUP BY 1 
		ORDER BY 1
	`
	
	plans := parseExplain(t, pool, query)
	checkIndexUsage(t, plans, "events", "idx_events_event_time")
}
