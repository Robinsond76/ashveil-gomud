package company

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33h3 wiring: the brawl world (four companions on the road,
// 920101) with a scripted passage to the verge (920102), written as the
// shipped passages are: refuse in a battle, then MoveRoom.

const passageScript = `
function onCommand(cmd, rest, user, room) {
    if ( cmd != "pull" ) {
        return false;
    }
    if ( user.InBattle() ) {
        SendUserMessage(user.UserId(), "Not while you are fighting.");
        return true;
    }
    SendUserMessage(user.UserId(), "A hidden door swings open and you step through.");
    user.MoveRoom(920102);
    return true;
}
`

func relocationBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	path := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "rooms", "brawl", "920101.js")
	require.NoError(t, os.WriteFile(path, []byte(passageScript), 0600))
	scripting.PruneVMs(true)
	t.Cleanup(func() { scripting.PruneVMs(true) })
	return b
}

// pull runs the passage through the real command path, scripts and all.
func (b *brawl) pull() string {
	b.t.Helper()
	*b.messages = nil
	_, err := usercommands.TryCommand("pull", "lever", b.aria.UserId, 0)
	require.NoError(b.t, err)
	events.ProcessEvents()
	return companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
}

// rounds runs n of the company's round ticks, as the game loop does.
func (b *brawl) rounds(n int) string {
	b.t.Helper()
	*b.messages = nil
	for i := 0; i < n; i++ {
		b.round++
		events.AddToQueue(events.NewRound{RoundNumber: b.round})
		events.ProcessEvents()
	}
	return companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
}

func separation(t *testing.T, companionID int) *domain.Separation {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	c, ok := findCompanion(record, companionID)
	require.True(t, ok)
	return c.Separation
}

func TestScriptedPassageTakesTheCompanyAndSeparatesTheAbsent(t *testing.T) {
	b := relocationBrawl(t)
	ysolde := b.companion(4)
	require.True(t, nativeRuntime{}.Relocate(ysolde.InstanceId, 920103))
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	out := b.pull()

	assert.Equal(t, 920102, b.aria.Character.RoomId)
	assert.Contains(t, out, "A hidden door swings open")
	assert.Contains(t, out, domain.CompanyFollows)
	for id := 1; id <= 3; id++ {
		assert.Equal(t, 920102, b.companion(id).Character.RoomId, "#%d came along", id)
		assert.Contains(t, rooms.LoadRoom(920102).GetMobs(rooms.FindCharmed), b.companion(id).InstanceId)
		assert.NotContains(t, b.road.GetMobs(rooms.FindCharmed), b.companion(id).InstanceId)
	}
	assert.Contains(t, out, "Ysolde was not with you and is separated from the company, and will find the way back to you in about 60 seconds")
	sep := separation(t, 4)
	require.NotNil(t, sep)
	assert.Equal(t, domain.SeparatedByMove, sep.Reason)
	_, tracked := module.instance(7, 4)
	assert.False(t, tracked, "off the map")
	assert.NotContains(t, rooms.LoadRoom(920103).GetMobs(rooms.FindCharmed), ysolde.InstanceId)
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())

	// Durable: the separation is on disk, with the state it left with.
	module.load()
	require.NoError(t, module.loadErr)
	require.NotNil(t, separation(t, 4), "saved")
	status := b.cmd("company", "status")
	assert.Contains(t, status, "separated; back in about 60 seconds")
}

func TestScriptedPassageIsRefusedInABattle(t *testing.T) {
	b := relocationBrawl(t)
	bandit := b.bandits["bandit cutthroat"][0]
	b.aria.Character.Aggro = &characters.Aggro{MobInstanceId: bandit}
	t.Cleanup(func() { b.aria.Character.Aggro = nil })

	out := b.pull()

	assert.Contains(t, out, "Not while you are fighting.")
	assert.Equal(t, 920101, b.aria.Character.RoomId)
	for id := 1; id <= 4; id++ {
		assert.Equal(t, 920101, b.companion(id).Character.RoomId)
		assert.Nil(t, separation(t, id))
	}
}

// A separated companion waits out its time (online only, across a logout
// and restart), never rejoins into a fight, then rejoins beside the leader
// with the health it left with.
func TestSeparatedCompanionRejoinsInTimeNeverIntoAFight(t *testing.T) {
	b := relocationBrawl(t)
	ysolde := b.companion(4)
	ysolde.Character.Health = 9
	require.True(t, nativeRuntime{}.Relocate(ysolde.InstanceId, 920103))
	b.pull()
	require.NotNil(t, separation(t, 4))
	round := util.GetRoundCount()

	// Ten rounds online, then a logout and a restart, then login.
	b.rounds(10)
	assert.Equal(t, domain.DefaultSeparationRounds-10, separation(t, 4).RoundsLeft)
	plugins := func() {
		events.AddToQueue(events.PlayerDespawn{UserId: 7})
		events.ProcessEvents()
		for _, instance := range module.instances[7] {
			nativeRuntime{}.Detach(7, instance)
		}
		module.instances = map[int]map[int]int{}
		module.load()
		require.NoError(t, module.loadErr)
		events.AddToQueue(events.PlayerSpawn{UserId: 7, RoomId: b.aria.Character.RoomId})
		events.ProcessEvents()
	}
	plugins()
	_, tracked := module.instance(7, 4)
	assert.False(t, tracked, "a relog doesn't bring it back early")
	require.NotNil(t, separation(t, 4))
	assert.Equal(t, domain.DefaultSeparationRounds-10, separation(t, 4).RoundsLeft, "the logout's save kept the count; no time passed offline")
	for id := 1; id <= 3; id++ {
		assert.Equal(t, 920102, b.companion(id).Character.RoomId)
	}

	// Its time runs out in a fight: it waits.
	b.aria.Character.Aggro = &characters.Aggro{MobInstanceId: 999999}
	b.rounds(domain.DefaultSeparationRounds - 10)
	require.NotNil(t, separation(t, 4), "never into a fight")
	assert.Zero(t, separation(t, 4).RoundsLeft)
	_, tracked = module.instance(7, 4)
	assert.False(t, tracked)

	b.aria.Character.Aggro = nil
	out := b.rounds(1)
	assert.Nil(t, separation(t, 4))
	assert.Contains(t, out, "Ysolde finds the way back and rejoins you.")
	back := b.companion(4)
	assert.Equal(t, 920102, back.Character.RoomId, "beside the leader, wherever they are")
	assert.Equal(t, 9, back.Character.Health, "as it left, with no recovery while away")
	assert.Equal(t, round, util.GetRoundCount(), "the rounds were the test's, never the world clock's")
}

// A companion left behind by an ordinary walk (here: moved away, as a
// door it couldn't pass would leave it) is separated after two rounds.
func TestStraySweepSeparatesACompanionLeftBehind(t *testing.T) {
	b := relocationBrawl(t)
	oswin := b.companion(2)
	require.True(t, nativeRuntime{}.Relocate(oswin.InstanceId, 920103))

	assert.NotContains(t, b.rounds(1), "lost sight of you")
	assert.Nil(t, separation(t, 2), "one round is an ordinary follow")
	out := b.rounds(1)
	assert.Contains(t, out, "Brother Oswin has lost sight of you and is separated from the company")
	require.NotNil(t, separation(t, 2))
	assert.Equal(t, domain.SeparatedStray, separation(t, 2).Reason)
}

// A fallen companion is untouched by a move: still dead, its rescue time
// as it was, never separated or respawned.
func TestPassageLeavesTheFallenAlone(t *testing.T) {
	b := relocationBrawl(t)
	garrick := b.companion(3)
	_, err := mobcommands.Suicide("", garrick, rooms.LoadRoom(garrick.Character.RoomId))
	require.NoError(t, err)
	events.ProcessEvents()
	record, _ := module.registry.Get(7)
	dead, _ := findCompanion(record, 3)
	require.True(t, dead.Dead())
	remaining := dead.Death.Remaining

	b.pull()

	record, _ = module.registry.Get(7)
	dead, _ = findCompanion(record, 3)
	require.True(t, dead.Dead(), "still fallen")
	assert.Nil(t, dead.Separation)
	assert.Equal(t, remaining, dead.Death.Remaining)
	_, tracked := module.instance(7, 3)
	assert.False(t, tracked, "not respawned")
}

// A quest's room reward takes the company along through the real hook.
func TestQuestRelocationTakesTheCompany(t *testing.T) {
	b := relocationBrawl(t)
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "quests")
	require.NoError(t, os.MkdirAll(dir, 0755))
	quest := "questid: 9303\nname: Summons\ndescription: Go.\nsteps:\n  - id: start\n    description: Begin.\n  - id: end\n    description: Done.\nrewards:\n  roomid: 920102\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "9303-summons.yaml"), []byte(quest), 0600))
	quests.LoadDataFiles()
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(dir, "9303-summons.yaml"))
		quests.LoadDataFiles()
	})
	ysolde := b.companion(4)
	require.True(t, nativeRuntime{}.Relocate(ysolde.InstanceId, 920103))

	out := completeQuest(b, "9303")

	assert.Equal(t, 920102, b.aria.Character.RoomId)
	assert.Contains(t, out, domain.CompanyFollows)
	for id := 1; id <= 3; id++ {
		assert.Equal(t, 920102, b.companion(id).Character.RoomId)
	}
	assert.Contains(t, out, "was not with you and is separated")
	require.NotNil(t, separation(t, 4))
}

// The shipped passages a player opens themselves refuse in a battle
// before they move anyone (Phase 33h3).
func TestShippedPassagesRefuseInABattle(t *testing.T) {
	shipped := shippedWorld(t)
	for _, rel := range []string{
		"rooms/frostfang/26.js", "rooms/frostfang/51.js", "rooms/catacombs/136.js",
		"rooms/dark_forest/565.js", "rooms/dark_forest/568.js",
	} {
		data, err := os.ReadFile(filepath.Join(shipped, rel))
		require.NoError(t, err, rel)
		src := string(data)
		onCommand := strings.Index(src, "function onCommand(")
		require.GreaterOrEqual(t, onCommand, 0, rel)
		guard := strings.Index(src[onCommand:], "user.InBattle()")
		move := strings.Index(src[onCommand:], "MoveRoom(")
		require.GreaterOrEqual(t, guard, 0, "%s refuses in a battle", rel)
		assert.Less(t, guard, move, "%s refuses before it moves anyone", rel)
	}
}

// Review finding (33h3): an admin teleport through the real command brings
// the company standing with the player and separates the one elsewhere.
func TestAdminTeleportTakesTheCompany(t *testing.T) {
	b := relocationBrawl(t)
	b.aria.Role = "admin"
	t.Cleanup(func() { b.aria.Role = "user" })
	ysolde := b.companion(4)
	require.True(t, nativeRuntime{}.Relocate(ysolde.InstanceId, 920103))

	out := b.cmd("teleport", "920102")

	assert.Equal(t, 920102, b.aria.Character.RoomId)
	assert.Contains(t, out, domain.CompanyFollows)
	for id := 1; id <= 3; id++ {
		assert.Equal(t, 920102, b.companion(id).Character.RoomId, "#%d came along", id)
	}
	assert.Contains(t, out, "Ysolde was not with you and is separated")
	require.NotNil(t, separation(t, 4))
}
