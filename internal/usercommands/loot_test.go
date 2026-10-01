package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookReportsPublicAndExpiredSpoils(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	floorDropWorld(t)
	start := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(start) })
	u := users.NewUserRecord(7, 1)
	room := rooms.NewEmptyRoom()
	room.Tags = []string{rooms.TagLit}
	c := reviewCorpse("bandit", 8, nil, nil)
	c.Gold, c.RoundCreated, c.BattleSpoils = 1, 500, true
	room.Corpses = []rooms.Corpse{c}
	for _, tc := range []struct{ period, want string }{{"2 hours", "now public"}, {"4 hours", "expired"}} {
		util.SetRoundCount(gametime.GetDate(500).AddPeriod(tc.period))
		text := captureUserText(t, func() { _, err := Look("bandit corpse", u, room, 0); require.NoError(t, err) })
		assert.Contains(t, text, tc.want)
	}
}

func TestLootClaimPublicExpiryAndSameNamedCorpses(t *testing.T) {
	u := users.NewUserRecord(7, 1)
	u.Character.CompanyCargo = true
	spec := items.ItemSpec{ItemId: 989701, Name: "loot token", Weight: 100}
	items.SetTestItemSpec(&spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(spec.ItemId) })
	item := items.New(spec.ItemId)
	item.Sharpen(1, 4)
	start := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(start) })
	util.SetRoundCount(500)
	c := rooms.Corpse{MobId: 1, ClaimUserId: 8, BattleSpoils: true, RoundCreated: 500, Character: *characters.New(), Items: []items.Item{item}, Gold: 9}
	c.Character.Name = "bandit"
	second := c
	second.Items = []items.Item{items.New(spec.ItemId)}
	room := &rooms.Room{Corpses: []rooms.Corpse{c, second}}
	gold := u.Character.Gold
	_, err := Loot("", u, room, 0)
	require.NoError(t, err)
	assert.Empty(t, u.Character.Items)
	assert.Equal(t, gold, u.Character.Gold)
	date := gametime.GetDate(c.RoundCreated)
	util.SetRoundCount(date.AddPeriod("2 hours"))
	_, err = Loot("", u, room, 0)
	require.NoError(t, err)
	require.Len(t, u.Character.Items, 2)
	assert.Equal(t, gold+18, u.Character.Gold)
	assert.Equal(t, item.UUID, u.Character.Items[0].UUID)
	assert.Equal(t, 4, u.Character.Items[0].SharpStrikes)
	room.Corpses = []rooms.Corpse{c}
	util.SetRoundCount(date.AddPeriod("4 hours"))
	_, err = Get("all corpse#1", u, room, 0)
	require.NoError(t, err)
	require.Len(t, u.Character.Items, 2)
	// Item consumption and management are refused throughout a fight.
	u.Character.SetAggro(0, 1, characters.DefaultAttack)
	_, err = Loot("", u, room, events.CmdSkipScripts)
	require.NoError(t, err)
	require.Len(t, u.Character.Items, 2)
}

func TestAutoLootRequiresExplicitOptIn(t *testing.T) {
	u := users.NewUserRecord(7, 1)
	assert.False(t, u.Character.AutoLoot)
	_, err := AutoLoot("on", u, nil, 0)
	require.NoError(t, err)
	assert.True(t, u.Character.AutoLoot)
	_, err = AutoLoot("off", u, nil, 0)
	require.NoError(t, err)
	assert.False(t, u.Character.AutoLoot)
}
