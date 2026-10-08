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
	shapeBeat                // Name rolls a stiff shoulder. "..."
	shapeSplit               // "...," Name says, "..."
	shapeGlance              // Name glances at Other. "..." (a reply)
	shapeAnswer              // "...," Name answers. (a reply, or a last word)
)

// Moods a line is said in. A gesture or a nod must fit the words beside
// it: nobody claps a shoulder while complaining, or smiles over the fallen.
const (
	MoodWarm  = 1  // pleased: an opinion that approves
	MoodPlain = 0  // ordinary talk
	MoodSour  = -1 // put out: an opinion that disapproves
)

// beats are small gestures that fit any place a company talks (a fire, a
// road, a won fight), by temperament. None names a thing that may be
// absent, a body part that needs a pronoun, or a feeling at odds with
// ordinary talk.
var beats = map[string][]string{
	"stoic":    {"rolls a stiff shoulder", "checks the straps on a pack", "tests the edge of a blade", "sits still a moment", "looks out past the others"},
	"cheerful": {"knocks the dirt from a boot", "hums a few notes", "stretches both arms", "leans in", "shakes out a cloak"},
	"grim":     {"wipes down a blade", "scowls at nothing", "spits", "works a knot from a strap", "stares at the ground"},
	"boastful": {"flexes a sore hand", "stands a little taller", "rubs a scuff from a buckle", "plants both feet", "taps the hilt of a weapon"},
	"wry":      {"raises an eyebrow", "smiles thinly", "snorts softly", "shrugs", "picks at a loose thread"},
	"devout":   {"touches a holy charm", "is quiet a moment", "lets out a slow breath", "traces a sign in the air", "folds both hands"},
}

var defaultBeats = []string{"shifts on the spot", "looks around", "coughs"}

// sourBeats open a line said in displeasure.
var sourBeats = map[string][]string{
	"stoic":    {"frowns", "says nothing for a breath"},
	"cheerful": {"stops smiling", "pulls a face"},
	"grim":     {"spits", "scowls"},
	"boastful": {"snorts", "folds both arms"},
	"wry":      {"sighs", "rolls both eyes"},
	"devout":   {"looks away", "lets out a slow breath"},
}

// griefBeats open a line said over a fallen comrade, whatever the temper.
var griefBeats = []string{"stares at the ground", "lets out a slow breath", "is quiet a moment", "rubs a tired face"}

// glances turn a reply toward the one it answers.
var glances = []string{"glances at %s", "turns toward %s", "looks across at %s"}

// echoes are how a voice agrees without repeating what was just said, by
// temperament and mood.
var echoes = map[int]map[string]string{
	MoodWarm: {
		"stoic":    "gives %s a short nod",
		"cheerful": "grins at %s and nods along",
		"grim":     "grunts, agreeing with %s",
		"boastful": "claps %s on the shoulder",
		"wry":      "nods at %s, for once agreeing",
		"devout":   "nods along with %s",
	},
	MoodSour: {
		"stoic":    "gives %s a short nod",
		"cheerful": "sighs and nods along with %s",
		"grim":     "grunts, agreeing with %s",
		"boastful": "snorts, agreeing with %s",
		"wry":      "says %s has the right of it",
		"devout":   "nods gravely with %s",
	},
}

// Narrator sets spoken lines down one at a time. The zero value is ready.
type Narrator struct {
	n        int
	last     shape
	lastName string
	lastMood int
	spoke    map[string]bool   // who has spoken already
	gestures map[string]bool   // gestures used already
	heard    map[string]string // spoken text -> who said it first
}

func name(n string) string { return `<ansi fg="cyan">` + n + `</ansi>` }

func pickBy(seed string, n int) int {
	h := fnv.New32a()
	h.Write([]byte(seed))
	return int(h.Sum32() % uint32(n))
}

// moodOf is the mood a line is said in: its own, or grief over the fallen.
func moodOf(s Said) int {
	if s.Mood != MoodPlain {
		return s.Mood
	}
	if s.Ctx == CtxFall {
		return MoodSour
	}
	return MoodPlain
}

// actionVerbs are delivered verbs that are not speech: "Pell grins" can
// open a line as a gesture, but nobody grins words.
var actionVerbs = map[string]string{"grins": "with a grin", "laughs": "with a laugh"}

// lightVerbs do not fit a sour or grieving line.
var lightVerbs = map[string]bool{"grins": true, "laughs": true, "quips": true, "announces": true}

// Line is one voice, set down as a sentence or two.
func (n *Narrator) Line(s Said) string {
	text := strings.TrimSpace(s.Text)
	if n.heard == nil {
		n.heard = map[string]string{}
		n.spoke = map[string]bool{}
		n.gestures = map[string]bool{}
	}
	mood := moodOf(s)
	key := strings.ToLower(text)
	if first, said := n.heard[key]; said && first != s.Name {
		return n.finish(shapeBeat, s.Name, mood, name(s.Name)+" "+fmt.Sprintf(echoOf(s.Personality, mood), name(first))+".")
	}
	n.heard[key] = s.Name

	seed := s.LineID + "|" + text
	verb := s.Verb
	if verb == "" {
		verb = "says"
	}
	if mood == MoodSour && lightVerbs[verb] {
		verb = "says"
	}
	// A reply answers the line before it; an opinion that crosses the one
	// before it turns on that speaker. Anything else stands on its own:
	// glancing at someone, then saying something unrelated, reads wrong.
	turnsOn := n.n > 0 && n.lastName != "" && n.lastName != s.Name &&
		(s.Reply || mood != MoodPlain && n.lastMood != MoodPlain && mood != n.lastMood)
	lastWord := n.n > 0 && n.lastName != s.Name && n.spoke[s.Name]
	var choices []shape
	switch {
	case turnsOn:
		choices = []shape{shapeGlance, shapeAnswer, shapeQuote, shapeSplit, shapeBeat}
	case lastWord:
		choices = []shape{shapeAnswer, shapeQuote, shapeBeat, shapeSplit}
	default:
		choices = []shape{shapeQuote, shapeBeat, shapeSplit, shapeTag}
	}
	first, rest, sentence, splittable := splitText(text)
	start := pickBy(seed, len(choices))
	var chosen shape
	for i := 0; i < len(choices); i++ {
		c := choices[(start+i)%len(choices)]
		if c == n.last && n.n > 0 {
			continue
		}
		if c == shapeSplit && !splittable {
			continue
		}
		if c == shapeAnswer && !s.Reply && !lastWord {
			continue
		}
		chosen = c
		break
	}
	nm := name(s.Name)
	speech := verb
	if _, ok := actionVerbs[verb]; ok {
		speech = "says"
	}
	var out string
	switch chosen {
	case shapeSplit:
		if sentence {
			v := speech
			if strings.HasSuffix(first, "?") {
				v = "asks"
			}
			out = fmt.Sprintf(`"%s" %s %s. "%s"`, closeQuote(first), nm, v, rest)
		} else {
			out = fmt.Sprintf(`"%s," %s %s, "%s"`, first, nm, speech, rest)
		}
	case shapeQuote:
		if with, ok := actionVerbs[verb]; ok {
			out = fmt.Sprintf(`"%s" %s says %s.`, closeQuote(text), nm, with)
		} else {
			out = fmt.Sprintf(`"%s" %s %s.`, closeQuote(text), nm, verb)
		}
	case shapeAnswer:
		v := "adds"
		switch {
		case strings.HasSuffix(text, "?"):
			v = "asks"
		case s.Reply:
			v = "answers"
		}
		out = fmt.Sprintf(`"%s" %s %s.`, closeQuote(text), nm, v)
	case shapeGlance:
		out = fmt.Sprintf(`%s %s. "%s"`, nm, glanceAt(s, mood, n.lastMood, seed, name(n.lastName)), text)
	case shapeBeat:
		out = fmt.Sprintf(`%s %s. "%s"`, nm, n.beat(s.Personality, mood, s.Ctx, seed), text)
	default:
		if _, ok := actionVerbs[verb]; ok {
			out = fmt.Sprintf(`%s %s. "%s"`, nm, verb, text)
		} else {
			out = fmt.Sprintf(`%s %s, "%s"`, nm, verb, text)
		}
	}
	return n.finish(chosen, s.Name, mood, out)
}

func (n *Narrator) finish(sh shape, who string, mood int, out string) string {
	n.last = sh
	n.lastName = who
	n.lastMood = mood
	n.spoke[who] = true
	n.n++
	return out
}

// glanceAt is how a speaker turns to the one before: a plain look for a
// reply, a sour look when crossing someone pleased, and a shrug past
// someone displeased.
func glanceAt(s Said, mood, lastMood int, seed, other string) string {
	if !s.Reply && mood != lastMood {
		if mood == MoodSour {
			return fmt.Sprintf("gives %s a sour look", other)
		}
		return fmt.Sprintf("pays %s no mind", other)
	}
	return fmt.Sprintf(glances[pickBy(seed, len(glances))], other)
}

// beat is a gesture for the speaker, not one already used in this run.
func (n *Narrator) beat(personality string, mood int, ctx, seed string) string {
	g := pickBeat(personality, mood, ctx, seed)
	for i := 0; n.gestures[g] && i < 8; i++ {
		g = pickBeat(personality, mood, ctx, fmt.Sprint(seed, i))
	}
	n.gestures[g] = true
	return g
}

func pickBeat(personality string, mood int, ctx, seed string) string {
	list := beats[personality]
	switch {
	case ctx == CtxFall:
		list = griefBeats
	case mood == MoodSour && len(sourBeats[personality]) > 0:
		list = sourBeats[personality]
	}
	if len(list) == 0 {
		list = defaultBeats
	}
	return list[pickBy(seed+"|beat", len(list))]
}

func echoOf(personality string, mood int) string {
	if mood == MoodSour {
		if e, ok := echoes[MoodSour][personality]; ok {
			return e
		}
	} else if e, ok := echoes[MoodWarm][personality]; ok {
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
// sentence break, else at the first comma. A tag between sentences needs
// a full thought on each side (four words), so it never lands just before
// a short punchline ("...a horse," Odo says. "Oh, I did."); a comma split
// keeps two words on each side.
func splitText(text string) (first, rest string, sentence, ok bool) {
	words := func(s string) int { return len(strings.Fields(s)) }
	for i := 0; i+1 < len(text); i++ {
		c := text[i]
		if (c == '.' || c == '?' || c == '!') && text[i+1] == ' ' {
			a, b := text[:i+1], strings.TrimSpace(text[i+2:])
			if strings.HasSuffix(a, "..") || words(a) < 4 || words(b) < 4 {
				continue
			}
			return a, b, true, true
		}
	}
	for i := 0; i+1 < len(text); i++ {
		if text[i] == ',' && text[i+1] == ' ' {
			a, b := text[:i], strings.TrimSpace(text[i+2:])
			// A comma split holds one clause: not after a sentence break
			// ("Look at us. Heroes,"), and never inside a list.
			if words(a) < 2 || words(b) < 2 || strings.ContainsAny(a, ".?!") {
				return "", "", false, false
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

// Say renders one line for a speaker outside an exchange (an opinion), in
// the given mood (MoodWarm for approval, MoodSour for disapproval).
func (n *Narrator) Say(name, personality, id, text string, mood int) string {
	return n.Line(Said{Name: name, Verb: verbFor(personality, id, text), Text: text, LineID: id, Personality: personality, Mood: mood})
}
