package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39g wiring: the Alchemist's flasks through the real strategy pass,
// the shipped spell scripts and the combat round, in the brawl world. Aria is
// the Alchemist (a player's character, so its spells come from its spell
// book); nobody lands a blow, so what moves is what the flasks do.

// alchemistBrawl is a fight with Aria a level-level Alchemist of the route
// (blank for none) knowing the given spells, with a full satchel and no mana.
func alchemistBrawl(t *testing.T, class string, level int, spells ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	loadShippedBuff(t, "13-poisoned.yaml")
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	classes.SetProvider(&fakeClassStore{state: classes.State{Class: class}})
	t.Cleanup(func() { classes.SetProvider(nil) })
	b.withArchetypes("alchemist")
	b.unplaced()
	forceBlows(t, false)
	noCounters(t)
	c := b.aria.Character
	c.Level = level
	c.SetSkill("cast", 1)
	c.SpellBook = map[string]int{}
	for _, id := range spells {
		c.SpellBook[id] = 1
	}
	c.ManaMax.Value, c.Mana = 0, 0
	c.FlasksSpent = 0
	return b
}

// hurtOne is one round with the given company member's health overridden.
func (b *brawl) hurtOne(id, hp int) {
	b.t.Helper()
	b.classRounds(1, map[int]int{id: hp})
}

func TestAnAlchemistDefaultsToHealerAndThrowsADraughtWithNoMana(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	assert.Regexp(t, `You\s+alchemist\s+healer`, b.cmd("strategy", ""))
	assert.Contains(t, b.cmd("strategy", ""), "Healing Draught (a flask)")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	before := flasks.Remaining(b.aria.Character)
	assert.Equal(t, 6, before, "six flasks at level 1")
	b.hurtOne(1, 150)
	assert.Greater(t, tamsin.Health, 150+9, "a Minor Heal's worth: 8 and 2d4")
	assert.Equal(t, before-1, flasks.Remaining(b.aria.Character), "one flask thrown")
	assert.Zero(t, b.aria.Character.Mana, "no mana was spent")
	assert.NotEqual(t, characters.SpellCast, b.aria.Character.Aggro.Type, "no chant: it landed as it was thrown")
}

func TestADraughtHealsThePlayersOwnCompanyAtItsWorstOff(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	b.startWitchFight()
	b.classRounds(1, map[int]int{1: 400, 2: 100})
	assert.Greater(t, b.companion(2).Character.Health, 100+9, "the most hurt ally is tended first")
	assert.Equal(t, 400, b.companion(1).Character.Health, "one flask a round")
}

func TestAnEmptySatchelMeansNoDraughtsAndNoMana(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	b.aria.Character.FlasksSpent = flasks.Capacity(b.aria.Character)
	require.Zero(t, flasks.Remaining(b.aria.Character))
	b.startWitchFight()
	b.hurtOne(1, 150)
	assert.Equal(t, 150, b.companion(1).Character.Health, "nothing to throw")
	assert.Equal(t, flasks.Capacity(b.aria.Character), b.aria.Character.FlasksSpent)
}

func TestAnAntidoteCuresPoisonAndBleedingBeforeAnyoneIsInDanger(t *testing.T) {
	b := alchemistBrawl(t, "", 3, "draught", "antidote")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	require.NoError(t, tamsin.AddBuff(13, false))
	require.NoError(t, tamsin.AddBuff(status.Bleeding, false))
	require.True(t, tamsin.HasBuffFlag("poison"))
	b.classRounds(1, nil)
	assert.False(t, tamsin.HasBuffFlag("poison"), "poison is cured")
	assert.False(t, status.Live(tamsin, status.Bleeding), "bleeding is stopped")
	assert.Equal(t, flasks.Capacity(b.aria.Character)-1, flasks.Remaining(b.aria.Character), "one flask thrown")
}

func TestAnAntidoteWaitsWhileSomeoneIsNearDeath(t *testing.T) {
	b := alchemistBrawl(t, "", 3, "draught", "antidote")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	require.NoError(t, tamsin.AddBuff(13, false))
	b.classRounds(1, map[int]int{2: 100})
	assert.True(t, tamsin.HasBuffFlag("poison"), "the heal comes first")
	assert.Greater(t, b.companion(2).Character.Health, 109)
}

// scorched are the foes a round's fire reached: the round's hold stands each
// foe at 800 health, so anything under it took fire.
func scorched(b *brawl) (n int) {
	for _, m := range b.livingBandits() {
		if m.Character.Health < 800 {
			n++
		}
	}
	return n
}

func TestFireFlaskBurnsTwoFoesAndKeepsTwoFlasksForHealing(t *testing.T) {
	b := alchemistBrawl(t, "", 6, "draught", "fireflask")
	b.startWitchFight()                                                  // the opening round throws the first flask
	b.aria.Character.FlasksSpent = flasks.Capacity(b.aria.Character) - 4 // four left
	b.classRounds(1, nil)
	assert.Equal(t, 2, scorched(b), "the flask reaches the foe aimed at and one beside it")
	assert.Equal(t, 3, flasks.Remaining(b.aria.Character))
	// Down to the two held back: no more fire, with no one hurt.
	b.aria.Character.FlasksSpent = flasks.Capacity(b.aria.Character) - flasks.Keep
	b.classRounds(1, nil)
	assert.Zero(t, scorched(b), "the last flasks are for heals")
	assert.Equal(t, flasks.Keep, flasks.Remaining(b.aria.Character))
}

func TestABracingTonicBlessesAnAllyBeforeTheFirstBlow(t *testing.T) {
	b := alchemistBrawl(t, "", 8, "draught", "tonic")
	b.startWitchFight() // the opening round: one tonic, before the blows
	blessed := 0
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		if c.RT != nil && c.RT.Bless > 0 {
			blessed++
		}
	}
	assert.Equal(t, 1, blessed, "one tonic a round")
	assert.Equal(t, flasks.Capacity(b.aria.Character)-1, flasks.Remaining(b.aria.Character), "one flask thrown")
}

func TestAnAlchemistThrowsNothingItHasNotLearned(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught") // level 1: no antidote, fire or tonic yet
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	require.NoError(t, tamsin.AddBuff(13, false))
	b.classRounds(2, nil)
	assert.True(t, tamsin.HasBuffFlag("poison"))
	assert.Equal(t, flasks.Capacity(b.aria.Character), flasks.Remaining(b.aria.Character), "no flask thrown")
}

func TestAnApothecaryDraughtHealsMoreSplashesAndCleans(t *testing.T) {
	healed := func(class string, level int) (int, *brawl) {
		b := alchemistBrawl(t, class, level, "draught")
		b.startWitchFight()
		b.hurtOne(1, 100)
		return b.companion(1).Character.Health - 100, b
	}
	// A draught's heal is rolled, so compare totals over several throws
	// (39h review: one throw each flaked about 1 run in 10).
	plain, potent := 0, 0
	for i := 0; i < 8; i++ {
		p, _ := healed("", 10)
		q, _ := healed("apothecary", 10)
		plain, potent = plain+p, potent+q
	}
	assert.Greater(t, potent, plain, "Potent draughts heal 30% more")

	// Splash draught (20): the next most hurt ally takes 30% as much.
	b := alchemistBrawl(t, "apothecary", 20, "draught")
	b.startWitchFight()
	b.classRounds(1, map[int]int{1: 100, 2: 300})
	assert.Greater(t, b.companion(1).Character.Health, 100+9)
	assert.Greater(t, b.companion(2).Character.Health, 300, "the spray reaches the next most hurt ally")

	// Clean draught (25): a harmful status comes off the patient.
	c := alchemistBrawl(t, "apothecary", 25, "draught")
	c.startWitchFight()
	tamsin := &c.companion(1).Character
	require.NoError(t, tamsin.AddBuff(status.Hobbled, false))
	require.True(t, status.Live(tamsin, status.Hobbled))
	c.hurtOne(1, 100)
	assert.False(t, status.Live(tamsin, status.Hobbled), "Clean draught clears it")
}

func TestABombardiersFlasksBurnHotterWiderAndLeaveFoesAlight(t *testing.T) {
	// One brawl at a time: each owns the shared class provider and listeners.
	run := func(class string, level int) (hit, burning int) {
		t.Helper()
		t.Run(class+"/run", func(t *testing.T) {
			b := alchemistBrawl(t, class, level, "draught", "fireflask")
			b.startWitchFight()
			b.aria.Character.FlasksSpent = 0
			b.classRounds(1, nil)
			for _, m := range b.livingBandits() {
				if status.Live(&m.Character, status.Burning) {
					burning++
				}
			}
			hit = scorched(b)
		})
		return hit, burning
	}
	plainHit, plainBurning := run("", 20)
	bombHit, bombBurning := run("bombardier", 20)
	assert.Zero(t, plainBurning, "no pitch for a plain Alchemist")
	assert.Equal(t, 2, plainHit)
	assert.Equal(t, 3, bombHit, "Wide throw reaches a third foe")
	assert.Equal(t, 3, bombBurning, "Pitch and tar leaves them alight")
}

func TestHotterFlasksBurnHarderThanAPlainOne(t *testing.T) {
	total := func(class string) (sum int) {
		t.Run(class+"/total", func(t *testing.T) {
			for i := 0; i < 20; i++ {
				t.Run("", func(t *testing.T) {
					b := alchemistBrawl(t, class, 10, "draught", "fireflask")
					b.startWitchFight()
					b.classRounds(1, nil)
					for _, m := range b.livingBandits() {
						sum += 800 - m.Character.Health
					}
				})
			}
		})
		return sum
	}
	assert.Greater(t, total("bombardier"), total(""), "Hotter flasks: 100% of Sparks against 70%")
}

func TestAMutagenistsTonicHardensAnAllyAtThePriceOfItsHealth(t *testing.T) {
	b := alchemistBrawl(t, "mutagenist", 10, "draught", "tonic")
	b.startWitchFight()
	b.classRounds(1, nil)
	var armored *characters.Character
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		if c.RT != nil && c.RT.Bark == 10 {
			armored = c
		}
	}
	require.NotNil(t, armored, "one ally is hardened")
	assert.Equal(t, 1000-100, armored.Health, "a tenth of its health")

	free := alchemistBrawl(t, "mutagenist", 25, "draught", "tonic")
	free.startWitchFight()
	free.classRounds(1, nil)
	for _, c := range []*characters.Character{free.aria.Character, &free.companion(1).Character, &free.companion(2).Character, &free.companion(3).Character, &free.companion(4).Character} {
		if c.RT != nil && c.RT.Bark > 0 {
			assert.Equal(t, 15, c.RT.Bark, "Strong mutagen")
			assert.Equal(t, 1000, c.Health, "Pure mutagen costs no health")
		}
	}
}

func TestTheSatchelSurvivesTheCompanionsSnapshotAndRespawn(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	mob := b.companion(1)
	mob.Character.HPArchetype = "alchemist"
	mob.Character.Level = 7
	mob.Character.FlasksSpent = 3
	state, ok := nativeRuntime{}.Snapshot(mob.InstanceId)
	require.True(t, ok)
	assert.Equal(t, 3, state.FlasksSpent)
	fresh := &mobs.Mob{}
	fresh.Character = *characters.New()
	applyState(fresh, state)
	assert.Equal(t, 3, fresh.Character.FlasksSpent, "what was thrown stays thrown across a respawn")
	assert.Equal(t, 3, state.Clone().FlasksSpent)
}

func TestTheBrewCommandRefillsEveryAlchemistFromTheLeadersReagents(t *testing.T) {
	b := alchemistBrawl(t, "", 6, "draught")
	tamsin := b.companion(1)
	tamsin.Character.HPArchetype = "alchemist"
	tamsin.Character.Level = 6
	b.aria.Character.FlasksSpent = 2
	tamsin.Character.FlasksSpent = 3
	out := b.cmd("brew", "")
	assert.Contains(t, out, "You have no reagents")
	assert.Equal(t, 2, b.aria.Character.FlasksSpent)

	for i := 0; i < 4; i++ {
		b.aria.Character.Items = append(b.aria.Character.Items, items.New(flasks.ReagentItemID))
	}
	out = b.cmd("brew", "")
	assert.Contains(t, out, "You brew 2 flasks from 2 reagents.")
	assert.Contains(t, out, "brews 2 flasks from 2 reagents.", "the companion's satchel is refilled from the same pack")
	assert.Zero(t, b.aria.Character.FlasksSpent)
	assert.Equal(t, 1, tamsin.Character.FlasksSpent, "two reagents were all that remained")
	assert.Zero(t, flasks.Reagents(b.aria.Character))
	assert.Contains(t, out, "Reagents carried: 0.")
}

func TestBrewIsRefusedInTheMiddleOfABattle(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	b.aria.Character.FlasksSpent = 1
	b.startWitchFight()
	assert.Contains(t, b.cmd("brew", ""), "Not in the middle of a battle")
	assert.Equal(t, 1, b.aria.Character.FlasksSpent)
}

func TestBrewSaysWhenNoOneThrowsFlasks(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("warrior")
	assert.Contains(t, b.cmd("brew", ""), "No one in your company throws flasks")
}

func TestCastingADraughtByHandSpendsAFlaskNotMana(t *testing.T) {
	b := alchemistBrawl(t, "", 1, "draught")
	before := flasks.Remaining(b.aria.Character)
	b.cmd("cast", "draught")
	assert.Equal(t, before-1, flasks.Remaining(b.aria.Character), "one flask thrown")
	assert.Zero(t, b.aria.Character.Mana)

	b.aria.Character.FlasksSpent = flasks.Capacity(b.aria.Character)
	out := b.cmd("cast", "draught")
	assert.Contains(t, out, "satchel is empty")
	assert.Contains(t, out, "brew")
	assert.Equal(t, flasks.Capacity(b.aria.Character), b.aria.Character.FlasksSpent)
}
