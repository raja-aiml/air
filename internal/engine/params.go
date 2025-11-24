package engine

import (
	"fmt"
	"time"
)

// Params wraps map[string]any with type-safe accessors.
type Params map[string]any

// String returns a string parameter value, or the default if not found.
func (p Params) String(key, defaultVal string) string {
	if v, ok := p[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

// StringRequired returns a string parameter value, or an error if not found.
func (p Params) StringRequired(key string) (string, error) {
	if v, ok := p[key].(string); ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("required parameter %q not provided", key)
}

// Bool returns a bool parameter value, or the default if not found.
func (p Params) Bool(key string, defaultVal bool) bool {
	if v, ok := p[key].(bool); ok {
		return v
	}
	return defaultVal
}

// Int returns an int parameter value, or the default if not found.
func (p Params) Int(key string, defaultVal int) int {
	switch v := p[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultVal
}

// Duration returns a duration parameter value, or the default if not found.
func (p Params) Duration(key string, defaultVal time.Duration) time.Duration {
	switch v := p[key].(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

// StringSlice returns a string slice parameter value, or the default if not found.
func (p Params) StringSlice(key string, defaultVal []string) []string {
	if v, ok := p[key].([]string); ok {
		return v
	}
	if v, ok := p[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

// Has returns true if the key exists in the params.
func (p Params) Has(key string) bool {
	_, ok := p[key]
	return ok
}
