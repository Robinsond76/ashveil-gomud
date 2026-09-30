package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/windup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// windUpWorld is a room with an ogre (8401) winding up at a foe (8402),
// the wind-up state reset after the test.
func windUpWorld(t *testing.T) (ogre, foe int, lines *[]string, stream *[]combatstream.Event) {
	t.Helper()
	room := &rooms.Room{RoomId: 990401}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
	o := engagementMob(t, 8401, 50, room.RoomId)
	o.Character.Name = "hill ogre"
	o.Character.SetAggro(0, 8402, characters.DefaultAttack, 0)
	engagementMob(t, 8402, 50, room.RoomId).Character.Name = "bandit"
	t.Cleanup(func() {
		windUps, windUpCooldown, landing = map[int]*windUpState{}, map[int]int{}, nil
	})

	events.ProcessEvents() // nothing left over from an earlier test
	var got []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.RoomId == room.RoomId {
			got = append(got, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	var evs []combatstream.Event
	t.Cleanup(combatstream.UseForTest(combatstream.New()))
	off := combatstream.Default().Subscribe(func(e combatstream.Event) { evs = append(evs, e) })
	t.Cleanup(off)
	return 8401, 8402, &got, &evs
}

// The power is handed over once, and only for the landing mob.
func TestWindUpPowerOnce(t *testing.T) {
	ogre, _, _, _ := windUpWorld(t)
	a, _ := windup.Get("crushing-blow")
	landing = &landingState{mobId: ogre, ability: a}
	if _, ok := windUpPower(ogre + 1); ok {
		t.Fatal("another mob's blow is ordinary")
	}
	p, ok := windUpPower(ogre)
	require.True(t, ok)
	assert.Equal(t, "Crushing Blow", p.Name)
	assert.Equal(t, 2, p.Multiplier)
	assert.Equal(t, status.KnockedDown, p.Status)
	_, again := windUpPower(ogre)
	assert.False(t, again, "one strike: a second blow in the turn is ordinary")
}

// A landing whose swing never happened is told as wasted; one that struck
// is not.
func TestFinishLandingWastesAnUnstruckBlow(t *testing.T) {
	ogre, _, lines, stream := windUpWorld(t)
	a, _ := windup.Get("crushing-blow")

	landing = &landingState{mobId: ogre, ability: a, struck: true}
	finishLanding()
	events.ProcessEvents()
	assert.Empty(t, *lines, "a blow that struck is told by its own lines")
	assert.Nil(t, landing)

	landing = &landingState{mobId: ogre, ability: a}
	finishLanding()
	events.ProcessEvents()
	require.Len(t, *lines, 1)
	assert.Contains(t, (*lines)[0], "fall, with nothing left to strike. (Crushing Blow wasted)")
	require.Len(t, *stream, 1)
	assert.Equal(t, combatstream.WindUpLand, (*stream)[0].Kind)
	assert.Equal(t, combatstream.OutcomeWasted, (*stream)[0].Outcome)
}

// The turn: winding up spends it; the landing re-aims at the named target
// with no wait; the cooldown counts its turns; a mob without windups never
// starts.
func TestWindUpTurnLandsThenCoolsDown(t *testing.T) {
	ogreId, foeId, _, _ := windUpWorld(t)
	a, _ := windup.Get("crushing-blow")
	ogre := mustMob(t, ogreId)
	windUps[ogreId] = &windUpState{ability: a, turnsLeft: 1, mobId: foeId}
	ogre.Character.SetAggro(0, 999, characters.DefaultAttack, 3) // re-aimed meanwhile

	assert.False(t, windUpTurn(ogre), "it lands: the swing follows")
	assert.True(t, isLanding(ogreId))
	assert.Equal(t, foeId, ogre.Character.Aggro.MobInstanceId, "at the target it named")
	assert.Zero(t, ogre.Character.Aggro.RoundsWaiting)
	assert.Equal(t, windup.Cooldown, windUpCooldown[ogreId])

	for i := windup.Cooldown; i > 0; i-- {
		assert.False(t, windUpTurn(ogre), "an ordinary swing while it cools down")
	}
	assert.Zero(t, windUpCooldown[ogreId])
	assert.False(t, windUpTurn(ogre), "no windups on this mob: it never starts one")
}

// A foe gone, fallen, or no longer on a plain attack loses its wind-up at
// the round's start.
func TestWindUpRoundPrunes(t *testing.T) {
	ogreId, _, _, _ := windUpWorld(t)
	a, _ := windup.Get("crushing-blow")
	ogre := mustMob(t, ogreId)
	windUps[ogreId] = &windUpState{ability: a, turnsLeft: 1}
	windUpRound()
	assert.Contains(t, windUps, ogreId, "a foe still fighting keeps it")

	ogre.Character.Aggro.Type = characters.SpellCast
	windUpRound()
	assert.NotContains(t, windUps, ogreId, "casting ends it")

	windUps[ogreId] = &windUpState{ability: a, turnsLeft: 1}
	windUpCooldown[ogreId] = 1
	ogre.Character.Health = 0
	windUpRound()
	assert.NotContains(t, windUps, ogreId)
	assert.NotContains(t, windUpCooldown, ogreId)
}

// A foe that loses its turn to a status loses its wind-up, with no one
// credited.
func TestWindUpLostTurn(t *testing.T) {
	ogreId, _, lines, stream := windUpWorld(t)
	a, _ := windup.Get("crushing-blow")
	windUps[ogreId] = &windUpState{ability: a, turnsLeft: 1}
	windUpLostTurn(mustMob(t, ogreId))
	events.ProcessEvents()
	assert.NotContains(t, windUps, ogreId)
	assert.Equal(t, windup.Cooldown, windUpCooldown[ogreId])
	require.Len(t, *lines, 1)
	assert.True(t, strings.HasSuffix(strings.TrimSpace((*lines)[0]), "(Crushing Blow interrupted)"), (*lines)[0])
	assert.Empty(t, *stream, "no interrupt to credit")
}

func mustMob(t *testing.T, id int) *mobs.Mob {
	t.Helper()
	m := mobs.GetInstance(id)
	require.NotNil(t, m)
	return m
}
