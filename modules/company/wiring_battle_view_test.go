package company

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32g2 wiring: the web client's battle view, Company.Battle, through
// the real round (DoCombat on the brawl world) and the gmcp module's
// company feed as a web client receives it. companyview.RefreshUser stands
// in for the game loop's refresh after each round.

// battleViews captures every Company.Battle sent, per user.
func battleViews(t *testing.T) map[int][]map[string]any {
	t.Helper()
	gmcp.AcceptGMCPForTest(users.GetConnectionId(7)) // a web client
	// A fresh login: the feed forgets what it sent user 7 in an earlier test.
	events.AddToQueue(events.PlayerSpawn{UserId: 7})
	events.ProcessEvents()
	got := map[int][]map[string]any{}
	id := events.RegisterListener(gmcp.GMCPOut{}, func(e events.Event) events.ListenerReturn {
		if out, ok := e.(gmcp.GMCPOut); ok && out.Module == "Company.Battle" {
			var body map[string]any
			if raw, ok := out.Payload.([]byte); ok {
				_ = json.Unmarshal(raw, &body)
			}
			got[out.UserId] = append(got[out.UserId], body)
		}
		return events.Cancel // no connection to deliver to
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(gmcp.GMCPOut{}, id) })
	return got
}

func (b *brawl) refresh(userIds ...int) {
	b.t.Helper()
	for _, id := range userIds {
		companyview.RefreshUser(id)
	}
	events.ProcessEvents()
}

func lastView(views map[int][]map[string]any, userId int) map[string]any {
	list := views[userId]
	if len(list) == 0 {
		return nil
	}
	return list[len(list)-1]
}

func viewEnemies(view map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	list, _ := view["enemies"].([]any)
	for _, e := range list {
		m := e.(map[string]any)
		out[m["id"].(string)] = m
	}
	return out
}

// TestBattleViewFollowsTheBattle: the view is {} before the fight; when
// the battle begins it carries the bandits with cells and health words
// and the company's targets by member key; it updates as bandits fall,
// each moving to fallen; a login re-sends it; the end sends {}. The clock
// never moves.
func TestBattleViewFollowsTheBattle(t *testing.T) {
	b := newBrawl(t)
	views := battleViews(t)
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	b.refresh(7)
	require.NotEmpty(t, views[7])
	assert.Empty(t, lastView(views, 7), "no battle: {}")

	require.Contains(t, b.cmd("formation", "move me 1 1"), "Placed") // front row, first column
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.refresh(7)
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	view := lastView(views, 7)
	require.NotEmpty(t, view, "the battle's view is sent")
	assert.NotEmpty(t, view["group"])
	enemies := viewEnemies(view)
	// Every bandit once: on the grid, or fallen already (a cutthroat can
	// fall in the first round).
	seen := map[string]bool{}
	for id := range enemies {
		seen[id] = true
	}
	for _, f := range listOf(view["fallen"]) {
		id := f.(map[string]any)["id"].(string)
		assert.False(t, seen[id], "fallen and on the grid")
		seen[id] = true
	}
	assert.Len(t, seen, 5, "every bandit, each once")
	assert.NotEmpty(t, enemies)
	captain := ""
	for id, e := range enemies {
		assert.NotEmpty(t, e["label"])
		assert.Contains(t, []string{"unhurt", "scratched", "wounded", "badly wounded", "near death"}, e["health"])
		cell := e["cell"].(map[string]any)
		assert.GreaterOrEqual(t, cell["row"].(float64), 0.0)
		if id == "m:"+strconv.Itoa(b.bandits["bandit captain"][0]) {
			captain = id
		}
	}
	require.NotEmpty(t, captain)
	// Placed at the front, Aria reaches the front of the bandits' ranks
	// with her bare hands, as scout marks it, and not whoever stands
	// behind the front.
	reachable := 0
	for _, e := range enemies {
		r, ok := e["reach"].(bool)
		require.True(t, ok, "a placed player sees reach on every enemy")
		if r {
			reachable++
			assert.Equal(t, 0.0, e["cell"].(map[string]any)["row"], "only the front row")
		}
	}
	assert.Positive(t, reachable)
	aims := map[string]string{}
	for _, a := range view["company"].([]any) {
		m := a.(map[string]any)
		aims[m["key"].(string)] = m["target"].(string)
	}
	assert.Equal(t, captain, aims["leader"], "Aria strikes the captain")
	for key := range aims {
		assert.Regexp(t, `^(leader|companion:\d+)$`, key, "members by key, never by name")
	}
	targeted := false
	for _, e := range enemies {
		if tgt, _ := e["target"].(string); tgt == "leader" || len(tgt) > 10 && tgt[:10] == "companion:" {
			targeted = true
		}
	}
	assert.True(t, targeted, "the bandits' targets name company members by key")

	// A refresh with nothing new sends nothing; a login re-sends the view.
	n := len(views[7])
	b.refresh(7)
	assert.Len(t, views[7], n, "unchanged: nothing sent")
	events.AddToQueue(events.PlayerSpawn{UserId: 7})
	events.ProcessEvents()
	b.refresh(7)
	require.Len(t, views[7], n+1, "a login (or copyover) re-sends it")
	assert.Equal(t, lastView(views, 7)["group"], view["group"])

	// Fight on: each fallen bandit leaves the grid for the fallen line.
	sawFallen := false
	for i := 0; i < 300 && len(b.livingBandits()) > 0; i++ {
		b.toughen()
		b.fight()
		b.refresh(7)
		v := lastView(views, 7)
		if len(v) == 0 {
			break
		}
		if fallen, _ := v["fallen"].([]any); len(fallen) > 0 {
			sawFallen = true
			on := viewEnemies(v)
			for _, f := range fallen {
				id := f.(map[string]any)["id"].(string)
				assert.NotContains(t, on, id, "a fallen bandit is off the grid")
				assert.NotEmpty(t, f.(map[string]any)["label"])
			}
		}
	}
	assert.True(t, sawFallen, "a fall was shown")
	require.Empty(t, b.livingBandits())
	b.fight() // the round that closes the battle
	b.refresh(7)
	_, inBattle = battle.Current(7)
	require.False(t, inBattle)
	assert.Empty(t, lastView(views, 7), "the end clears the view")

	assert.Equal(t, turn, util.GetTurnCount())
	assert.Equal(t, round, util.GetRoundCount())
}

// TestBattleViewPerPlayer: with the bandits split into groups of one,
// Aria's view names the groups waiting their turn; Brom, fighting the next
// group in his own battle, gets his own view (his group, no company of
// his), and a player in no battle gets {}.
func TestBattleViewPerPlayer(t *testing.T) {
	b := newBrawl(t)
	views := battleViews(t)
	b.looseBandits()
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.toughen()
	b.fight() // the other groups find Aria and wait
	b.refresh(7)
	aria := lastView(views, 7)
	require.NotEmpty(t, aria)
	assert.Len(t, viewEnemies(aria), 1, "a group of one")
	waiting, _ := aria["waiting"].([]any)
	assert.NotEmpty(t, waiting, "the groups waiting their turn are named")

	brom := users.NewUserRecord(8, 2)
	brom.Username = "brom"
	brom.Password = "$2a$test"
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.Level = 3
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	brom.Character.HealthMax.Value = 1000
	brom.Character.Health = 1000
	users.SetTestUser(brom)
	gmcp.AcceptGMCPForTest(users.GetConnectionId(8))
	b.road.AddPlayer(brom.UserId)
	t.Cleanup(func() { b.road.RemovePlayer(8) })

	b.refresh(8)
	assert.Empty(t, lastView(views, 8), "Brom is in no battle yet")
	// The waiting bandits have a few HP: a crit from Brom would end his
	// battle in the round it began, before the loop could see it.
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}
	for i := 0; i < 10; i++ {
		if _, ok := battle.Current(8); ok {
			break
		}
		b.toughen()
		b.fight()
	}
	bromBattle, ok := battle.Current(8)
	require.True(t, ok, "Brom is fighting the next group")
	b.refresh(7, 8)
	mine := lastView(views, 8)
	require.NotEmpty(t, mine, "Brom gets his own view")
	for id := range viewEnemies(mine) {
		var instance int
		_, err := fmt.Sscanf(id, "m:%d", &instance)
		require.NoError(t, err)
		assert.True(t, bromBattle.Has(instance), "only his battle's group")
	}
	for _, a := range listOf(mine["company"]) {
		assert.Equal(t, "leader", a.(map[string]any)["key"], "Brom has no companions")
	}
	if ariaBattle, ok := battle.Current(7); ok {
		for id := range viewEnemies(lastView(views, 7)) {
			var instance int
			_, err := fmt.Sscanf(id, "m:%d", &instance)
			require.NoError(t, err)
			assert.True(t, ariaBattle.Has(instance), "Aria's view shows only her battle's group")
		}
	}
}

func listOf(v any) []any {
	list, _ := v.([]any)
	return list
}

// companyAims is a view's company targets, by member key.
func companyAims(view map[string]any) map[string]string {
	out := map[string]string{}
	for _, a := range listOf(view["company"]) {
		m := a.(map[string]any)
		out[m["key"].(string)] = m["target"].(string)
	}
	return out
}

// TestBattleViewCompanionsOthersDarkAndTelnet (32g2 review finding 4):
// through real rounds, companions' targets are sent by member key and
// change as they turn to new foes; a bandit set on another player in the
// room names them under others; in the dark the view carries nothing but
// that; and a connection that hasn't accepted GMCP gets no view.
func TestBattleViewCompanionsOthersDarkAndTelnet(t *testing.T) {
	b := newBrawl(t)
	views := battleViews(t)
	b.aimAt("bandit captain")

	// Companions' targets, and their changes as bandits fall.
	aimSets := map[string]bool{}
	sawCompanion := false
	for i := 0; i < 6 && len(b.livingBandits()) > 1; i++ {
		b.toughen()
		b.fight()
		b.refresh(7)
		for key, target := range companyAims(lastView(views, 7)) {
			if key != "leader" {
				sawCompanion = true
				assert.Regexp(t, `^companion:\d+$`, key)
				assert.Regexp(t, `^m:\d+$`, target)
			}
		}
		raw, _ := json.Marshal(lastView(views, 7)["company"])
		aimSets[string(raw)] = true
	}
	assert.True(t, sawCompanion, "companions' targets are in the view, by key")
	assert.Greater(t, len(aimSets), 1, "a companion's new target is sent")

	// A bandit set on Brom, who isn't in Aria's company.
	brom := users.NewUserRecord(8, 2)
	brom.Username = "brom"
	brom.Password = "$2a$test"
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	users.SetTestUser(brom)
	b.road.AddPlayer(brom.UserId)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	living := b.livingBandits()
	require.NotEmpty(t, living)
	cur, ok := battle.Current(7)
	require.True(t, ok)
	var bandit = living[0]
	for _, m := range living {
		if cur.Has(m.InstanceId) {
			bandit = m
			break
		}
	}
	bandit.Character.SetAggro(8, 0, characters.DefaultAttack)
	b.refresh(7)
	view := lastView(views, 7)
	assert.Equal(t, []any{map[string]any{"id": "u:8", "name": "Brom"}}, view["others"], "an outsider is named under others")
	assert.Equal(t, "u:8", viewEnemies(view)["m:"+strconv.Itoa(bandit.InstanceId)]["target"])

	// In the dark: nothing but that.
	biome := b.road.Biome
	b.road.Biome = "cave"
	t.Cleanup(func() { b.road.Biome = biome })
	b.refresh(7)
	dark := lastView(views, 7)
	assert.Equal(t, true, dark["dark"], "too dark to make them out")
	assert.Empty(t, dark["enemies"])
	assert.Nil(t, dark["fallen"])
	assert.Nil(t, dark["others"])
	b.road.Biome = biome

	// A telnet client that hasn't accepted GMCP: nothing.
	gmcp.RefuseGMCPForTest(users.GetConnectionId(7))
	t.Cleanup(func() { gmcp.AcceptGMCPForTest(users.GetConnectionId(7)) })
	n := len(views[7])
	b.fight()
	b.refresh(7)
	assert.Len(t, views[7], n, "no view without GMCP")
}
