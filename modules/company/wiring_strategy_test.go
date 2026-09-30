package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Phase 32d: the real strategy module and its command.
	_ "github.com/GoMudEngine/GoMud/modules/strategy"
)

// Phase 32d wiring: each character fights by its strategy, through the
// real strategy command, attack, and combat round (shipped config,
// DoCombat).

// fakeArchetypes stands in for modules/archetype (which this package's
// tests don't load): Aria's archetype, and the shipped companion spells.
type fakeArchetypes struct{ player string }

func (fakeArchetypes) CanTrain(int, string) (bool, string)      { return true, "" }
func (fakeArchetypes) CanLearnSpell(int, string) (bool, string) { return true, "" }
func (fakeArchetypes) Exists(id string) bool {
	return id == "warrior" || id == "cleric" || id == "wizard" || id == "ranger" || id == "rogue"
}
func (fakeArchetypes) ArchetypeName(id string) (string, bool) { return strings.Title(id), true }
func (f fakeArchetypes) PlayerArchetype(int) (string, bool)   { return f.player, f.player != "" }
func (fakeArchetypes) CompanionSpells(id string, level int) []string {
	switch id {
	case "cleric":
		if level >= 5 {
			return []string{"heal", "tend", "healall"}
		}
		return []string{"heal", "tend"} // Phase 30b: tend (never auto-cast)
	case "wizard":
		if level >= 5 {
			return []string{"mm", "sparks"}
		}
		return []string{"mm"}
	}
	return nil
}

// withArchetypes gives Aria an archetype and her companions the shipped
// ones (Tamsin and Garrick warriors, Oswin a cleric, Ysolde a ranger).
func (b *brawl) withArchetypes(player string) {
	b.withArchetypesFor(player, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
}

// withArchetypesFor is withArchetypes with the companions' archetypes given.
func (b *brawl) withArchetypesFor(player string, companions map[int]string) {
	b.t.Helper()
	archetypes.SetProvider(fakeArchetypes{player: player})
	b.t.Cleanup(func() { archetypes.SetProvider(nil) })
	for id, arch := range companions {
		if err := module.registry.SetCompanionArchetype(7, id, arch); err != nil {
			require.ErrorIs(b.t, err, domain.ErrArchetypeAlreadySet)
		}
	}
}

// unplaced takes everyone out of the formation, so everyone reaches every
// foe (an unplaced attacker fails open at the gates): the rules alone
// decide.
func (b *brawl) unplaced() {
	b.t.Helper()
	for _, who := range []string{"me", "tamsin", "oswin", "garrick", "ysolde"} {
		// A member not placed is refused with an error: that's fine here.
		_, _ = usercommands.TryCommand("formation", "clear "+who, b.aria.UserId, events.CmdSkipScripts)
	}
	events.ProcessEvents()
	f, _ := domain.FormationFor(7)
	for _, key := range []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2), domain.CompanionMemberKey(3), domain.CompanionMemberKey(4)} {
		_, _, placed := f.Find(key)
		require.False(b.t, placed, "%s is placed", key)
	}
}

// shapeBandits sets each bandit's health so every rule has one answer:
// the captain leads (toughest), the bruiser has the most health left, the
// slinger the lowest fraction, and the first cutthroat the least health.
func (b *brawl) shapeBandits() (captain, bruiser, slinger, cutA, cutB int) {
	b.t.Helper()
	set := func(id, max, hp int) {
		m := mobs.GetInstance(id)
		m.Character.HealthMax.Value, m.Character.Health = max, hp
	}
	captain, bruiser, slinger = b.bandits["bandit captain"][0], b.bandits["bandit bruiser"][0], b.bandits["bandit slinger"][0]
	cutA, cutB = b.bandits["bandit cutthroat"][0], b.bandits["bandit cutthroat"][1]
	set(captain, 500, 400)
	set(bruiser, 450, 450)
	set(slinger, 100, 30)
	set(cutA, 10, 5)
	set(cutB, 10, 10)
	return
}

func aimOf(c *characters.Character) int {
	if c.Aggro == nil {
		return 0
	}
	return c.Aggro.MobInstanceId
}

func TestStrategiesSetTheFirstAims(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, bruiser, slinger, cutA, _ := b.shapeBandits()

	assert.Contains(t, b.cmd("strategy", "tamsin strongest"), "Tamsin Reed will go for the foe with the most health left")
	assert.Contains(t, b.cmd("strategy", "garrick target leader"), "Garrick Vane will go for their leader")
	b.cmd("strategy", "ysolde wounded")
	b.cmd("strategy", "me nearest")
	list := b.cmd("strategy", "")
	assert.Regexp(t, `Brother Oswin\s+cleric\s+healer\s+weakest`, list)
	assert.Regexp(t, `You\s+-\s+fighter\s+nearest`, list)

	b.cmd("attack", fmt.Sprintf("#%d", slinger))
	assert.Equal(t, captain, aimOf(b.aria.Character), "Aria: the nearest (front-left)")
	assert.Equal(t, bruiser, aimOf(&b.companion(1).Character), "Tamsin: the strongest")
	assert.Equal(t, cutA, aimOf(&b.companion(2).Character), "Oswin: the weakest (his default)")
	assert.Equal(t, captain, aimOf(&b.companion(3).Character), "Garrick: their leader")
	assert.Equal(t, slinger, aimOf(&b.companion(4).Character), "Ysolde: the most wounded")

	// The upkeep keeps them there (sticky aims) through a round.
	b.toughen()
	b.fight()
	assert.Equal(t, bruiser, aimOf(&b.companion(1).Character))
	assert.Equal(t, captain, aimOf(&b.companion(3).Character))

	// When a target falls, the rule picks again: Tamsin's strongest is now
	// the captain.
	b.road.RemoveMob(bruiser)
	mobs.DestroyInstance(bruiser)
	b.toughen()
	b.fight()
	assert.Equal(t, captain, aimOf(&b.companion(1).Character), "Tamsin turns to the next strongest")
}

func TestAssistFollowsThePlayer(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, slinger, _, _ := b.shapeBandits()
	b.cmd("strategy", "ysolde assist")
	b.cmd("strategy", "me strongest") // the bruiser

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	assert.Equal(t, aimOf(b.aria.Character), aimOf(&b.companion(4).Character), "Ysolde starts on Aria's foe")

	b.aria.Character.SetAggro(0, slinger, characters.DefaultAttack) // as if Aria's foe changed
	b.toughen()
	b.fight()
	assert.Equal(t, slinger, aimOf(&b.companion(4).Character), "Ysolde turns with Aria")
}

func TestDefendGoesForTheFoeOnTheMostHurt(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("strategy", "garrick defend")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.hardenBandits() // the slinger must outlast the rounds (it has 30 HP)
	b.toughen()
	b.fight()

	// Tamsin is badly hurt, and the slinger strikes her: Garrick's foe.
	tamsin := b.companion(1)
	tamsin.Character.Health = 200
	slinger := mobs.GetInstance(b.bandits["bandit slinger"][0])
	slinger.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	b.fight()
	garrick := b.companion(3)
	target := mobs.GetInstance(aimOf(&garrick.Character))
	require.NotNil(t, target, "Garrick has a foe")
	require.NotNil(t, target.Character.Aggro)
	assert.Equal(t, tamsin.InstanceId, target.Character.Aggro.MobInstanceId, "Garrick's foe strikes Tamsin, the most hurt")
}

func TestAPlayerAloneAimsByTheirRule(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.cmd("strategy", "me strongest")
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	room := b.into(920103)
	pair[0].Character.HealthMax.Value, pair[0].Character.Health = 100, 100
	pair[1].Character.HealthMax.Value, pair[1].Character.Health = 120, 120

	b.cmd("attack", "ruffians")
	assert.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character), "the strongest of the group")

	// Weaken the other below the first: alone, she stays on her foe
	// (sticky), then turns by her rule only when it falls.
	pair[1].Character.Health = 50
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.fight()
	assert.Equal(t, pair[1].InstanceId, aimOf(b.aria.Character), "a kept aim is sticky")
	room.RemoveMob(pair[1].InstanceId)
	mobs.DestroyInstance(pair[1].InstanceId)
	b.fight()
	assert.Equal(t, pair[0].InstanceId, aimOf(b.aria.Character), "alone, she turns to the next by her rule")
}

// castEvents are the stream's cast events by kind, for one caster.
func castEvents(events []combatstream.Event, kind combatstream.Kind, name string) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind && e.Source.Name == name {
			n++
		}
	}
	return n
}

func TestAClericCompanionHealsTheHurt(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	stream := b.listen()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	for _, m := range b.livingBandits() { // aims are set: now no one falls
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	b.toughen()
	b.fight()
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type, "no one hurt: Oswin swings")
	assert.Equal(t, 20, oswin.Character.Mana)
	aim := aimOf(&oswin.Character)
	require.NotZero(t, aim)

	// Aria below half: Oswin heals her, with mana and a chant.
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 300
	out := b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "Oswin chants")
	assert.Equal(t, "heal", oswin.Character.Aggro.SpellInfo.SpellId)
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds, "on Aria, the most hurt")
	assert.Equal(t, 17, oswin.Character.Mana, "Minor Heal costs 3")
	assert.Contains(t, out, "Brother Oswin begins a low prayer.", "the spell's own chant line")
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastStart, "Brother Oswin"))

	// Two rounds of chanting, then the heal lands and he turns back to his
	// foe without a "turns toward".
	healed := false
	for i := 0; i < 3 && !healed; i++ {
		b.aria.Character.HealthMax.Value = 1000
		out = b.fight()
		for _, e := range *stream {
			if e.Kind == combatstream.Heal && e.Source.Name == "Brother Oswin" {
				healed = true
			}
		}
	}
	require.True(t, healed, "the heal landed")
	assert.NotContains(t, out, "Brother Oswin turns toward")
	// The heal landed this round (after this round's strategy pass), so he
	// is back on his foe until the next round's pass.
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.DefaultAttack, oswin.Character.Aggro.Type)
	assert.Equal(t, aim, aimOf(&oswin.Character), "back to his foe")
}

func TestAWizardPlayerCastsWithNoCommand(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	b.unplaced()
	_, bruiser, _, _, _ := b.shapeBandits()
	stream := b.listen()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
	assert.Regexp(t, `You\s+wizard\s+caster\s+weakest\s+Magic Missile \(6 mana\)`, b.cmd("strategy", ""))
	b.cmd("strategy", "me strongest")

	b.cmd("attack", fmt.Sprintf("#%d", bruiser))
	require.Equal(t, bruiser, aimOf(b.aria.Character))
	// Phase 30d1: no blow lands, so none breaks her chant.
	forceBlows(t, false)
	b.toughen()
	out := b.fight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, characters.SpellCast, b.aria.Character.Aggro.Type, "she casts with no command")
	assert.Equal(t, "mm", b.aria.Character.Aggro.SpellInfo.SpellId)
	assert.Equal(t, []int{bruiser}, b.aria.Character.Aggro.SpellInfo.TargetMobInstanceIds, "at her rule's choice")
	assert.Equal(t, 14, b.aria.Character.Mana)
	assert.Contains(t, out, "You begin to chant")
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastStart, "Aria"))

	// The spell ends (cast or fizzled) the next round, and she turns back
	// to the bruiser without a word.
	b.toughen()
	out = b.fight()
	assert.Equal(t, 1, castEvents(*stream, combatstream.CastComplete, "Aria"))
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, characters.DefaultAttack, b.aria.Character.Aggro.Type)
	assert.Equal(t, bruiser, aimOf(b.aria.Character))
	assert.NotContains(t, out, "You turn toward")

	// Out of mana, she swings.
	b.aria.Character.Mana = 0
	b.toughen()
	b.fight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, characters.DefaultAttack, b.aria.Character.Aggro.Type, "no mana: she swings")

	// A fighter never casts.
	b.aria.Character.Mana = 20
	b.cmd("strategy", "me fighter")
	assert.Equal(t, 20, b.aria.Character.Mana)
}

func TestCompanionManaComesBackOutOfCombat(t *testing.T) {
	b := newBrawl(t)
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 5
	hooks.AutoHeal(events.NewRound{RoundNumber: 3})
	assert.Equal(t, 5+oswin.Character.ManaPerRound(), oswin.Character.Mana, "out of combat, every third round")

	oswin.Character.Mana = 5
	hooks.AutoHeal(events.NewRound{RoundNumber: 4})
	assert.Equal(t, 5, oswin.Character.Mana, "only every third round")

	oswin.Character.SetAggro(0, b.bandits["bandit captain"][0], characters.DefaultAttack)
	hooks.AutoHeal(events.NewRound{RoundNumber: 6})
	assert.Equal(t, 5, oswin.Character.Mana, "not in combat")
}

// Phase 32d: in a battle only flee takes her out; the setup can't change.
func TestInABattleOnlyFleeWorks(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	b.fight()
	before, _ := domain.FormationFor(7)

	assert.Contains(t, b.cmd("formation", "move tamsin 3 3"), "The battle is under way")
	after, _ := domain.FormationFor(7)
	assert.Equal(t, before, after, "the formation is unchanged")
	assert.Contains(t, b.cmd("formation", ""), "Company formation", "reading it is fine")

	assert.Contains(t, b.cmd("strategy", "tamsin leader"), "The battle is under way")
	assert.NotContains(t, b.cmd("strategy", ""), "leader ", "unchanged")

	assert.Contains(t, b.cmd("east", ""), "Only flee takes you out of it.")
	assert.Equal(t, 920101, b.aria.Character.RoomId)
	// Through the real dispatch, aliases included (32d review).
	for _, c := range [][2]string{{"wear", "sword"}, {"wield", "sword"}, {"unequip", "all"}, {"drink", "water"}, {"eat", "bread"}, {"use", "whetstone"},
		{"company", "eat"}, {"company", "drink"}, {"company", "meal"}} { // 32f's company meals too
		assert.Contains(t, b.cmd(c[0], c[1]), "The battle is under way", c[0])
	}
	assert.Contains(t, b.cmd("break", ""), "Only flee takes you out of it.")

	// Flee still works (made certain: she is far quicker than they are).
	for _, m := range b.livingBandits() {
		m.Character.Stats.Speed.ValueAdj = 0
	}
	b.aria.Character.Stats.Speed.ValueAdj = 100000
	b.cmd("flee", "")
	assert.Contains(t, b.fight(), "You break away and flee east.")
	_, inBattle := battle.Current(7)
	assert.False(t, inBattle, "the flight ended her battle at once")

	// Out of the battle, the formation can change again.
	assert.Contains(t, b.cmd("formation", "move tamsin 3 3"), "Placed Tamsin Reed")
}
