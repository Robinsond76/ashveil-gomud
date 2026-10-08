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
	first := plain(n.Line(said("Maren", "boastful", "rough", "Sleeping in dirt? I have standards.")))
	second := plain(n.Line(said("Hild", "boastful", "rough", "Sleeping in dirt? I have standards.")))
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

func TestSplitNeedsTwoWordsEachSide(t *testing.T) {
	if _, _, _, ok := splitText("Fine. Go."); ok {
		t.Fatal("too short to split")
	}
	a, b, sentence, ok := splitText("Sleeping in dirt? I have standards.")
	if !ok || !sentence || a != "Sleeping in dirt?" || b != "I have standards." {
		t.Fatal(a, b, sentence, ok)
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
