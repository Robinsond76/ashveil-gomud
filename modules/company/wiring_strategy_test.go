package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
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
			return []string{"heal", "healall"}
		}
		return []string{"heal"}
	case "wizard":
		return []string{"mm"}
	}
	return nil
}

// withArchetypes gives Aria an archetype and her companions the shipped
// ones (Tamsin and Garrick warriors, Oswin a cleric, Ysolde a ranger).
func (b *brawl) withArchetypes(player string) {
	b.t.Helper()
	archetypes.SetProvider(fakeArchetypes{player: player})
	b.t.Cleanup(func() { archetypes.SetProvider(nil) })
	for id, arch := range map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"} {
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
