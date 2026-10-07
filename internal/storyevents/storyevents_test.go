package storyevents

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func goodEvent() Event {
	return Event{
		ID: "test-event", Title: "Test",
		Triggers: []Trigger{{Kind: TriggerRoom, Room: 5}},
		Pages: map[string]Page{
			"start": {Text: "A page.", Choices: []Choice{
				{Label: "Go on", Next: "second"},
				{Label: "Leave"},
			}},
			"second": {Text: "Another.", Choices: []Choice{{Label: "End"}}},
		},
	}
}

func problems(e Event) string { return strings.Join(e.Validate(Lookups{}), "\n") }

func TestAGoodEventValidates(t *testing.T) {
	assert.Empty(t, goodEvent().Validate(Lookups{}))
}

func TestValidationRejectsEachBrokenRule(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Event)
		want   string
	}{
		{"bad id", func(e *Event) { e.ID = "Bad Id" }, "id"},
		{"no title", func(e *Event) { e.Title = "" }, "title"},
		{"no trigger", func(e *Event) { e.Triggers = nil }, "no triggers"},
		{"unknown trigger", func(e *Event) { e.Triggers[0].Kind = "sunrise" }, "unknown kind"},
		{"room trigger without a room", func(e *Event) { e.Triggers[0].Room = 0 }, "room"},
		{"arrival without a zone", func(e *Event) { e.Triggers[0] = Trigger{Kind: TriggerArrival} }, "needs a zone"},
		{"chance over 100", func(e *Event) { e.Triggers[0].Chance = 101 }, "chance"},
		{"missing start", func(e *Event) { e.Start = "nope" }, "start page"},
		{"unreachable page", func(e *Event) {
			e.Pages["island"] = Page{Text: "x", Choices: []Choice{{Label: "End"}}}
		}, "cannot be reached"},
		{"loop", func(e *Event) {
			e.Pages["second"] = Page{Text: "Back.", Choices: []Choice{{Label: "Again", Next: "start"}, {Label: "End"}}}
		}, "loop"},
		{"missing next", func(e *Event) { e.Pages["start"].Choices[0].Next = "ghost" }, "does not exist"},
		{"empty text", func(e *Event) { e.Pages["second"] = Page{Choices: []Choice{{Label: "End"}}} }, "text"},
		{"no choices", func(e *Event) { e.Pages["second"] = Page{Text: "x"} }, "choices"},
		{"no way out", func(e *Event) {
			e.Pages["second"] = Page{Text: "x", Choices: []Choice{{Label: "Climb", Require: Requirement{Classes: []string{"rogue"}}}}}
		}, "always take"},
		{"costly way out", func(e *Event) {
			e.Pages["second"] = Page{Text: "x", Choices: []Choice{{Label: "Pay", Do: []Outcome{{Kind: OutcomeGold, Amount: -5}}}}}
		}, "always take"},
		{"risky way out", func(e *Event) {
			e.Pages["second"] = Page{Text: "x", Choices: []Choice{{Label: "Dare", Risk: &Risk{Pct: 10}, FailText: "no"}}}
		}, "always take"},
		{"risk without fail", func(e *Event) {
			e.Pages["start"].Choices[1].Risk = &Risk{Pct: 30}
		}, "fail text"},
		{"fail without risk", func(e *Event) { e.Pages["start"].Choices[1].FailText = "x" }, "need a risk"},
		{"unknown outcome", func(e *Event) { e.Pages["start"].Choices[1].Do = []Outcome{{Kind: "teleport"}} }, "unknown kind"},
		{"huge gold", func(e *Event) { e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeGold, Amount: MaxGold + 1}} }, "gold amount"},
		{"wound too deep", func(e *Event) { e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeWound, Pct: 90}} }, "wound pct"},
		{"bad need", func(e *Event) {
			e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeNeed, Stat: "joy", Amount: 5}}
		}, "need stat"},
		{"one foe", func(e *Event) {
			e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeBattle, Foes: []Foe{{Mob: 1}}}}
		}, "2 to"},
		{"battle then continue", func(e *Event) {
			e.Pages["start"].Choices[0].Do = []Outcome{{Kind: OutcomeBattle, Foes: []Foe{{Mob: 1}, {Mob: 1}}}}
		}, "ends the event"},
		{"bad who", func(e *Event) {
			e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeFlag, Flag: "x", Who: "everyone"}}
		}, "who"},
		{"bad flag", func(e *Event) { e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeFlag, Flag: "Bad Flag"}} }, "flag"},
		{"bad picture", func(e *Event) { e.Picture = "../etc/passwd" }, "picture"},
		{"bad alignment", func(e *Event) { e.Pages["start"].Choices[0].Require.Alignment = "lawful" }, "alignment"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := goodEvent()
			// Pages and choices are shared slices: rebuild them per case.
			e.Pages = map[string]Page{
				"start":  {Text: "A page.", Choices: []Choice{{Label: "Go on", Next: "second"}, {Label: "Leave"}}},
				"second": {Text: "Another.", Choices: []Choice{{Label: "End"}}},
			}
			c.mutate(&e)
			assert.Contains(t, problems(e), c.want)
		})
	}
}

func TestValidationChecksIDsAgainstTheWorld(t *testing.T) {
	e := goodEvent()
	e.Pages["start"].Choices[1].Do = []Outcome{{Kind: OutcomeItem, Item: 999}, {Kind: OutcomeMove, Room: 888}}
	e.Pages["start"].Choices[0].Require = Requirement{Skill: "nonsense"}
	got := strings.Join(e.Validate(Lookups{
		Item:  func(int) bool { return false },
		Room:  func(id int) bool { return id == 5 },
		Skill: func(string) bool { return false },
	}), "\n")
	assert.Contains(t, got, "item 999 does not exist")
	assert.Contains(t, got, "move room 888 does not exist")
	assert.Contains(t, got, `unknown skill "nonsense"`)
}

func TestParseRejectsUnknownFields(t *testing.T) {
	_, err := Parse([]byte("- id: a\n  titel: typo\n"))
	assert.Error(t, err)
	list, err := Parse([]byte("- id: a\n  title: A\n"))
	require.NoError(t, err)
	assert.Equal(t, "a", list[0].ID)
}

func TestCatalogKeepsSoundEventsAndNamesTheRest(t *testing.T) {
	bad := goodEvent()
	bad.ID = "bad-one"
	bad.Triggers = nil
	dup := goodEvent()
	cat, probs := NewCatalog([]Event{goodEvent(), bad, dup}, Lookups{})
	assert.Equal(t, []string{"test-event"}, cat.IDs())
	joined := strings.Join(probs, "\n")
	assert.Contains(t, joined, `event "bad-one": no triggers`)
	assert.Contains(t, joined, "defined twice")
}

func TestTriggersMatchByKind(t *testing.T) {
	e := goodEvent()
	e.Triggers = []Trigger{
		{Kind: TriggerRoom, Room: 5},
		{Kind: TriggerTag, Tag: "lair-door"},
		{Kind: TriggerArrival, Zone: "Brindle Downs"},
		{Kind: TriggerCamp, Zone: "Brindle Downs"},
	}
	cat, _ := NewCatalog([]Event{e}, Lookups{})
	hit := func(kind string, room int, zone string, tags ...string) bool {
		return len(cat.Triggered(kind, room, zone, tags)) == 1
	}
	assert.True(t, hit(TriggerRoom, 5, "x"))
	assert.False(t, hit(TriggerRoom, 6, "x"))
	assert.True(t, hit(TriggerTag, 9, "x", "other", "Lair-Door"))
	assert.False(t, hit(TriggerTag, 9, "x", "other"))
	assert.True(t, hit(TriggerArrival, 9, "brindle downs"))
	assert.False(t, hit(TriggerArrival, 9, "Alderbrook"))
	assert.True(t, hit(TriggerCamp, 9, "Brindle Downs"))
	assert.False(t, hit(TriggerCamp, 9, "Alderbrook"), "a camp trigger needs its zone")
}

func members() []Facts {
	return []Facts{
		{Key: "leader", Name: "Aldous", Leader: true, Archetype: "warrior", Level: 8, Alignment: 30, Skills: map[string]int{"cooking": 1}},
		{Key: "c1", Name: "Eder", Archetype: "ranger", Level: 6, Personality: "devout", Alignment: -40, Skills: map[string]int{"cooking": 3}},
		{Key: "c2", Name: "Sera", Archetype: "rogue", Class: "nightblade", Level: 9, Personality: "wry", Skills: map[string]int{"cooking": 3}},
	}
}

func TestBestPicksTheMemberWhoDoesItBest(t *testing.T) {
	co := Company{Flags: map[string]bool{}}
	got, ok := Best(Requirement{Skill: "cooking", SkillLevel: 2}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Sera", got.Name, "equal skill goes to the higher level")

	got, ok = Best(Requirement{Classes: []string{"ranger", "rogue"}}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Sera", got.Name, "no skill named: the higher level")

	got, ok = Best(Requirement{Classes: []string{"nightblade"}}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Sera", got.Name, "an advanced class matches too")

	got, ok = Best(Requirement{Personality: "devout"}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Eder", got.Name)

	_, ok = Best(Requirement{Skill: "cooking", SkillLevel: 4}, members(), co)
	assert.False(t, ok, "nobody cooks that well")

	got, ok = Best(Requirement{Alignment: "evil"}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Eder", got.Name)

	got, ok = Best(Requirement{Leader: true, Alignment: "good"}, members(), co)
	require.True(t, ok)
	assert.Equal(t, "Aldous", got.Name)

	_, ok = Best(Requirement{MinLevel: 10}, members(), co)
	assert.False(t, ok)
}

func TestTiesGoToTheEarlierMember(t *testing.T) {
	a := Facts{Key: "a", Name: "A", Level: 5, Skills: map[string]int{"x": 2}}
	b := Facts{Key: "b", Name: "B", Level: 5, Skills: map[string]int{"x": 2}}
	got, _ := Best(Requirement{Skill: "x"}, []Facts{a, b}, Company{})
	assert.Equal(t, "A", got.Name)
}

func TestMemberTagsAreARequirementHook(t *testing.T) {
	m := members()
	m[1].Tags = []string{"background:soldier"}
	got, ok := Best(Requirement{Tag: "Background:Soldier"}, m, Company{})
	require.True(t, ok)
	assert.Equal(t, "Eder", got.Name)
	_, ok = Best(Requirement{Tag: "background:thief"}, m, Company{})
	assert.False(t, ok)
}

func TestCompanyRequirements(t *testing.T) {
	co := Company{Gold: 20, Items: map[int]int{7: 1}, Flags: map[string]bool{"seen": true}}
	for _, c := range []struct {
		name string
		req  CompanyRequirement
		want bool
	}{
		{"none", CompanyRequirement{}, true},
		{"gold enough", CompanyRequirement{Gold: 20}, true},
		{"gold short", CompanyRequirement{Gold: 21}, false},
		{"item carried", CompanyRequirement{Item: 7}, true},
		{"item missing", CompanyRequirement{Item: 8}, false},
		{"flag set", CompanyRequirement{Flag: "seen"}, true},
		{"flag unset", CompanyRequirement{Flag: "other"}, false},
		{"not flag set", CompanyRequirement{NotFlag: "seen"}, false},
		{"not flag unset", CompanyRequirement{NotFlag: "other"}, true},
	} {
		assert.Equal(t, c.want, c.req.Meets(co), c.name)
	}
	_, ok := Best(Requirement{CompanyRequirement: CompanyRequirement{Gold: 99}}, members(), co)
	assert.False(t, ok, "a company field that fails closes the choice")
}

func TestRiskFallsWithRankAndStaysInRange(t *testing.T) {
	r := Risk{Pct: 50, LessPerLevel: 10}
	assert.Equal(t, 50, r.RiskPct(0))
	assert.Equal(t, 20, r.RiskPct(3))
	assert.Equal(t, 0, r.RiskPct(9), "never below zero")
	assert.Equal(t, MaxRiskPct, Risk{Pct: MaxRiskPct}.RiskPct(-4), "never above the cap")
}

func TestDescribeNamesWhatAChoiceNeeds(t *testing.T) {
	got := Requirement{Skill: "cooking", SkillLevel: 2, Classes: []string{"cleric", "shaman"}}.Describe()
	assert.Contains(t, got, "someone with cooking 2")
	assert.Contains(t, got, "a cleric or a shaman")
	assert.Equal(t, "Aldous digs.", Fill("{who} digs.", "Aldous"))
}
