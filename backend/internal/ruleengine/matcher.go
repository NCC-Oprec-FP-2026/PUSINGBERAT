package ruleengine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/NCC-Oprec-FP-2026/PUSINGBERAT/internal/domain"
)

func ruleConditionsMatch(rule RuleDefinition, event *domain.ParsedEvent) (bool, error) {
	if !logTypeAllowed(rule.LogTypes, event) {
		return false, nil
	}

	for _, condition := range rule.Conditions {
		ok, err := conditionMatches(condition, event)
		if err != nil || !ok {
			return ok, err
		}
	}

	return true, nil
}

func logTypeAllowed(logTypes []string, event *domain.ParsedEvent) bool {
	if len(logTypes) == 0 {
		return true
	}

	eventLogType := fieldValue(event, "extra.log_type")
	for _, logType := range logTypes {
		if strings.EqualFold(strings.TrimSpace(logType), eventLogType) {
			return true
		}
	}
	return false
}

func conditionMatches(condition Condition, event *domain.ParsedEvent) (bool, error) {
	actual := fieldValue(event, condition.Field)
	expected := condition.Value

	switch condition.Operator {
	case "contains":
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected)), nil
	case "equals", "eq":
		return strings.EqualFold(actual, expected), nil
	case "not_equals", "neq":
		return !strings.EqualFold(actual, expected), nil
	case "regex":
		return regexp.MatchString(expected, actual)
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(actual), strings.ToLower(expected)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(actual), strings.ToLower(expected)), nil
	case "greater_than", "gt":
		return compareNumeric(actual, expected, func(a, b float64) bool { return a > b })
	case "less_than", "lt":
		return compareNumeric(actual, expected, func(a, b float64) bool { return a < b })
	default:
		return false, fmt.Errorf("unsupported operator %q", condition.Operator)
	}
}

func compareNumeric(actual, expected string, compare func(float64, float64) bool) (bool, error) {
	actualNumber, err := strconv.ParseFloat(actual, 64)
	if err != nil {
		return false, err
	}
	expectedNumber, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false, err
	}
	return compare(actualNumber, expectedNumber), nil
}

func fieldValue(event *domain.ParsedEvent, field string) string {
	normalized := strings.ToLower(strings.TrimSpace(field))
	switch normalized {
	case "raw_line":
		return event.RawLine
	case "message":
		if event.Message == nil {
			return ""
		}
		return *event.Message
	case "hostname", "host":
		if event.Hostname == nil {
			return ""
		}
		return *event.Hostname
	case "process":
		if event.Process == nil {
			return ""
		}
		return *event.Process
	case "pid":
		if event.PID == nil {
			return ""
		}
		return strconv.Itoa(*event.PID)
	case "log_level", "level":
		if event.LogLevel == nil {
			return ""
		}
		return *event.LogLevel
	default:
		if strings.HasPrefix(normalized, "extra.") {
			key := strings.TrimPrefix(normalized, "extra.")
			var extra map[string]any
			if len(event.Extra) == 0 || json.Unmarshal(event.Extra, &extra) != nil {
				return ""
			}
			if value, ok := extra[key]; ok && value != nil {
				return fmt.Sprint(value)
			}
		}
		return ""
	}
}
