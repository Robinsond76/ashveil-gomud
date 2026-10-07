package townsfolk

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(24 * 3600)

func catalog(t *testing.T, lines ...Line) Catalog {
	t.Helper()
	cat, problems := NewCatalog(lines)
	require.Empty(t, problems)
	return cat
}

func gossip() NPC { return NPC{MobID: 1, Tags: []string{"gossip"}, Zone: "Alderbrook"} }

func TestValidateRejectsBadLines(t *testing.T) {
	for name, tc := range map[string]struct {
		line Line
		want string
	}{
		"empty text":          {Line{ID: "a", Kind: chronicle.Boss}, "text is empty"},
		"bad id":              {Line{ID: "A b", Kind: chronicle.Boss, Text: "x"}, "id must be"},
		"unknown placeholder": {Line{ID: "a", Kind: chronicle.Boss, Text: "{boss} fell"}, "unknown placeholder {boss}"},
		"unknown kind":        {Line{ID: "a", Kind: "bogus", Text: "x"}, "unknown deed kind"},
		"bad ref":             {Line{ID: "a", Kind: chronicle.Boss, Ref: "Hollow King", Text: "x"}, "ref"},
		"days too long":       {Line{ID: "a", Kind: chronicle.Boss, Days: 91, Text: "x"}, "days must be"},
		"nothing to say":      {Line{ID: "a", Text: "x"}, "needs a kind"},
		"state with sets":     {Line{ID: "a", Time: "day", Sets: "mark", Text: "x"}, "belong to a deed line"},
		"state placeholder":   {Line{ID: "a", Time: "day", Text: "hello {who}"}, "no placeholders"},
		"bad time":            {Line{ID: "a", Time: "noon", Text: "x"}, "time must be"},
		"deed with weather":   {Line{ID: "a", Kind: chronicle.Boss, Weather: []string{"rain"}, Text: "x"}, "no weather"},
		"bad flag":            {Line{ID: "a", Kind: chronicle.Boss, Flag: "Bad Flag", Text: "x"}, "flag"},
		"multi-line":          {Line{ID: "a", Kind: chronicle.Boss, Text: "a\nb"}, "one line"},
		"semicolon":           {Line{ID: "a", Kind: chronicle.Boss, Text: "a; emote dances"}, "semicolons"},
		"uppercase weather":   {Line{ID: "a", Weather: []string{"Rain"}, Text: "x"}, "lowercase"},
	} {
		t.Run(name, func(t *testing.T) {
			problems := tc.line.Validate()
			require.NotEmpty(t, problems)
			assert.Contains(t, strings.Join(problems, ";"), tc.want)
		})
	}
	good := Line{ID: "ok", Kind: chronicle.Boss, Ref: "mob:12", Text: "{who} slew {subject} at {place}.", Sets: "slayer"}
	assert.Empty(t, good.Validate())
}

func TestCatalogKeepsSoundLinesAndNamesTheRest(t *testing.T) {
	cat, problems := NewCatalog([]Line{
		{ID: "b", Kind: chronicle.Boss, Text: "x"},
		{ID: "a", Kind: chronicle.Relic, Text: "y"},
		{ID: "b", Kind: chronicle.Relic, Text: "dup"},
		{ID: "bad", Text: "z"},
	})
	assert.Equal(t, 2, cat.Len())
	assert.Equal(t, "a", cat.Lines()[0].ID, "id order")
	joined := strings.Join(problems, ";")
	assert.Contains(t, joined, `"b": defined twice`)
	assert.Contains(t, joined, `"bad"`)
}

func TestParseRefusesUnknownFields(t *testing.T) {
	_, err := Parse([]byte("- id: a\n  kind: boss\n  text: x\n  typo: 1\n"))
	assert.Error(t, err)
	list, err := Parse([]byte("- id: a\n  kind: boss\n  text: x\n"))
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestChooseTellsTheNewestUnheardDeedOnce(t *testing.T) {
	cat := catalog(t,
		Line{ID: "boss", Kind: chronicle.Boss, Text: "{who} put down {subject} at {place}."},
		Line{ID: "relic", Kind: chronicle.Relic, Text: "{who} found {subject}."},
	)
	now := 100 * day
	entries := []chronicle.Entry{ // newest first
		{Seq: 3, At: now - 1*day, Kind: chronicle.Relic, Members: []string{"Mara"}, Subject: "the Ember Crown"},
		{Seq: 2, At: now - 2*day, Kind: chronicle.Boss, Members: []string{"Mara", "Tobin"}, Subject: "the Hollow King", Place: "the Crypt"},
		{Seq: 1, At: now - 3*day, Kind: chronicle.Fell, Members: []string{"Ysolde"}},
	}
	heard := map[int]bool{}
	ctx := Context{NPC: gossip(), Now: now, Entries: entries, Heard: func(s int) bool { return heard[s] }}

	c, ok := cat.Choose(ctx)
	require.True(t, ok)
	assert.Equal(t, "Mara found the Ember Crown.", c.Text)
	assert.Equal(t, 3, c.Entry.Seq)

	heard[3] = true
	c, ok = cat.Choose(ctx)
	require.True(t, ok)
	assert.Equal(t, "Mara and Tobin put down the Hollow King at the Crypt.", c.Text)

	heard[2] = true
	_, ok = cat.Choose(ctx) // the fall has no line: silence
	assert.False(t, ok)
}

func TestChooseHonoursWindowTagsAndZones(t *testing.T) {
	now := 100 * day
	old := chronicle.Entry{Seq: 1, At: now - 20*day, Kind: chronicle.Boss, Members: []string{"Mara"}}
	fresh := chronicle.Entry{Seq: 2, At: now - 2*day, Kind: chronicle.Boss, Members: []string{"Mara"}}

	cat := catalog(t, Line{ID: "boss", Kind: chronicle.Boss, Text: "default window"})
	_, ok := cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{old}})
	assert.False(t, ok, "20 days is past the 14-day default")
	_, ok = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{fresh}})
	assert.True(t, ok)

	cat = catalog(t, Line{ID: "boss", Kind: chronicle.Boss, Days: 30, Text: "long memory"})
	_, ok = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{old}})
	assert.True(t, ok, "a line may keep a deed 30 days")

	cat = catalog(t, Line{ID: "boss", Kind: chronicle.Boss, Tags: []string{"priest"}, Text: "x"})
	_, ok = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{fresh}})
	assert.False(t, ok, "a gossip does not say a priest's line")
	cat = catalog(t, Line{ID: "boss", Kind: chronicle.Boss, Tags: []string{"GOSSIP"}, Text: "x"})
	_, ok = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{fresh}})
	assert.True(t, ok, "tags match without case")

	cat = catalog(t, Line{ID: "boss", Kind: chronicle.Boss, Zones: []string{"Hollowweb"}, Text: "x"})
	_, ok = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{fresh}})
	assert.False(t, ok, "a line for another zone")
	_, ok = cat.Choose(Context{NPC: NPC{Tags: []string{"gossip"}, Zone: "hollowweb"}, Now: now, Entries: []chronicle.Entry{fresh}})
	assert.True(t, ok)
}

func TestARefLineBeatsAPlainKindLine(t *testing.T) {
	cat := catalog(t,
		Line{ID: "any-boss", Kind: chronicle.Boss, Text: "a boss fell"},
		Line{ID: "hollow-king", Kind: chronicle.Boss, Ref: "mob:12", Text: "the Hollow King is dead"},
	)
	now := 100 * day
	king := chronicle.Entry{Seq: 1, At: now, Kind: chronicle.Boss, Ref: "mob:12"}
	other := chronicle.Entry{Seq: 2, At: now, Kind: chronicle.Boss, Ref: "mob:99"}
	c, _ := cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{king}})
	assert.Equal(t, "the Hollow King is dead", c.Text)
	c, _ = cat.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{other}})
	assert.Equal(t, "a boss fell", c.Text)

	only := catalog(t, Line{ID: "hollow-king", Kind: chronicle.Boss, Ref: "mob:12", Text: "x"})
	_, ok := only.Choose(Context{NPC: gossip(), Now: now, Entries: []chronicle.Entry{other}})
	assert.False(t, ok, "a ref line never tells another deed")
}

func TestChooseReadsFlagsAndMemberTags(t *testing.T) {
	now := 100 * day
	e := chronicle.Entry{Seq: 1, At: now, Kind: chronicle.Story, Keys: []string{"leader", "companion:2"}}
	cat := catalog(t,
		Line{ID: "needs-flag", Kind: chronicle.Story, Flag: "slayer", Text: "slayer"},
		Line{ID: "no-flag", Kind: chronicle.Story, NoFlag: "slayer", Text: "stranger"},
		Line{ID: "devout", Kind: chronicle.Story, MemberTag: "devout", Text: "devout"},
	)
	flags := map[string]bool{}
	ctx := Context{
		NPC: gossip(), Now: now, Entries: []chronicle.Entry{e},
		Flag:      func(f string) bool { return flags[f] },
		MemberTag: func(key, tag string) bool { return key == "companion:2" && tag == "devout" },
		Rand:      func(n int) int { return n - 1 }, // the last candidate
	}
	c, ok := cat.Choose(ctx)
	require.True(t, ok)
	assert.Contains(t, []string{"stranger", "devout"}, c.Text, "needs-flag is held back")
	for i := 0; i < 5; i++ {
		c, _ = cat.Choose(ctx)
		assert.NotEqual(t, "slayer", c.Text)
	}
	flags["slayer"] = true
	for i := 0; i < 5; i++ {
		c, _ = cat.Choose(ctx)
		assert.NotEqual(t, "stranger", c.Text, "no_flag lines stop once the flag is set")
	}
	ctx.MemberTag = nil
	only := catalog(t, Line{ID: "devout", Kind: chronicle.Story, MemberTag: "devout", Text: "devout"})
	_, ok = only.Choose(ctx)
	assert.False(t, ok, "a member-tag line needs a tag source")
}

func TestStateLinesFallBackAndSilenceIsAnOption(t *testing.T) {
	cat := catalog(t,
		Line{ID: "boss", Kind: chronicle.Boss, Text: "deed"},
		Line{ID: "rain", Weather: []string{"rain", "storm"}, Text: "wet"},
		Line{ID: "night", Time: "night", Text: "late"},
		Line{ID: "day", Time: "day", Text: "quiet"},
	)
	base := Context{NPC: gossip(), Now: 100 * day}
	c, ok := cat.Choose(base)
	require.True(t, ok)
	assert.Equal(t, "quiet", c.Text, "daytime, no weather: the day line")
	assert.Nil(t, c.Entry, "a state line tells no deed")

	night := base
	night.Night = true
	c, _ = cat.Choose(night)
	assert.Equal(t, "late", c.Text)

	night.Weather = "rain"
	night.Rand = func(n int) int { return 0 }
	c, _ = cat.Choose(night)
	assert.Contains(t, []string{"wet", "late"}, c.Text)
	seen := map[string]bool{}
	for r := 0; r < 2; r++ {
		night.Rand = func(n int) int { return r % n }
		c, _ = cat.Choose(night)
		seen[c.Text] = true
	}
	assert.True(t, seen["wet"] && seen["late"], "both the weather and the hour can be spoken of: %v", seen)

	// A deed always beats a state line.
	withDeed := base
	withDeed.Entries = []chronicle.Entry{{Seq: 1, At: 100 * day, Kind: chronicle.Boss}}
	c, _ = cat.Choose(withDeed)
	assert.Equal(t, "deed", c.Text)

	// Nothing fits: silence.
	only := catalog(t, Line{ID: "rain", Weather: []string{"rain"}, Text: "wet"})
	_, ok = only.Choose(base)
	assert.False(t, ok)
	_, ok = (Catalog{}).Choose(base)
	assert.False(t, ok)
}

func TestSpeakNeedsAProviderListenersAndTags(t *testing.T) {
	SetProvider(nil)
	_, ok := Speak(gossip(), []int{1})
	assert.False(t, ok, "no module: silence")

	p := &fakeProvider{}
	SetProvider(p)
	t.Cleanup(func() { SetProvider(nil) })
	_, ok = Speak(gossip(), nil)
	assert.False(t, ok, "nobody to hear")
	_, ok = Speak(NPC{MobID: 2}, []int{1})
	assert.False(t, ok, "an NPC without townsfolk tags is no talker")
	assert.Zero(t, p.calls)
	sp, ok := Speak(gossip(), []int{1})
	assert.True(t, ok)
	assert.Equal(t, "hi", sp.Text)
}

type fakeProvider struct{ calls int }

func (f *fakeProvider) Speak(NPC, []int) (Speech, bool) {
	f.calls++
	return Speech{Text: "hi"}, true
}

func TestFillUsesFallbacks(t *testing.T) {
	assert.Equal(t, "The company met it in these parts", Fill("{who} met {subject} in {place}", chronicle.Entry{}))
	assert.Equal(t, "Mara, Tobin met Orc, Chief at the gate", Fill("{who} met {subject} at {place}", chronicle.Entry{Members: []string{"Mara; Tobin"}, Subject: "Orc; Chief", Place: "the gate"}))
	assert.NotContains(t, Fill("{who} met {subject}", chronicle.Entry{Members: []string{"Mara"}, Subject: "x; emote dances"}), ";", "values cannot split a command")
	assert.Equal(t, 0, DaysAgo(10, 5))
	assert.Equal(t, 3, DaysAgo(0, 3*day+5))
}

func TestALineForAMembersBackgroundBeatsAPlainOne(t *testing.T) {
	now := 100 * day
	e := chronicle.Entry{Seq: 1, At: now, Kind: chronicle.Boss, Subject: "the Hollow King"}
	cat := catalog(t,
		Line{ID: "plain", Kind: chronicle.Boss, Text: "plain"},
		Line{ID: "soldier", Kind: chronicle.Boss, MemberTag: "trade-soldier", Text: "soldier"},
	)
	ctx := Context{
		NPC: gossip(), Now: now, Entries: []chronicle.Entry{e}, Leader: "Mara",
		MemberTag: func(key, tag string) bool { return key == "leader" && tag == "trade-soldier" },
	}
	for i := 0; i < 3; i++ {
		ctx.Rand = func(n int) int { return i % n }
		c, ok := cat.Choose(ctx)
		require.True(t, ok)
		assert.Equal(t, "soldier", c.Text)
	}
	ctx.MemberTag = func(string, string) bool { return false }
	c, ok := cat.Choose(ctx)
	require.True(t, ok)
	assert.Equal(t, "plain", c.Text, "without the background the plain line is told")
}

func TestABackgroundLineHeardLatelyJoinsThePlainOnes(t *testing.T) {
	now := 100 * day
	e := chronicle.Entry{Seq: 1, At: now, Kind: chronicle.Boss, Subject: "the Hollow King"}
	cat := catalog(t,
		Line{ID: "plain", Kind: chronicle.Boss, Text: "plain"},
		Line{ID: "soldier", Kind: chronicle.Boss, MemberTag: "trade-soldier", Text: "soldier"},
	)
	ctx := Context{
		NPC: gossip(), Now: now, Entries: []chronicle.Entry{e}, Leader: "Mara",
		MemberTag: func(key, tag string) bool { return key == "leader" && tag == "trade-soldier" },
		LineHeard: func(id string) bool { return id == "soldier" },
	}
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		ctx.Rand = func(n int) int { return i % n }
		c, ok := cat.Choose(ctx)
		require.True(t, ok)
		seen[c.Text] = true
	}
	assert.Equal(t, map[string]bool{"plain": true, "soldier": true}, seen, "both lines can be told")
}

// Phase 78: a background line greets the member who has it, once: {who}
// names that member, not everyone named on the deed.
func TestABackgroundLineGreetsOnlyTheMemberWhoHasIt(t *testing.T) {
	now := 100 * day
	e := chronicle.Entry{Seq: 1, At: now, Kind: chronicle.Boss, Subject: "the Hollow King",
		Members: []string{"Mara", "Oswin", "Tess"}, Keys: []string{"leader", "companion:2", "companion:3"}}
	cat := catalog(t,
		Line{ID: "soldier", Kind: chronicle.Boss, MemberTag: "trade-soldier", Text: "A soldier, {who}."},
	)
	ctx := Context{
		NPC: gossip(), Now: now, Entries: []chronicle.Entry{e}, Leader: "Mara",
		MemberTag: func(key, tag string) bool { return (key == "companion:2" || key == "companion:3") && tag == "trade-soldier" },
	}
	c, ok := cat.Choose(ctx)
	require.True(t, ok)
	assert.Equal(t, "A soldier, Oswin.", c.Text, "the first member with the background, alone")
	assert.Equal(t, []string{"Mara", "Oswin", "Tess"}, c.Entry.Members, "the deed told is the whole deed")
}
