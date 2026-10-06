package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func floorDropWorld(t *testing.T) {
	t.Helper()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Death.CorpsesEnabled = true
	gameplay.Death.CorpseItems = false
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

func reviewCorpse(name string, claim int, held []items.Item, worn *items.Item) rooms.Corpse {
	c := rooms.Corpse{ClaimUserId: claim, MobId: 94912, Character: *characters.New(), Items: held}
	c.Character.Name = name
	if worn != nil {
		c.Character.Equipment.Weapon = *worn
	}
	return c
}

// 33d review: 33d dropped get's CorpseItems gate so a claimant could loot,
// opening every corpse in floor-drop worlds: anyone took the gear a body
// kept (failed drop rolls, perma-gear).
func TestFloorDropWorldCorpseKeepsItsWornGear(t *testing.T) {
	floorDropWorld(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 94911, Name: "guard sword", Type: items.Weapon, Weight: 1})
	t.Cleanup(func() { items.RemoveTestItemSpec(94911) })
	sword := items.New(94911)
	for _, command := range []string{"all corpse", "sword corpse", "sword from guard corpse"} {
		t.Run(command, func(t *testing.T) {
			room := rooms.NewEmptyRoom()
			stranger := users.NewUserRecord(94901, 0)
			stranger.Character.RoomId = room.RoomId
			room.AddCorpse(reviewCorpse("guard", 0, nil, &sword))
			_, err := Get(command, stranger, room, 0)
			require.NoError(t, err)
			assert.Empty(t, stranger.Character.Items)
			assert.Len(t, room.Corpses[0].Character.GetAllWornItems(), 1)
		})
	}
	t.Run("claimant takes held loot only", func(t *testing.T) {
		room := rooms.NewEmptyRoom()
		owner := users.NewUserRecord(94902, 0)
		owner.Character.RoomId = room.RoomId
		items.SetTestItemSpec(&items.ItemSpec{ItemId: 94913, Name: "coin purse", Weight: 1})
		t.Cleanup(func() { items.RemoveTestItemSpec(94913) })
		room.AddCorpse(reviewCorpse("guard", owner.UserId, []items.Item{items.New(94913)}, &sword))
		_, err := Get("all corpse", owner, room, 0)
		require.NoError(t, err)
		require.Len(t, owner.Character.Items, 1)
		assert.Equal(t, 94913, owner.Character.Items[0].ItemId)
		assert.Len(t, room.Corpses[0].Character.GetAllWornItems(), 1, "worn gear stays on the body")
		_, err = Get("sword corpse", owner, room, 0)
		require.NoError(t, err)
		assert.Len(t, owner.Character.Items, 1)
	})
}

// 33d review: several same-named corpses resolved to the first, so a
// claimant could never reach a second claimed bandit corpse.
func TestClaimantReachesEachSameNamedClaimedCorpse(t *testing.T) {
	floorDropWorld(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 94914, Name: "bandit token", Weight: 1})
	t.Cleanup(func() { items.RemoveTestItemSpec(94914) })
	room := rooms.NewEmptyRoom()
	owner := users.NewUserRecord(94903, 0)
	owner.Character.RoomId = room.RoomId
	room.AddCorpse(reviewCorpse("bandit", 0, nil, nil))                                // the claimant's own earlier solo kill
	room.AddCorpse(reviewCorpse("bandit", 94999, []items.Item{items.New(94914)}, nil)) // an ally's claim
	room.AddCorpse(reviewCorpse("bandit", owner.UserId, []items.Item{items.New(94914)}, nil))
	room.AddCorpse(reviewCorpse("bandit", owner.UserId, []items.Item{items.New(94914)}, nil))
	for i := 0; i < 3; i++ {
		_, err := Get("all corpse", owner, room, 0)
		require.NoError(t, err)
	}
	assert.Len(t, owner.Character.Items, 2, "both of the claimant's corpses, never the ally's")
	assert.Len(t, room.Corpses[1].Items, 1)
}

// 33d review: look named the claimant by raw player id, or not at all.
func TestLookNamesTheLootClaimant(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	floorDropWorld(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 94915, Name: "bandit token", Weight: 1})
	t.Cleanup(func() { items.RemoveTestItemSpec(94915) })
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	room := rooms.NewEmptyRoom()
	room.Tags = []string{rooms.TagLit} // the light must not depend on the shared clock
	claimant := users.NewUserRecord(94904, 0)
	claimant.Character.Name = "Borin"
	claimant.Character.RoomId = room.RoomId
	users.SetTestUser(claimant)
	viewer := users.NewUserRecord(94905, 0)
	viewer.Character.Name = "Aria"
	viewer.Character.RoomId = room.RoomId
	users.SetTestUser(viewer)
	room.AddCorpse(reviewCorpse("bandit", claimant.UserId, []items.Item{items.New(94915)}, nil))
	var text []string
	freshEvents(t)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == viewer.UserId {
			text = append(text, m.Text)
		}
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	_, err := Look("bandit corpse", viewer, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	out := strings.Join(text, "\n")
	assert.Contains(t, out, "claimed by")
	assert.Contains(t, out, "Borin")
	assert.NotContains(t, out, "#94904")
}

// 33d review (independent): an unclaimed corpse matched get's last word and
// returned early, so "get leather cap" beside a "captain of the guard" corpse
// never searched the floor.
func TestUnclaimedCorpseNameNeverShadowsAFloorItem(t *testing.T) {
	floorDropWorld(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 94916, Name: "leather cap", Type: items.Head, Weight: 1})
	t.Cleanup(func() { items.RemoveTestItemSpec(94916) })
	room := rooms.NewEmptyRoom()
	user := users.NewUserRecord(94906, 0)
	user.Character.RoomId = room.RoomId
	room.AddCorpse(reviewCorpse("captain of the guard", 0, nil, nil))
	room.AddItem(items.New(94916), false)
	_, err := Get("leather cap", user, room, 0)
	require.NoError(t, err)
	require.Len(t, user.Character.Items, 1)
	assert.Empty(t, room.Items)
}

// 33d review (independent): look reached only the first same-named corpse.
func TestLookPrefersTheViewersOwnClaim(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	floorDropWorld(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 94917, Name: "bandit token", Weight: 1})
	t.Cleanup(func() { items.RemoveTestItemSpec(94917) })
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	room := rooms.NewEmptyRoom()
	room.Tags = []string{rooms.TagLit}
	ally := users.NewUserRecord(94907, 0)
	ally.Character.Name = "Aria"
	users.SetTestUser(ally)
	viewer := users.NewUserRecord(94908, 0)
	viewer.Character.Name = "Bob"
	viewer.Character.RoomId = room.RoomId
	users.SetTestUser(viewer)
	room.AddCorpse(reviewCorpse("bandit", ally.UserId, []items.Item{items.New(94917)}, nil))
	room.AddCorpse(reviewCorpse("bandit", viewer.UserId, []items.Item{items.New(94917)}, nil))
	var text []string
	freshEvents(t)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == viewer.UserId {
			text = append(text, m.Text)
		}
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	_, err := Look("bandit corpse", viewer, room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(text, "\n"), "claimed by you")
}
