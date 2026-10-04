package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33e wiring: automatic class abilities through the real strategy
// command, attack, and combat round (shipped config, DoCombat).

// abilityBrawl is a brawl whose company has archetypes, stands unplaced,
// and has fought one round already with nobody falling; every ability is
// ready, and a tackle always lands unless a test scripts otherwise.
func abilityBrawl(t *testing.T, companions map[int]string) (*brawl, *[]combatstream.Event) {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypesFor("", companions)
	b.unplaced()
	hooks.ResetAbilitiesForTest()
	t.Cleanup(hooks.ResetAbilitiesForTest)
	t.Cleanup(hooks.UseAbilityRollForTest(func(int) int { return 0 }))
	// The shipped statuses, applied as the world loop applies buffs.
	loadStatusBuffs(t)
	buffId := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffId) })
	return b, b.listen()
}

// start opens the fight on the bandit captain, with everyone hard to kill.
func (b *brawl) start() {
	b.t.Helper()
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits()
	b.toughen()
}

// since are the events after the first n.
func since(stream []combatstream.Event, n int) []combatstream.Event {
	return append([]combatstream.Event(nil), stream[n:]...)
}

// abilityEvents are the ability events by a named source.
func abilityEvents(stream []combatstream.Event, source string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range stream {
		if e.Kind == combatstream.Ability && e.Source.Name == source {
			out = append(out, e)
		}
	}
	return out
}

// swingsBy are the weapon attack events by a named source.
func swingsBy(stream []combatstream.Event, source string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range stream {
		if e.Kind == combatstream.Attack && e.Source.Name == source {
			out = append(out, e)
		}
	}
	return out
}

func TestAWarriorCompanionTacklesInTheRealRound(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	// Only Tamsin uses abilities here: the command turns the others off.
	assert.Contains(t, b.cmd("strategy", "garrick abilities off"), "Garrick Vane will use no class abilities")
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)

	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	assert.Regexp(t, `Tamsin Reed tackles the .* to the ground\. \(knocked down\)`, out)
	tackles := abilityEvents(round, "Tamsin Reed")
	require.Len(t, tackles, 1)
	assert.Equal(t, "Tackle", tackles[0].Status)
	assert.Equal(t, combatstream.OutcomeSucceeded, tackles[0].Outcome)
	assert.Equal(t, foe.InstanceId, tackles[0].Target.MobInstanceId)
	assert.Empty(t, swingsBy(round, "Tamsin Reed"), "the tackle was her turn: no swing")
	assert.True(t, status.Live(&foe.Character, status.KnockedDown), "the foe is down")
	assert.Empty(t, abilityEvents(round, "Garrick Vane"), "abilities off")
	assert.Empty(t, abilityEvents(round, "Ysolde"), "abilities off")

	// The next three rounds she rests the ability and swings.
	swung := 0
	for i := 0; i < 3; i++ {
		b.toughen()
		b.hardenBandits()
		n = len(*stream)
		b.fight()
		round = since(*stream, n)
		assert.Empty(t, abilityEvents(round, "Tamsin Reed"), "cooldown, round %d", i+2)
		swung += len(swingsBy(round, "Tamsin Reed"))
	}
	assert.NotZero(t, swung, "she swings while the tackle rests")
	// Four rounds on, with her foe back on its feet, she tackles again; a
	// miss still takes her turn.
	t.Cleanup(hooks.UseAbilityRollForTest(func(int) int { return 99 }))
	// (No status left from the dice: nobody down, stunned, or staggered.)
	for _, m := range b.livingBandits() {
		status.Clear(&m.Character)
	}
	status.Clear(&tamsin.Character)
	b.toughen()
	b.hardenBandits()
	n = len(*stream)
	out = b.fight()
	round = since(*stream, n)
	require.Len(t, abilityEvents(round, "Tamsin Reed"), 1)
	assert.Equal(t, combatstream.OutcomeFailed, abilityEvents(round, "Tamsin Reed")[0].Outcome)
	assert.Regexp(t, `Tamsin Reed lunges to tackle the .*, and misses\. \(tackle missed\)`, out)
	assert.Empty(t, swingsBy(round, "Tamsin Reed"))
}

func TestATackleBreaksAnEnemyChant(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "cleric", 4: "cleric"})
	b.start()
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)
	// The foe begins a long chant at Aria.
	foe.Character.SetCast(5, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}, TargetMobInstanceIds: []int{}})

	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	require.Len(t, abilityEvents(round, "Tamsin Reed"), 1)
	assert.Contains(t, out, "chant breaks off under the blow")
	interrupted := false
	for _, e := range round {
		if e.Kind == combatstream.Interrupt && e.Source.Name == "Tamsin Reed" && e.Outcome == combatstream.OutcomeSucceeded {
			interrupted = true
		}
	}
	assert.True(t, interrupted, "an interrupt credited to Tamsin")
}

func TestARogueStrikesAnOpeningAndARangerAims(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "rogue", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.cmd("strategy", "garrick abilities off")
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Equipment.Weapon = items.New(10004) // a dagger
	tamsin.Character.Equipment.Offhand = items.Item{}
	b.start()
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)

	// A steady foe: no opening. Ysolde aims on the round her sling is
	// ready (a sling waits a round between shots).
	var out string
	var round []combatstream.Event
	for i := 0; i < 3; i++ {
		b.toughen()
		b.hardenBandits()
		n := len(*stream)
		out = b.fight()
		round = since(*stream, n)
		assert.Empty(t, abilityEvents(round, "Tamsin Reed"), "no opening on a foe on its feet")
		if len(abilityEvents(round, "Ysolde")) > 0 {
			break
		}
		assert.Empty(t, swingsBy(round, "Ysolde"), "no aim while the sling waits, and no shot")
	}
	require.Len(t, abilityEvents(round, "Ysolde"), 1)
	assert.Equal(t, "Aimed Shot", abilityEvents(round, "Ysolde")[0].Status)
	assert.Regexp(t, `Ysolde takes careful aim at the .*\. \(aimed shot\)`, out)
	shots := swingsBy(round, "Ysolde")
	require.NotEmpty(t, shots, "the aimed shot is her shot")
	assert.True(t, shots[0].Crit, "every blow lands here: her first is a crit")
	require.NotNil(t, b.companion(4).Character.Aggro)
	assert.NotEqual(t, characters.BackStab, b.companion(4).Character.Aggro.Type, "back to a plain attack after the round")

	// Her foe knocked down: Tamsin strikes the opening until a blow lands.
	landed := false
	for i := 0; i < 12 && !landed; i++ {
		foe = mobs.GetInstance(aimOf(&tamsin.Character))
		require.NotNil(t, foe)
		foe.AddBuff(status.KnockedDown, "combat")
		b.toughen()
		b.hardenBandits()
		hooks.ResetAbilitiesForTest()
		n := len(*stream)
		out = b.fight()
		round = since(*stream, n)
		require.Len(t, abilityEvents(round, "Tamsin Reed"), 1, "round %d", i)
		assert.Equal(t, "Opening Strike", abilityEvents(round, "Tamsin Reed")[0].Status)
		assert.Regexp(t, `Tamsin Reed sees an opening on the .*\. \(opening strike\)`, out)
		for _, e := range swingsBy(round, "Tamsin Reed") {
			if e.Outcome != combatstream.OutcomeMiss {
				landed = true
				assert.True(t, e.Crit, "the opening's landed blow is a crit")
			}
		}
		require.NotNil(t, tamsin.Character.Aggro)
		assert.NotEqual(t, characters.BackStab, tamsin.Character.Aggro.Type)
	}
	assert.True(t, landed, "an opening strike landed")

	// A rogue with a club has no opening strike.
	tamsin.Character.Equipment.Weapon = items.New(10010)
	foe = mobs.GetInstance(aimOf(&tamsin.Character))
	foe.AddBuff(status.KnockedDown, "combat")
	b.toughen()
	hooks.ResetAbilitiesForTest()
	n := len(*stream)
	b.fight()
	assert.Empty(t, abilityEvents(since(*stream, n), "Tamsin Reed"), "a club can't strike an opening")
}

func TestAPlayerUsesTheirOwnAbility(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "cleric", 2: "cleric", 3: "cleric", 4: "cleric"})
	b.aria.Character.SetSkill("brawling", 1)
	b.start()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	require.Len(t, abilityEvents(round, "Aria"), 1)
	assert.Regexp(t, `You tackle the .* to the ground\. \(knocked down\)`, out)
	assert.Empty(t, swingsBy(round, "Aria"), "her tackle was her turn")

}

func TestTwoHealersDoNotHealTheSameAlly(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "cleric", 4: "ranger"})
	oswin, garrick := b.companion(2), b.companion(3)
	for _, m := range []*mobs.Mob{oswin, garrick} {
		m.Character.ManaMax.Value, m.Character.Mana = 40, 40
	}
	b.start()
	// Aria alone is hurt: one heal, not two.
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 300
	n := len(*stream)
	b.fight()
	round := since(*stream, n)
	heals := 0
	for _, e := range round {
		if e.Kind == combatstream.CastStart && e.SpellId == "heal" {
			heals++
		}
	}
	assert.Equal(t, 1, heals, "the second healer leaves Aria to the first")
	chanting := 0
	for _, m := range []*mobs.Mob{oswin, garrick} {
		if m.Character.Aggro != nil && m.Character.Aggro.Type == characters.SpellCast {
			chanting++
		}
	}
	assert.Equal(t, 1, chanting)

	// Next round the first heal is still chanting: it still covers her.
	b.aria.Character.HealthMax.Value = 1000
	n = len(*stream)
	b.fight()
	for _, e := range since(*stream, n) {
		assert.False(t, e.Kind == combatstream.CastStart && e.SpellId == "heal", "no second heal while the first chants")
	}
}

func TestAManaReserveHoldsAttackSpells(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "wizard", 4: "ranger"})
	garrick := b.companion(3)
	garrick.Character.ManaMax.Value, garrick.Character.Mana = 40, 20
	assert.Contains(t, b.cmd("strategy", "garrick reserve 50"), "only while 50% of their mana would be left")
	b.start()
	n := len(*stream)
	b.fight()
	assert.Zero(t, castEvents(since(*stream, n), combatstream.CastStart, "Garrick Vane"), "20-6 would leave less than half")
	assert.Equal(t, 20, garrick.Character.Mana)

	garrick.Character.Mana = 40
	b.toughen()
	n = len(*stream)
	b.fight()
	assert.Equal(t, 1, castEvents(since(*stream, n), combatstream.CastStart, "Garrick Vane"), "40-6 leaves more than half")
}

// alwaysLand makes every blow hit, and no defense or crit roll interfere:
// only an ability's crit is a crit.
func alwaysLand(t *testing.T) {
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

// A tackle breaks a foe's wind-up, credited to the tackler.
func TestATackleBreaksAWindUp(t *testing.T) {
	b, ogre := ogreBrawl(t, "tamsin abilities off", "garrick abilities off", "ysolde abilities off")
	hooks.ResetAbilitiesForTest()
	t.Cleanup(hooks.ResetAbilitiesForTest)
	t.Cleanup(hooks.UseAbilityRollForTest(func(int) int { return 0 }))
	noCounters(t)
	windUpDice(t, 0)
	stream := b.listen()

	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, b.captain().InstanceId, characters.DefaultAttack)
	b.fight()
	require.True(t, hooks.WindingUp(ogre.InstanceId))

	// Now Aria can tackle, and goes for the ogre.
	b.aria.Character.SetSkill("brawling", 1)
	b.ogreOn(ogre, 0)
	b.aria.Character.SetAggro(0, ogre.InstanceId, characters.DefaultAttack)
	out := b.fight()
	assert.Regexp(t, `You tackle the hill ogre to the ground\. \(knocked down\)`, out)
	assert.Contains(t, out, "(Crushing Blow interrupted)")
	broken := interruptsOf(*stream, key(ogre))
	require.NotEmpty(t, broken)
	assert.Equal(t, 7, broken[0].Source.UserId, "Aria's tackle")
	assert.False(t, hooks.WindingUp(ogre.InstanceId))
	assert.Empty(t, windUpsOf(*stream, combatstream.WindUpLand, key(ogre)), "no blow")
}

// 33e review: an ability follows the formation as a swing does. A foe the
// enemy front row shields gets none (the front-row foe would take the
// blow), and a tackle needs hand-to-hand reach; the front-row foe itself
// is tackled.
func TestAbilitiesFollowTheFormation(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "cleric", 4: "cleric"})
	b.start()
	party, ok := enemyparty.PartyOf(b.road, b.captain().InstanceId)
	require.True(t, ok)
	var back, front *mobs.Mob
	for row := 1; row < domain.FormationRows && back == nil; row++ {
		for col := 0; col < domain.FormationCols && back == nil; col++ {
			bk, ok1 := mobparty.InstanceIdFromMemberKey(party.Formation.At(row, col))
			fr, ok2 := mobparty.InstanceIdFromMemberKey(party.Formation.At(0, col))
			if ok1 && ok2 {
				back, front = mobs.GetInstance(bk), mobs.GetInstance(fr)
			}
		}
	}
	require.NotNil(t, back, "a foe behind a front-row foe")
	require.NotNil(t, front)
	_, col, _ := party.Formation.Find(mobparty.MemberKeyFor(back.InstanceId))

	// Tamsin in the front row of that column, with a spear's reach.
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	record.Formation = domain.Formation{}
	require.NoError(t, record.Formation.Place(domain.CompanionMemberKey(1), 0, col))
	module.registry.Put(record)
	const spearID = 99331
	items.SetTestItemSpec(&items.ItemSpec{ItemId: spearID, Name: "test spear", Type: items.Weapon, Subtype: items.Stabbing, Hands: 2, Reach: true,
		Damage: items.Damage{DiceRoll: "1d2", Attacks: 1, DiceCount: 1, SideCount: 2}})
	t.Cleanup(func() { items.RemoveTestItemSpec(spearID) })
	tamsin := b.companion(1)
	tamsin.Character.Equipment.Weapon = items.New(spearID)
	tamsin.Character.Equipment.Offhand = items.Item{}

	tamsin.Character.SetAggro(0, back.InstanceId, characters.DefaultAttack, 0)
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	b.fight()
	round := since(*stream, n)
	assert.Empty(t, abilityEvents(round, "Tamsin Reed"), "no tackle past the front row")
	assert.False(t, status.Live(&back.Character, status.KnockedDown))

	tamsin.Character.SetAggro(0, front.InstanceId, characters.DefaultAttack, 0)
	b.toughen()
	b.hardenBandits()
	n = len(*stream)
	b.fight()
	tackles := abilityEvents(since(*stream, n), "Tamsin Reed")
	require.Len(t, tackles, 1, "the front-row foe is within reach")
	assert.Equal(t, front.InstanceId, tackles[0].Target.MobInstanceId)
}

// 33e review: nobody uses an ability while the company prepares to
// retreat.
func TestNoAbilitiesWhileTheCompanyPreparesToRetreat(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.aria.Character.SetSkill("brawling", 1)
	b.start()
	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 99 }))
	assert.Contains(t, b.cmd("retreat", "east"), "begins an ordered retreat")
	n := len(*stream)
	b.fight()
	for _, e := range since(*stream, n) {
		assert.NotEqual(t, combatstream.Ability, e.Kind, "no ability while withdrawing: %s %s", e.Source.Name, e.Status)
	}
}

// A member casting this round uses no ability: Aria, a caster who can
// also tackle, chants her spell instead.
func TestACasterCastingUsesNoAbility(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "cleric", 2: "cleric", 3: "cleric", 4: "cleric"})
	b.aria.Character.SetSkill("brawling", 1)
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 40, 40
	b.cmd("strategy", "me caster")
	b.start()
	n := len(*stream)
	b.fight()
	round := since(*stream, n)
	assert.Equal(t, 1, castEvents(round, combatstream.CastStart, "Aria"))
	assert.Empty(t, abilityEvents(round, "Aria"))
}

// A successful tackle is queued as a buff. The next warrior in the same
// ability pass must see that pending knockdown and retain its physical turn.
func TestWarriorsDoNotDuplicateASuccessfulTackle(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	b.fight()
	var tackles int
	for _, e := range *stream {
		if e.Kind == combatstream.Ability && e.Status == "Tackle" {
			tackles++
			assert.Equal(t, combatstream.OutcomeSucceeded, e.Outcome)
		}
	}
	assert.Equal(t, 1, tackles, "one foe can be knocked down once in the ability pass")
	assert.NotEmpty(t, swingsBy(*stream, "Garrick Vane"), "the second warrior swings")
}
