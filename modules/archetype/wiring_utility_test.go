package archetype

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Wiring (17b): the real picklock and go user commands, the real walking
// step seam, the real archetypes seam, and the real cast command path.

// wired registers a fixed-roll module as the archetypes provider and as a
// walking step listener, the way init() does.
func wired(t *testing.T, roll int) *ArchetypeModule {
	t.Helper()
	m, _, _ := utilModule(t, roll)
	m.now = util.GetRoundCount
	m.cast = nativeCast
	archetypes.SetProvider(m)
	remove := walking.AddStepListener(m.onStep)
	t.Cleanup(func() {
		archetypes.SetProvider(nil)
		remove()
	})
	return m
}

func lockpickUser(t *testing.T, m *ArchetypeModule, id, roomID int, archetype string) *users.UserRecord {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 8, Name: "lockpicks", NameSimple: "lockpicks", Type: items.Lockpicks})
	u := trainee(t, id, roomID)
	u.Character.StoreItem(items.New(8))
	if archetype != "" {
		m.choose(u, archetype, true)
	}
	return u
}

// wrongPin is the first pin direction that does NOT match the lock.
func wrongPin(lockID string, difficulty int) string {
	seq := util.GetLockSequence(lockID, difficulty, string(configs.GetServerConfig().Seed))
	if seq[0] == 'U' {
		return "DOWN"
	}
	return "UP"
}

// pickOnce starts (or continues) picking and answers one wrong pin.
func pickOnce(t *testing.T, u *users.UserRecord, room *rooms.Room, target, lockID string, difficulty int) {
	t.Helper()
	_, err := usercommands.Picklock(target, u, room, 0)
	require.NoError(t, err)
	p := u.GetPrompt()
	require.NotNil(t, p)
	require.NotEmpty(t, p.Questions)
	p.Questions[0].Answer(wrongPin(lockID, difficulty))
	_, err = usercommands.Picklock(target, u, room, 0)
	require.NoError(t, err)
}

func trapBuffs(list []events.Buff) int {
	n := 0
	for _, b := range list {
		if b.BuffId == 13 {
			n++
		}
	}
	return n
}

func TestWiringPicklockSpringsArmedTrapButNotDisarmedOne(t *testing.T) {
	m := wired(t, 100)
	room := trapRoom(t, 97200)
	u := lockpickUser(t, m, 90, 97200, "rogue")
	u.Character.StoreItem(items.New(8))

	sprung := captureBuffs(t, func() { pickOnce(t, u, room, "chest", "97200-chest", 5) })
	assert.Equal(t, 1, trapBuffs(sprung), "a broken pick springs an armed trap")

	u.ClearPrompt()
	assert.Contains(t, m.disarm(u, room, "chest"), "disarm the trap")

	u.Character.StoreItem(items.New(8))
	safe := captureBuffs(t, func() { pickOnce(t, u, room, "chest", "97200-chest", 5) })
	assert.Zero(t, trapBuffs(safe), "a disarmed trap stays quiet")
}

func TestWiringPicklockWarnsBeforeFirstPin(t *testing.T) {
	m := wired(t, 100)
	room := trapRoom(t, 97210)
	u := lockpickUser(t, m, 91, 97210, "rogue")

	text := captureText(t, func() {
		_, err := usercommands.Picklock("chest", u, room, 0)
		require.NoError(t, err)
	})
	assert.Contains(t, text, "Your instincts prickle")

	u.ClearPrompt()
	text = captureText(t, func() {
		_, err := usercommands.Picklock("chest", u, room, 0)
		require.NoError(t, err)
	})
	assert.NotContains(t, text, "instincts", "one free sense per lock")
}

func TestWiringPicklockNoWarningWithAutoskillOffOrNoRogue(t *testing.T) {
	m := wired(t, 100)
	room := trapRoom(t, 97220)
	off := lockpickUser(t, m, 92, 97220, "rogue")
	m.setAutoskill(92, utilityTraps, false)
	warrior := lockpickUser(t, m, 93, 97220, "warrior")
	for _, u := range []*users.UserRecord{off, warrior} {
		text := captureText(t, func() {
			_, err := usercommands.Picklock("chest", u, room, 0)
			require.NoError(t, err)
		})
		assert.NotContains(t, text, "instincts")
	}
}

// walkRooms links an entry room (id) north to a destination (id+1).
func walkRooms(t *testing.T, id int, destBiome string, dest func(r *rooms.Room)) (*rooms.Room, *rooms.Room) {
	t.Helper()
	from := &rooms.Room{RoomId: id, Zone: "ArchetypeWalk", Title: "Hall", Biome: "testhall",
		Exits: map[string]exit.RoomExit{"north": {RoomId: id + 1}}}
	to := &rooms.Room{RoomId: id + 1, Zone: "ArchetypeWalk", Title: "Beyond", Biome: destBiome,
		Exits: map[string]exit.RoomExit{"south": {RoomId: id}}}
	if dest != nil {
		dest(to)
	}
	rooms.SetTestRoom(from)
	rooms.SetTestRoom(to)
	t.Cleanup(func() {
		rooms.RemoveTestRoom(id)
		rooms.RemoveTestRoom(id + 1)
	})
	return from, to
}

func walker(t *testing.T, id, roomID int) *users.UserRecord {
	t.Helper()
	u := trainee(t, id, roomID)
	u.Character.ActionPoints = 100
	if r := rooms.LoadRoom(roomID); r != nil {
		r.AddPlayer(id)
	}
	return u
}

func testBiomes(t *testing.T) {
	t.Helper()
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "testhall", Name: "Hall", LitArea: true, Indoor: true})
	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "testcave", Name: "Cave", DarkArea: true})
	t.Cleanup(func() {
		rooms.RemoveTestBiome("testhall")
		rooms.RemoveTestBiome("testcave")
	})
}

func trappedChest(r *rooms.Room) {
	r.Containers = map[string]rooms.Container{"chest": {Lock: gamelock.Lock{Difficulty: 5, TrapBuffIds: []int{13}}}}
}

func TestWiringGoAutoSensesTrapOnEntry(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from, _ := walkRooms(t, 97300, "testhall", trappedChest)
	u := walker(t, 94, 97300)
	m.choose(u, "rogue", true)

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Equal(t, 97301, u.Character.RoomId)
	assert.Contains(t, text, "Your instincts prickle: there is a trap on")
}

func TestWiringAutoSenseSilentForTeleportOrAutoskillOff(t *testing.T) {
	testBiomes(t)
	m := wired(t, 100)
	from, _ := walkRooms(t, 97310, "testhall", trappedChest)
	u := walker(t, 95, 97310)
	m.choose(u, "rogue", true)

	text := captureText(t, func() { require.NoError(t, rooms.MoveToRoom(u.UserId, 97311)) })
	assert.NotContains(t, text, "instincts", "only walking steps trigger auto-sense")

	require.NoError(t, rooms.MoveToRoom(u.UserId, 97310))
	m.setAutoskill(95, utilityTraps, false)
	text = captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Equal(t, 97311, u.Character.RoomId)
	assert.NotContains(t, text, "instincts")
}

func TestWiringGoIntoDarknessAutoCastsForPlayerWizard(t *testing.T) {
	testBiomes(t)
	m := wired(t, 50)
	from, _ := walkRooms(t, 97400, "testcave", nil)
	u := walker(t, 96, 97400)
	m.choose(u, "wizard", true)
	u.Character.Mana = 50
	cost := 10 // floatinglight
	require.True(t, u.Character.HasSpell("floatinglight"))

	text := captureText(t, func() {
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
	})
	assert.Equal(t, 97401, u.Character.RoomId)
	assert.Contains(t, text, "conjuring a floating light")
	require.NotNil(t, u.Character.Aggro, "the real cast path started the spell")
	assert.Equal(t, "floatinglight", u.Character.Aggro.SpellInfo.SpellId)
	assert.Equal(t, 50-cost, u.Character.Mana, "mana spent by the cast command")
}

func TestWiringGoIntoDarknessCompanionWizardLights(t *testing.T) {
	testBiomes(t)
	m := wired(t, 50)
	from, _ := walkRooms(t, 97410, "testcave", nil)
	u := walker(t, 97, 97410)
	m.choose(u, "warrior", true)
	bran := withCompanion(t, 97, 97901, 97410, 1, "wizard") // still in the origin room as the leader steps
	bran.Character.Mana = 30

	var text string
	buffsSeen := captureBuffs(t, func() {
		text = captureText(t, func() {
			_, err := usercommands.Go("north", u, from, 0)
			require.NoError(t, err)
		})
	})
	assert.Contains(t, text, "Bran conjures a floating light")
	assert.Equal(t, 20, bran.Character.Mana)
	found := false
	for _, b := range buffsSeen {
		if b.MobInstanceId == 97901 && b.BuffId == 1000 {
			found = true
		}
	}
	assert.True(t, found, "the companion gets the party light buff")
	assert.Nil(t, u.Character.Aggro, "the leader keeps walking")
}

func TestWiringAutoLightSkips(t *testing.T) {
	testBiomes(t)

	t.Run("lit room", func(t *testing.T) {
		m := wired(t, 50)
		from, _ := walkRooms(t, 97420, "testhall", nil)
		u := walker(t, 98, 97420)
		m.choose(u, "wizard", true)
		u.Character.Mana = 50
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Nil(t, u.Character.Aggro)
		assert.Equal(t, 50, u.Character.Mana)
	})

	t.Run("autoskill off", func(t *testing.T) {
		m := wired(t, 50)
		from, _ := walkRooms(t, 97430, "testcave", nil)
		u := walker(t, 99, 97430)
		m.choose(u, "wizard", true)
		m.setAutoskill(99, utilityLight, false)
		u.Character.Mana = 50
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Nil(t, u.Character.Aggro)
	})

	t.Run("not enough mana", func(t *testing.T) {
		m := wired(t, 50)
		from, _ := walkRooms(t, 97440, "testcave", nil)
		u := walker(t, 100, 97440)
		m.choose(u, "wizard", true)
		u.Character.Mana = 3
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Nil(t, u.Character.Aggro)
		assert.Equal(t, 3, u.Character.Mana)
	})

	t.Run("no wizard", func(t *testing.T) {
		m := wired(t, 50)
		from, _ := walkRooms(t, 97450, "testcave", nil)
		u := walker(t, 101, 97450)
		m.choose(u, "rogue", true)
		u.Character.Mana = 50
		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Nil(t, u.Character.Aggro)
	})

	t.Run("cooldown", func(t *testing.T) {
		m := wired(t, 50)
		from, _ := walkRooms(t, 97460, "testcave", nil)
		u := walker(t, 102, 97460)
		m.choose(u, "warrior", true)
		bran := withCompanion(t, 102, 97902, 97460, 1, "wizard")
		bran.Character.Mana = 50

		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Equal(t, 40, bran.Character.Mana, "first dark step lights")

		// Back and forth again within the cooldown: no second light.
		require.NoError(t, rooms.MoveToRoom(u.UserId, 97460))
		bran.Character.RoomId = 97460
		_, err = usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Equal(t, 40, bran.Character.Mana, "cooldown")
	})

	t.Run("companion already carries a party light", func(t *testing.T) {
		buffs.SetTestFlag(rooms.FlagPartyLight)
		buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1000, Name: "Floating Light", TriggerRate: "10 rounds", TriggerCount: 1, Flags: []string{rooms.FlagPartyLight}})
		m := wired(t, 50)
		from, _ := walkRooms(t, 97470, "testcave", nil)
		u := walker(t, 103, 97470)
		m.choose(u, "wizard", true)
		u.Character.Mana = 50
		bran := withCompanion(t, 103, 97903, 97470, 1, "wizard")
		require.NoError(t, bran.Character.AddBuff(1000, false))
		require.True(t, bran.Character.HasBuffFlag(rooms.FlagPartyLight))
		bran.Character.Mana = 50 // after AddBuff, whose Validate caps mana

		_, err := usercommands.Go("north", u, from, 0)
		require.NoError(t, err)
		assert.Nil(t, u.Character.Aggro, "no cast: a following companion's light is already up")
		assert.Equal(t, 50, bran.Character.Mana, "and the companion spends nothing")
	})
}

// TestShippedTrappedTollBox loads the real Dunmar West Gate room file and
// checks its toll box is a locked, trapped proving lock.
func TestShippedTrappedTollBox(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default", "rooms", "dunmar", "2001.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var room rooms.Room
	require.NoError(t, yaml.Unmarshal(data, &room))
	box, ok := room.Containers["toll box"]
	require.True(t, ok)
	assert.True(t, box.Lock.IsLocked())
	assert.Equal(t, []int{13}, box.Lock.TrapBuffIds)
	locks := trappedLocks(&room)
	require.Len(t, locks, 1)
	assert.Equal(t, "2001-toll box", locks[0].ID)
}
