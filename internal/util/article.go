package util

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// firstVisible is the byte index of the first rune outside <...> tags, or
// -1 when there is none.
func firstVisible(s string) int {
	inTag := false
	for i, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>' && inTag:
			inTag = false
		case !inTag:
			return i
		}
	}
	return -1
}

// Article puts "the " before a name whose first visible letter is
// lowercase ("the bandit captain"). A capitalised name is proper and is
// returned as is ("Garrick Vane"), as is one that already begins "the ".
// Phase 29c.
func Article(name string) string {
	i := firstVisible(name)
	if i < 0 {
		return name
	}
	r, _ := utf8.DecodeRuneInString(name[i:])
	if unicode.IsUpper(r) || strings.HasPrefix(name[i:], "the ") {
		return name
	}
	return "the " + name
}

// CapitalizeFirst upper-cases a line's first visible letter, skipping
// <...> tags. Phase 29c.
func CapitalizeFirst(line string) string {
	i := firstVisible(line)
	if i < 0 {
		return line
	}
	r, size := utf8.DecodeRuneInString(line[i:])
	return line[:i] + string(unicode.ToUpper(r)) + line[i+size:]
}
