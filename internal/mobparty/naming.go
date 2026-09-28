package mobparty

import (
	"fmt"
	"strings"
)

// Phase 32c: a group's name. Every enemy group is named when it forms and
// keeps that name while it stands (mobs.Mob.GroupName carries it). A name
// is either authored (a room's spawn list, or a module naming what it
// spawns) or generated here: a collective noun and the plural of the
// group's most common kind, "a band of ruffians". Players start a fight by
// naming a group (Keywords), never a member.

// DefaultNoun is the collective noun of a group whose members have none.
const DefaultNoun = "band"

// Naming is how a group is named and matched.
type Naming struct {
	Name     string // as shown mid-sentence: "a band of ruffians", "the Rat King's court", or a lone mob's own name
	Noun     string // the collective noun ("band"), "" for an authored or lone name
	Kind     string // the plural of the most common member ("ruffians"), "" for a lone mob
	Keyword  string // the word the room tells players to type: the kind, or an authored name's last word
	Desc     string // an authored description, if any
	Authored bool
	Solo     bool // a group of one, named by its only member
}

// NameGroup names a group from its members. A member's GroupName (set when
// the group formed) wins, so a group's name holds while members fall. A
// group of one with no name is its member. Otherwise the name is generated
// from the most common kind, a tie going to the toughest (highest EHP).
func NameGroup(members []MobSummary) Naming {
	if len(members) == 0 {
		return Naming{}
	}
	for _, m := range members {
		if m.GroupName != "" {
			return fromName(m.GroupName, members)
		}
	}
	if len(members) == 1 {
		return Naming{Name: members[0].Name, Keyword: strings.ToLower(members[0].Name), Solo: true}
	}
	return Generate(members)
}

// Generate makes a fresh name for members, ignoring any name they carry:
// "a band of ruffians" for a group whose most common member is a ruffian.
func Generate(members []MobSummary) Naming {
	count := map[string]int{}
	best := map[string]float64{}
	noun := map[string]string{}
	var order []string
	for _, m := range members {
		if _, ok := count[m.Name]; !ok {
			order = append(order, m.Name)
		}
		count[m.Name]++
		if m.EHP > best[m.Name] {
			best[m.Name] = m.EHP
		}
		if noun[m.Name] == "" && m.Noun != "" {
			noun[m.Name] = m.Noun
		}
	}
	kind := ""
	for _, k := range order {
		if kind == "" || count[k] > count[kind] || (count[k] == count[kind] && best[k] > best[kind]) {
			kind = k
		}
	}
	n := noun[kind]
	if n == "" {
		n = DefaultNoun
	}
	plural := Plural(kind)
	return Naming{
		Name:    fmt.Sprintf("%s %s of %s", Article(n), n, plural),
		Noun:    n,
		Kind:    plural,
		Keyword: strings.ToLower(plural),
	}
}

// fromName is the naming of a group that already carries a name: generated
// names ("a band of ruffians") keep their noun and kind, so they match the
// same words.
func fromName(name string, members []MobSummary) Naming {
	desc := ""
	for _, m := range members {
		if m.GroupDesc != "" {
			desc = m.GroupDesc
			break
		}
	}
	if art, rest, ok := strings.Cut(name, " "); ok && (art == "a" || art == "an") {
		if noun, kind, ok := strings.Cut(rest, " of "); ok && !strings.Contains(noun, " ") {
			return Naming{Name: name, Noun: noun, Kind: kind, Keyword: strings.ToLower(kind), Desc: desc}
		}
	}
	words := nameWords(name)
	kw := strings.ToLower(name)
	if len(words) > 0 {
		kw = words[len(words)-1]
	}
	return Naming{Name: name, Keyword: kw, Desc: desc, Authored: true}
}

// Keywords are the words and phrases that name the group: its noun, its
// kind, its name without an article, and each word of an authored name.
// A member's own (singular) name is not among them, unless the group is a
// lone mob.
func (n Naming) Keywords() []string {
	var out []string
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return
		}
		for _, have := range out {
			if have == s {
				return
			}
		}
		out = append(out, s)
	}
	add(n.Noun)
	add(n.Kind)
	add(StripArticle(n.Name))
	if n.Authored {
		for _, w := range nameWords(n.Name) {
			add(w)
		}
	}
	return out
}

// Matches reports whether search names the group: exactly one of its
// keywords (case aside).
func (n Naming) Matches(search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	search = StripArticle(search)
	for _, k := range n.Keywords() {
		if k == search {
			return true
		}
	}
	return false
}

// MatchesPrefix reports whether search begins one of the group's keywords
// (a player typing "ruff" for "ruffians"). Callers try exact names first.
func (n Naming) MatchesPrefix(search string) bool {
	search = StripArticle(strings.ToLower(strings.TrimSpace(search)))
	if len(search) < 2 {
		return false
	}
	for _, k := range n.Keywords() {
		if strings.HasPrefix(k, search) {
			return true
		}
	}
	return false
}

// StripArticle drops a leading "a ", "an ", or "the ".
func StripArticle(s string) string {
	lower := strings.ToLower(s)
	for _, art := range []string{"a ", "an ", "the "} {
		if strings.HasPrefix(lower, art) {
			return s[len(art):]
		}
	}
	return s
}

// nameWords are the lower-case words of a name worth matching, without
// articles, "of", and possessive endings.
func nameWords(name string) []string {
	var out []string
	for _, w := range strings.Fields(strings.ToLower(name)) {
		w = strings.TrimSuffix(strings.TrimSuffix(w, "'s"), "'")
		switch w {
		case "", "a", "an", "the", "of":
			continue
		}
		out = append(out, w)
	}
	return out
}

// WithOrdinal names the nth group of the same name in a room (n from 1):
// "a band of ruffians", "a second band of ruffians"; an authored name
// "the Rat King's court (second)".
func WithOrdinal(name string, n int) string {
	if n <= 1 {
		return name
	}
	ord := Ordinal(n)
	if art, rest, ok := strings.Cut(name, " "); ok && (art == "a" || art == "an") {
		return fmt.Sprintf("a %s %s", ord, rest)
	}
	return fmt.Sprintf("%s (%s)", name, ord)
}

// Ordinal is "second" for 2, up to "tenth", then "12th" style.
func Ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"}
	if n >= 1 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%dth", n)
}

// CountWord is "two" for 2, up to "ten", then digits.
func CountWord(n int) string {
	words := []string{"no", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%d", n)
}

// Article is "an" before a vowel sound (by spelling), else "a".
func Article(word string) string {
	if word == "" {
		return "a"
	}
	switch strings.ToLower(word)[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}

// Capitalize upper-cases the first letter.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ListKinds lists names by kind, in first-seen order: "two ruffians, a
// cutpurse, and a rat".
func ListKinds(names []string) string {
	count := map[string]int{}
	var order []string
	for _, n := range names {
		if _, ok := count[n]; !ok {
			order = append(order, n)
		}
		count[n]++
	}
	parts := make([]string, len(order))
	for i, n := range order {
		if count[n] == 1 {
			parts[i] = Article(n) + " " + n
		} else {
			parts[i] = CountWord(count[n]) + " " + Plural(n)
		}
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}

// irregular plurals, by the last word of a name.
var irregular = map[string]string{
	"man": "men", "woman": "women", "child": "children", "mouse": "mice", "louse": "lice",
	"goose": "geese", "foot": "feet", "tooth": "teeth", "ox": "oxen", "person": "people",
	"sheep": "sheep", "deer": "deer", "fish": "fish", "moose": "moose", "bison": "bison",
	"elf": "elves", "wolf": "wolves", "thief": "thieves", "dwarf": "dwarves", "knife": "knives",
	"leaf": "leaves", "self": "selves", "half": "halves", "calf": "calves", "loaf": "loaves",
	"wife": "wives", "life": "lives", "scarf": "scarves", "werewolf": "werewolves",
	"undead": "undead", "cactus": "cacti", "fungus": "fungi", "octopus": "octopuses",
}

// Plural pluralises a name by its last word: "bandit cutthroat" is
// "bandit cutthroats", "dire wolf" is "dire wolves".
func Plural(name string) string {
	if name == "" {
		return name
	}
	head, last := "", name
	if i := strings.LastIndex(name, " "); i >= 0 {
		head, last = name[:i+1], name[i+1:]
	}
	return head + pluralWord(last)
}

func pluralWord(w string) string {
	lower := strings.ToLower(w)
	if p, ok := irregular[lower]; ok {
		if w != lower { // keep a leading capital
			return Capitalize(p)
		}
		return p
	}
	// compounds: "guardsman" -> "guardsmen" (and "-woman" -> "-women")
	if strings.HasSuffix(lower, "man") && len(lower) > 3 {
		return w[:len(w)-3] + "men"
	}
	n := len(lower)
	switch {
	case strings.HasSuffix(lower, "ss"), strings.HasSuffix(lower, "sh"), strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "s"):
		return w + "es"
	case n > 1 && lower[n-1] == 'y' && !strings.ContainsRune("aeiou", rune(lower[n-2])):
		return w[:n-1] + "ies"
	}
	return w + "s"
}
