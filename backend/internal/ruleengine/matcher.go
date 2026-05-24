package ruleengine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

// ---------------------------------------------------------------------------
// Matcher — condition evaluation against ParsedEvents
// ---------------------------------------------------------------------------

// MatchAllConditions evaluates every condition in the slice against the given
// event using AND logic (short-circuit).  Returns true only when ALL
// conditions match.  Returns false with a nil error when a condition simply
// doesn't match.  Returns false with a non-nil error only on malformed
// input (e.g., an invalid regex pattern).
func MatchAllConditions(conditions []domain.RuleCondition, ev *domain.ParsedEvent) (bool, error) {
	for i := range conditions {
		matched, err := matchCondition(&conditions[i], ev)
		if err != nil {
			return false, fmt.Errorf("condition[%d] (%s %s): %w",
				i, conditions[i].Field, conditions[i].Operator, err)
		}
		if !matched {
			return false, nil // short-circuit: AND fails on first mismatch
		}
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Single condition evaluation
// ---------------------------------------------------------------------------

// matchCondition resolves the field value from the event and delegates to
// the appropriate operator function.
func matchCondition(cond *domain.RuleCondition, ev *domain.ParsedEvent) (bool, error) {
	fieldVal := resolveField(cond.Field, ev)

	switch cond.Operator {
	case "equals":
		return opEquals(fieldVal, cond.Value), nil
	case "contains":
		return opContains(fieldVal, cond.Value), nil
	case "starts_with":
		return opStartsWith(fieldVal, cond.Value), nil
	case "ends_with":
		return opEndsWith(fieldVal, cond.Value), nil
	case "regex":
		return opRegex(fieldVal, cond.Value)
	case "gt":
		return opGT(fieldVal, cond.Value)
	case "lt":
		return opLT(fieldVal, cond.Value)
	default:
		return false, fmt.Errorf("unsupported operator %q", cond.Operator)
	}
}

// ---------------------------------------------------------------------------
// Field resolution
// ---------------------------------------------------------------------------

// resolveField extracts the string value of a named field from a ParsedEvent.
// Pointer fields that are nil resolve to the empty string.
func resolveField(field string, ev *domain.ParsedEvent) string {
	switch field {
	case "message":
		return derefStr(ev.Message)
	case "hostname":
		return derefStr(ev.Hostname)
	case "process":
		return derefStr(ev.Process)
	case "pid":
		return derefInt(ev.PID)
	case "log_level":
		return derefStr(ev.LogLevel)
	case "raw_line":
		return ev.RawLine
	default:
		// Future-proofing: unknown fields silently resolve to "" so that
		// a rule referencing a field not yet supported doesn't crash.
		return ""
	}
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// derefInt converts a *int to its string representation, or "" if nil.
func derefInt(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}

// ---------------------------------------------------------------------------
// Operator implementations
// ---------------------------------------------------------------------------

// opEquals performs a case-sensitive exact string comparison.
func opEquals(fieldVal, expected string) bool {
	return fieldVal == expected
}

// opContains checks whether fieldVal contains the substring.
func opContains(fieldVal, substr string) bool {
	return strings.Contains(fieldVal, substr)
}

// opStartsWith checks whether fieldVal starts with the prefix.
func opStartsWith(fieldVal, prefix string) bool {
	return strings.HasPrefix(fieldVal, prefix)
}

// opEndsWith checks whether fieldVal ends with the suffix.
func opEndsWith(fieldVal, suffix string) bool {
	return strings.HasSuffix(fieldVal, suffix)
}

// opRegex compiles the pattern and matches it against fieldVal.
// Compiled patterns are cached in a sync.Map for performance so that the
// same regex is only compiled once across the lifetime of the process.
func opRegex(fieldVal, pattern string) (bool, error) {
	re, err := getCompiledRegex(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	return re.MatchString(fieldVal), nil
}

// opGT parses both operands as float64 and returns true when
// fieldVal > expected.  Non-numeric values yield an error.
func opGT(fieldVal, expected string) (bool, error) {
	fv, err := strconv.ParseFloat(fieldVal, 64)
	if err != nil {
		return false, fmt.Errorf("gt: field value %q is not numeric: %w", fieldVal, err)
	}
	ev, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false, fmt.Errorf("gt: expected value %q is not numeric: %w", expected, err)
	}
	return fv > ev, nil
}

// opLT parses both operands as float64 and returns true when
// fieldVal < expected.  Non-numeric values yield an error.
func opLT(fieldVal, expected string) (bool, error) {
	fv, err := strconv.ParseFloat(fieldVal, 64)
	if err != nil {
		return false, fmt.Errorf("lt: field value %q is not numeric: %w", fieldVal, err)
	}
	ev, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false, fmt.Errorf("lt: expected value %q is not numeric: %w", expected, err)
	}
	return fv < ev, nil
}

// ---------------------------------------------------------------------------
// Regex cache
// ---------------------------------------------------------------------------

// regexCache stores compiled *regexp.Regexp objects keyed by their pattern
// string.  This avoids repeated compilation of the same pattern across
// thousands of event evaluations.
var regexCache sync.Map // map[string]*regexp.Regexp

// getCompiledRegex returns a compiled regex for the given pattern, using
// the cache when available.
func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	// Store and return.  A race here is benign: two goroutines might both
	// compile the same pattern, but sync.Map.Store is safe and the result
	// is identical.
	regexCache.Store(pattern, re)
	return re, nil
}

// ResolveField is an exported alias for resolveField, used by the engine
// package for alert template interpolation (e.g. replacing {{hostname}}).
func ResolveField(field string, ev *domain.ParsedEvent) string {
	return resolveField(field, ev)
}
