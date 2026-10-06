package archetype

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wiring (36a): Scribe identifies Rare and better gear at a camp rest (the
// provider seam the camping module calls) and in the field with the real
// scribe command.

const (
	scribeSwordID = 97711
	scribeMaceID  = 97712
)

func scribeItem(t *testing.T, rarity items.Rarity) items.Item {
	return scribeItemAs(t, scribeSwordID, "test blade", rarity)
}

func scribeItemAs(t *testing.T, id int, name string, rarity items.Rarity) items.Item {
	t.Helper()
	spec := &items.ItemSpec{ItemId: id, Name: name, NameSimple: name, Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6}, Value: 100}
	items.SetTestItemSpec(spec)
	t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	itm := items.New(id)
	itm.ApplyRoll(items.Rolled{
		Version: items.RollVersion, ILvl: 20, Quality: items.QualityStandard, Rarity: rarity, Identified: !rarity.DropsUnidentified(),
		BaseValue: 100, Name: "Gloomfang",
		Affixes: []items.RolledAffix{{ID: "strength", Label: "Mighty", Mechanic: "statmod:strength", Value: 2, Tier: 1, MinValue: 1, MaxValue: 2}},
	})
	return itm
}

func scribeRoom(t *testing.T, id int) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: id, Zone: "ScribeTest", Title: "Camp"}
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	return r
}

func scribeLeader(t *testing.T, id, roomID, rank int) *users.UserRecord {
	t.Helper()
	u := trainee(t, id, roomID)
	u.Character.Skills = map[string]int{"scribe": rank}
	u.Character.Mana, u.Character.ManaMax.Value = 50, 50
	return u
}

func pack(u *users.UserRecord, list ...items.Item) {
	u.Character.Items = nil
	for _, itm := range list {
		u.Character.Items = append(u.Character.Items, itm)
	}
}

func unreadCount(c *characters.Character) int {
	n := 0
	for _, itm := range c.Items {
		if !itm.IsIdentified() {
			n++
		}
	}
	return n
}

func TestCampRestScribeRankBoundariesOnRealPackItems(t *testing.T) {
	m := registered(t)
	scribeRoom(t, 97701)
	cases := []struct {
		rank   int
		rarity items.Rarity
		read   bool
	}{
		{0, items.RarityRare, false},
		{1, items.RarityRare, true}, {1, items.RarityEpic, false},
		{2, items.RarityEpic, true}, {2, items.RarityLegendary, false},
		{3, items.RarityLegendary, true}, {3, items.RaritySet, true},
		{4, items.RaritySet, true},
	}
	for i, c := range cases {
		u := scribeLeader(t, 400+i, 97701, c.rank)
		pack(u, scribeItem(t, c.rarity))
		lines := m.CampIdentify(u.UserId)
		assert.Equal(t, c.read, u.Character.Items[0].IsIdentified(), "rank %d, %s", c.rank, c.rarity)
		if c.read {
			require.NotEmpty(t, lines)
			assert.Contains(t, lines[0], "You read the company's new finds")
			assert.Equal(t, 2, u.Character.Items[0].GetSpec().StatMods.Get("strength"), "its affixes now count")
		} else {
			assert.Empty(t, lines)
		}
	}
}

func TestCampRestNeedsNoManaAndAutoskillTurnsItOff(t *testing.T) {
	m := registered(t)
	scribeRoom(t, 97702)
	u := scribeLeader(t, 410, 97702, 4)
	u.Character.Mana = 0
	pack(u, scribeItem(t, items.RarityRare), scribeItem(t, items.RarityEpic))

	assert.Contains(t, m.setAutoskill(410, utilityScribe, false), "now off")
	assert.Empty(t, m.CampIdentify(410))
	assert.Equal(t, 2, unreadCount(u.Character), "autoskill scribe off: nothing is read")

	assert.Contains(t, m.setAutoskill(410, utilityScribe, true), "now on")
	lines := m.CampIdentify(410)
	assert.Len(t, lines, 1)
	assert.Zero(t, unreadCount(u.Character))
	assert.Zero(t, u.Character.Mana, "the camp rest costs no mana")
	assert.Empty(t, m.CampIdentify(410), "a second rest finds nothing left")
}

func TestCampRestRevealsWornGearEvenWithoutAScribe(t *testing.T) {
	m := registered(t)
	scribeRoom(t, 97703)
	u := scribeLeader(t, 411, 97703, 0)
	worn := scribeItem(t, items.RarityEpic)
	u.Character.Equipment.Weapon = worn
	pack(u, scribeItem(t, items.RarityRare))

	lines := m.CampIdentify(411)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "worn gear gives up its secrets")
	assert.True(t, u.Character.Equipment.Weapon.IsIdentified(), "an item worn through a rest reveals itself")
	assert.Equal(t, 1, unreadCount(u.Character), "the pack's item needs a Scribe")

	m.setAutoskill(411, utilityScribe, false)
	u.Character.Equipment.Weapon = scribeItem(t, items.RarityRare)
	assert.NotEmpty(t, m.CampIdentify(411), "the worn-gear rule is not an autoskill")
	assert.True(t, u.Character.Equipment.Weapon.IsIdentified())
}

func TestCampRestUsesTheBestScribeAndNamesThem(t *testing.T) {
	m := registered(t)
	scribeRoom(t, 97704)
	u := scribeLeader(t, 412, 97704, 1)
	bran := withCompanion(t, 412, 97921, 97704, 8, "wizard")
	bran.Character.Skills = map[string]int{"scribe": 2}
	pack(u, scribeItem(t, items.RarityEpic))

	lines := m.CampIdentify(412)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Bran reads the company's new finds", "the companion's higher rank reads the Epic, and is named")
	assert.True(t, u.Character.Items[0].IsIdentified())

	// A tie goes to the leader.
	u.Character.Skills["scribe"] = 2
	pack(u, scribeItem(t, items.RarityEpic))
	lines = m.CampIdentify(412)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "You read the company's new finds")

	// A companion carries things too: theirs are read.
	bran.Character.Items = []items.Item{scribeItem(t, items.RarityRare)}
	assert.NotEmpty(t, m.CampIdentify(412))
	assert.True(t, bran.Character.Items[0].IsIdentified())
}

func TestScribeCommandSpendsManaAndReadsTheItem(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97705)
	u := scribeLeader(t, 413, 97705, 2)
	pack(u, scribeItemAs(t, scribeMaceID, "test mace", items.RarityEpic), scribeItem(t, items.RarityRare))

	text := captureText(t, func() {
		handled, err := m.scribeCommand("mace", u, room, 0)
		require.NoError(t, err)
		assert.True(t, handled)
	})
	assert.Contains(t, text, "You read the")
	assert.Contains(t, text, "(15 mana)")
	assert.Contains(t, text, "+2 strength")
	assert.Equal(t, 35, u.Character.Mana)
	assert.True(t, u.Character.Items[0].IsIdentified())
	assert.False(t, u.Character.Items[1].IsIdentified(), "only the named item")

	text = captureText(t, func() { _, _ = m.scribeCommand("blade", u, room, 0) })
	assert.Contains(t, text, "(8 mana)")
	assert.Equal(t, 27, u.Character.Mana)

	text = captureText(t, func() { _, _ = m.scribeCommand("blade", u, room, 0) })
	assert.Contains(t, text, "needs no reading", "an item already read is not read twice")
	assert.Equal(t, 27, u.Character.Mana)
	text = captureText(t, func() { _, _ = m.scribeCommand("lantern", u, room, 0) })
	assert.Contains(t, text, `carries a "lantern"`)
}

func TestScribeCommandRefusals(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97706)
	say := func(u *users.UserRecord, rest string) string {
		return captureText(t, func() { _, _ = m.scribeCommand(rest, u, room, 0) })
	}

	u := scribeLeader(t, 414, 97706, 1)
	pack(u, scribeItemAs(t, scribeMaceID, "test mace", items.RarityEpic))
	assert.Contains(t, say(u, "mace"), "takes Scribe rank 2; the best here is rank 1")
	assert.Equal(t, 50, u.Character.Mana)
	assert.False(t, u.Character.Items[0].IsIdentified())

	u.Character.Skills["scribe"] = 2
	u.Character.Mana = 14
	assert.Contains(t, say(u, "mace"), "takes 15 mana")
	assert.Equal(t, 14, u.Character.Mana, "a refusal spends nothing")
	assert.False(t, u.Character.Items[0].IsIdentified())

	u.Character.Mana = 50
	u.Character.Aggro = &characters.Aggro{MobInstanceId: 7}
	assert.Contains(t, say(u, "mace"), "battle", "never in battle")
	assert.Equal(t, 50, u.Character.Mana)
	assert.False(t, u.Character.Items[0].IsIdentified())
	u.Character.Aggro = nil

	none := scribeLeader(t, 415, 97706, 0)
	pack(none, scribeItem(t, items.RarityRare))
	assert.Contains(t, say(none, "blade"), "Nobody in your company has Scribe training")
	assert.False(t, none.Character.Items[0].IsIdentified())
}

func TestScribeCommandNamesACompanionScribeAndSpendsTheirMana(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97707)
	u := scribeLeader(t, 416, 97707, 0)
	bran := withCompanion(t, 416, 97922, 97707, 10, "wizard")
	bran.Character.Skills = map[string]int{"scribe": 3}
	bran.Character.Mana = 30
	pack(u, scribeItemAs(t, scribeMaceID, "test mace", items.RarityLegendary))

	text := captureText(t, func() { _, _ = m.scribeCommand("test mace", u, room, 0) })
	assert.Contains(t, text, "Bran reads the")
	assert.Contains(t, text, "(25 mana)")
	assert.Equal(t, 5, bran.Character.Mana, "the Scribe's own mana pays")
	assert.Equal(t, 50, u.Character.Mana)
	assert.True(t, u.Character.Items[0].IsIdentified())

	// Naming the member works, and a member without training is refused.
	pack(u, scribeItem(t, items.RarityRare))
	bran.Character.Skills = map[string]int{}
	text = captureText(t, func() { _, _ = m.scribeCommand("bran blade", u, room, 0) })
	assert.Contains(t, text, "Bran has no Scribe training")
	bran.Character.Skills = map[string]int{"scribe": 1}
	bran.Character.Mana = 3
	text = captureText(t, func() { _, _ = m.scribeCommand("bran blade", u, room, 0) })
	assert.Contains(t, text, "Bran lacks the mana")
	bran.Character.Mana = 10
	text = captureText(t, func() { _, _ = m.scribeCommand("bran blade", u, room, 0) })
	assert.Contains(t, text, "Bran reads the")
	assert.Equal(t, 2, bran.Character.Mana)
}

func TestScribeListShowsWhatWaits(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97708)
	u := scribeLeader(t, 417, 97708, 1)
	pack(u, scribeItemAs(t, scribeMaceID, "test mace", items.RarityEpic), scribeItem(t, items.RarityRare))
	text := captureText(t, func() { _, _ = m.scribeCommand("", u, room, 0) })
	assert.Contains(t, text, "Your best Scribe here is you (rank 1)")
	assert.Contains(t, text, "test mace")
	assert.Contains(t, text, "(needs rank 2)")
	assert.Contains(t, text, "test blade")
}

func TestScribeIsAShippedCasterSkill(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97709)
	room.SkillTraining = map[string]rooms.TrainingRange{"scribe": {Min: 1, Max: 4}}
	for _, arch := range []string{"warrior", "rogue", "ranger"} {
		u := trainee(t, 420, 97709)
		m.choose(u, arch, true)
		text := captureText(t, func() { _, _ = usercommands.Train("scribe", u, room, 0) })
		assert.Zero(t, u.Character.GetSkillLevel("scribe"), "%s cannot train Scribe: %s", arch, text)
	}
	for _, arch := range []string{"wizard", "cleric"} {
		u := trainee(t, 421, 97709)
		m.choose(u, arch, true)
		_, _ = usercommands.Train("scribe", u, room, 0)
		assert.Equal(t, 1, u.Character.GetSkillLevel("scribe"), "%s trains Scribe", arch)
	}
	assert.True(t, m.autoskillOn(421, utilityScribe))
	assert.Contains(t, m.autoskillList(421), "scribe")
}

// A companion's worn gear reveals itself at camp under their own name, with
// its capitals kept.
func TestCampRestNamesACompanionsWornGearWithItsCase(t *testing.T) {
	m := registered(t)
	scribeRoom(t, 97708)
	scribeLeader(t, 417, 97708, 0)
	bran := withCompanion(t, 417, 97923, 97708, 8, "wizard")
	bran.Character.Equipment.Weapon = scribeItem(t, items.RarityEpic)

	lines := m.CampIdentify(417)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "By the fire, Bran's worn gear")
	assert.True(t, bran.Character.Equipment.Weapon.IsIdentified())
}

// Copies of one base item are read one at a time, by instance.
func TestScribeReadsOneOfTwoCopiesOfTheSameBaseItem(t *testing.T) {
	m := registered(t)
	room := scribeRoom(t, 97709)
	u := scribeLeader(t, 418, 97709, 1)
	first := scribeItem(t, items.RarityRare)
	second := scribeItem(t, items.RarityRare)
	pack(u, first, second)

	captureText(t, func() { _, _ = m.scribeCommand("test blade", u, room, 0) })
	assert.Equal(t, 1, unreadCount(u.Character), "only one copy was read")
	captureText(t, func() { _, _ = m.scribeCommand("test blade", u, room, 0) })
	assert.Equal(t, 0, unreadCount(u.Character))
}
