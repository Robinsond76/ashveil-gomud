package items

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func dataRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

func shippedSets(t *testing.T) map[string]*SetSpec {
	t.Helper()
	out := map[string]*SetSpec{}
	files, err := filepath.Glob(filepath.Join(dataRoot(), "sets", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		s := &SetSpec{}
		require.NoError(t, yaml.Unmarshal(data, s), f)
		require.NoError(t, s.Validate(), f)
		assert.Equal(t, s.SetId+".yaml", filepath.Base(f))
		out[s.SetId] = s
	}
	return out
}

// bossMobs reads the shipped mob files for the ids of every boss.
func bossMobs(t *testing.T) map[int]bool {
	t.Helper()
	out := map[int]bool{}
	require.NoError(t, filepath.Walk(filepath.Join(dataRoot(), "mobs"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var m struct {
			MobId int  `yaml:"mobid"`
			Boss  bool `yaml:"boss"`
		}
		require.NoError(t, yaml.Unmarshal(data, &m))
		if m.Boss {
			out[m.MobId] = true
		}
		return nil
	}))
	return out
}

// Every shipped relic: a named, tier 4-6 piece with a real signature or a
// real set, a fixed level and a boss that exists; no boss's relics add up to
// more than a sure drop; every set is a small complete set of distinct slots.
func TestShippedRelicsAreCompleteAndSourced(t *testing.T) {
	specs := shippedSpecs(t)
	sets := shippedSets(t)
	bosses := bossMobs(t)
	chance := map[int]int{}
	pieces := map[string][]*ItemSpec{}
	legendaries := 0
	var ids []int
	for id, spec := range specs {
		if spec.Relic != nil {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	require.GreaterOrEqual(t, len(ids), 12, "the shipped relics")
	for _, id := range ids {
		spec := specs[id]
		r := spec.Relic
		assert.GreaterOrEqual(t, id, 50000, "relics have their own id range: %s", spec.Name)
		assert.Equal(t, spec.Name, strings.TrimSpace(spec.Name))
		assert.True(t, bosses[r.Mob], "%s drops from mob %d, which is not a boss", spec.Name, r.Mob)
		chance[r.Mob] += r.Chance
		assert.NotEmpty(t, r.Lore, spec.Name)
		assert.Contains(t, []int{4, 5, 6}, spec.Tier, spec.Name)
		if r.IsSet() {
			set, ok := sets[r.Set]
			require.True(t, ok, "%s names set %q", spec.Name, r.Set)
			assert.NotEmpty(t, set.Name)
			pieces[r.Set] = append(pieces[r.Set], spec)
			continue
		}
		legendaries++
		assert.NotEmpty(t, r.Signature, spec.Name)
		assert.NoError(t, classes.ValidateGearEffects(r.Effects), spec.Name)
		// The signature says what it does, from its numbers.
		assert.NotEmpty(t, classes.DescribeGearEffects(r.Effects), spec.Name)
	}
	assert.GreaterOrEqual(t, legendaries, 5)
	for boss, total := range chance {
		assert.LessOrEqual(t, total, 100, "boss %d's relic chances", boss)
		assert.Greater(t, total, 0)
	}
	require.Len(t, pieces, len(sets), "every set has pieces")
	for id, list := range pieces {
		assert.GreaterOrEqual(t, len(list), 3, "set %s is 3 or 4 pieces", id)
		assert.LessOrEqual(t, len(list), 4, "set %s is 3 or 4 pieces", id)
		slots := map[ItemType]bool{}
		for _, p := range list {
			assert.False(t, slots[p.Type], "set %s has two %s pieces", id, p.Type)
			slots[p.Type] = true
			assert.Equal(t, list[0].Relic.Mob, p.Relic.Mob, "a set drops from one boss")
			assert.Equal(t, list[0].Tier, p.Tier, "a set is one tier")
			assert.Equal(t, list[0].Relic.ILvl, p.Relic.ILvl)
		}
		for _, b := range sets[id].Bonuses {
			assert.LessOrEqual(t, b.Pieces, len(list), "set %s asks for more pieces than it has", id)
		}
	}
}

func testRelic(id int, name string, typ ItemType, r RelicSpec) *ItemSpec {
	spec := &ItemSpec{ItemId: id, Name: name, Type: typ, Subtype: Wearable, Tier: 5, DamageReduction: 3, Relic: &r}
	if typ == Weapon {
		spec.Subtype, spec.Hands = Slashing, 1
		spec.Damage = Damage{Attacks: 1, DiceCount: 1, SideCount: 6}
		spec.DamageReduction = 0
	}
	return spec
}

func withRelics(t *testing.T, specs ...*ItemSpec) {
	t.Helper()
	for _, s := range specs {
		SetTestItemSpec(s)
	}
	t.Cleanup(func() {
		for _, s := range specs {
			RemoveTestItemSpec(s.ItemId)
		}
	})
}

// A Legendary's signature and a set's bonuses add up from what is worn:
// the set's second bonus only at its piece count, a relic worn twice
// counts once, and plain gear grants nothing.
func TestGearEffectsFollowWhatIsWorn(t *testing.T) {
	withRelics(t,
		testRelic(990001, "test blade", Weapon, RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: 1, Chance: 5}),
		testRelic(990002, "test helm", Head, RelicSpec{Set: "testset", ILvl: 30, Mob: 1, Chance: 5}),
		testRelic(990003, "test mail", Body, RelicSpec{Set: "testset", ILvl: 30, Mob: 1, Chance: 5}),
		testRelic(990004, "test boots", Feet, RelicSpec{Set: "testset", ILvl: 30, Mob: 1, Chance: 5}),
		&ItemSpec{ItemId: 990005, Name: "plain hat", Type: Head, Subtype: Wearable},
	)
	SetTestSet(&SetSpec{SetId: "testset", Name: "Test Regalia", Bonuses: []SetBonus{
		{Pieces: 2, Effects: map[string]int{classes.Armor: 3}},
		{Pieces: 3, Effects: map[string]int{classes.Armor: 2, classes.Attack: 4}},
	}})
	t.Cleanup(func() { RemoveTestSet("testset") })

	it := func(id int) Item { return New(id) }
	fx, sets := GearEffects([]Item{it(990005), {}})
	assert.Empty(t, fx, "plain gear grants nothing")
	assert.Empty(t, sets)

	fx, _ = GearEffects([]Item{it(990001)})
	assert.Equal(t, map[string]int{classes.Wounded: 20}, fx)
	fx, _ = GearEffects([]Item{it(990001), it(990001)})
	assert.Equal(t, 20, fx[classes.Wounded], "a relic worn twice counts once")

	fx, sets = GearEffects([]Item{it(990002)})
	assert.Empty(t, fx, "one piece is no bonus")
	require.Len(t, sets, 1)
	assert.Equal(t, 1, sets[0].Worn)
	assert.Equal(t, 3, sets[0].Total)

	fx, sets = GearEffects([]Item{it(990002), it(990003)})
	assert.Equal(t, map[string]int{classes.Armor: 3}, fx, "two pieces: the first bonus")
	assert.Len(t, sets[0].Active, 1)

	fx, sets = GearEffects([]Item{it(990002), it(990003), it(990004), it(990001), it(990002)})
	assert.Equal(t, map[string]int{classes.Armor: 5, classes.Attack: 4, classes.Wounded: 20}, fx, "three pieces: both bonuses, and the signature")
	assert.Len(t, sets[0].Active, 2)
	assert.Equal(t, 3, sets[0].Worn, "a repeated piece counts once")

	lines := SetProgressText(sets)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Test Regalia: 3 of 3 pieces")
	assert.Contains(t, lines[0], "+4 Attack")
}

func TestRelicDescriptionStatesTheRealEffect(t *testing.T) {
	withRelics(t,
		testRelic(990011, "test blade", Weapon, RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, Lore: "Old and grey.", ILvl: 30, Mob: 1, Chance: 5}),
		testRelic(990012, "test helm", Head, RelicSpec{Set: "descset", Lore: "Worn by wardens.", ILvl: 30, Mob: 1, Chance: 5}),
		testRelic(990013, "test mail", Body, RelicSpec{Set: "descset", ILvl: 30, Mob: 1, Chance: 5}),
	)
	SetTestSet(&SetSpec{SetId: "descset", Name: "Warden's Kit", Bonuses: []SetBonus{
		{Pieces: 2, Effects: map[string]int{classes.Armor: 3}},
		{Pieces: 3, Effects: map[string]int{classes.Evasion: 4}},
	}})
	t.Cleanup(func() { RemoveTestSet("descset") })

	blade := New(990011)
	text := blade.RelicDescription()
	assert.Contains(t, text, "Reaping")
	assert.Contains(t, text, "20% more to a foe at or below half health", "the sentence is built from the effect's number")
	assert.Contains(t, text, "Old and grey.")
	assert.Contains(t, blade.GetLongDescriptionFor(0), "Reaping", "look shows it")

	helm := New(990012)
	text = helm.RelicDescription()
	assert.Contains(t, text, "Warden's Kit")
	assert.Contains(t, text, "2 worn: +3% damage reduction")
	assert.Contains(t, text, "3 worn: +4 Evasion")
	assert.Contains(t, text, "Worn by wardens.")

	assert.Empty(t, (&Item{ItemId: 1}).RelicDescription(), "an item that is no relic has none")
}

// Treasure pays (owner decision 2026-10-06): a relic sells like any other
// find, from its own value, and its name stands whatever it rolled.
func TestRelicsSellAndKeepTheirName(t *testing.T) {
	withRelics(t, testRelic(990021, "Test Reaper", Weapon, RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: 1, Chance: 5}))
	relic := New(990021)
	assert.False(t, relic.IsSpecialForSale(), "merchants buy relics")
	relic.ApplyRoll(Rolled{Version: RollVersion, Tier: 5, ILvl: 30, Quality: QualityExquisite, Rarity: RarityLegendary, Identified: true,
		Affixes: []RolledAffix{{ID: "keen", Label: "Keen", Mechanic: "statmod:strength", Value: 2}}})
	assert.False(t, relic.IsSpecialForSale(), "even rolled")
	assert.Equal(t, "Test Reaper", relic.RollName("Test Reaper"), "the authored name stands, not a quality or affix name")
}

func TestRelicValidationRefusesBadRelics(t *testing.T) {
	good := RelicSpec{Signature: "Reaping", Effects: map[string]int{classes.Wounded: 20}, ILvl: 30, Mob: 1, Chance: 5}
	tweak := func(f func(*ItemSpec, *RelicSpec)) error {
		r := good
		spec := testRelic(1, "x", Weapon, r)
		spec.Tier = 6
		f(spec, spec.Relic)
		return spec.Validate()
	}
	assert.NoError(t, tweak(func(*ItemSpec, *RelicSpec) {}))
	assert.Error(t, tweak(func(s *ItemSpec, _ *RelicSpec) { s.Tier = 3 }), "relics are tier 4 and up")
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Effects = map[string]int{"made-up": 3} }), "an effect gear may not grant")
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Effects = nil }), "a legendary needs an effect")
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Signature = "" }), "and a signature name")
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Mob = 0 }), "and a boss")
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Chance = 101 }))
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.ILvl = 0 }))
	assert.Error(t, tweak(func(_ *ItemSpec, r *RelicSpec) { r.Set = "x" }), "a set piece carries no signature of its own")
	assert.Error(t, tweak(func(s *ItemSpec, _ *RelicSpec) { s.Type, s.Subtype = Potion, Usable }), "only equipment")

	assert.Error(t, (&SetSpec{SetId: "a", Name: "A", Bonuses: []SetBonus{{Pieces: 2, Effects: map[string]int{classes.Armor: 3}}}}).Validate(), "a set needs two bonuses")
	assert.Error(t, (&SetSpec{SetId: "a", Name: "A", Bonuses: []SetBonus{{Pieces: 3, Effects: map[string]int{classes.Armor: 3}}, {Pieces: 4, Effects: map[string]int{classes.Armor: 4}}}}).Validate(), "the first bonus is at 2 pieces")
	assert.Error(t, (&SetSpec{SetId: "a", Name: "A", Bonuses: []SetBonus{{Pieces: 2, Effects: map[string]int{"nope": 3}}, {Pieces: 3, Effects: map[string]int{classes.Armor: 4}}}}).Validate())
}
