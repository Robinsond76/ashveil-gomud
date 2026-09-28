package util

import (
	"regexp"
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
// returned as is ("Garrick Vane"), as is one that already begins with an
// article ("the rat", "a stray dog") or not with a letter ("#3").
// Phase 29c.
func Article(name string) string {
	i := firstVisible(name)
	if i < 0 {
		return name
	}
	r, _ := utf8.DecodeRuneInString(name[i:])
	if !unicode.IsLower(r) {
		return name
	}
	for _, article := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(name[i:], article) {
			return name
		}
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

var (
	narrationTag  = regexp.MustCompile(`<[^>]*>`)
	narrationCaps = regexp.MustCompile(`\b[A-Z]{2,}\b`)
)

// NarrationVoiceProblem reports what, if anything, breaks the Phase 29c
// narration voice in one line of combat text: an exclamation mark or an
// ALL-CAPS word outside markup. "" when the line is fine.
func NarrationVoiceProblem(line string) string {
	visible := narrationTag.ReplaceAllString(line, "")
	if strings.Contains(visible, "!") {
		return "exclamation mark"
	}
	if w := narrationCaps.FindString(visible); w != "" {
		return "ALL-CAPS word " + w
	}
	return ""
}
