package light

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 900 rounds a game day: a game hour is 37 rounds.
const testDay = 900

func TestBurnCountsWholeGameHoursFromWhenLit(t *testing.T) {
	h, next := burn(0, 500, testDay)
	assert.Equal(t, 0, h, "a lantern with no start burns nothing yet")
	assert.Equal(t, uint64(500), next, "and starts counting now")

	h, next = burn(100, 136, testDay)
	assert.Equal(t, 0, h)
	assert.Equal(t, uint64(100), next, "less than an hour keeps its start")

	h, next = burn(100, 100+37*3+5, testDay)
	assert.Equal(t, 3, h, "three whole hours")
	assert.Equal(t, uint64(100+37*3), next, "the part-hour carries over")

	h, next = burn(900, 100, testDay)
	assert.Equal(t, 0, h, "a start after now (a reset counter) burns nothing")
	assert.Equal(t, uint64(100), next)
}

// fakeHolder is a character's lantern and buffs, without a game.
type fakeHolder struct {
	held   items.Item
	packed []items.Item
	buff   map[int]bool
}

func (f *fakeHolder) offhand() items.Item     { return f.held }
func (f *fakeHolder) setOffhand(i items.Item) { f.held = i }
func (f *fakeHolder) hasBuff(id int) bool     { return f.buff[id] }
func (f *fakeHolder) addBuff(id, _ int)       { f.buff[id] = true }
func (f *fakeHolder) removeBuff(id int)       { delete(f.buff, id) }
func (f *fakeHolder) dousePacked() {
	for i := range f.packed {
		f.packed[i].Lit = false
	}
}

func TestALitLanternBurnsOilAndGivesLight(t *testing.T) {
	f := &fakeHolder{held: items.Item{ItemId: lanternItemID, Uses: 3, Lit: true, LastUsedRound: 1000}, buff: map[int]bool{}}
	assert.Empty(t, tickLantern(f, 1000+37, testDay))
	assert.Equal(t, 2, f.held.Uses, "one game hour burns one hour of oil")
	assert.True(t, f.buff[lanternBuffID], "a lit lantern gives light")

	msg := tickLantern(f, 1000+37*3, testDay)
	assert.Contains(t, msg, "gutters and goes out")
	assert.Equal(t, emptyLantern, f.held.Uses, "empty is -1, never 0 (items.Validate would refill a 0)")
	assert.False(t, f.held.Lit)
	assert.False(t, f.buff[lanternBuffID], "out of oil, it gives no light")
}

func TestADarkOrStowedLanternBurnsNothingAndGivesNoLight(t *testing.T) {
	f := &fakeHolder{held: items.Item{ItemId: lanternItemID, Uses: 5}, buff: map[int]bool{lanternBuffID: true}}
	tickLantern(f, 99999, testDay)
	assert.Equal(t, 5, f.held.Uses, "a dark lantern burns nothing")
	assert.False(t, f.buff[lanternBuffID], "and its light is gone")

	packed := &fakeHolder{packed: []items.Item{{ItemId: lanternItemID, Uses: 5, Lit: true}}, buff: map[int]bool{}}
	tickLantern(packed, 99999, testDay)
	assert.False(t, packed.packed[0].Lit, "a lit lantern stowed in the pack goes out")
	assert.False(t, packed.buff[lanternBuffID])
}

// world sets up the item and buff specs and a player in a room, and
// captures what they are told.
func world(t *testing.T) (*users.UserRecord, *rooms.Room, *[]string) {
	t.Helper()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: lanternItemID, Name: "lantern", NameSimple: "lantern", Type: items.Offhand, Uses: 24})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: torchItemID, Name: "torch", NameSimple: "torch", Type: items.Object})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: lampOilItemID, Name: "flask of lamp oil", NameSimple: "lamp oil", Type: items.Object})
	t.Cleanup(func() {
		items.RemoveTestItemSpec(lanternItemID)
		items.RemoveTestItemSpec(torchItemID)
		items.RemoveTestItemSpec(lampOilItemID)
	})
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: lanternBuffID, Name: "Lantern light", TriggerRate: "1 round", TriggerCount: 1, Flags: []string{rooms.FlagLightSource}})
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: torchBuffID, Name: "Torchlight", TriggerRate: "1 real minute", TriggerCount: 15, Flags: []string{rooms.FlagLightSource}})

	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(7, 1)
	user.Character = characters.New()
	user.Character.Name = "Wren"
	users.SetTestUser(user)
	room := &rooms.Room{RoomId: 91050, Zone: "Deep"}
	room.SetTestOccupants([]int{7}, nil)

	told := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == 7 {
			told = append(told, m.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return user, room, &told
}

func said(told *[]string) string {
	events.ProcessEvents()
	out := strings.Join(*told, "")
	*told = (*told)[:0]
	return out
}

// The real commands: light lantern, douse lantern, fill lantern and light
// torch, through the module's handlers and the core fill command.
func TestLanternCommands(t *testing.T) {
	user, room, told := world(t)
	m := &LightModule{}
	c := user.Character

	_, _ = m.userCommand("lantern", user, room, 0)
	assert.Contains(t, said(told), "You have no lantern.")

	lantern := items.New(lanternItemID)
	require.Equal(t, 24, lantern.Uses, "a new lantern comes full")
	c.Items = append(c.Items, lantern)
	_, _ = m.userCommand("lantern", user, room, 0)
	assert.Contains(t, said(told), "Hold your lantern first")

	c.Items = nil
	c.Equipment.Offhand = lantern
	_, _ = m.userCommand("lantern", user, room, 0)
	assert.Contains(t, said(told), "You light your")
	assert.True(t, c.Equipment.Offhand.Lit)
	assert.Equal(t, util.GetRoundCount(), c.Equipment.Offhand.LastUsedRound, "the burn counts from when it was lit")
	assert.True(t, c.HasBuffFlag(rooms.FlagLightSource), "a lit lantern is a light source")

	_, _ = m.userCommand("", user, room, 0)
	assert.Contains(t, said(told), "Your lantern is lit, with oil for about 24 hours.")

	_, _ = m.douse("lantern", user, room, 0)
	assert.Contains(t, said(told), "You douse your")
	assert.False(t, c.Equipment.Offhand.Lit)
	assert.False(t, liveBuff(c.GetBuffs(lanternBuffID)), "a doused lantern gives no light")

	// Relit at once, before the old buff is pruned, it shines again.
	_, _ = m.userCommand("lantern", user, room, 0)
	said(told)
	assert.True(t, liveBuff(c.GetBuffs(lanternBuffID)))
	tickLantern(userHolder{user}, util.GetRoundCount(), testDay)
	assert.True(t, liveBuff(c.GetBuffs(lanternBuffID)), "the round tick keeps a relit lantern's light")
	_, _ = m.douse("lantern", user, room, 0)
	said(told)

	_, _ = m.douse("torch", user, room, 0)
	assert.Contains(t, said(told), "can't be put out")

	// An empty lantern won't light; lamp oil through the core fill command
	// fills it.
	l := c.Equipment.Offhand
	l.Uses = emptyLantern
	c.Equipment.Offhand = l
	_, _ = m.userCommand("lantern", user, room, 0)
	assert.Contains(t, said(told), "out of oil")
	usercommands.RegisterFillHandler(fillLantern)
	handled, err := usercommands.Fill("lantern", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, said(told), "You have no lamp oil.")
	c.Items = append(c.Items, items.New(lampOilItemID))
	_, _ = usercommands.Fill("lantern", user, room, 0)
	assert.Contains(t, said(told), "You fill your")
	assert.Equal(t, 24, c.Equipment.Offhand.Uses)
	_, found := findItem(c.Items, lampOilItemID)
	assert.False(t, found, "the flask is used up")
	_, _ = usercommands.Fill("lantern", user, room, 0)
	assert.Contains(t, said(told), "already full")
}

func TestATorchIsUsedUpAndBurns(t *testing.T) {
	user, room, told := world(t)
	m := &LightModule{}
	c := user.Character

	_, _ = m.userCommand("torch", user, room, 0)
	assert.Contains(t, said(told), "You have no torch.")

	c.Items = append(c.Items, items.New(torchItemID))
	_, _ = m.userCommand("torch", user, room, 0)
	assert.Contains(t, said(told), "You light a")
	_, found := findItem(c.Items, torchItemID)
	assert.False(t, found, "lighting a torch uses it up")
	assert.True(t, liveBuff(c.GetBuffs(torchBuffID)))
	assert.True(t, c.HasBuffFlag(rooms.FlagLightSource))

	_, _ = m.userCommand("", user, room, 0)
	assert.Contains(t, said(told), "You carry a burning torch.")
}

// A save from before light gear: its lantern lit whenever worn, through the
// permanent Illumination buff. Loading it (Validate(true), as users.LoadUser
// and copyover do) drops that buff, since the lantern no longer grants it,
// and the lantern comes back full of oil and dark.
func TestAnOldSavesAlwaysOnLanternLightIsDropped(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: lanternItemID, Name: "lantern", NameSimple: "lantern", Type: items.Offhand, Uses: 24})
	t.Cleanup(func() { items.RemoveTestItemSpec(lanternItemID) })
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1, Name: "Illumination", TriggerRate: "5 real minutes", TriggerCount: 1, Flags: []string{rooms.FlagLightSource}})

	c := characters.New()
	c.Equipment.Offhand = items.Item{ItemId: lanternItemID}
	require.NoError(t, c.AddBuff(1, true))
	require.True(t, liveBuff(c.GetBuffs(1)))

	require.NoError(t, c.Validate(true))
	assert.False(t, liveBuff(c.GetBuffs(1)), "the old permanent lantern light is gone")
	assert.Equal(t, 24, c.Equipment.Offhand.Uses, "and it comes full")
	assert.False(t, c.Equipment.Offhand.Lit)
}
