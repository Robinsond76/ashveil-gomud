package banter

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Prose: companion speech is written like a page of a book, not a log of
// "Name says, ...". A Narrator takes spoken lines one after another and
// varies how each is set down: the quote first, a short gesture before it,
// a speech tag in the middle, or a glance at the one who spoke before. It
// never uses one shape twice running, and it never lets two voices say the
// same words (the second nods along instead). The shape of a line is fixed
// by the line and its place, so a line always reads the same way there.

type shape int

const (
	shapeTag    shape = iota // Name says, "..."
	shapeQuote               // "...," Name says.
	shapeBeat                // Name turns a stick in the embers. "..."
	shapeSplit               // "...," Name says, "..."
	shapeGlance              // Name glances at Other. "..."
	shapeAnswer              // "...," Name answers.
)

// beats are small gestures that fit any place a company talks (a fire, a
// road, a won fight), by temperament. None names a thing that may be
// absent.
var beats = map[string][]string{
	"stoic":    {"rolls a stiff shoulder", "checks the straps on a pack", "keeps watching the dark"},
	"cheerful": {"stretches out and smiles", "knocks the dirt from a boot", "tilts a face to the air"},
	"grim":     {"scowls at the ground", "wipes down a blade", "spits and looks away"},
	"boastful": {"squares the shoulders", "flexes a sore hand", "stands a little taller"},
	"wry":      {"raises an eyebrow", "leans back on an elbow", "rubs the bridge of a nose"},
	"devout":   {"touches a charm at the throat", "bows the head a moment", "lets out a slow breath"},
}

var defaultBeats = []string{"shifts where seated", "glances around", "clears a throat"}

var glances = []string{"glances at %s", "turns toward %s", "looks across at %s"}

// echoes are how a voice agrees without repeating what was just said.
var echoes = map[string]string{
	"stoic":    "gives %s a short nod",
	"cheerful": "grins at %s and nods along",
	"grim":     "grunts, agreeing with %s",
	"boastful": "claps %s on the shoulder",
	"wry":      "lifts a brow at %s, agreeing",
	"devout":   "bows to %s in agreement",
}

// Narrator sets spoken lines down one at a time. The zero value is ready.
type Narrator struct {
	n        int
	last     shape
	lastName string
	heard    map[string]string // spoken text -> who said it first
}

func name(n string) string { return `<ansi fg="cyan">` + n + `</ansi>` }

func pickBy(seed string, n int) int {
	h := fnv.New32a()
	h.Write([]byte(seed))
	return int(h.Sum32() % uint32(n))
}

// Line is one voice, set down as a sentence or two.
func (n *Narrator) Line(s Said) string {
	text := strings.TrimSpace(s.Text)
	if n.heard == nil {
		n.heard = map[string]string{}
	}
	key := strings.ToLower(text)
	if first, said := n.heard[key]; said && first != s.Name {
		return n.finish(shapeBeat, s.Name, name(s.Name)+" "+fmt.Sprintf(echoOf(s.Personality), name(first))+".")
	}
	n.heard[key] = s.Name

	seed := s.LineID + "|" + text
	verb := s.Verb
	if verb == "" {
		verb = "says"
	}
	var choices []shape
	if n.n == 0 || n.lastName == s.Name {
		choices = []shape{shapeQuote, shapeBeat, shapeSplit, shapeTag}
	} else {
		choices = []shape{shapeGlance, shapeAnswer, shapeQuote, shapeSplit, shapeBeat}
	}
	first, rest, sentence, splittable := splitText(text)
	start := pickBy(seed, len(choices))
	var chosen shape
	for i := 0; i < len(choices); i++ {
		c := choices[(start+i)%len(choices)]
		if c == n.last && len(choices) > 1 && n.n > 0 {
			continue
		}
		if c == shapeSplit && !splittable {
			continue
		}
		chosen = c
		break
	}
	nm := name(s.Name)
	var out string
	switch chosen {
	case shapeSplit:
		if sentence {
			v := verb
			if strings.HasSuffix(first, "?") {
				v = "asks"
			}
			out = fmt.Sprintf(`"%s" %s %s. "%s"`, closeQuote(first), nm, v, rest)
		} else {
			out = fmt.Sprintf(`"%s," %s %s, "%s"`, first, nm, verb, rest)
		}
	case shapeQuote:
		out = fmt.Sprintf(`"%s" %s %s.`, closeQuote(text), nm, verb)
	case shapeAnswer:
		v := "adds"
		if n.n > 0 && !strings.HasSuffix(text, "?") {
			v = []string{"answers", "adds"}[pickBy(seed, 2)]
		} else if strings.HasSuffix(text, "?") {
			v = "asks"
		}
		out = fmt.Sprintf(`"%s" %s %s.`, closeQuote(text), nm, v)
	case shapeGlance:
		g := glances[pickBy(seed, len(glances))]
		if n.lastName == "" || n.lastName == s.Name {
			g = ""
		}
		if g == "" {
			out = fmt.Sprintf(`%s %s. "%s"`, nm, pickBeat(s.Personality, seed), text)
		} else {
			out = fmt.Sprintf(`%s %s. "%s"`, nm, fmt.Sprintf(g, name(n.lastName)), text)
		}
	case shapeBeat:
		out = fmt.Sprintf(`%s %s. "%s"`, nm, pickBeat(s.Personality, seed), text)
	default:
		out = fmt.Sprintf(`%s %s, "%s"`, nm, verb, text)
	}
	return n.finish(chosen, s.Name, out)
}

func (n *Narrator) finish(sh shape, who, out string) string {
	n.last = sh
	n.lastName = who
	n.n++
	return out
}

func pickBeat(personality, seed string) string {
	list := beats[personality]
	if len(list) == 0 {
		list = defaultBeats
	}
	return list[pickBy(seed+"|beat", len(list))]
}

func echoOf(personality string) string {
	if e, ok := echoes[personality]; ok {
		return e
	}
	return "nods along with %s"
}

// closeQuote ends a spoken line so a speech tag can follow it: a final
// full stop becomes a comma; a question or exclamation keeps its mark.
func closeQuote(text string) string {
	t := strings.TrimSpace(text)
	switch {
	case strings.HasSuffix(t, "..."), strings.HasSuffix(t, "?"), strings.HasSuffix(t, "!"), strings.HasSuffix(t, ","):
		return t
	case strings.HasSuffix(t, "."):
		return strings.TrimSuffix(t, ".") + ","
	}
	return t + ","
}

// splitText finds a place to set a speech tag inside a line: at the first
// sentence break, else at the first comma, keeping two words on each side.
func splitText(text string) (first, rest string, sentence, ok bool) {
	words := func(s string) int { return len(strings.Fields(s)) }
	for i := 0; i+1 < len(text); i++ {
		c := text[i]
		if (c == '.' || c == '?' || c == '!') && text[i+1] == ' ' {
			a, b := text[:i+1], strings.TrimSpace(text[i+2:])
			if strings.HasSuffix(a, "..") || words(a) < 2 || words(b) < 2 {
				continue
			}
			return a, b, true, true
		}
	}
	for i := 0; i+1 < len(text); i++ {
		if text[i] == ',' && text[i+1] == ' ' {
			a, b := text[:i], strings.TrimSpace(text[i+2:])
			if words(a) < 2 || words(b) < 2 {
				continue
			}
			return a, b, false, true
		}
	}
	return "", "", false, false
}

// Aside is a loyalty change set quietly at the end of a line.
func Aside(delta int) string {
	return fmt.Sprintf(` <ansi fg="black-bold">(loyalty %+d)</ansi>`, delta)
}

// Say renders one line for a speaker outside an exchange (an opinion).
func (n *Narrator) Say(name, personality, id, text string) string {
	return n.Line(Said{Name: name, Verb: verbFor(personality, id, text), Text: text, LineID: id, Personality: personality})
}
