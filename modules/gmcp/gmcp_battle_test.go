package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleBattle is the Phase 31 mock: a captain and two cutthroats in
// front, a slinger fallen, a hidden bruiser, and a cutthroat striking
// Brannoc, who isn't in the company.
func sampleBattle() battleFacts {
	return battleFacts{
		InBattle: true,
		Group:    "a band of cutthroats",
		Placed:   true,
		Enemies: []enemyFact{
			{Id: 412, Label: "the cutthroat captain", Standing: true, Row: 0, Col: 0, Health: 30, HealthMax: 50, Reach: true,
				Target: targetFact{Key: "leader"}},
			{Id: 413, Label: "the first cutthroat", Standing: true, Row: 0, Col: 1, Health: 20, HealthMax: 20, Reach: true,
				Target: targetFact{Key: "companion:2"}},
			{Id: 414, Label: "the second cutthroat", Standing: true, Row: 1, Col: 1, Health: 4, HealthMax: 20,
				Target: targetFact{UserId: 9, Name: "Brannoc"}},
			{Id: 415, Label: "the slinger", Seen: true},
			{Id: 418, Label: "the lurker"}, // gone, last seen hidden: not named
			{Id: 416, Label: "the bruiser", Standing: true, Hidden: true, Row: 1, Col: 0, Health: 40, HealthMax: 40,
				Target: targetFact{Key: "leader"}},
			{Id: 417, Label: "the third cutthroat", Standing: true, Row: 0, Col: 2, Health: 20, HealthMax: 20,
				Target: targetFact{UserId: 9, Name: "Brannoc"}},
		},
		Company: []aimFact{
			{Key: "leader", Target: 412},
			{Key: "companion:2", Target: 412},
			{Key: "companion:3", Target: 416}, // hidden: not shown
			{Key: "companion:4"},              // aiming at no one
		},
		Waiting: []string{"a pack of grey wolves"},
	}
}

// TestBattlePayload (32g2): the enemies the player can see with their
// cells, words, reach, and targets; the fallen off the grid; others by
// name once; the company's targets by member key, only on listed enemies.
func TestBattlePayload(t *testing.T) {
	raw, err := json.Marshal(buildBattle(sampleBattle()))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	assert.Equal(t, "a band of cutthroats", got["group"])
	enemies := got["enemies"].([]any)
	require.Len(t, enemies, 4, "the hidden bruiser and the fallen slinger are not on the grid")
	assert.Equal(t, map[string]any{"id": "m:412", "label": "the cutthroat captain", "cell": map[string]any{"row": 0.0, "col": 0.0},
		"health": "wounded", "reach": true, "target": "leader"}, enemies[0])
	assert.Equal(t, "companion:2", enemies[1].(map[string]any)["target"])
	third := enemies[2].(map[string]any)
	assert.Equal(t, "near death", third["health"])
	assert.Equal(t, false, third["reach"])
	assert.Equal(t, "u:9", third["target"])

	assert.Equal(t, []any{map[string]any{"id": "m:415", "label": "the slinger"}}, got["fallen"])
	assert.Equal(t, []any{map[string]any{"id": "u:9", "name": "Brannoc"}}, got["others"], "named once")
	assert.Equal(t, []any{
		map[string]any{"key": "leader", "target": "m:412"},
		map[string]any{"key": "companion:2", "target": "m:412"},
	}, got["company"], "no target on a hidden enemy, none for a member aiming at no one")
	assert.Equal(t, []any{"a pack of grey wolves"}, got["waiting"])
	assert.NotContains(t, string(raw), "\"30\"", "no numbers")
	assert.NotContains(t, string(raw), "health_max")
}

// TestBattlePayloadUnplacedAndNone: an unplaced player gets no reach;
// no battle is {}; a battle whose group is gone still says it is on.
func TestBattlePayloadUnplacedAndNone(t *testing.T) {
	f := sampleBattle()
	f.Placed = false
	raw, _ := json.Marshal(buildBattle(f))
	assert.NotContains(t, string(raw), "reach")

	raw, _ = json.Marshal(buildBattle(battleFacts{}))
	assert.Equal(t, "{}", string(raw))

	raw, _ = json.Marshal(buildBattle(battleFacts{InBattle: true, Enemies: []enemyFact{{Id: 5, Label: "the rat", Seen: true}}}))
	assert.JSONEq(t, `{"group":"the enemy","enemies":[],"fallen":[{"id":"m:5","label":"the rat"}],"focus":"none","saved_focus":"none","focus_ready":false}`, string(raw))
}

// TestBattleExtraSendsOnChange: through the feed, an unchanged battle
// sends nothing, a new health word sends once, and the end sends {}.
func TestBattleExtraSendsOnChange(t *testing.T) {
	facts := sampleBattle()
	f, out := testFeed()
	f.extras = []companyExtra{battleExtra(func(*users.UserRecord) battleFacts { return facts })}
	u := users.NewUserRecord(7, 1)

	f.updateExtras(u)
	f.updateExtras(u)
	require.Len(t, *out, 1)
	assert.Equal(t, "Company.Battle", (*out)[0].module)

	facts.Enemies[0].Health = 29 // still wounded
	f.updateExtras(u)
	require.Len(t, *out, 1, "the same word sends nothing")
	facts.Enemies[0].Health = 20
	f.updateExtras(u)
	require.Len(t, *out, 2, "badly wounded sends once")

	facts = battleFacts{}
	f.updateExtras(u)
	require.Len(t, *out, 3)
	assert.Empty(t, (*out)[2].body, "the end clears the view")
}

// TestBattleExtraNotBuiltWithoutGMCP: a telnet client that hasn't accepted
// GMCP gets no battle view.
func TestBattleExtraNotBuiltWithoutGMCP(t *testing.T) {
	built := false
	f, out := testFeed()
	f.accepting = func(int) bool { return false }
	f.extras = []companyExtra{battleExtra(func(*users.UserRecord) battleFacts { built = true; return sampleBattle() })}
	f.updateExtras(users.NewUserRecord(7, 1))
	assert.False(t, built)
	assert.Empty(t, *out)
}

// TestBattlePayloadDark (32g2 review finding 3): in the dark the view shows
// what scout does: nothing about the enemy but that it is too dark.
func TestBattlePayloadDark(t *testing.T) {
	f := sampleBattle()
	f.Dark = true
	raw, _ := json.Marshal(buildBattle(f))
	// Phase 30c: the company's own focus is still shown (it names no enemy).
	assert.JSONEq(t, `{"group":"the enemy","dark":true,"enemies":[],"focus":"none","saved_focus":"none","focus_ready":false}`, string(raw))
}

// TestSeenEnemies (32g2 review finding 2): whether each enemy was hidden
// when last seen in this battle, forgotten when a new battle begins or the
// player leaves.
func TestSeenEnemies(t *testing.T) {
	a, b := battleSeenKey{fight: 100}, battleSeenKey{fight: 101}
	s := newSeenEnemies()
	s.note(7, a, 21, true)
	s.note(7, a, 22, false)
	assert.True(t, s.hiddenLast(7, a, 21))
	assert.False(t, s.hiddenLast(7, a, 22))
	assert.False(t, s.hiddenLast(7, a, 23), "never seen: not known hidden")
	s.note(7, a, 21, false)
	assert.False(t, s.hiddenLast(7, a, 21), "the last sight counts")
	s.note(7, a, 21, true)
	assert.False(t, s.hiddenLast(8, a, 21))
	assert.False(t, s.hiddenLast(7, b, 21), "another battle starts afresh")
	s.note(7, b, 22, true)
	assert.False(t, s.hiddenLast(7, a, 21), "the old battle is forgotten")
	s.forget(7)
	assert.False(t, s.hiddenLast(7, b, 22))
	s.note(7, a, 1, true)
	s.note(9, a, 1, true)
	s.prune([]int{9})
	assert.False(t, s.hiddenLast(7, a, 1))
	assert.True(t, s.hiddenLast(9, a, 1))
}

// TestBattlePayloadFocus (30c): the focus the company aims by, the saved
// one, and whether an order may be given, dark or not.
func TestBattlePayloadFocus(t *testing.T) {
	f := sampleBattle()
	f.Focus, f.SavedFocus, f.FocusReady = "leader", "", true
	raw, _ := json.Marshal(buildBattle(f))
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "leader", got["focus"])
	assert.Equal(t, "none", got["saved_focus"])
	assert.Equal(t, true, got["focus_ready"])

	f.Dark, f.FocusReady = true, false
	raw, _ = json.Marshal(buildBattle(f))
	assert.JSONEq(t, `{"group":"the enemy","dark":true,"enemies":[],"focus":"leader","saved_focus":"none","focus_ready":false}`, string(raw))
}

// TestBattlePayloadGuards (30c2): each guardian's guards left and its set
// ward (blank: the most hurt), dark or not.
func TestBattlePayloadGuards(t *testing.T) {
	f := sampleBattle()
	f.Guards = []guardFact{{Key: "companion:1", Left: 2, Ward: "leader"}, {Key: "leader", Left: 0}}
	raw, _ := json.Marshal(buildBattle(f))
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, []any{
		map[string]any{"key": "companion:1", "left": 2.0, "ward": "leader"},
		map[string]any{"key": "leader", "left": 0.0, "ward": ""},
	}, got["guards"])

	f.Dark = true
	raw, _ = json.Marshal(buildBattle(f))
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Len(t, got["guards"], 2, "our own guards show in the dark")

	f = sampleBattle()
	raw, _ = json.Marshal(buildBattle(f))
	assert.NotContains(t, string(raw), "guards", "none without a guardian")
}

func TestSurrenderedFoeHasNoCellOrTargetAndDarkHidesIt(t *testing.T) {
	f := battleFacts{InBattle: true, Placed: true, Enemies: []enemyFact{{Id: 1, Label: "first bandit", Surrendered: true, Seen: true}, {Id: 2, Label: "second bandit", Standing: true, Health: 50, HealthMax: 100}}, Company: []aimFact{{Key: "leader", Target: 1}}}
	p := buildBattle(f).(battlePayload)
	require.Len(t, p.Surrendered, 1)
	assert.Equal(t, "first bandit", p.Surrendered[0].Label)
	require.Len(t, p.Enemies, 1)
	assert.Empty(t, p.Company)
	assert.Empty(t, p.Fallen)
	f.Dark = true
	p = buildBattle(f).(battlePayload)
	assert.Empty(t, p.Surrendered)
	assert.Empty(t, p.Enemies)
}

func TestBattleWithdrawalVisibleEvenInDark(t *testing.T) {
	f := sampleBattle()
	f.Retreat = &retreatFact{Exit: "east", Rounds: 2}
	for _, dark := range []bool{false, true} {
		f.Dark = dark
		raw, err := json.Marshal(buildBattle(f))
		require.NoError(t, err)
		var view map[string]any
		require.NoError(t, json.Unmarshal(raw, &view))
		assert.Equal(t, map[string]any{"exit": "east", "rounds": float64(2)}, view["retreat"])
	}
	raw, _ := json.Marshal(buildBattle(battleFacts{Retreat: f.Retreat}))
	assert.JSONEq(t, "{}", string(raw), "runtime order cannot create a battle view on its own")
}

func TestGatherWithdrawalCountdownFromRuntimeOrder(t *testing.T) {
	r := &rooms.Room{RoomId: 929933, Title: "withdrawal fixture"}
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(r.RoomId) })
	u := users.NewUserRecord(33, 1)
	u.Character.RoomId = r.RoomId
	battle.Begin(u.UserId, r.RoomId, 1, "retreat", nil)
	t.Cleanup(func() { battle.End(u.UserId) })
	u.Character.Aggro = &characters.Aggro{Type: characters.Retreat, RoundsWaiting: 1, RetreatInfo: &characters.RetreatInfo{RoomID: r.RoomId, ExitName: "east"}}
	f := gatherBattle(u)
	require.NotNil(t, f.Retreat)
	assert.Equal(t, 2, f.Retreat.Rounds)
	u.Character.Aggro.RoundsWaiting = 0
	f = gatherBattle(u)
	require.NotNil(t, f.Retreat)
	assert.Equal(t, 1, f.Retreat.Rounds)
	u.Character.RoomId++
	assert.False(t, gatherBattle(u).InBattle, "departed company no longer publishes the old battle")
}
