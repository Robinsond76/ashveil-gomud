package usercommands

import (
	"regexp"
	"strings"
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

// Looting says what was taken once per kind of corpse, not once per item.
func TestLootCombinesTakenThingsIntoOneLine(t *testing.T) {
	u := users.NewUserRecord(7, 1)
	u.Character.CompanyCargo = true
	for id, name := range map[int]string{989711: "wooden shield", 989712: "tattered pants", 989713: "leather cap"} {
		spec := items.ItemSpec{ItemId: id, Name: name, Weight: 100}
		items.SetTestItemSpec(&spec)
		id := id
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	start := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(start) })
	util.SetRoundCount(500)
	corpse := func(ids ...int) rooms.Corpse {
		c := rooms.Corpse{MobId: 1, ClaimUserId: 7, BattleSpoils: true, RoundCreated: 500, Character: *characters.New(), Gold: 3}
		c.Character.Name = "skeleton"
		for _, id := range ids {
			c.Items = append(c.Items, items.New(id))
		}
		return c
	}
	room := &rooms.Room{Corpses: []rooms.Corpse{corpse(989711, 989712), corpse(989711, 989713)}}
	text := plainLootText(captureUserText(t, func() { _, err := Loot("", u, room, 0); require.NoError(t, err) }))
	assert.Equal(t, 1, strings.Count(text, "You take"), text)
	assert.Contains(t, text, "6 gold")
	assert.Contains(t, text, "2 wooden shields")
	assert.Contains(t, text, "the tattered pants")
	assert.Contains(t, text, "the leather cap")
	assert.Contains(t, text, "skeleton corpses")

	// One corpse keeps the singular noun.
	room = &rooms.Room{Corpses: []rooms.Corpse{corpse(989713)}}
	text = plainLootText(captureUserText(t, func() { _, err := Get("all corpse", u, room, 0); require.NoError(t, err) }))
	assert.Equal(t, 1, strings.Count(text, "You take"), text)
	assert.Contains(t, text, "3 gold and the")
	assert.Contains(t, text, "skeleton corpse")
	assert.NotContains(t, text, "corpses")
}

func plainLootText(s string) string {
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
}

// Review of #197: names ending in "ss" or "us" and "x of y" names pluralise
// properly; names already plural stay; markup around the name survives.
func TestLootItemPhrasePlurals(t *testing.T) {
	cases := map[string]string{
		"wooden shield":        "wooden shields",
		"tattered pants":       "tattered pants",
		"iron greaves":         "iron greaves",
		"brigandine chausses":  "brigandine chausses",
		"fitted plate cuirass": "fitted plate cuirasses",
		"hound harness":        "hound harnesses",
		"celestial lotus":      "celestial lotuses",
		"vial of leadroot":     "vials of leadroot",
		"bolt of silk":         "bolts of silk",
		"history of frostfang": "histories of frostfang",
	}
	for in, want := range cases {
		assert.Equal(t, want, lootPlural(in), in)
	}
	got := plainLootText(lootItemPhrase(`<ansi fg="questflag">★</ansi>old key <ansi fg="black-bold">(cursed)</ansi>`, "old key", 2))
	assert.Equal(t, "2 ★old keys (cursed)", got)
	got = plainLootText(lootItemPhrase(`<ansi fg="r">o</ansi><ansi fg="g">rb</ansi>`, "orb", 3))
	assert.Equal(t, "the orb (x3)", got, "a name split by colour is counted, not mangled")
}

// Review of #197: looting several corpses with a full company says once what
// was left, after the one line of what was taken.
func TestLootSaysOnceWhatWasLeftBehind(t *testing.T) {
	setupCarry(t, map[int]int{7: 3000})
	room := testRoom()
	user := carrier(t, 7, "Dain", room)
	start := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(start) })
	util.SetRoundCount(500)
	for range 2 {
		c := rooms.Corpse{MobId: 1, ClaimUserId: 7, BattleSpoils: true, RoundCreated: 500, Character: *characters.New(), Gold: 2}
		c.Character.Name = "rat"
		c.Items = []items.Item{newItem(carryAnvil), newItem(carryPebble)}
		room.Corpses = append(room.Corpses, c)
	}
	text := plainLootText(captureUserText(t, func() { _, err := Loot("", user, room, 0); require.NoError(t, err) }))
	assert.Equal(t, 1, strings.Count(text, "You take"), text)
	assert.Equal(t, 1, strings.Count(text, "You leave"), text)
	assert.Contains(t, text, "You leave 2 things behind")
	assert.Less(t, strings.Index(text, "You take"), strings.Index(text, "You leave"), text)
}
