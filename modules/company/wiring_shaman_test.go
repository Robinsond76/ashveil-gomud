package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39c wiring: the Shaman's weather through the real strategy pass, the
// shipped spell scripts, the buff event and the combat round (shipped
// config and spells, DoCombat), in the brawl world.

// shamanBrawl is a fight begun on the bandits with Aria a level-level Shaman
// of the route (blank for none) knowing the given spells, with mana to
// spare. Nobody lands a blow, so no chant breaks.
func shamanBrawl(t *testing.T, class string, level int, spells ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	classes.SetProvider(&fakeClassStore{state: classes.State{Class: class}})
	t.Cleanup(func() { classes.SetProvider(nil) })
	b.withArchetypes("shaman")
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
	c.ManaMax.Value, c.Mana = 300, 300
	return b
}

// slingTheSlinger gives the bandit slinger a sling, so a fog or a chill has
// a foe worth it.
func (b *brawl) slingTheSlinger() {
	b.t.Helper()
	b.bandit("bandit slinger").Character.Equipment.Weapon = items.New(10014)
}

func (b *brawl) marked(buff int) int {
	n := 0
	for _, m := range b.livingBandits() {
		if status.Live(&m.Character, buff) {
			n++
		}
	}
	return n
}

// roundsUntil runs rounds, holding the bandits and the company up, until done.
func (b *brawl) roundsUntil(max int, done func() bool) string {
	b.t.Helper()
	var out string
	for i := 0; i < max && !done(); i++ {
		b.toughen()
		b.hold(nil)
		out += b.fight()
	}
	return out
}

func TestShamanDefaultsToCasterAndCallsFogAgainstAShooter(t *testing.T) {
	b := shamanBrawl(t, "", 1, "callfog", "gust")
	b.slingTheSlinger()
	assert.Regexp(t, `You\s+shaman\s+caster`, b.cmd("strategy", ""))
	stream := b.listen()
	// Call Fog has a one-round chant (no extra wait), so the opening round
	// casts it: no weather was up and a shooter stands.
	b.startWitchFight()
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastStart, "Aria"))
	assert.Equal(t, 300-6, b.aria.Character.Mana)
	out := b.roundsUntil(3, func() bool { return b.marked(status.Fogbound) > 0 })
	assert.Equal(t, stormcraft.Fog, battle.WeatherOf(7).Kind)
	assert.Equal(t, len(b.livingBandits()), b.marked(status.Fogbound), "every foe in the battle is fogbound")
	assert.Equal(t, 4, b.livingBandits()[0].Character.GetBuffs(status.Fogbound)[0].TriggersInitial, "3 rounds, one more than it lasts")
	assert.NotContains(t, out, "Gust", "it had no need of a second spell")
}

func TestShamanSavesItsWeatherForFoesThatShootOrCast(t *testing.T) {
	b := shamanBrawl(t, "", 1, "callfog", "gust")
	b.startWitchFight() // five bandits with no bows or spells
	agg := b.aria.Character.Aggro
	require.NotNil(t, agg)
	assert.Equal(t, "gust", agg.SpellInfo.SpellId, "nothing to fog: it casts its attack spell")
	assert.Equal(t, stormcraft.None, battle.WeatherOf(7).Kind)
}

func TestWeatherPassesAndTheMarksComeOff(t *testing.T) {
	b := shamanBrawl(t, "", 1, "callfog")
	b.slingTheSlinger()
	b.startWitchFight()
	out := b.roundsUntil(12, func() bool { return false })
	assert.Contains(t, out, "The fog thins and lifts.", "the call ran out, and the room is told")
	assert.Contains(t, out, "The fog thins around", "and each foe's mark with it")
	// With none up and a shooter standing, she calls it again.
	assert.Contains(t, out, "A gray fog rolls over the battle.")
}

func TestANewWeatherReplacesTheOldAndTheFoesOldMarkComesOff(t *testing.T) {
	b := shamanBrawl(t, "", 3, "callfog")
	b.slingTheSlinger()
	b.startWitchFight()
	b.roundsUntil(3, func() bool { return b.marked(status.Fogbound) > 0 })
	require.Equal(t, stormcraft.Fog, battle.WeatherOf(7).Kind)
	require.Positive(t, b.marked(status.Fogbound))

	got := scripting.GetActor(7, 0).CallWeather("chill")
	events.ProcessEvents()
	assert.Equal(t, true, got["landed"])
	assert.Equal(t, "fog", got["replaced"])
	assert.Equal(t, stormcraft.Chill, battle.WeatherOf(7).Kind)
	assert.Zero(t, b.marked(status.Fogbound), "the fog's mark came off")
	assert.Equal(t, len(b.livingBandits()), b.marked(status.Windchilled))
	assert.Equal(t, "same", scripting.GetActor(7, 0).CallWeather("chill")["reason"], "the same weather is not called twice")
	assert.Equal(t, "invalid", scripting.GetActor(7, 0).CallWeather("snow")["reason"])
}

func TestWeatherNeedsABattleAndTouchesNoWorldWeather(t *testing.T) {
	b := newBrawl(t)
	got := scripting.GetActor(7, 0).CallWeather("rain")
	assert.Equal(t, false, got["landed"], "no battle, no weather: %v", b.aria.Character.Name)
	assert.Equal(t, "", scripting.GetActor(7, 0).Weather())
}

func TestRainFeedsLightningAndLightningIsCalledAfterIt(t *testing.T) {
	b := shamanBrawl(t, "", 8, "rain", "lightning", "gust")
	b.startWitchFight()
	agg := b.aria.Character.Aggro
	require.NotNil(t, agg)
	assert.Equal(t, "rain", agg.SpellInfo.SpellId, "it knows Lightning, so Rain is worth its mana")
	out := b.roundsUntil(8, func() bool { return false })
	assert.Contains(t, out, "Rain drives down over the battle.")
	assert.Contains(t, out, "strikes", "then Lightning")
	assert.Regexp(t, `\(\d+ damage, rain\)`, out, "Lightning strikes harder in the rain")
}

func TestStormcallerLightningChainsToASecondFoe(t *testing.T) {
	b := shamanBrawl(t, "stormcaller", 10, "lightning")
	require.Equal(t, 50, b.aria.Character.ClassEffects().Int(classes.Chain))
	b.startWitchFight()
	require.Equal(t, "lightning", b.aria.Character.Aggro.SpellInfo.SpellId)
	assert.Len(t, b.aria.Character.Aggro.SpellInfo.TargetMobInstanceIds, 2, "a primary and a second foe")
	out := b.roundsUntil(3, func() bool { return false })
	assert.Contains(t, out, "The bolt leaps on to")
	assert.Contains(t, out, "chained)")
}

func TestPlainShamansLightningStrikesOneFoe(t *testing.T) {
	b := shamanBrawl(t, "", 8, "lightning")
	b.startWitchFight()
	assert.Len(t, b.aria.Character.Aggro.SpellInfo.TargetMobInstanceIds, 1)
}

func TestEarthspeakerStoneskinsAnAllyOnItsOwn(t *testing.T) {
	b := shamanBrawl(t, "earthspeaker", 10, "stoneskin", "gust")
	stream := b.listen()
	b.startWitchFight() // a one-round chant: armor before damage
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastStart, "Aria"))
	assert.Equal(t, 300-8, b.aria.Character.Mana, "Stoneskin costs 8")
	armored := 0
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		if c.RT != nil && c.RT.Bark == 10 {
			armored++
		}
	}
	assert.Equal(t, 1, armored, "one ally holds +10 armor")
}

func TestMistweaverFogCoversTheCompanyWithEvasion(t *testing.T) {
	b := shamanBrawl(t, "mistweaver", 10, "callfog")
	b.slingTheSlinger()
	b.startWitchFight() // the call; the round's auras were set before it
	assert.Zero(t, b.companion(1).Character.Aura.Evasion, "no fog yet at the round's start")
	b.toughen()
	b.hold(nil)
	b.fight()
	assert.Equal(t, stormcraft.Fog, battle.WeatherOf(7).Kind)
	assert.Equal(t, 5, b.companion(1).Character.Aura.Evasion, "Veil of mist: allies gain +5 Evasion while the fog lasts")
	assert.Equal(t, 5, b.aria.Character.Aura.Evasion)
}

func TestMistweaverWeatherLastsLonger(t *testing.T) {
	b := shamanBrawl(t, "mistweaver", 15, "callfog")
	require.Equal(t, 1, b.aria.Character.ClassEffects().Int(classes.WeatherLong))
	b.slingTheSlinger()
	b.startWitchFight()
	b.roundsUntil(3, func() bool { return b.marked(status.Fogbound) > 0 })
	assert.Equal(t, 5, b.livingBandits()[0].Character.GetBuffs(status.Fogbound)[0].TriggersInitial, "4 rounds, one more than it lasts")
}
