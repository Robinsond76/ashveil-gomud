package banter

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

var tags = regexp.MustCompile(`<[^>]+>`)

func plain(s string) string { return tags.ReplaceAllString(s, "") }

func said(name, personality, id, text string) Said {
	return Said{Name: name, Personality: personality, LineID: id, Text: text, Verb: verbFor(personality, id, text)}
}

func TestProseNeverRepeatsAShapeTwiceInARow(t *testing.T) {
	texts := []string{
		"I go where the road goes. It has not lied to me yet.",
		"Every wizard is a student. Some of us just admit it.",
		"Some days I am good, some days I am not. Mostly I am tired.",
		"A cold camp keeps a body honest.",
		"Is it just me, or does a meal taste twice as good out here?",
	}
	names := []string{"Maren", "Hild", "Brann"}
	for seed := 0; seed < 200; seed++ {
		var n Narrator
		var shapes []shape
		for i := 0; i < 4; i++ {
			n.Line(said(names[i%3], "stoic", fmt.Sprintf("l%d-%d", seed, i), texts[(seed+i)%len(texts)]))
			shapes = append(shapes, n.last)
		}
		for i := 1; i < len(shapes); i++ {
			if shapes[i] == shapes[i-1] {
				t.Fatalf("seed %d repeated shape %v: %v", seed, shapes[i], shapes)
			}
		}
	}
}

func TestProseUsesMoreThanOneShapeAndKeepsEveryWord(t *testing.T) {
	seen := map[shape]bool{}
	for i := 0; i < 100; i++ {
		var n Narrator
		text := "Every wizard is a student. Some of us just admit it."
		out := plain(n.Line(said("Maren", "wry", fmt.Sprint("x", i), text)))
		seen[n.last] = true
		if !strings.Contains(out, "Maren") {
			t.Fatalf("name missing: %q", out)
		}
		flat := strings.NewReplacer(`"`, "", ",", "", ".", "").Replace(out)
		for _, w := range strings.Fields("Every wizard is a student Some of us just admit it") {
			if !strings.Contains(flat, w) {
				t.Fatalf("lost %q in %q", w, out)
			}
		}
	}
	if len(seen) < 3 {
		t.Fatalf("expected varied shapes, got %v", seen)
	}
}

func TestProseNeverOpensWithNameSaysTwice(t *testing.T) {
	var n Narrator
	a := plain(n.Line(said("Hild", "grim", "a", "Keep the fire low.")))
	b := plain(n.Line(said("Brann", "cheerful", "b", "Cheerful as ever, are we.")))
	if a == "" || b == "" || strings.HasPrefix(a, "Hild mutters, \"") && strings.HasPrefix(b, "Brann laughs, \"") {
		t.Fatalf("both lines took the old template: %q / %q", a, b)
	}
}

func TestTwoVoicesNeverSayTheSameWords(t *testing.T) {
	var n Narrator
	first := plain(n.Say("Maren", "boastful", "rough", "Sleeping in dirt? I have standards.", MoodSour))
	second := plain(n.Say("Hild", "boastful", "rough", "Sleeping in dirt? I have standards.", MoodSour))
	if strings.Count(first+second, "standards") != 1 {
		t.Fatalf("the words were repeated: %q / %q", first, second)
	}
	if !strings.Contains(second, "Maren") {
		t.Fatalf("the second voice should answer to the first: %q", second)
	}
}

func TestQuoteFirstTurnsAFullStopIntoAComma(t *testing.T) {
	if got := closeQuote("A cold camp keeps a body honest."); got != "A cold camp keeps a body honest," {
		t.Fatal(got)
	}
	if got := closeQuote("Ready?"); got != "Ready?" {
		t.Fatal(got)
	}
}

func TestSplitNeedsAFullThoughtEachSide(t *testing.T) {
	for _, text := range []string{
		"Fine. Go.",
		"Sleeping in dirt? I have standards.",
		// Nor inside a list after a sentence break.
		"Look at us. Heroes, wanderers, and professional fire-watchers.",
	} {
		if a, b, _, ok := splitText(text); ok {
			t.Fatalf("split %q as %q / %q", text, a, b)
		}
	}
	// The tag never lands just before a punchline: it keeps the joke whole.
	a, b, sentence, ok := splitText("If I had known adventure meant this much walking, I would have asked for a horse. Oh, I did.")
	if !ok || sentence || b != "I would have asked for a horse. Oh, I did." {
		t.Fatal(a, b, sentence, ok)
	}
	a, b, sentence, ok = splitText("Gold is a poor god now. It will own you, in time.")
	if !ok || !sentence || a != "Gold is a poor god now." || b != "It will own you, in time." {
		t.Fatal(a, b, sentence, ok)
	}
	a, b, sentence, ok = splitText("Is it me, or is it getting colder?")
	if !ok || sentence || a != "Is it me" || b != "or is it getting colder?" {
		t.Fatal(a, b, sentence, ok)
	}
}

// looks are the ways one speaker turns to another.
var looks = regexp.MustCompile(`glances at|turns toward|looks across at|answers|sour look|no mind`)

func TestOnlyARepliesTurnsToTheLastSpeaker(t *testing.T) {
	texts := []string{
		"Another day on the road. I have had worse.",
		"The fire is good. That is enough.",
		"Do you think our luck will hold?",
		"Keep the fire low. There are eyes out there, I am sure of it.",
	}
	names := []string{"Odo", "Tamsin", "Ilse"}
	for seed := 0; seed < 300; seed++ {
		var n Narrator
		for i, text := range texts {
			s := said(names[i%3], Personalities[(seed+i)%len(Personalities)], fmt.Sprint(seed, "-", i), text)
			if out := plain(n.Line(s)); looks.MatchString(out) {
				t.Fatalf("seed %d: a line that answers nobody turns to someone: %q", seed, out)
			}
		}
	}
	turned := false
	for seed := 0; seed < 50 && !turned; seed++ {
		var n Narrator
		n.Line(said("Odo", "wry", fmt.Sprint("q", seed), "Is it me, or is it getting colder?"))
		reply := said("Tamsin", "stoic", fmt.Sprint("r", seed), "It is not you. Put on another layer.")
		reply.Reply = true
		turned = looks.MatchString(plain(n.Line(reply)))
	}
	if !turned {
		t.Fatal("a reply never turned to the one it answers")
	}
}

func TestGesturesFitTheMood(t *testing.T) {
	warm := map[string]bool{}
	for _, list := range beats {
		for _, b := range list {
			warm[b] = true
		}
	}
	for seed := 0; seed < 200; seed++ {
		for _, p := range Personalities {
			// A complaint never opens with a cheerful gesture, and agreeing
			// with one never claps a shoulder.
			var n Narrator
			first := plain(n.Say("Maren", p, fmt.Sprint("rough", seed), "Sleeping in dirt? I have standards.", MoodSour))
			echo := plain(n.Say("Hild", p, fmt.Sprint("rough", seed), "Sleeping in dirt? I have standards.", MoodSour))
			if strings.Contains(echo, "claps") || strings.Contains(echo, "grins") {
				t.Fatalf("a sour agreement reads warm: %q", echo)
			}
			for b := range warm {
				if sourOK(p, b) {
					continue
				}
				if strings.Contains(first, "Maren "+b+".") {
					t.Fatalf("a complaint opens with a plain gesture %q: %q", b, first)
				}
			}
			// Talk over the fallen never laughs, grins or quips.
			var g Narrator
			s := said("Pell", p, fmt.Sprint("fall", seed), "I cannot make a joke about it. Not about Odo. Not today.")
			s.Ctx = CtxFall
			out := plain(g.Line(s))
			for _, light := range []string{"laugh", "grin", "quips", "announces", "hums", "stretches"} {
				if strings.Contains(out, light) {
					t.Fatalf("grief reads light (%s): %q", light, out)
				}
			}
		}
	}
}

// sourOK reports whether a plain gesture is also a sour one for p.
func sourOK(p, b string) bool {
	for _, s := range sourBeats[p] {
		if s == b {
			return true
		}
	}
	return false
}

func TestAnExchangeMarksItsReplies(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	members := []Member{
		{ID: 1, Name: "Odo", Archetype: "wizard", Personality: "wry"},
		{ID: 2, Name: "Tamsin", Archetype: "ranger", Personality: "stoic"},
		{ID: 3, Name: "Hild", Archetype: "cleric", Personality: "devout"},
	}
	replies := 0
	for seed := int64(0); seed < 400; seed++ {
		said := p.Exchange(rand.New(rand.NewSource(seed)), Request{Contexts: []string{CtxCamp}, Members: members, Leader: "Robinson"})
		for i, s := range said {
			answers := p.lines[p.byID[s.LineID]].Reply
			want := i > 0 && answers != "" && answers == said[i-1].LineID
			if s.Reply != want {
				t.Fatalf("seed %d line %d (%s) Reply=%v, answers %q after %q", seed, i, s.LineID, s.Reply, answers, said[max(i-1, 0)].LineID)
			}
			if s.Reply {
				replies++
			}
		}
	}
	if replies == 0 {
		t.Fatal("no exchange had a reply")
	}
}

func TestAnExchangeNeverRepeatsALine(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	members := []Member{
		{ID: 1, Name: "Maren", Archetype: "warrior", Personality: "boastful"},
		{ID: 2, Name: "Hild", Archetype: "cleric", Personality: "boastful"},
		{ID: 3, Name: "Brann", Archetype: "ranger", Personality: "boastful"},
	}
	for seed := int64(0); seed < 300; seed++ {
		said := p.Exchange(rand.New(rand.NewSource(seed)), Request{Contexts: []string{CtxCamp}, Members: members, Leader: "Robinson", Recent: map[int]map[string]bool{}})
		ids := map[string]bool{}
		for _, s := range said {
			if s.LineID != "" && ids[s.LineID] {
				t.Fatalf("seed %d repeated %s", seed, s.LineID)
			}
			ids[s.LineID] = true
		}
	}
}

func TestAsideIsDim(t *testing.T) {
	if got := Aside(-2); !strings.Contains(got, "(loyalty -2)") || !strings.Contains(got, "black-bold") {
		t.Fatal(got)
	}
}
