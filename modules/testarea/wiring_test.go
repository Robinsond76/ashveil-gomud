package testarea

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	deathdomain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Everything a trip snapshots, as the server has it.
	_ "github.com/GoMudEngine/GoMud/modules/archetype"
	_ "github.com/GoMudEngine/GoMud/modules/camping"
	_ "github.com/GoMudEngine/GoMud/modules/company"
	_ "github.com/GoMudEngine/GoMud/modules/death"
	_ "github.com/GoMudEngine/GoMud/modules/encounters"
	_ "github.com/GoMudEngine/GoMud/modules/encumbrance"
	_ "github.com/GoMudEngine/GoMud/modules/expedition"
	_ "github.com/GoMudEngine/GoMud/modules/exposure"
	_ "github.com/GoMudEngine/GoMud/modules/mount"
	_ "github.com/GoMudEngine/GoMud/modules/strategy"
	_ "github.com/GoMudEngine/GoMud/modules/survival"
	_ "github.com/GoMudEngine/GoMud/modules/walking"
	_ "github.com/GoMudEngine/GoMud/modules/weather"
)

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func shippedWorld() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

func useDataDir(t *testing.T, dir string) {
	t.Helper()
	previous := configs.GetFilePathsConfig().DataFiles.String()
	set := func(value string) error {
		flat := map[string]any{}
		for k, v := range configs.Flatten(configs.GetOverrides()) {
			flat[k] = v
		}
		flat["FilePaths.DataFiles"] = value
		return configs.RestoreOverrides(flat)
	}
	require.NoError(t, set(dir))
	t.Cleanup(func() { require.NoError(t, set(previous)) })
}

// linkedWorld is a data directory made of a copy of the shipped world (so the
// test reads the real rooms, items and mobs) with its own users and
// plugin-data, so nothing a test saves reaches the repository.
func linkedWorld(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	shipped := shippedWorld()
	entries, err := os.ReadDir(shipped)
	require.NoError(t, err)
	for _, e := range entries {
		switch e.Name() {
		case "users", "plugin-data", "buffs":
			continue
		}
		if !e.IsDir() {
			data, err := os.ReadFile(filepath.Join(shipped, e.Name()))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, e.Name()), data, 0600))
			continue
		}
		require.NoError(t, os.CopyFS(filepath.Join(dir, e.Name()), os.DirFS(filepath.Join(shipped, e.Name()))), e.Name())
	}
	for _, d := range []string{"users", "plugin-data", "buffs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, d), 0755))
	}
	return dir
}

func freshEvents(t testing.TB) {
	t.Helper()
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
}

// trip is a loaded world with one admin standing in Dunmar's West Gate.
type trip struct {
	t        *testing.T
	dir      string
	user     *users.UserRecord
	messages []string
}

func newTrip(t *testing.T) *trip {
	t.Helper()
	dir := linkedWorld(t)
	useDataDir(t, dir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	skills.LoadDataFiles()
	spells.LoadSpellFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dir)
	buffs.RegisterFS(plugins.GetPluginRegistry())
	buffs.LoadDataFiles()

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
	})
	user := users.NewUserRecord(77, 1)
	user.Username = "robinsond76"
	user.Password = "$2a$test"
	user.Role = users.RoleAdmin
	user.Character.Name = "Robinson"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Level = 5
	user.Character.Gold = 1000
	user.Character.Validate()
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2001))
	// Arriving has its effects (a checkpoint at a church, say) before the
	// trip, as it does in play.
	events.ProcessEvents()
	t.Cleanup(func() {
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
	})

	tr := &trip{t: t, dir: dir, user: user}
	freshEvents(t)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == user.UserId {
			tr.messages = append(tr.messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return tr
}

func (tr *trip) run(command, rest string) string {
	tr.t.Helper()
	events.ProcessEvents()
	tr.messages = nil
	handled, err := usercommands.TryCommand(command, rest, tr.user.UserId, events.CmdSkipScripts)
	require.NoError(tr.t, err)
	require.True(tr.t, handled, command)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(tr.messages, "\n"), "")
}

// savedUserFile is the user file as the server would save it now.
func (tr *trip) savedUserFile() []byte {
	tr.t.Helper()
	require.NoError(tr.t, users.SaveUser(*tr.user))
	data, err := os.ReadFile(filepath.Join(tr.dir, "users", "77.yaml"))
	require.NoError(tr.t, err)
	return data
}

func (tr *trip) states() map[string][]byte {
	tr.t.Helper()
	states, err := userstate.CaptureAll(tr.user.UserId)
	require.NoError(tr.t, err)
	return states
}

// TestTripInAndBackLeavesTheCharacterAsItWas is the area's point: an admin
// with a class, a companion and a horse takes a trip, changes all of it (a
// class and level, gold and gear, new companions, a new horse), and comes
// back with the saved user file byte for byte as before and every module's
// state as before, in the room they left, with the shared clock untouched.
func TestTripInAndBackLeavesTheCharacterAsItWas(t *testing.T) {
	tr := newTrip(t)
	u := tr.user
	require.Contains(t, tr.run("archetype", "choose warrior confirm"), "Warrior")
	tr.run("company", "summon training dummy")
	require.Contains(t, tr.run("mount", "stable pack-horse"), "pack horse")
	tr.run("strategy", "")

	before := tr.savedUserFile()
	beforeStates := tr.states()
	beforeRoom := u.Character.RoomId
	require.Contains(t, beforeStates, "company")
	require.Contains(t, beforeStates, "mount")
	require.Contains(t, beforeStates, "archetype")

	out := tr.run("testarea", "")
	assert.Contains(t, out, "Test Area Hub", "the trip lands in the hub")
	assert.Equal(t, HubRoom, u.Character.RoomId)

	// Change everything that can be changed.
	assert.Contains(t, tr.run("testarea", "class paladin"), "Paladin")
	assert.Contains(t, tr.run("testarea", "level 18"), "level 18")
	assert.Equal(t, 18, u.Character.Level)
	tr.run("testarea", "gold 5000")
	assert.Contains(t, tr.run("testarea", "kit weapons"), "You take")
	assert.Contains(t, tr.run("testarea", "kit camp"), "You take")
	assert.Contains(t, tr.run("testarea", "companion add samurai 9"), "Recruited")
	assert.Contains(t, tr.run("testarea", "companion class #1 wizard"), "wizard")
	assert.Contains(t, tr.run("testarea", "companion level #1 14"), "level 14")
	tr.run("testarea", "stable")
	assert.Contains(t, tr.run("mount", "stable riding-horse"), "riding horse")
	assert.Contains(t, tr.run("testarea", "fight 3 2 58"), "foes")
	assert.Contains(t, tr.run("testarea", "clear"), "gone")
	assert.Contains(t, tr.run("testarea", "weather"), "weather")
	// A camp with a rest under way is left behind, timers and all.
	tr.run("testarea", "camp")
	assert.Contains(t, tr.run("camp", ""), "make camp")
	tr.run("camp", "fire")
	assert.Contains(t, tr.run("camp", "rest"), "settle in")
	assert.Contains(t, tr.states(), "camping", "the camp is part of the saved state while it stands")
	assert.NotEqual(t, before, tr.savedUserFile(), "the trip really changed the character")

	// Save and load while away: a restart does not lose the way back.
	plugins.Save()
	plugins.Load(tr.dir)
	_, still := registered.session(u.UserId)
	require.True(t, still, "the trip survives a restart")

	back := tr.run("testarea", "return")
	assert.Contains(t, back, "back as you were")
	assert.Equal(t, beforeRoom, u.Character.RoomId)
	assert.Equal(t, string(before), string(tr.savedUserFile()), "the saved character is byte-identical")
	afterStates := tr.states()
	assert.Equal(t, len(beforeStates), len(afterStates))
	for name, want := range beforeStates {
		assert.Equal(t, string(want), string(afterStates[name]), "module %s is as it was", name)
	}
	_, in := registered.session(u.UserId)
	assert.False(t, in, "the trip is over")
	assert.Contains(t, tr.run("testarea", "level 3"), "Start a trip first")

	// And the world is as it was: one companion standing beside the
	// leader, the one pack horse, the warrior class, the old level.
	members, ok := company.CompanyMembers(u.UserId)
	require.True(t, ok)
	require.Len(t, members, 1, "the recruit made in the area is gone")
	assert.Equal(t, 1, members[0].ID)
	assert.NotEqual(t, "wizard", members[0].Class)
	assert.Equal(t, 5, u.Character.Level)
	assert.Equal(t, 880, u.Character.Gold, "gold bought at the stable stays spent; area gold is gone")
	assert.Equal(t, "warrior", u.Character.ArchetypeID())
	instanceID, live := company.InstanceFor(u.UserId, 1)
	require.True(t, live, "the companion is standing again")
	assert.Equal(t, beforeRoom, mobs.GetInstance(instanceID).Character.RoomId)
}

// roundTrip is the saved user file and module states before a trip, to
// compare after it.
type roundTrip struct {
	room   int
	file   []byte
	states map[string][]byte
}

// withCompany gives the admin what a played character has: a class, a
// companion (which settles the company cargo, as a login does) and a horse.
func (tr *trip) withCompany() {
	tr.t.Helper()
	require.Contains(tr.t, tr.run("archetype", "choose warrior confirm"), "Warrior")
	tr.run("company", "summon training dummy")
	require.Contains(tr.t, tr.run("mount", "stable pack-horse"), "pack horse")
}

func (tr *trip) before() roundTrip {
	tr.t.Helper()
	return roundTrip{room: tr.user.Character.RoomId, file: tr.savedUserFile(), states: tr.states()}
}

func (tr *trip) assertAsBefore(b roundTrip) {
	tr.t.Helper()
	_, in := registered.session(tr.user.UserId)
	assert.False(tr.t, in, "the trip is over")
	assert.Equal(tr.t, b.room, tr.user.Character.RoomId)
	assert.Equal(tr.t, string(b.file), string(tr.savedUserFile()), "the saved character is byte-identical")
	after := tr.states()
	assert.Equal(tr.t, len(b.states), len(after))
	for name, want := range b.states {
		assert.Equal(tr.t, string(want), string(after[name]), "module %s is as it was", name)
	}
}

// TestCompanionGearDismissalAndCargoAreUndone: gear equipped on or removed
// from a companion in the area, and a companion dismissed there (which hands
// its gear to the leader), are all back as they were after the return, and
// no armory item survives anywhere.
func TestCompanionGearDismissalAndCargoAreUndone(t *testing.T) {
	tr := newTrip(t)
	u := tr.user
	tr.withCompany()
	// A fighter who can hold a weapon, recruited and armed before the trip.
	_, err := company.AdminRecruit(u.UserId, u.Character.RoomId, "warrior", 5)
	require.NoError(t, err)
	tr.run("look", "")
	kit, ok := kitItems("weapons")
	require.True(t, ok)
	require.GreaterOrEqual(t, len(kit), 2)
	own := items.New(kit[0])
	require.True(t, u.Character.StoreItem(own))
	require.NotContains(t, tr.run("company", "equip #2 "+own.ShorthandId()), "can't")
	b := tr.before()
	itemIDs := func() map[int]int {
		out := map[int]int{}
		for _, itm := range u.Character.Items {
			out[itm.ItemId]++
		}
		return out
	}
	inventoryBefore := itemIDs()

	tr.run("testarea", "")
	require.Contains(t, tr.run("testarea", "kit weapons"), "You take")
	var armory items.Item
	for _, itm := range u.Character.Items {
		if itm.ItemId == kit[1] {
			armory = itm
		}
	}
	require.NotZero(t, armory.ItemId)
	require.NotContains(t, tr.run("company", "equip #2 "+armory.ShorthandId()), "can't", "an armory weapon displaces the companion's own")
	require.NotContains(t, tr.run("company", "remove #2 weapon"), "can't")
	require.NotContains(t, tr.run("company", "equip #2 "+armory.ShorthandId()), "can't")
	require.Contains(t, tr.run("company", "dismiss #2"), "dismissed")
	inArea, _ := company.CompanyMembers(u.UserId)
	require.Len(t, inArea, 1, "the dismissal happened")
	assert.Positive(t, itemIDs()[kit[1]], "the dismissed companion's armory weapon came back to the leader")
	tr.run("testarea", "return")
	tr.assertAsBefore(b)

	assert.Equal(t, inventoryBefore, itemIDs(), "the leader carries exactly what they did; no armory item came back")
	members, ok := company.CompanyMembers(u.UserId)
	require.True(t, ok)
	require.Len(t, members, 2, "the dismissed companion is back")
}

// TestLeavingTheAreaEndsTheTrip: an admin teleport (or any move but the
// return) out of the area ends the trip at once, so the test character never
// stands in the world.
func TestLeavingTheAreaEndsTheTrip(t *testing.T) {
	tr := newTrip(t)
	u := tr.user
	tr.withCompany()
	b := tr.before()
	tr.run("testarea", "")
	tr.run("testarea", "gold 99999")
	require.Contains(t, tr.run("testarea", "kit weapons"), "You take")

	out := tr.run("teleport", "2002")
	assert.Contains(t, out, "trip is over")
	b.room = u.Character.RoomId
	assert.NotEqual(t, 2002, b.room, "the return puts them back where the trip began")
	assert.Equal(t, 2001, u.Character.RoomId)
	tr.assertAsBefore(b)
}

// TestDeathInTheAreaWakesInTheHub: a death on a trip wakes the admin in the
// hub, not at a church in the world, and the level it cost comes back on the
// return.
func TestDeathInTheAreaWakesInTheHub(t *testing.T) {
	tr := newTrip(t)
	u := tr.user
	tr.withCompany()
	b := tr.before()
	tr.run("testarea", "yard")
	require.Equal(t, 90002, u.Character.RoomId)

	provider, ok := deathdomain.Active()
	require.True(t, ok)
	u.Character.Health = -10
	tr.messages = nil
	provider.Respawn(u.UserId, true)
	events.ProcessEvents()
	assert.Equal(t, HubRoom, u.Character.RoomId, "the dead wake in the hub")
	assert.Equal(t, 4, u.Character.Level, "the death cost a level")
	_, in := registered.session(u.UserId)
	require.True(t, in, "the trip goes on")
	assert.NotContains(t, strings.Join(tr.messages, "\n"), "altar")

	tr.run("testarea", "return")
	assert.Equal(t, 5, u.Character.Level)
	tr.assertAsBefore(b)
}

// 38e review: the area recruits a hound and a stone golem, keeps each
// species (a creature's body is its template), refuses to turn the admin
// into one, hands out stone mortar, and the return undoes the recruits.
func TestCreaturesInTheArea(t *testing.T) {
	tr := newTrip(t)
	tr.withCompany()
	b := tr.before()
	tr.run("testarea", "")

	assert.Contains(t, tr.run("testarea", "companion add hound 12"), "Recruited")
	assert.Contains(t, tr.run("testarea", "companion add stone-golem"), "Recruited")
	assert.Contains(t, tr.run("testarea", "companion class brindle warrior"), "can't trade places")
	assert.Contains(t, tr.run("testarea", "companion class dummy hound"), "can't trade places")
	assert.Contains(t, tr.run("testarea", "companion level cairn 20"), "level 20")
	assert.Contains(t, tr.run("testarea", "class hound"), "creature species")
	ids, ok := kitItems("supplies")
	require.True(t, ok)
	assert.Contains(t, ids, creatures.RepairItemID, "stone mortar is in the supplies kit")

	tr.run("testarea", "return")
	tr.assertAsBefore(b)
}
