package company

import "strings"

// SplitNameMatches sorts items into those whose name equals selector
// (case-insensitively) and those whose name merely contains it. selector must
// already be lowercased and trimmed. Each caller applies its own policy to
// the two lists: whether a lone exact hit wins, whether duplicates are
// ambiguous, whether a substring counts when an exact hit exists.
func SplitNameMatches[T any](items []T, selector string, name func(T) string) (exact, partial []T) {
	for _, item := range items {
		n := name(item)
		if strings.EqualFold(n, selector) {
			exact = append(exact, item)
		} else if strings.Contains(strings.ToLower(n), selector) {
			partial = append(partial, item)
		}
	}
	return exact, partial
}
