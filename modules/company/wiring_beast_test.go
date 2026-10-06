package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/beasts"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39e wiring: the Beast Tamer's bonded beast through the real battle
// pass, ability pass and combat round (shipped config, DoCombat), in the
// brawl world. Tamsin is the Beast Tamer; every other member's abilities are
// off and every blow lands.

// beastBrawl is a brawl whose first companion is a level-level Beast Tamer
// (of a route, when given); the fight has begun on the captain.
func beastBrawl(t *testing.T, level int, route string) (*brawl, *[]combatstream.Event) {
	t.Helper()
	return beastBrawlWith(t, level, route, nil)
}

// beastBrawlWith is beastBrawl with a setup run before the fight begins.
func beastBrawlWith(t *testing.T, level int, route string, before func(*brawl)) (*brawl, *[]combatstream.Event) {
	t.Helper()
	hooks.ResetBeastsForTest()
	domain.ResetBeastsForTest()
	t.Cleanup(domain.ResetBeastsForTest)
	t.Cleanup(hooks.ResetBeastsForTest)
	// Beasts stand hard to kill, as toughen makes the company.
	t.Cleanup(beasts.UseSpawnHookForTest(func(m *mobs.Mob) {
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}))
	b, stream := abilityBrawl(t, map[int]string{1: "beasttamer", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.HPArchetype = "beasttamer"
	if route != "" {
		tamsin.Character.SetClassState(route, nil)
	}
	if before != nil {
		before(b)
	}
	b.start()
	return b, stream
}

// beast is Tamsin's standing beast.
func (b *brawl) beast() *mobs.Mob {
	b.t.Helper()
	tamer, ok := beasts.Of(7, domain.CompanionMemberKey(1))
	require.True(b.t, ok)
	mob, standing := beasts.Live(tamer)
	require.True(b.t, standing, "the beast stands")
	return mob
}

func TestABeastTakesItsOwnTurnBesideItsTamer(t *testing.T) {
	b, stream := beastBrawl(t, 3, "")
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	beast := b.beast()
	assert.Equal(t, "Ash", beast.Character.Name)
	assert.NotEmpty(t, swingsBy(round, "Ash"), "the beast bites on its own turn\n%s", out)
	assert.NotEmpty(t, swingsBy(round, "Tamsin Reed"), "the Tamer still strikes")
	require.NotEmpty(t, abilityEvents(round, "Tamsin Reed"))
	assert.Equal(t, "Sic", abilityEvents(round, "Tamsin Reed")[0].Status)
	assert.Contains(t, out, "(sic)")
	// It holds a cell in the formation but takes no company slot.
	f, _ := domain.FormationFor(7)
	_, _, placed := f.Find(domain.BeastMemberKey(domain.CompanionMemberKey(1)))
	assert.True(t, placed, "the beast stands in the formation")
	assert.Len(t, b.companyInstances(), 4, "still a company of leader and four")
}

func TestAFallenBeastIsWoundedNotDestroyedAndMendsAtRest(t *testing.T) {
	b, _ := beastBrawl(t, 3, "")
	b.hardenBandits()
	b.fight()
	ash := b.beast()
	id := ash.InstanceId
	ash.Character.Health = 0
	hooks.ResetAbilitiesForTest()
	b.toughen()
	b.hardenBandits()
	out := b.fight()
	assert.Contains(t, out, "goes down, wounded")
	tamsin := b.companion(1)
	require.NotNil(t, tamsin.Character.Beast)
	assert.True(t, tamsin.Character.Beast.Wounded, "the record keeps the wound")
	_, standing := domain.BeastOf(id)
	assert.False(t, standing, "the wounded beast leaves the battle")
	assert.Nil(t, mobs.GetInstance(id))
	// Rest closes the wound.
	assert.True(t, beasts.Recover(&tamsin.Character))
	assert.False(t, tamsin.Character.Beast.Wounded)
	assert.Zero(t, tamsin.Character.Beast.Damage)
}

func TestSicGivesTheBeastAttackAndPackSenseGivesTheTamerEvasion(t *testing.T) {
	b, _ := beastBrawl(t, 8, "")
	b.hardenBandits()
	base := b.companion(1).Character.Evasion()
	b.fight()
	tamsin := b.companion(1)
	assert.Equal(t, 5, tamsin.Character.RT.PackSense, "Pack Sense while the beast stands")
	assert.Equal(t, base+5, tamsin.Character.Evasion())
	assert.Equal(t, 10, b.beast().Character.RT.Sic, "Sic's Attack on the beast's strike")
}

func TestRallyHealsAHurtBeastWithNoMana(t *testing.T) {
	b, stream := beastBrawl(t, 3, "")
	b.hardenBandits()
	b.fight()
	ash := b.beast()
	ash.Character.Health = 1
	tamsin := b.companion(1)
	mana := tamsin.Character.Mana
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	assert.Contains(t, out, "(rally")
	statuses := []string{}
	for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
		statuses = append(statuses, e.Status)
	}
	assert.Contains(t, statuses, "Rally")
	assert.Greater(t, ash.Character.Health, 1)
	assert.Equal(t, 1, tamsin.Character.RT.Rallies)
	assert.Equal(t, mana, tamsin.Character.Mana, "no mana")
}

func TestADrakeBreathesFireOnAFoeAndItsNeighbours(t *testing.T) {
	b, stream := beastBrawl(t, 10, "dragon-tamer")
	b.hardenBandits()
	var out string
	n := len(*stream)
	for i := 0; i < 4; i++ {
		b.toughen()
		b.hardenBandits()
		out += b.fight()
	}
	drake := b.beast()
	assert.Equal(t, "drake", drake.Character.RT.Beast.Kind)
	assert.Contains(t, out, "(breath")
	breaths := 0
	for _, e := range abilityEvents(since(*stream, n), drake.Character.Name) {
		if e.Status == "Breath" {
			breaths++
		}
	}
	assert.Equal(t, 1, breaths, "every 3 rounds: once in 4")
}

func TestABearGuardsAHurtAllyAndCountsTheGuard(t *testing.T) {
	b, _ := beastBrawl(t, 10, "bearward")
	b.hardenBandits()
	b.fight()
	assert.Equal(t, 3, b.beast().Character.RT.Beast.Guards)
	guards := 0
	for i := 0; i < 6 && guards == 0; i++ {
		b.companion(2).Character.Health = 1
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, b.companion(2).InstanceId, characters.DefaultAttack)
		}
		b.toughen()
		b.companion(2).Character.Health = 1
		b.hardenBandits()
		out := b.fight()
		if strings.Contains(out, "steps in") {
			guards++
		}
	}
	assert.Positive(t, guards, "the bear guarded a hurt ally")
	assert.Positive(t, b.beast().Character.RT.Beast.GuardsUsed, "the bear counts the guard")
}

func TestAWarhoundsBiteHobblesAWoundedFoe(t *testing.T) {
	b, _ := beastBrawl(t, 10, "houndmaster")
	b.hardenBandits()
	b.fight()
	hound := b.beast()
	assert.Equal(t, "warhound", hound.Character.RT.Beast.Kind)
	assert.True(t, hound.Character.RT.Beast.Hobble)
	hobbled := false
	for i := 0; i < 5 && !hobbled; i++ {
		for _, m := range b.livingBandits() {
			m.Character.Health = max(1, m.Character.HealthMax.Value/4)
		}
		b.toughen()
		out := b.fight()
		hobbled = strings.Contains(out, "(hobbled)")
		_ = out
	}
	assert.True(t, hobbled, "a wounded foe is hobbled by the bite")
}

func TestAPlayerBeastTamerSendsItsBeastIn(t *testing.T) {
	hooks.ResetBeastsForTest()
	domain.ResetBeastsForTest()
	t.Cleanup(domain.ResetBeastsForTest)
	t.Cleanup(beasts.UseSpawnHookForTest(func(m *mobs.Mob) {
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}))
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.withArchetypesFor("beasttamer", map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"tamsin", "garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	b.start()
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	tamer, ok := beasts.Of(7, domain.LeaderMemberKey)
	require.True(t, ok)
	beast, standing := beasts.Live(tamer)
	require.True(t, standing, "the player's beast stands\n%s", out)
	assert.NotEmpty(t, swingsBy(since(*stream, n), beast.Character.Name), "it bites on its own turn")
	assert.Equal(t, 7, beast.Character.RT.Beast.OwnerUser)
	assert.Contains(t, out, "(sic)")
}

// Review fix: the beast command's health is the health the beast stands
// with in battle.
func TestTheBeastCommandShowsTheHealthTheBeastFightsWith(t *testing.T) {
	b, _ := beastBrawl(t, 12, "bearward")
	stood := 0
	t.Cleanup(beasts.UseSpawnHookForTest(func(m *mobs.Mob) {
		stood = m.Character.HealthLimit()
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}))
	b.hardenBandits()
	b.fight()
	require.Positive(t, stood, "the beast stood")
	assert.Equal(t, stood, beasts.HealthLimit(&b.companion(1).Character))
}

// Review fix: a beast sent down the ordinary death path (suicide) is wounded
// on its Tamer's record, never killed with a corpse or a reward.
func TestABeastOnTheDeathPathIsWoundedNotKilled(t *testing.T) {
	b, _ := beastBrawl(t, 3, "")
	b.hardenBandits()
	b.fight()
	ash := b.beast()
	id := ash.InstanceId
	_, err := mobcommands.Suicide("", ash, rooms.LoadRoom(ash.Character.RoomId))
	require.NoError(t, err)
	tamsin := b.companion(1)
	require.NotNil(t, tamsin.Character.Beast)
	assert.True(t, tamsin.Character.Beast.Wounded, "the record keeps the wound")
	_, standing := domain.BeastOf(id)
	assert.False(t, standing)
	assert.Nil(t, mobs.GetInstance(id))
}

// Review fix: "strategy [member] abilities off" holds a Tamer's Sic and
// Rally back, as the help says; the beast still takes its own turn.
func TestAbilitiesOffHoldsSicBack(t *testing.T) {
	b, stream := beastBrawlWith(t, 3, "", func(b *brawl) {
		require.Contains(t, b.cmd("strategy", "tamsin abilities off"), "class abilities")
	})
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	assert.Empty(t, abilityEvents(round, "Tamsin Reed"), "no Sic\n%s", out)
	assert.NotContains(t, out, "(sic)")
	assert.NotEmpty(t, swingsBy(round, "Ash"), "the beast still bites on its own turn")
}

// Review fix: a battle a wounded beast sits out says so, once.
func TestAWoundedBeastSitsTheBattleOutAndSaysSo(t *testing.T) {
	b, _ := beastBrawl(t, 3, "")
	tamsin := b.companion(1)
	tamsin.Character.EnsureBeast("Ash").Wounded = true
	b.hardenBandits()
	out := b.fight()
	out += b.fight()
	assert.Equal(t, 1, strings.Count(out, "sits this battle out"), out)
	_, standing := beasts.Live(beasts.Tamer{Leader: 7, Key: domain.CompanionMemberKey(1)})
	assert.False(t, standing)
}
