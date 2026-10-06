package gmcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 54: the battle feed carries the sigil the battle began in, so the
// battle screen's header and the Combat tab can show it.
func TestBattleFeedCarriesTheSigil(t *testing.T) {
	if sigilFact(battle.Battle{}) != nil {
		t.Error("no sigil, no fact")
	}
	exp := time.Now().Add(10 * time.Minute).Unix()
	f := sigilFact(battle.Battle{Sigil: sigils.Stillness, SigilExpires: exp})
	if f == nil || f.Kind != "stillness" || f.Name != "stillness sigil" || f.Expires != exp || !strings.Contains(f.Effect, "chants") {
		t.Fatalf("stillness: %+v", f)
	}
	raw, _ := json.Marshal(buildBattle(battleFacts{InBattle: true, Sigil: f}))
	if !strings.Contains(string(raw), `"sigil":{"kind":"stillness","name":"stillness sigil"`) {
		t.Errorf("payload: %s", raw)
	}
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true, Dark: true, Sigil: f}))
	if !strings.Contains(string(raw), `"sigil":{"kind":"stillness"`) {
		t.Errorf("the dark hides the foes, not the sigil: %s", raw)
	}
	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true}))
	if strings.Contains(string(raw), "sigil") {
		t.Errorf("no sigil: %s", raw)
	}
}

// Phase 54: Room.Info lists the sigils lit in the room, for everyone, marks
// the viewer's own, and omits the field when none is lit or one has faded.
func TestRoomInfoCarriesTheSigilsLitHere(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "gmcpsig", Name: "Meadow", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("gmcpsig") })
	room := &rooms.Room{RoomId: 990451, Zone: "Meadow", Biome: "gmcpsig"}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(990451) })

	owner := users.NewUserRecord(7, 1)
	owner.Character.Name = "Aria"
	owner.Character.RoomId = room.RoomId
	viewer := users.NewUserRecord(8, 2)
	viewer.Character.RoomId = room.RoomId
	users.SetTestUser(owner)
	users.SetTestUser(viewer)
	t.Cleanup(func() { users.RemoveTestUser(7); users.RemoveTestUser(8) })

	g := &GMCPRoomModule{}
	info := func(u *users.UserRecord) (GMCPRoomModule_Payload, string) {
		data, _ := g.GetRoomNode(u, `Room.Info`)
		p, ok := data.(GMCPRoomModule_Payload)
		require.True(t, ok)
		raw, err := json.Marshal(p)
		require.NoError(t, err)
		return p, string(raw)
	}

	_, raw := info(viewer)
	assert.NotContains(t, raw, "sigils", "no sigil, no field")

	owner.Character.Sigil = sigils.Lay(sigils.Ward, room.RoomId, time.Now())
	p, raw := info(viewer)
	require.Len(t, p.Sigils, 1)
	assert.Equal(t, "ward", p.Sigils[0].Kind)
	assert.Equal(t, "Aria", p.Sigils[0].Owner)
	assert.False(t, p.Sigils[0].Mine, "the viewer's company did not lay it")
	assert.Contains(t, raw, `"sigils":[{"kind":"ward","name":"ward sigil"`)
	assert.True(t, func() bool { q, _ := info(owner); return q.Sigils[0].Mine }(), "the layer's own is marked")

	owner.Character.Sigil = sigils.Lay(sigils.Ward, room.RoomId, time.Now().Add(-time.Hour))
	_, raw = info(viewer)
	assert.NotContains(t, raw, "sigils", "a faded sigil is gone")
}
