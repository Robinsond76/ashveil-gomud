package lifestory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func shipped(t *testing.T) *Data {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "lifestory.yaml"))
	require.NoError(t, err)
	require.NoError(t, LoadBytes(raw))
	t.Cleanup(func() { SetData(nil) })
	return Current()
}

func TestShippedLifeStoryValidates(t *testing.T) {
	d := shipped(t)
	for _, id := range Stages {
		s := d.Stage(id)
		require.NotNil(t, s, id)
		assert.GreaterOrEqual(t, len(s.Options), 5, "%s has five or six answers", id)
		assert.LessOrEqual(t, len(s.Options), 6, id)
	}
	for _, o := range d.Stage(StageTrade).Options {
		assert.NotZero(t, o.Keepsake, "%s gives a keepsake", o.ID)
	}
}

// walk visits every combination of answers, choosing each stat the answer
// still offers.
func walk(d *Data, visit func(Picks)) {
	var rec func(stage int, picks Picks)
	rec = func(stage int, picks Picks) {
		if stage == len(Stages) {
			visit(picks)
			return
		}
		id := Stages[stage]
		for _, o := range d.Stage(id).Options {
			offered := Offered(o, picks)
			choices := offered
			if len(choices) == 0 {
				choices = []string{``}
			}
			for _, st := range choices {
				next := Picks{}
				for k, v := range picks {
					next[k] = v
				}
				next[id] = o.ID
				if st != `` {
					next[StatKey(id)] = st
				}
				rec(stage+1, next)
			}
		}
	}
	rec(0, Picks{})
}

// The life story gives +3 across its three stages at most, and never more
// than +2 to one stat, whatever is chosen.
func TestStatBonusNeverExceedsLimits(t *testing.T) {
	d := shipped(t)
	count := 0
	walk(d, func(p Picks) {
		count++
		require.NoError(t, d.Check(p), "%v", p)
		total := 0
		for st, v := range StatBonus(p) {
			assert.LessOrEqual(t, v, MaxStatBonus, "%s in %v", st, p)
			total += v
		}
		assert.LessOrEqual(t, total, 3, "%v", p)
		assert.Equal(t, 3, total, "every shipped option names two stats, so all three +1s land: %v", p)
	})
	assert.Greater(t, count, 1000)
}

func TestStatBonusCapsHandEditedPicks(t *testing.T) {
	p := Picks{
		StatKey(StageHomeland): `strength`, StatKey(StageUpbringing): `strength`, StatKey(StageTrade): `strength`,
	}
	assert.Equal(t, map[string]int{`strength`: 2}, StatBonus(p))
	assert.Empty(t, StatBonus(Picks{StatKey(StageHomeland): `luck`}), "an unknown stat adds nothing")
	assert.Empty(t, StatBonus(nil))
}

func TestOfferedHidesCappedStats(t *testing.T) {
	o := Option{Stats: []string{`strength`, `speed`}}
	assert.Equal(t, []string{`strength`, `speed`}, Offered(o, Picks{}))
	full := Picks{StatKey(StageHomeland): `strength`, StatKey(StageUpbringing): `strength`}
	assert.Equal(t, []string{`speed`}, Offered(o, full))
	full[StatKey(StageTrade)] = `speed`
	assert.Equal(t, []string{`speed`}, Offered(o, full))
	assert.Empty(t, Offered(Option{Stats: []string{`strength`}}, full))
}

func TestCheckRejectsBadPicks(t *testing.T) {
	d := shipped(t)
	good := Picks{
		StageHomeland: `river-farms`, StatKey(StageHomeland): `vitality`,
		StageUpbringing: `farmhand`, StatKey(StageUpbringing): `strength`,
		StageTrade: `soldier`, StatKey(StageTrade): `vitality`,
	}
	require.NoError(t, d.Check(good))

	bad := Picks{}
	for k, v := range good {
		bad[k] = v
	}
	delete(bad, StageTrade)
	assert.Error(t, d.Check(bad), "a stage left out")

	bad = Picks{}
	for k, v := range good {
		bad[k] = v
	}
	bad[StageTrade] = `wizard-king`
	assert.Error(t, d.Check(bad), "an answer that doesn't exist")

	bad = Picks{}
	for k, v := range good {
		bad[k] = v
	}
	bad[StatKey(StageHomeland)] = `mysticism`
	assert.Error(t, d.Check(bad), "a stat the answer doesn't name")

	bad = Picks{}
	for k, v := range good {
		bad[k] = v
	}
	delete(bad, StatKey(StageHomeland))
	assert.Error(t, d.Check(bad), "no stat chosen while one was offered")
}

func TestBackstoryUsesNameAndPronouns(t *testing.T) {
	d := shipped(t)
	picks := Picks{StageHomeland: `river-farms`, StageUpbringing: `farmhand`, StageTrade: `soldier`}
	she := Pronouns{Subject: `she`, Object: `her`, Possessive: `her`}
	text := d.Backstory(picks, `Mara`, she)
	assert.Contains(t, text, `Mara was born on the river farms`)
	assert.Contains(t, text, `Her childhood was fields`)
	assert.Contains(t, text, `left her hard to tire`)
	assert.NotContains(t, text, `{`)
	assert.NotContains(t, text, `}`)

	// every answer fills cleanly for every pronoun set
	for _, p := range []Pronouns{
		she, {Subject: `he`, Object: `him`, Possessive: `his`}, {Subject: `they`, Object: `them`, Possessive: `their`},
	} {
		for _, id := range Stages {
			for _, o := range d.Stage(id).Options {
				got := d.Backstory(Picks{id: o.ID}, `Mara`, p)
				assert.NotEmpty(t, got)
				assert.False(t, strings.ContainsAny(got, `{}`), got)
				assert.True(t, strings.HasSuffix(got, `.`), got)
			}
		}
	}
}

func TestBackground(t *testing.T) {
	assert.Equal(t, ``, Background(nil))
	assert.Equal(t, `soldier`, Background(PicksWithBackground(`soldier`)))
}

func TestValidateRejectsMalformedData(t *testing.T) {
	load := func() *Data {
		raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "lifestory.yaml"))
		require.NoError(t, err)
		var d Data
		require.NoError(t, yaml.Unmarshal(raw, &d))
		return &d
	}
	require.NoError(t, load().Validate())

	d := load()
	d.Stages = d.Stages[:2]
	assert.Error(t, d.Validate(), "a stage missing")

	d = load()
	d.Stage(StageHomeland).Options[0].Stats = []string{`luck`}
	assert.Error(t, d.Validate())

	d = load()
	d.Stage(StageHomeland).Options[0].Stats = []string{`strength`, `speed`, `smarts`}
	assert.Error(t, d.Validate())

	d = load()
	d.Stage(StageHomeland).Options[0].Keepsake = 5
	assert.Error(t, d.Validate(), "only the trade gives a keepsake")

	d = load()
	d.Stage(StageTrade).Options[0].Text = `{name} knew {nobody}.`
	assert.Error(t, d.Validate())

	d = load()
	d.Stage(StageTrade).Options[1].ID = d.Stage(StageTrade).Options[0].ID
	assert.Error(t, d.Validate(), "a repeated id")
}

func TestTagsNameEachStagePick(t *testing.T) {
	picks := Picks{StageHomeland: "hill-clans", StageUpbringing: "farmhand", StageTrade: "soldier", "trade-stat": "strength"}
	assert.Equal(t, []string{"homeland-hill-clans", "upbringing-farmhand", "trade-soldier"}, Tags(picks))
	assert.Equal(t, []string{"trade-soldier"}, Tags(PicksWithBackground("soldier")))
	assert.Empty(t, Tags(nil), "no life story, no tags")
}

func TestTagNameReadsAPickAsThePlayerSeesIt(t *testing.T) {
	shipped(t)
	for tag, want := range map[string]string{
		"trade-soldier":          "a soldier",
		"homeland-hill-clans":    "the hill clans",
		"Upbringing-Hunters-Get": "a hunter's get",
	} {
		got, ok := TagName(tag)
		assert.True(t, ok, tag)
		assert.Equal(t, want, got, tag)
	}
	for _, tag := range []string{"trade-dancer", "soldier", "", "trade-"} {
		_, ok := TagName(tag)
		assert.False(t, ok, tag)
	}
}
