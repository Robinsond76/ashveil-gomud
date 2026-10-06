// Package modconfig coerces the loosely typed values a module reads from
// plug.Config.Get into Go values.
//
// Module config lives under Modules.<name>.* in config.yaml and arrives as
// any: ints, int64s, float64s, strings and nested maps keyed by string or by
// any, depending on which YAML decoder produced them. Every module used to
// carry its own copy of these coercions, and the copies drifted (some
// truncated fractional floats, some rejected them). This package is the one
// shared set of rules:
//
//   - Int rejects a fractional or out-of-range float rather than truncating
//     it, so a mistyped "2.5" falls back to the module default instead of
//     silently becoming 2.
//   - Strings are trimmed before parsing; a blank string is not a number.
//   - Map lowercases keys so config lookups are case-insensitive.
package modconfig

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Int coerces raw to an int. It accepts int, int64, uint64, an integral
// float64 within the int32 range, and a trimmed numeric string. The bool is
// false when raw is missing, blank, non-numeric or fractional.
func Int(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		if v > math.MaxInt32 {
			return 0, false
		}
		return int(v), true
	case float64:
		if v == math.Trunc(v) && math.Abs(v) <= math.MaxInt32 {
			return int(v), true
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

// IntOr is Int with a fallback for values that do not coerce.
func IntOr(raw any, fallback int) int {
	if n, ok := Int(raw); ok {
		return n
	}
	return fallback
}

// Float coerces raw to a float64 from float64, int, int64 or a trimmed
// numeric string.
func Float(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// FloatOr is Float with a fallback for values that do not coerce.
func FloatOr(raw any, fallback float64) float64 {
	if f, ok := Float(raw); ok {
		return f
	}
	return fallback
}

// String returns raw when it is a string and "" otherwise. It does not
// trim; callers that want a trimmed value trim it themselves, so a
// deliberate " " stays distinguishable from a missing key.
func String(raw any) string {
	s, _ := raw.(string)
	return s
}

// Strings coerces a []string or []any config list to its trimmed,
// non-blank string entries, in order. Non-string entries are skipped.
func Strings(raw any) []string {
	var out []string
	switch v := raw.(type) {
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// Map returns a nested config map with lowercased string keys, accepting
// either map[string]any or the map[any]any a YAML decoder may produce. Keys
// that are not strings are dropped. Anything else yields nil.
func Map(raw any) map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[strings.ToLower(k)] = item
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = item
			}
		}
		return out
	}
	return nil
}

// Duration coerces raw to a time.Duration: a Go duration string such as
// "90s" or "1h30m", or a number of whole seconds as int, int64, uint64, an
// integral float64 or a numeric string. The bool is false for anything
// else, including a negative or fractional value.
func Duration(raw any) (time.Duration, bool) {
	if text, ok := raw.(string); ok {
		if d, err := time.ParseDuration(strings.TrimSpace(text)); err == nil {
			return d, d >= 0
		}
	}
	if n, ok := Int(raw); ok && n >= 0 {
		return time.Duration(n) * time.Second, true
	}
	return 0, false
}

// Bool coerces raw to a bool from a bool or a string strconv.ParseBool
// accepts ("true", "1", "yes" is not one). Anything else is false.
func Bool(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && b
	}
	return false
}
