// --- internal/ruleengine/engine.go ---

package ruleengine

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Threshold window types
// ---------------------------------------------------------------------------

// windowKey uniquely identifies a threshold counter. For rules with a
// group_by field, the GroupValue partitions counters (e.g. per-hostname).
type windowKey struct {
	RuleID     string
	GroupValue string
}

// windowCounter stores the list of match timestamps within the sliding
// window for a single (rule_id, group_value) pair.
type windowCounter struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// ---------------------------------------------------------------------------
// Engine
// ---------------------------------------------------------------------------

// Engine evaluates parsed events against the loaded rule set. It owns the
// in-memory threshold window state and sends generated alerts to alertChan.
//
// Thread safety: Engine.Evaluate may be called from a single goroutine (the
// persistence worker) or from multiple goroutines. The threshold map is
// protected by a sync.Mutex for safe concurrent access.
type Engine struct {
	loader    *RuleLoader
	alertChan chan<- *domain.Alert

	windowsMu sync.Mutex
	windows   map[windowKey]*windowCounter
}

// NewEngine creates a rule evaluation engine that sends generated alerts
// to the provided channel.
func NewEngine(loader *RuleLoader, alertChan chan<- *domain.Alert) *Engine {
	return &Engine{
		loader:    loader,
		alertChan: alertChan,
		windows:   make(map[windowKey]*windowCounter),
	}
}

// ---------------------------------------------------------------------------
// Evaluate — main entry point
// ---------------------------------------------------------------------------

// Evaluate checks the given event against every enabled rule. For each rule
// that matches (and whose threshold is met, if configured), an Alert is
// generated and sent to the alert channel.
//
// logType is the log_type of the event's log source (e.g. "syslog"). It is
// used for fast log-type filtering before condition evaluation.
func (e *Engine) Evaluate(event *domain.ParsedEvent, logType string) {
	rules := e.loader.GetRules()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Fast exit: if the rule restricts log_types, check membership.
		if len(rule.LogTypes) > 0 && !containsLogType(rule.LogTypes, logType) {
			continue
		}

		// Evaluate all conditions (AND logic, short-circuit).
		matched, err := MatchAllConditions(rule.Conditions, event)
		if err != nil {
			slog.Debug("engine: condition evaluation error",
				"rule_id", rule.ID,
				"err", err,
			)
			continue
		}
		if !matched {
			continue
		}

		// --- Threshold check ---
		if rule.Threshold != nil {
			if !e.checkThreshold(rule, event) {
				continue // threshold not yet reached
			}
		}

		// --- Generate and dispatch alert ---
		alert := e.generateAlert(rule, event)

		// Non-blocking send: if the alert channel is full, drop the alert
		// rather than blocking the evaluation goroutine.
		select {
		case e.alertChan <- alert:
			slog.Info("engine: alert generated",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"severity", rule.Severity,
				"title", alert.Title,
			)
		default:
			slog.Warn("engine: alert channel full, dropping alert",
				"rule_id", rule.ID,
				"title", alert.Title,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// Threshold sliding window
// ---------------------------------------------------------------------------

// checkThreshold implements the sliding window counter for threshold-based
// rules. Returns true when the match count within the window reaches the
// configured threshold count.
func (e *Engine) checkThreshold(rule *domain.RuleDefinition, event *domain.ParsedEvent) bool {
	// Build the window key from the rule ID and the group_by field value.
	groupValue := ""
	if rule.Threshold.GroupBy != "" {
		groupValue = ResolveField(rule.Threshold.GroupBy, event)
	}

	key := windowKey{
		RuleID:     rule.ID,
		GroupValue: groupValue,
	}

	// Get or create the counter for this key.
	e.windowsMu.Lock()
	counter, exists := e.windows[key]
	if !exists {
		counter = &windowCounter{}
		e.windows[key] = counter
	}
	e.windowsMu.Unlock()

	// Lock the individual counter for eviction + append.
	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Duration(rule.Threshold.WindowSeconds) * time.Second)

	// Evict timestamps outside the window. We reuse the backing array to
	// avoid allocations on the hot path.
	fresh := counter.timestamps[:0]
	for _, t := range counter.timestamps {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}

	// Append the current match.
	fresh = append(fresh, now)

	// FIX: Trigger exactly once per window reach and reset the counter
	if len(fresh) >= rule.Threshold.Count {
		counter.timestamps = fresh[:0] // Clear timestamps to prevent alert spam
		return true
	}

	counter.timestamps = fresh
	return false
}

// ---------------------------------------------------------------------------
// Alert generation
// ---------------------------------------------------------------------------

// generateAlert creates a domain.Alert from a matched rule and event,
// interpolating {{field}} placeholders in the alert title and description.
func (e *Engine) generateAlert(rule *domain.RuleDefinition, event *domain.ParsedEvent) *domain.Alert {
	title := interpolateTemplate(rule.Alert.Title, rule, event)
	desc := interpolateTemplate(rule.Alert.Description, rule, event)

	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}

	rawLine := event.RawLine
	var rawLinePtr *string
	if rawLine != "" {
		rawLinePtr = &rawLine
	}

	logSourceID := event.LogSourceID

	return &domain.Alert{
		RuleName:    rule.Name,
		EventID:     &event.ID,
		LogSourceID: &logSourceID,
		Severity:    rule.Severity,
		Title:       title,
		Description: descPtr,
		RawLine:     rawLinePtr,
		TriggeredAt: time.Now().UTC(),
	}
}

// interpolateTemplate replaces {{placeholder}} tokens in a template string
// with resolved values from the event and rule threshold configuration.
//
// Supported placeholders:
//   - {{count}}          — threshold count (from rule definition)
//   - {{window_seconds}} — threshold window size (from rule definition)
//   - Any ParsedEvent field name (e.g. {{hostname}}, {{message}}, {{process}})
func interpolateTemplate(tmpl string, rule *domain.RuleDefinition, event *domain.ParsedEvent) string {
	if tmpl == "" {
		return ""
	}

	result := tmpl

	// Replace threshold-specific placeholders first.
	if rule.Threshold != nil {
		result = strings.ReplaceAll(result, "{{count}}", strconv.Itoa(rule.Threshold.Count))
		result = strings.ReplaceAll(result, "{{window_seconds}}", strconv.Itoa(rule.Threshold.WindowSeconds))
	}

	// Replace event field placeholders by scanning for {{...}} patterns.
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start + 2 // adjust to absolute position past "}}"

		fieldName := result[start+2 : end-2]
		fieldValue := ResolveField(fieldName, event)

		result = result[:start] + fieldValue + result[end:]
	}

	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// containsLogType checks whether the target log type is in the allowed list.
func containsLogType(allowed []string, target string) bool {
	for _, lt := range allowed {
		if lt == target {
			return true
		}
	}
	return false
}

// AlertChanSize is the recommended buffer size for the alert channel.
const AlertChanSize = 100

// NewAlertChan creates a buffered alert channel with the standard size.
func NewAlertChan() chan *domain.Alert {
	return make(chan *domain.Alert, AlertChanSize)
}

// SetAlertRuleID sets the RuleID field on an alert. Convenience for the
// dispatcher to attach the DB UUID (the engine uses string IDs from YAML).
func SetAlertRuleID(alert *domain.Alert, ruleID uuid.UUID) {
	alert.RuleID = &ruleID
}

// Loader returns the engine's rule loader (for callers that need it).
func (e *Engine) Loader() *RuleLoader {
	return e.loader
}

// WindowCount returns the number of active threshold windows (testing/debug).
func (e *Engine) WindowCount() int {
	e.windowsMu.Lock()
	defer e.windowsMu.Unlock()
	return len(e.windows)
}

// String returns a human-readable summary of the engine state.
func (e *Engine) String() string {
	return fmt.Sprintf("Engine{rules=%d, windows=%d}",
		e.loader.RuleCount(), e.WindowCount())
}
