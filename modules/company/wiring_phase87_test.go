package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 87: a ranger's readied Aimed Shot survives its target falling
// earlier in the round. The kill turns her onto another living foe through
// the real reassignment, and the shot (with its bonus) goes with her.
func TestAnAimedShotFollowsItsFallenTarget(t *testing.T) {
	b, _ := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.start()
	b.fight()
	ysolde := b.companion(4)
	room := rooms.LoadRoom(ysolde.Character.RoomId)
	require.NotNil(t, room)
	require.NotNil(t, ysolde.Character.Aggro)

	lost := mobs.GetInstance(aimOf(&ysolde.Character))
	require.NotNil(t, lost)
	ysolde.Character.Aggro.Type = characters.BackStab
	ysolde.Character.Aggro.StrikeBonus = 7
	lost.Character.Health = 0 // the Wizard's spell got there first

	require.True(t, hooks.ReassignCompanionTargetForTest(ysolde, room), "another foe stands to shoot")
	assert.NotEqual(t, lost.InstanceId, aimOf(&ysolde.Character))
	next := mobs.GetInstance(aimOf(&ysolde.Character))
	require.NotNil(t, next)
	assert.Greater(t, next.Character.Health, 0)
	assert.Equal(t, characters.BackStab, ysolde.Character.Aggro.Type, "the aimed shot is not wasted")
	assert.Equal(t, 7, ysolde.Character.Aggro.StrikeBonus)
}

// Phase 87: a heal still chanting when the last foe falls says it goes on,
// instead of landing unexplained after the battle summary.
func TestAHealThatOutlastsTheFightSaysSo(t *testing.T) {
	b, _ := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	heard := b.ariaHears()
	oswin := b.companion(2)
	oswin.Character.SpellBook["heal"] = 1
	oswin.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "heal"})
	room := rooms.LoadRoom(b.aria.Character.RoomId)
	require.NotNil(t, room)

	hooks.NoteFinishingChantsForTest(b.aria, room)
	events.ProcessEvents()
	all := strings.Join(*heard, "")
	assert.Regexp(t, `The fight is over, but .*Oswin.* finishes Minor Heal\. \(chant finishing\)`, all)

	// A harmful spell has nothing left to hit: no such line.
	*heard = nil
	oswin.Character.SpellBook["mm"] = 1
	oswin.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "mm"})
	hooks.NoteFinishingChantsForTest(b.aria, room)
	events.ProcessEvents()
	assert.NotContains(t, strings.Join(*heard, ""), "finishes Magic Missile")
}

// Phase 87 review: an Aimed Shot that kills its target was loosed. The
// kill's turn onto the next foe must not carry the spent shot on, read it
// as never loosed ("finds no target"), or refund its wait.
func TestAKillingAimedShotIsSpent(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "", 30)
	b.cmd("strategy", "tamsin abilities off")
	b.start()
	b.companion(4).Character.Equipment.Weapon = items.New(10014)
	out := rangerRounds(b, stream, 4, isolate(b))
	require.Contains(t, out, "takes careful aim at the first cutthroat")
	assert.NotContains(t, out, "finds no target", "the shot flew and killed")
	assert.Equal(t, 1, strings.Count(out, "takes careful aim at the"), "a loosed Aimed Shot keeps its wait: once in four rounds")
}
