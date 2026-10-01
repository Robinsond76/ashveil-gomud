package archetype

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wiring (33f2): Read the Trail and Keen Eye through the real go command and
// walking step seam, the track and specialists commands, and the resolver
// other modules call through internal/archetypes.

// banditCamp puts a hostile pair of bandits in room id.
func banditCamp(t *testing.T, id, firstInstance int) {
	t.Helper()
	camp := &rooms.Room{RoomId: id, Zone: "ArchetypeWalk", Title: "Camp", Biome: "testhall",
		Exits: map[string]exit.RoomExit{"west": {RoomId: id - 1}}}
	rooms.SetTestRoom(camp)
	t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	for i := 0; i < 2; i++ {
		mob := &mobs.Mob{InstanceId: firstInstance + i, MobId: 9701, Hostile: true, Character: *characters.New()}
		mob.Character.Name = "bandit"
		mob.Character.RoomId = id
		mob.Character.Health, mob.Character.HealthMax.Value = 10, 10
		mobs.SetTestInstance(mob)
		instance := mob.InstanceId
		t.Cleanup(func() { mobs.RemoveTestInstance(instance) })
		camp.AddMob(instance)
	}
}

// trailRooms: a hall (id) north to a clearing (id+1), whose east exit leads
// to a bandit camp (id+2).
func trailRooms(t *testing.T, id int) *rooms.Room {
	t.Helper()
	from, _ := walkRooms(t, id, "testhall", func(r *rooms.Room) {
		r.Exits["east"] = exit.RoomExit{RoomId: id + 2}
	})
	banditCamp(t, id+2, id*10)
	return from
}

func TestWiringGoReadsTheTrailByCompanionRanger(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from := trailRooms(t, 97600)
	u := walker(t, 120, 97600)
	m.choose(u, "warrior", true)
	withCompanion(t, 120, 97920, 97600, 20, "ranger") // level 20: tracking 3

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Equal(t, 97601, u.Character.RoomId)
	assert.Contains(t, text, "Bran reads the trail:")
	assert.Regexp(t, `\(2\), east\.`, text, "level 3 names the group and counts it")
}

func TestWiringTrailReadsLessAtLevelOneAndNothingWithoutATracker(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from := trailRooms(t, 97610)
	u := walker(t, 121, 97610)
	m.choose(u, "ranger", true)
	u.Character.Skills = map[string]int{"track": 1}

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "You read the trail: fresh tracks, east.")

	// A warrior with no ranger reads nothing.
	from2 := trailRooms(t, 97620)
	w := walker(t, 122, 97620)
	m.choose(w, "warrior", true)
	text = captureText(t, func() {
		_, err := usercommands.Go("north", w, from2, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "the trail")
}

func TestWiringTrailAutoskillOffAndTrackCommand(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from := trailRooms(t, 97630)
	u := walker(t, 123, 97630)
	m.choose(u, "ranger", true)
	u.Character.Skills = map[string]int{"track": 2}
	assert.Contains(t, m.setAutoskill(123, archetypes.UtilityTrail, false), "now off")

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "the trail", "autoskill off: no reading on the move")

	clearing := rooms.LoadRoom(97631)
	text = captureText(t, func() {
		_, err := m.trackCommand("", u, clearing, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "Nobody", "a bare track still reads on demand")
	text = captureText(t, func() {
		_, err := m.trackCommand("bandit", u, clearing, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Usage: track")
}

func TestWiringTrackCommandNeedsATrackerAndIgnoresPeacefulMobs(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	_, clearing := walkRooms(t, 97640, "testhall", func(r *rooms.Room) {
		r.Exits["east"] = exit.RoomExit{RoomId: 97642}
	})
	banditCamp(t, 97642, 976420)
	for _, id := range rooms.LoadRoom(97642).GetMobs() {
		mobs.GetInstance(id).Hostile = false
	}
	u := walker(t, 124, 97641)
	m.choose(u, "warrior", true)
	text := captureText(t, func() {
		_, err := m.trackCommand("", u, clearing, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Nobody here in your company can read a trail")

	u2 := walker(t, 126, 97641)
	m.choose(u2, "ranger", true)
	u2.Character.Skills = map[string]int{"track": 4}
	text = captureText(t, func() {
		_, err := m.trackCommand("", u2, clearing, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "There are no fresh tracks of enemies nearby.", "peaceful mobs leave no report")
}

// secretRooms: a hall (id) north to a room (id+1) with a secret west exit
// to a cache (id+2).
func secretRooms(t *testing.T, id int) *rooms.Room {
	t.Helper()
	from, _ := walkRooms(t, id, "testhall", func(r *rooms.Room) {
		r.Exits["west"] = exit.RoomExit{RoomId: id + 2, Secret: true}
	})
	cache := &rooms.Room{RoomId: id + 2, Zone: "ArchetypeWalk", Title: "Cache", Biome: "testhall"}
	rooms.SetTestRoom(cache)
	t.Cleanup(func() { rooms.RemoveTestRoom(id + 2) })
	return from
}

func TestWiringKeenEyeSpotsASecretExitAndRemembersIt(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from := secretRooms(t, 97650)
	u := walker(t, 127, 97650)
	m.choose(u, "warrior", true)
	withCompanion(t, 127, 97927, 97650, 30, "rogue")

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Bran spots a hidden way:")
	assert.True(t, u.Character.KnowsSecretExit(97651, "west"), "remembered on the character (durable)")
	assert.Contains(t, u.Character.KnownSecretExits, "97651:west")

	details := rooms.GetDetails(rooms.LoadRoom(97651), u)
	_, shown := details.VisibleExits["west"]
	assert.True(t, shown, "a spotted secret exit shows in the leader's exits")
}

func TestWiringKeenEyeWithoutARogueRarelySpots(t *testing.T) {
	testBiomes(t)
	m := wired(t, 1) // the lowest roll
	from := secretRooms(t, 97660)
	u := walker(t, 128, 97660)
	m.choose(u, "warrior", true)
	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "hidden way")
	assert.Empty(t, u.Character.KnownSecretExits)
	details := rooms.GetDetails(rooms.LoadRoom(97661), u)
	_, shown := details.VisibleExits["west"]
	assert.False(t, shown)
}

func TestBestSpecialistIsOwnedPresentAndSwitchable(t *testing.T) {
	m := wired(t, 100)
	room := trapRoom(t, 97670)
	u := trainee(t, 129, room.RoomId)
	m.choose(u, "warrior", true)
	bran := withCompanion(t, 129, 97929, room.RoomId, 10, "ranger")

	sp, ok := archetypes.BestSpecialist(129, archetypes.UtilityPathfinder)
	require.True(t, ok)
	assert.Equal(t, archetypes.Specialist{Name: "Bran", Level: 2}, sp)

	bran.Character.RoomId = 1 // separated
	_, ok = archetypes.BestSpecialist(129, archetypes.UtilityPathfinder)
	assert.False(t, ok, "a companion elsewhere does no work")
	bran.Character.RoomId = room.RoomId

	bran.Character.Health = -1 // downed
	_, ok = archetypes.BestSpecialist(129, archetypes.UtilityPathfinder)
	assert.False(t, ok, "a downed companion does no work")
	bran.Character.Health = 10

	m.setAutoskill(129, archetypes.UtilityPathfinder, false)
	_, ok = archetypes.BestSpecialist(129, archetypes.UtilityPathfinder)
	assert.False(t, ok, "switched off")

	_, ok = archetypes.BestSpecialist(130, archetypes.UtilityPathfinder)
	assert.False(t, ok, "another player's companion never serves them")
}

func TestSpecialistsViewNamesWhoDoesWhat(t *testing.T) {
	m := wired(t, 100)
	room := trapRoom(t, 97680)
	u := trainee(t, 131, room.RoomId)
	m.choose(u, "wizard", true)
	u.Character.Skills = map[string]int{"cast": 2}
	withCompanion(t, 131, 97931, room.RoomId, 20, "rogue")

	text := captureText(t, func() {
		_, err := m.specialistsCommand("", u, room, 0)
		require.NoError(t, err)
	})
	lines := strings.Split(text, "\n")
	find := func(prefix string) string {
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), prefix) {
				return l
			}
		}
		return ""
	}
	assert.Contains(t, find("Keen Eye"), "Bran, level 3")
	assert.Contains(t, find("Haggle"), "Bran, level 3")
	assert.Contains(t, find("Weather Sense"), "you, level 2")
	assert.Contains(t, find("Read the Trail"), "nobody here (a ranger's skill)")
}
