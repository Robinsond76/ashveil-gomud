package banter

import (
	"math/rand"
	"strings"
	"testing"
)

func mustPool(t *testing.T) *Pool {
	t.Helper()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestShippedPoolIsLargeAndConsistent(t *testing.T) {
	p := mustPool(t)
	if p.Len() < 300 {
		t.Fatalf("pool has %d lines, want at least 300", p.Len())
	}
	for _, l := range p.Lines() {
		for _, c := range l.Ctx {
			switch c {
			case CtxCamp, CtxRested, CtxWin, CtxClose, CtxFall, CtxFlawless:
			default:
				t.Errorf("%q: unknown context %q", l.Text, c)
			}
		}
		if l.Reply == "" && len(l.Ctx) == 0 {
			t.Errorf("%q has no context and is not a reply", l.Text)
		}
		for _, a := range l.Arch {
			if GroupOf(a) == "" && !isGroup(a) && !isArchetype(a) {
				t.Errorf("%q: unknown archetype tag %q", l.Text, a)
			}
		}
		if strings.Contains(l.Text, "{") {
			for _, ph := range []string{"{other}", "{fallen}", "{leader}", "{name}"} {
				l.Text = strings.ReplaceAll(l.Text, ph, "")
			}
			if strings.ContainsAny(l.Text, "{}") {
				t.Errorf("%q has an unknown placeholder", l.Text)
			}
		}
	}
}

func isGroup(a string) bool {
	switch a {
	case "martial", "arcane", "faith", "skirmisher":
		return true
	}
	return false
}

func isArchetype(a string) bool { _, ok := groups[a]; return ok }

// Every personality, group, and alignment bucket has enough to say in each
// context, so exchanges rarely fall back and repeats are rare.
func TestEveryTagHasLinesInEveryContext(t *testing.T) {
	p := mustPool(t)
	count := func(ctx string, m Member) int {
		n := 0
		for _, l := range p.Lines() {
			if inContext(l, ctx) && l.matches(m) && l.Reply == "" {
				n++
			}
		}
		return n
	}
	for _, ctx := range []string{CtxCamp, CtxRested, CtxWin, CtxClose, CtxFall, CtxFlawless} {
		min := 3
		if ctx == CtxCamp {
			min = 12
		}
		for _, pers := range Personalities {
			if n := count(ctx, Member{Personality: pers, Archetype: "warrior", Alignment: 0}); n < min {
				t.Errorf("%s/%s: %d lines, want >= %d", ctx, pers, n, min)
			}
		}
		for arch := range groups {
			if n := count(ctx, Member{Personality: "stoic", Archetype: arch, Alignment: 0}); n < min {
				t.Errorf("%s/%s: %d lines, want >= %d", ctx, arch, n, min)
			}
		}
		for _, a := range []int{50, 0, -50} {
			if n := count(ctx, Member{Personality: "stoic", Archetype: "warrior", Alignment: a}); n < min {
				t.Errorf("%s/alignment %d: %d lines, want >= %d", ctx, a, n, min)
			}
		}
	}
}

func team() []Member {
	return []Member{
		{ID: 1, Name: "Hild Marrow", Archetype: "warrior", Personality: "stoic", Alignment: 40},
		{ID: 2, Name: "Brann", Archetype: "wizard", Personality: "wry", Alignment: 0},
		{ID: 3, Name: "Sister Ovel", Archetype: "cleric", Personality: "devout", Alignment: 70},
	}
}

func TestExchangeShapes(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(7))
	sizes := map[int]int{}
	for i := 0; i < 300; i++ {
		said := p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: team(), Leader: "Captain", Fallen: "Hild"})
		if len(said) < 2 || len(said) > 4 {
			t.Fatalf("exchange of %d lines: %+v", len(said), said)
		}
		sizes[len(said)]++
		if said[0].Member == said[1].Member {
			t.Fatalf("the same member answered themself: %+v", said)
		}
		for _, s := range said {
			if strings.Contains(s.Text, "{") || s.Text == "" || s.Verb == "" {
				t.Fatalf("bad line %+v", s)
			}
		}
	}
	if sizes[2] == 0 || sizes[3] == 0 {
		t.Errorf("sizes %v: want both 2 and 3 line exchanges", sizes)
	}
}

func TestPairNeverSpeaksAlone(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(1))
	if p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: team()[:1]}) != nil {
		t.Error("one member made an exchange")
	}
}

func TestRepliesFollowTheirPrompt(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(3))
	replied := 0
	for i := 0; i < 400; i++ {
		said := p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: team()})
		for j, s := range said {
			line := p.lines[p.byID[s.LineID]]
			if line.Reply == "" {
				continue
			}
			replied++
			if j == 0 {
				t.Fatalf("a reply opened the exchange: %+v", s)
			}
			if said[j-1].LineID != line.Reply {
				t.Fatalf("reply %q follows %q, wants %q", s.LineID, said[j-1].LineID, line.Reply)
			}
		}
	}
	if replied == 0 {
		t.Error("no call-and-response happened")
	}
}

func TestBattleContextFallsBackToWin(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(5))
	said := p.Exchange(rng, Request{Contexts: []string{"nonsense", CtxWin}, Members: team()})
	if len(said) < 2 {
		t.Fatal("no fallback exchange")
	}
}

func TestFallLinesNameTheFallen(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(11))
	named := false
	for i := 0; i < 100; i++ {
		for _, s := range p.Exchange(rng, Request{Contexts: []string{CtxFall}, Members: team(), Fallen: "Corin", Leader: "Captain"}) {
			if strings.Contains(s.Text, "Corin") {
				named = true
			}
		}
	}
	if !named {
		t.Error("no fall line named the fallen member")
	}
}

func TestTagsGateLines(t *testing.T) {
	p, err := NewPool([]Line{
		{Text: "evil only", Ctx: []string{CtxCamp}, Align: []string{"evil"}},
		{Text: "general", Ctx: []string{CtxCamp}},
		{Text: "stoic only", Ctx: []string{CtxCamp}, Pers: []string{"stoic"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	good := []Member{
		{ID: 1, Name: "A", Archetype: "warrior", Personality: "wry", Alignment: 50},
		{ID: 2, Name: "B", Archetype: "rogue", Personality: "wry", Alignment: 50},
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		for _, s := range p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: good}) {
			if s.Text != "general" {
				t.Fatalf("a gated line was said: %q", s.Text)
			}
		}
	}
}

func TestRecentLinesAreNotRepeated(t *testing.T) {
	p, err := NewPool([]Line{
		{ID: "a", Text: "one", Ctx: []string{CtxCamp}},
		{ID: "b", Text: "two", Ctx: []string{CtxCamp}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ms := []Member{{ID: 1, Name: "A", Personality: "wry"}, {ID: 2, Name: "B", Personality: "wry"}}
	recent := map[int]map[string]bool{1: {"a": true}, 2: {"a": true}}
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 30; i++ {
		for _, s := range p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: ms, Recent: recent}) {
			if s.LineID == "a" {
				t.Fatal("a recent line was repeated")
			}
		}
	}
	// With everything used up, there is nothing to say.
	recent = map[int]map[string]bool{1: {"a": true, "b": true}, 2: {"a": true, "b": true}}
	if p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: ms, Recent: recent}) != nil {
		t.Error("exchange with every line used")
	}
}

func TestNewPoolRejectsBadData(t *testing.T) {
	cases := map[string][]Line{
		"no text":   {{Ctx: []string{"camp"}}},
		"bad pers":  {{Text: "x", Pers: []string{"grumpy"}}},
		"bad align": {{Text: "x", Align: []string{"lawful"}}},
		"bad reply": {{Text: "x", Reply: "nope"}},
		"dup id":    {{ID: "a", Text: "x"}, {ID: "a", Text: "y"}},
	}
	for name, lines := range cases {
		if _, err := NewPool(lines); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestBucketAndEnabled(t *testing.T) {
	for a, want := range map[int]string{100: "good", 20: "good", 19: "neutral", 0: "neutral", -19: "neutral", -20: "evil", -100: "evil"} {
		if got := Bucket(a); got != want {
			t.Errorf("Bucket(%d) = %s, want %s", a, got, want)
		}
	}
	if !Enabled(nil) || !Enabled(true) || Enabled(false) {
		t.Error("Enabled: want on unless false")
	}
}

func TestShortName(t *testing.T) {
	if got := (Member{Name: "Hild Marrow"}).short(); got != "Hild" {
		t.Errorf("got %q", got)
	}
	if got := (Member{Name: "a dark acolyte"}).short(); got != "a dark acolyte" {
		t.Errorf("got %q", got)
	}
}

// Repeats within a long camp session stay rare when the caller records
// what was said.
func TestRepeatsAreRareWithAWindow(t *testing.T) {
	p := mustPool(t)
	rng := rand.New(rand.NewSource(9))
	recent := map[int]map[string]bool{}
	var order = map[int][]string{}
	repeats, total := 0, 0
	for i := 0; i < 200; i++ {
		said := p.Exchange(rng, Request{Contexts: []string{CtxCamp}, Members: team(), Recent: recent, Leader: "Captain"})
		for _, s := range said {
			total++
			if recent[s.Member][s.LineID] {
				repeats++
			}
			if recent[s.Member] == nil {
				recent[s.Member] = map[string]bool{}
			}
			recent[s.Member][s.LineID] = true
			order[s.Member] = append(order[s.Member], s.LineID)
			if len(order[s.Member]) > 12 {
				delete(recent[s.Member], order[s.Member][0])
				order[s.Member] = order[s.Member][1:]
			}
		}
	}
	if repeats != 0 {
		t.Errorf("%d of %d lines repeated inside the window", repeats, total)
	}
}
