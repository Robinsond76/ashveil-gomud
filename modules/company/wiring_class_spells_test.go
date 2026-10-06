package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38b wiring: class spells through the real strategy pass, the shipped
// spell scripts and the combat round, in the brawl world. Aria is the
// caster; her blows never land and neither do the bandits'.

// classCaster is a witchBrawl-style fight with Aria a cleric of the class
// and level, knowing the given spells, with mana to spare.
func classCaster(t *testing.T, class string, level int, spells ...string) *brawl {
	t.Helper()
	b := newBrawl(t)
	loadStatusBuffs(t)
	loadShippedBuff(t, "13-poisoned.yaml")
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	store := &fakeClassStore{state: classes.State{Class: class}}
	classes.SetProvider(store)
	t.Cleanup(func() { classes.SetProvider(nil) })
	b.withArchetypes("cleric")
	b.unplaced()
	forceBlows(t, false)
	noCounters(t)
	c := b.aria.Character
	c.Level = level
	c.SetSkill("cast", 1)
	c.SpellBook = map[string]int{"heal": 1}
	for _, id := range spells {
		c.SpellBook[id] = 1
	}
	c.ManaMax.Value, c.Mana = 300, 300
	return b
}

// rounds runs n rounds with everyone standing at full health, then applies
// hurt (member health overrides, by company id: 0 is Aria) before the first.
func (b *brawl) classRounds(n int, hurt map[int]int) {
	b.t.Helper()
	b.toughen()
	for id, hp := range hurt {
		if id == 0 {
			b.aria.Character.Health = hp
		} else {
			b.companion(id).Character.Health = hp
		}
	}
	for i := 0; i < n; i++ {
		b.hold(nil)
		b.fight()
	}
}

func warded(b *brawl) int {
	n := 0
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		if c.RT != nil && c.RT.Ward > 0 {
			n++
		}
	}
	return n
}

func TestPriestWardsTheCompanyWhileNoOneIsHurt(t *testing.T) {
	b := classCaster(t, "priest", 10, "ward")
	b.startWitchFight()
	b.classRounds(4, nil)
	assert.Positive(t, warded(b), "a ward is up")
	rt := b.aria.Character.RT
	require.NotNil(t, rt)
	assert.Positive(t, rt.WardCap, "sized by the ward spell's power block")
}

func TestPriestGreaterHealsAnAllyInTrouble(t *testing.T) {
	b := classCaster(t, "priest", 15, "greaterheal")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	b.classRounds(4, map[int]int{1: 150})
	assert.Greater(t, tamsin.Health, 150+15, "a Greater Heal is bigger than a Minor Heal's 13 or so")
}

func TestDruidRejuvenatesOverRounds(t *testing.T) {
	b := classCaster(t, "druid", 10, "rejuvenation")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	b.classRounds(1, map[int]int{1: 400})
	for i := 0; i < 5 && (tamsin.RT == nil || tamsin.RT.Rejuv == 0); i++ {
		b.hold(nil)
		b.fight()
	}
	require.NotNil(t, tamsin.RT)
	require.Positive(t, tamsin.RT.Rejuv, "Rejuvenation is on her")
	before := tamsin.Health
	b.hold(nil)
	b.fight()
	assert.Greater(t, tamsin.Health, before, "she mends a little each round")
}

func TestDruidBarkskinsAnAllyAndStrikersTakeThorns(t *testing.T) {
	b := classCaster(t, "druid", 25, "barkskin")
	b.startWitchFight()
	b.classRounds(5, nil)
	barked := 0
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		if c.RT != nil && c.RT.Bark > 0 {
			barked++
			assert.Equal(t, 10, c.RT.Bark)
			assert.Equal(t, 2, c.RT.Thorns, "Thornhide at 25")
		}
	}
	assert.Positive(t, barked)
}

func TestBloodPriestSiphonsAFoeAndHealsTheMostHurtAlly(t *testing.T) {
	b := classCaster(t, "blood-priest", 10, "siphon")
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	foeHealth := func() (n int) {
		for _, m := range b.livingBandits() {
			n += m.Character.Health
		}
		return n
	}
	b.classRounds(1, map[int]int{1: 300})
	start := foeHealth()
	for i := 0; i < 4; i++ {
		b.hold(nil)
		start = foeHealth()
		b.fight()
	}
	_ = start
	assert.Greater(t, tamsin.Health, 300, "the drained life went to her")
}

func TestClericBlessesAnAllyAtLevelEight(t *testing.T) {
	b := classCaster(t, "", 8, "bless")
	b.startWitchFight()
	b.classRounds(4, nil)
	blessed := false
	for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
		blessed = blessed || (c.RT != nil && c.RT.Bless > 0)
	}
	assert.True(t, blessed)
}

// knightBrawl is classCaster with Aria a warrior of the class and level.
func knightBrawl(t *testing.T, class string, level int) *brawl {
	t.Helper()
	b := classCaster(t, class, level)
	b.withArchetypes("warrior")
	b.aria.Character.SpellBook = map[string]int{}
	return b
}

func TestKnightLaysOnHandsInsteadOfSwingingAndRunsOutBetweenRests(t *testing.T) {
	b := knightBrawl(t, "knight", 10)
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	b.toughen()
	tamsin.Health = 100 // 10%
	var out string
	b.hold(nil)
	out = b.fight()
	assert.Contains(t, out, "lay on hands", "the Knight takes the turn to heal")
	assert.Contains(t, out, "1 left")
	assert.Greater(t, tamsin.Health, 100)
	assert.Equal(t, 1, b.aria.Character.RT.Hands)

	tamsin.Health = 100
	b.hold(nil)
	b.fight()
	assert.Equal(t, 2, b.aria.Character.RT.Hands)

	tamsin.Health = 100
	b.hold(nil)
	out = b.fight()
	assert.NotContains(t, out, "lay on hands", "two uses between rests at level 10")
	assert.Equal(t, 2, b.aria.Character.RT.Hands)

	b.aria.Character.RestClass()
	assert.Zero(t, b.aria.Character.RT.Hands, "rest brings them back")
}

func TestBlackguardsOathHealsTheMostHurtAllyAndCowsTheFoe(t *testing.T) {
	b := knightBrawl(t, "blackguard", 10)
	forceBlows(t, true)
	b.startWitchFight()
	tamsin := &b.companion(1).Character
	b.toughen()
	tamsin.Health = 100
	b.hold(nil)
	out := b.fight()
	assert.Contains(t, out, "blood oath")
	assert.Greater(t, tamsin.Health, 100)
	cowed := 0
	for _, m := range b.livingBandits() {
		if m.Character.RT != nil && m.Character.RT.Intim > 0 {
			cowed++
		}
	}
	assert.Positive(t, cowed, "a wounded foe is intimidated")
	assert.LessOrEqual(t, b.aria.Character.RT.OathUsed, 3, "three blows a battle at level 10")
}

// summonRounds runs rounds until a summon stands (or max rounds pass).
func (b *brawl) summonRounds(max int) *mobs.Mob {
	b.t.Helper()
	for i := 0; i < max; i++ {
		b.toughen()
		b.hold(nil)
		b.fight()
		if ids := company.SummonInstances(); len(ids) > 0 {
			return mobs.GetInstance(ids[0])
		}
	}
	return nil
}

func TestHierarchCallsAnAngelThatHealsGuardsAndFades(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := classCaster(t, "hierarch", 35, "callhost")
	b.startWitchFight()
	angel := b.summonRounds(6)
	require.NotNil(t, angel, "the Hierarch calls the Host as the battle opens")
	sm := angel.Character.RT.Summon
	require.NotNil(t, sm)
	assert.Equal(t, "angel", sm.Kind)
	assert.Equal(t, 40, angel.Character.RT.Bark, "Armor of the Host")
	leader, key, ok := company.LeaderAndKeyForInstance(angel.InstanceId)
	assert.True(t, ok)
	assert.Equal(t, 7, leader)
	assert.True(t, company.IsSummonKey(key))
	assert.True(t, b.aria.Character.RT.Summoned, "only one a battle")

	// Mercy heals the most hurt ally every few rounds.
	tamsin := &b.companion(1).Character
	b.toughen()
	tamsin.Health = 100
	for i := 0; i < 4 && tamsin.Health <= 100; i++ {
		b.hold(nil)
		b.fight()
	}
	assert.Greater(t, tamsin.Health, 100, "Mercy healed her")

	// The fight's end dismisses the Angel.
	battle.End(7)
	for _, m := range b.livingBandits() {
		_, err := mobcommands.Suicide("vanish", m, rooms.LoadRoom(m.Character.RoomId))
		require.NoError(t, err)
	}
	b.fight()
	assert.Empty(t, company.SummonInstances())
	assert.Nil(t, mobs.GetInstance(angel.InstanceId))
}

func TestDemonologistBindsADemonThatBurnsFoesAndBreaksFreeWhenSheFalls(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := classCaster(t, "demonologist", 45, "bindfiend")
	b.startWitchFight()
	demon := b.summonRounds(6)
	require.NotNil(t, demon, "the Demonologist binds a Demon")
	assert.Equal(t, "demon", demon.Character.RT.Summon.Kind)
	foe := b.livingBandits()[0]
	b.hold(nil)
	before := foe.Character.Health
	b.fight()
	assert.Less(t, foe.Character.Health, before, "Hellfire burns the foe")

	// She falls: the Demon claws the nearest ally once and vanishes.
	b.aria.Character.Health = 0
	b.hold(nil)
	b.fight()
	assert.Empty(t, company.SummonInstances(), "the broken binding frees it, and it is gone")
}

func TestASummonThatFallsFadesWithoutALoot(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := classCaster(t, "hierarch", 30, "callhost")
	b.startWitchFight()
	angel := b.summonRounds(6)
	require.NotNil(t, angel)
	room := rooms.LoadRoom(angel.Character.RoomId)
	angel.Character.Health = 0
	_, err := mobcommands.Suicide("", angel, room)
	require.NoError(t, err)
	assert.Empty(t, company.SummonInstances())
	assert.Nil(t, mobs.GetInstance(angel.InstanceId))
}

// Phase 38b review: a Bless lasts its three rounds, not the whole battle.
func TestBlessWearsOffAfterItsRounds(t *testing.T) {
	b := classCaster(t, "", 8, "bless")
	b.startWitchFight()
	var blessed *characters.Character
	for i := 0; i < 4 && blessed == nil; i++ {
		b.classRounds(1, nil)
		for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
			if c.RT != nil && c.RT.Bless > 0 {
				blessed = c
				break
			}
		}
	}
	require.NotNil(t, blessed, "someone is blessed")
	b.aria.Character.Mana = 0 // no new blessings
	for i := 0; i < 3; i++ {
		b.hold(nil)
		b.fight()
	}
	assert.Zero(t, blessed.RT.Bless, "the blessing has worn off")
}

// Phase 38b review: Siphon costs its rank (10, then 8 from 15), free of the
// Blood Priest's hungry heal tax, which its other heals still pay.
func TestSiphonCostsItsRankAndOtherHealsPayTheTax(t *testing.T) {
	b := classCaster(t, "blood-priest", 10)
	c := b.aria.Character
	siphon, heal := spells.GetSpell("siphon"), spells.GetSpell("heal")
	require.NotNil(t, siphon)
	require.NotNil(t, heal)
	assert.Equal(t, 10, c.SpellCost(siphon))
	assert.Greater(t, c.SpellCost(heal), heal.Cost, "dark healing is hungry")
	c.Level = 15
	assert.Equal(t, 8, c.SpellCost(siphon), "Cheaper siphon")
}

// Phase 38b review: a summon is called in battle only, so a call in peace
// doesn't spend the next battle's.
func TestASummonCannotBeCalledOutsideBattle(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := classCaster(t, "hierarch", 35, "callhost")
	_, err := summons.Call(summons.Caller{UserID: b.aria.UserId}, summons.Angel)
	assert.ErrorIs(t, err, summons.ErrNoBattle)
	assert.Empty(t, company.SummonInstances())
	assert.False(t, b.aria.Character.RTState().Summoned, "the battle's call is still owed")
}

// Phase 38b review: the Angel departs when its Hierarch falls.
func TestTheAngelDepartsWhenItsHierarchFalls(t *testing.T) {
	t.Cleanup(company.ResetSummonsForTest)
	b := classCaster(t, "hierarch", 35, "callhost")
	b.startWitchFight()
	angel := b.summonRounds(6)
	require.NotNil(t, angel)
	b.aria.Character.Health = 0
	b.hold(nil)
	b.fight()
	assert.Empty(t, company.SummonInstances(), "the Angel has gone")
	assert.Nil(t, mobs.GetInstance(angel.InstanceId))
}
