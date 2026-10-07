package items

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv2 "gopkg.in/yaml.v2"
	yamlv3 "gopkg.in/yaml.v3"
)

const awakeBladeID = 99811

func awakenBlade(t *testing.T) *ItemSpec {
	t.Helper()
	spec := testRelic(awakeBladeID, "Test Edge", Weapon, RelicSpec{
		Signature: "Edge", Effects: map[string]int{classes.Wounded: 20}, ILvl: 20, Mob: 7, Chance: 5,
		Awakenings: []AwakeningSpec{
			{Name: "Orc-Taker", Kind: AwakenSlay, Races: []string{"goblin"}, Count: 3, Target: "goblins", Effects: map[string]int{classes.Damage: 1}},
			{Name: "Chief's End", Kind: AwakenLair, Mob: 7, Target: "the chief", Effects: map[string]int{classes.Wounded: 5}},
			{Name: "Far Hold", Kind: AwakenPlace, Zone: "Far Keep", Target: "Far Keep", Effects: map[string]int{classes.Attack: 3}},
		},
	})
	SetTestItemSpec(spec)
	t.Cleanup(func() { RemoveTestItemSpec(awakeBladeID) })
	return spec
}

func TestAwakeningSpecsAreHeldToTheGearCaps(t *testing.T) {
	base := func() RelicSpec {
		return RelicSpec{Signature: "S", Effects: map[string]int{classes.Wounded: 20}, ILvl: 20, Mob: 7, Chance: 5}
	}
	spec := &ItemSpec{Type: Weapon, Tier: 5}
	slay := func(fx map[string]int) AwakeningSpec {
		return AwakeningSpec{Name: "A", Kind: AwakenSlay, Races: []string{"ogre"}, Count: 5, Target: "ogres", Effects: fx}
	}
	cases := []struct {
		name string
		edit func(r *RelicSpec)
		err  string
	}{
		{"valid", func(r *RelicSpec) { r.Awakenings = []AwakeningSpec{slay(map[string]int{classes.Wounded: 10})} }, ""},
		{"more than three", func(r *RelicSpec) {
			for i := 0; i < 4; i++ {
				a := slay(map[string]int{classes.Attack: 1})
				a.Name = string(rune('A' + i))
				r.Awakenings = append(r.Awakenings, a)
			}
		}, "at most 3"},
		{"over a third of the cap", func(r *RelicSpec) { r.Awakenings = []AwakeningSpec{slay(map[string]int{classes.Attack: 5})} }, "at most 4"},
		{"base plus awakened over the cap", func(r *RelicSpec) {
			r.Awakenings = []AwakeningSpec{slay(map[string]int{classes.Wounded: 10}), {Name: "B", Kind: AwakenLair, Mob: 7, Target: "x", Effects: map[string]int{classes.Wounded: 10}}}
		}, "over the cap"},
		{"one-shot effects cannot awaken", func(r *RelicSpec) { r.Awakenings = []AwakeningSpec{slay(map[string]int{classes.Bargain: 1})} }, "at most 0"},
		{"unknown effect", func(r *RelicSpec) { r.Awakenings = []AwakeningSpec{slay(map[string]int{"nonsense": 1})} }, "not an effect"},
		{"slay needs races", func(r *RelicSpec) {
			a := slay(map[string]int{classes.Attack: 1})
			a.Races = nil
			r.Awakenings = []AwakeningSpec{a}
		}, "needs races"},
		{"races are lowercase", func(r *RelicSpec) {
			a := slay(map[string]int{classes.Attack: 1})
			a.Races = []string{"Ogre"}
			r.Awakenings = []AwakeningSpec{a}
		}, "lowercase"},
		{"lair needs a boss", func(r *RelicSpec) {
			r.Awakenings = []AwakeningSpec{{Name: "A", Kind: AwakenLair, Target: "x", Effects: map[string]int{classes.Attack: 1}}}
		}, "needs a boss"},
		{"place needs a zone", func(r *RelicSpec) {
			r.Awakenings = []AwakeningSpec{{Name: "A", Kind: AwakenPlace, Target: "x", Effects: map[string]int{classes.Attack: 1}}}
		}, "needs a zone"},
		{"unknown kind", func(r *RelicSpec) {
			r.Awakenings = []AwakeningSpec{{Name: "A", Kind: "dance", Target: "x", Effects: map[string]int{classes.Attack: 1}}}
		}, "not slay"},
		{"repeated name", func(r *RelicSpec) {
			a := slay(map[string]int{classes.Attack: 1})
			r.Awakenings = []AwakeningSpec{a, a}
		}, "repeats"},
		{"grants nothing", func(r *RelicSpec) { r.Awakenings = []AwakeningSpec{slay(nil)} }, "grants nothing"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := base()
			c.edit(&r)
			err := r.validate(spec)
			if c.err == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.err)
		})
	}
}

func TestAwakeningProgressWakesPowersAndShowsOnTheItem(t *testing.T) {
	awakenBlade(t)
	itm := New(awakeBladeID)
	assert.True(t, itm.HasAwakenings())
	assert.Equal(t, "0 of 3 awakened", itm.AwakeningSummary())
	assert.Nil(t, itm.AwakenedEffects())
	lines := itm.AwakeningLines()
	require.Len(t, lines, 3)
	assert.Equal(t, "Sleeping, Orc-Taker (0 of 3): slay 3 goblins while it is worn, and it wakes with +1 damage on every landed blow.", lines[0])
	assert.Contains(t, lines[1], "defeat the chief while it is worn")
	assert.Contains(t, lines[2], "reach Far Keep while it is worn")

	assert.False(t, itm.AdvanceAwakening(0, 1))
	assert.False(t, itm.AdvanceAwakening(0, 1))
	assert.Equal(t, 2, itm.AwakeningProgress(0))
	assert.Contains(t, itm.AwakeningLines()[0], "(2 of 3)")
	assert.Equal(t, "0 of 3 awakened; next: Orc-Taker (2 of 3)", itm.AwakeningStatus())

	assert.True(t, itm.AdvanceAwakening(0, 1), "the third goblin wakes it")
	assert.True(t, itm.Awakened(0))
	assert.False(t, itm.AdvanceAwakening(0, 1), "a woken awakening does not wake twice")
	assert.Equal(t, 3, itm.AwakeningProgress(0), "progress is capped at the need")
	assert.Equal(t, 1, itm.AwakenedMask())
	assert.Equal(t, map[string]int{classes.Damage: 1}, itm.AwakenedEffects())
	assert.Equal(t, "Awakened, Orc-Taker: +1 damage on every landed blow.", itm.AwakeningLines()[0])

	// A single-count awakening wakes at once.
	assert.True(t, itm.AdvanceAwakening(1, 1))
	assert.Equal(t, 3, itm.AwakenedMask())
	assert.Equal(t, 5, itm.AwakenedEffects()[classes.Wounded])
	assert.Equal(t, "2 of 3 awakened; next: Far Hold (0 of 1)", itm.AwakeningStatus())

	// The item's look and relic lines carry the progress.
	assert.Contains(t, strings.Join(itm.RelicLines(), "\n"), "Awakened, Orc-Taker")
	assert.Contains(t, itm.RelicDescription(), "Sleeping, Far Hold")
}

func TestAwakeningProgressIsPerItemAndSurvivesCopiesAndSaves(t *testing.T) {
	awakenBlade(t)
	a, b := New(awakeBladeID), New(awakeBladeID)
	a.AdvanceAwakening(0, 2)
	copied := a
	a.AdvanceAwakening(0, 1)
	assert.Equal(t, 2, copied.AwakeningProgress(0), "a copy made earlier keeps its own count")
	assert.Zero(t, b.AwakeningProgress(0), "another blade is untouched")

	for name, roundTrip := range map[string]func(Item) Item{
		"yaml.v3": func(i Item) Item {
			data, err := yamlv3.Marshal(i)
			require.NoError(t, err)
			var out Item
			require.NoError(t, yamlv3.Unmarshal(data, &out))
			return out
		},
		"yaml.v2": func(i Item) Item {
			data, err := yamlv2.Marshal(i)
			require.NoError(t, err)
			var out Item
			require.NoError(t, yamlv2.Unmarshal(data, &out))
			return out
		},
	} {
		got := roundTrip(a)
		assert.Equal(t, 3, got.AwakeningProgress(0), name)
		assert.True(t, got.Awakened(0), name)
		assert.Equal(t, a.AwakenedMask(), got.AwakenedMask(), name)
	}
	data, err := yamlv3.Marshal(b)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "awaken", "an untouched relic saves nothing extra")
}

func TestAwakenedPowersJoinTheWornGearEffects(t *testing.T) {
	awakenBlade(t)
	itm := New(awakeBladeID)
	fx, _ := GearEffects([]Item{itm})
	assert.Equal(t, map[string]int{classes.Wounded: 20}, fx)
	itm.AdvanceAwakening(1, 1)
	itm.AdvanceAwakening(0, 3)
	fx, _ = GearEffects([]Item{itm})
	assert.Equal(t, map[string]int{classes.Wounded: 25, classes.Damage: 1}, fx)
}

// Awakening is no profit path: a merchant prices the relic from its spec.
func TestAwakeningDoesNotChangeSaleValue(t *testing.T) {
	spec := awakenBlade(t)
	spec.Value = 1000
	itm := New(awakeBladeID)
	before := itm.SaleBaseValue()
	for idx := range spec.Relic.Awakenings {
		itm.AdvanceAwakening(idx, 100)
	}
	assert.Equal(t, 7, itm.AwakenedMask())
	assert.Equal(t, before, itm.SaleBaseValue())
	assert.Equal(t, 1000, itm.SaleBaseValue())
	assert.False(t, itm.IsSpecialForSale(), "an awakened relic stays as sellable as before")
}

func TestAwakeningMatchesOnlyItsOwnDeeds(t *testing.T) {
	spec := awakenBlade(t)
	slay, lair, place := spec.Relic.Awakenings[0], spec.Relic.Awakenings[1], spec.Relic.Awakenings[2]
	assert.True(t, slay.Matches(AwakenSlay, "Goblin"))
	assert.False(t, slay.Matches(AwakenSlay, "ogre"))
	assert.False(t, slay.Matches(AwakenLair, "goblin"))
	assert.True(t, lair.Matches(AwakenLair, "7"))
	assert.False(t, lair.Matches(AwakenLair, "8"))
	assert.True(t, place.Matches(AwakenPlace, "far keep"))
	assert.False(t, place.Matches(AwakenPlace, "Near Keep"))
}

// Phase 67: every shipped relic has two or three awakenings, each wired to
// something that exists: a boss, a race, a zone with rooms.
func TestShippedRelicsAllCarryRealAwakenings(t *testing.T) {
	specs := shippedSpecs(t)
	bosses := bossMobs(t)
	races := map[string]bool{}
	raceFiles, err := filepath.Glob(filepath.Join(dataRoot(), "races", "*.yaml"))
	require.NoError(t, err)
	for _, f := range raceFiles {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var r struct {
			Name string `yaml:"name"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &r))
		races[strings.ToLower(r.Name)] = true
	}
	zones := map[string]bool{}
	roomFiles, err := filepath.Glob(filepath.Join(dataRoot(), "rooms", "*", "*.yaml"))
	require.NoError(t, err)
	for _, f := range roomFiles {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var r struct {
			Zone string `yaml:"zone"`
		}
		require.NoError(t, yamlv2.Unmarshal(data, &r))
		zones[strings.ToLower(r.Zone)] = true
	}
	relics := 0
	for id, spec := range specs {
		if spec.Relic == nil {
			continue
		}
		relics++
		list := spec.Relic.Awakenings
		assert.GreaterOrEqual(t, len(list), 2, "%s has two or three awakenings", spec.Name)
		assert.LessOrEqual(t, len(list), MaxAwakenings, spec.Name)
		kinds := map[string]bool{}
		for _, a := range list {
			kinds[a.Kind] = true
			assert.NotEmpty(t, classes.DescribeGearEffects(a.Effects), "%d %s", id, a.Name)
			switch a.Kind {
			case AwakenSlay:
				for _, race := range a.Races {
					assert.True(t, races[race], "%s: %s names race %q, which does not exist", spec.Name, a.Name, race)
				}
			case AwakenLair:
				assert.True(t, bosses[a.Mob], "%s: %s names mob %d, which is not a boss", spec.Name, a.Name, a.Mob)
			case AwakenPlace:
				assert.True(t, zones[strings.ToLower(a.Zone)], "%s: %s names zone %q, which has no rooms", spec.Name, a.Name, a.Zone)
				assert.LessOrEqual(t, a.Need(), 1, "%s: a place awakening asks once, so walking back and forth earns nothing", a.Name)
			}
		}
		assert.GreaterOrEqual(t, len(kinds), 2, "%s mixes its deeds", spec.Name)
	}
	assert.GreaterOrEqual(t, relics, 16)
}
