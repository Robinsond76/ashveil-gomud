package mount

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33g: personal load is worn equipment; shared cargo, packs, and
// horses affect company capacity without changing agility burden.
// Driven through plugins.Load and the real commands, as the 32f test is.
func TestMountsAndCargoNeverLightenBurden(t *testing.T) {
	dataDir := t.TempDir()
	useLogisticsDataDir(t, dataDir)
	writeLogisticsWorld(t, dataDir)
	races.LoadDataFiles()
	items.LoadDataFiles()
	buffs.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	mount.SetProvider(registered)
	t.Cleanup(func() { mount.SetProvider(registered) })

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(33, 1)
	user.Username = "brannoc"
	user.Password = "$2a$test"
	user.Character.Name = "Brannoc"
	user.Character.RaceId = 1
	user.Character.Level = 5
	user.Character.Gold = 1000
	user.Character.Validate()
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2001))
	t.Cleanup(func() {
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
	})
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == user.UserId {
			messages = append(messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	run := func(command, rest string) string {
		t.Helper()
		events.ProcessEvents()
		messages = nil
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return logisticsTags.ReplaceAllString(strings.Join(messages, "\n"), "")
	}
	companyCapacity := func() int {
		t.Helper()
		load, ok := encumbrance.CurrentLoad(user.UserId)
		require.True(t, ok)
		return load.CapacityGrams
	}
	c := user.Character
	run("cargo", "")
	armor := items.ItemSpec{ItemId: 989751, Name: "burden armor", Type: items.Body, Weight: 10500}
	items.SetTestItemSpec(&armor)
	t.Cleanup(func() { items.RemoveTestItemSpec(armor.ItemId) })
	c.Equipment.Body = items.New(armor.ItemId)

	// Worn armor burdens the leader; saddle and water remain cargo.
	c.StoreItem(items.New(34))
	c.StoreItem(items.New(30015))
	require.Equal(t, 10500, c.PersonalGrams())
	agility, burden := c.AgilityCapacityGrams(), c.Burden()
	require.Greater(t, burden, 0.0, "10.5 kg burdens Brannoc (capacity %d g)", agility)
	require.Less(t, burden, 1.0, "but not fully (capacity %d g)", agility)

	// A satchel adds company capacity and cargo weight, with no personal burden.
	before := companyCapacity()
	c.StoreItem(items.New(31))
	assert.Equal(t, before+5000, companyCapacity())
	assert.Equal(t, agility, c.AgilityCapacityGrams(), "a pack is company cargo room")
	assert.Equal(t, burden, c.Burden(), "the shared satchel does not burden the leader")

	// A horse: much more company capacity, the same burden.
	before = companyCapacity()
	assert.Contains(t, run("mount", "stable pack-horse"), "pack horse")
	assert.Greater(t, companyCapacity(), before)
	assert.Equal(t, agility, c.AgilityCapacityGrams())
	assert.Equal(t, burden, c.Burden(), "a horse never lightens the rider")

	// Saddling moves 9 kg from cargo to the herd, preserving worn burden.
	run("mount", "saddle pack pack saddle")
	assert.Equal(t, 10500, c.PersonalGrams())
	assert.Equal(t, burden, c.Burden())

	// The old stow command is guidance; shared cargo never affects worn burden.
	grams := c.PersonalGrams()
	assert.Contains(t, run("cargo", "put waterskin"), "already belong")
	assert.Equal(t, grams, c.PersonalGrams())
	assert.NotEmpty(t, encumbrance.CargoContents(user.UserId))
	assert.Equal(t, agility, c.AgilityCapacityGrams(), "cargo adds no agility capacity")
	assert.Equal(t, characters.BurdenFor(c.PersonalGrams(), agility), c.Burden())
}
