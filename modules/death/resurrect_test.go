package death

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// riteWorld is the test world with keepers configured, a living player,
// and a fake company with one dead companion, "Bran".
type riteWorld struct {
	*testWorld
	keepers map[int]int // room -> keeper template present
	raised  []string
	raiseFn func(selector string, roomID int) (company.ResurrectionResult, error)
}

func newRiteWorld(t *testing.T) *riteWorld {
	t.Helper()
	w := &riteWorld{testWorld: newTestWorld(t), keepers: map[int]int{18: 4, 2007: 65, 2005: 66}}
	w.module.cfg = parseSettings(configMap(map[string]any{
		"Settlements": []any{
			map[string]any{"Zone": "Frostfang", "Kind": "city", "ServiceRoomId": 18, "ServiceMobId": 4},
			map[string]any{"Zone": "Dunmar", "Kind": "city", "ServiceRoomId": 2007, "ServiceMobId": 65},
			map[string]any{"Zone": "Old Kings Road", "Kind": "village", "ServiceRoomId": 2005, "ServiceMobId": 66},
			map[string]any{"Zone": "Bleakmoor", "Kind": "city", "ServiceRoomId": 3002},
		},
	}))
	w.rooms[3002].Tags = []string{"church"}
	w.user.Character.Health = w.user.Character.HealthMax.Value
	w.module.keeper = func(room *rooms.Room, mobID int) (string, bool) {
		if w.keepers[room.RoomId] == mobID && mobID > 0 {
			return "Sister Maren", true
		}
		return "", false
	}
	w.module.deadCompanions = func(int) []company.DeadCompanionView {
		return []company.DeadCompanionView{{ID: 2, Name: "Bran", Level: 4, Remaining: 9000}}
	}
	w.raiseFn = func(selector string, roomID int) (company.ResurrectionResult, error) {
		return company.ResurrectionResult{ID: 2, Name: "Bran", Level: 3, Spawned: true}, nil
	}
	w.module.raise = func(_ int, selector string, roomID int) (company.ResurrectionResult, error) {
		w.raised = append(w.raised, selector)
		return w.raiseFn(selector, roomID)
	}
	return w
}

func (w *riteWorld) rite(t *testing.T, roomID int, rest string) string {
	t.Helper()
	w.user.Character.RoomId = roomID
	handled, err := w.module.resurrectCommand(rest, w.user, w.rooms[roomID], 0)
	require.NoError(t, err)
	require.True(t, handled)
	return w.text()
}

func TestParseKeeper(t *testing.T) {
	s := parseSettings(configMap(map[string]any{
		"Settlements": []any{
			map[string]any{"Zone": "Dunmar", "Kind": "city", "ServiceRoomId": 2007, "ServiceMobId": "65"},
			map[string]any{"Zone": "Frostfang", "Kind": "city", "ServiceRoomId": 18},
		},
	}))
	d, _ := s.registry.ServiceAt(2007)
	assert.Equal(t, 65, d.ServiceMobID)
	f, _ := s.registry.ServiceAt(18)
	assert.Zero(t, f.ServiceMobID)
}

func TestResurrectRefusedOutsideService(t *testing.T) {
	w := newRiteWorld(t)
	for _, room := range []int{2001, 2002, 17} {
		assert.Contains(t, w.rite(t, room, "bran"), "no one here who can call back the dead", room)
	}
	// A registered room that lost its tag isn't a service room either.
	w.rooms[2007].Tags = nil
	assert.Contains(t, w.rite(t, 2007, "bran"), "no one here who can call back the dead")
	assert.Empty(t, w.raised)
}

func TestResurrectNeedsKeeper(t *testing.T) {
	w := newRiteWorld(t)
	delete(w.keepers, 2007) // Sister Maren is away (or dead)
	assert.Contains(t, w.rite(t, 2007, "bran"), "no one to perform the rite")
	assert.Contains(t, w.rite(t, 3002, "bran"), "no one to perform the rite", "no keeper configured")
	assert.Empty(t, w.raised)
}

func TestResurrectNotWhileFighting(t *testing.T) {
	w := newRiteWorld(t)
	w.user.Character.Aggro = &characters.Aggro{MobInstanceId: 5}
	assert.Contains(t, w.rite(t, 2007, "bran"), "Not while you are fighting")
	assert.Empty(t, w.raised)
}

func TestResurrectAtChurchAndShaman(t *testing.T) {
	w := newRiteWorld(t)
	out := w.rite(t, 2007, "bran")
	assert.Contains(t, out, "Bran draws breath again")
	assert.Contains(t, tagPattern.ReplaceAllString(out, ""), "now level 3")
	out = w.rite(t, 2005, "bran")
	assert.Contains(t, out, "Bran draws breath again", "a village shaman")
	assert.Equal(t, []string{"bran", "bran"}, w.raised)

	w.raiseFn = func(string, int) (company.ResurrectionResult, error) {
		return company.ResurrectionResult{ID: 2, Name: "Bran", Level: 3}, nil
	}
	assert.Contains(t, w.rite(t, 2007, "bran"), "rejoin you when you next return")
}

func TestResurrectErrors(t *testing.T) {
	w := newRiteWorld(t)
	for err, want := range map[error]string{
		company.ErrUnknownMember: `None of your company answers to "bran".`,
		company.ErrNotDead:       "bran is not dead.",
		company.ErrCompanionLost: "It is too late",
		errors.New("disk full"):  "The rite falters",
	} {
		w.raiseFn = func(string, int) (company.ResurrectionResult, error) { return company.ResurrectionResult{}, err }
		assert.Contains(t, w.rite(t, 2007, "bran"), want)
	}
}

func TestResurrectListing(t *testing.T) {
	w := newRiteWorld(t)
	out := w.rite(t, 2002, "")
	assert.Contains(t, out, "#2 Bran, level 4: 2h 30m")
	w.module.deadCompanions = func(int) []company.DeadCompanionView { return nil }
	assert.Contains(t, w.rite(t, 2002, ""), "None of your company lies dead.")
}
