package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32c wiring: named enemy groups, `attack <group>`, and a battle that
// plays out on its own, through the real commands and round (shipped
// config, spawned rooms, DoCombat).

// into moves Aria (alone: her companions dismissed) into a room.
func (b *brawl) into(roomId int) *rooms.Room {
	b.t.Helper()
	b.cmd("company", "dismiss all")
	room := rooms.LoadRoom(roomId)
	require.NotNil(b.t, room)
	b.road.RemovePlayer(7)
	b.aria.Character.RoomId = room.RoomId
	room.AddPlayer(7)
	b.t.Cleanup(func() { room.RemovePlayer(7) })
	return room
}

// groupLines are the group lines of Aria's room, as look renders them.
func (b *brawl) groupLines() []string {
	b.t.Helper()
	return rooms.GetDetails(rooms.LoadRoom(b.aria.Character.RoomId), b.aria).VisibleGroups
}

func TestSpawnedGroupsAreNamedInTheRoom(t *testing.T) {
	b := newBrawl(t)
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	for _, m := range pair {
		assert.Equal(t, "a band of ruffians", m.GroupName, "named once, when the group formed")
	}
	b.into(920103)
	assert.Equal(t, []string{`<ansi fg="mobname">A band of ruffians</ansi> (2).`}, b.groupLines())

	mixed := spawnedHostiles(t, 920104)
	require.Len(t, mixed, 2)
	name := mixed[0].GroupName
	assert.Equal(t, name, mixed[1].GroupName)
	assert.Contains(t, []string{"a band of ruffians", "a band of big rats"}, name)
	b.into(920104)
	lines := b.groupLines()
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "(2): a ")
}

func TestAGroupKeepsItsNameAsMembersFall(t *testing.T) {
	b := newBrawl(t)
	mixed := spawnedHostiles(t, 920104)
	require.Len(t, mixed, 2)
	room := b.into(920104)
	name := mixed[0].GroupName
	// The member the group is named for falls; the other keeps the name.
	var gone, left *mobs.Mob
	for _, m := range mixed {
		if strings.Contains(name, m.Character.Name) {
			gone = m
		} else {
			left = m
		}
	}
	require.NotNil(t, gone)
	room.RemoveMob(gone.InstanceId)
	mobs.DestroyInstance(gone.InstanceId)
	room.Prepare(false)
	assert.Equal(t, name, left.GroupName)
	lines := b.groupLines()
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], fmt.Sprintf("(1): a %s.", left.Character.Name))
	kind := name[strings.LastIndex(name, " ")+1:]
	assert.Contains(t, b.cmd("attack", kind), "You go for the "+name[strings.Index(name, " ")+1:]+".", "the survivor answers to its group's name")
}

func TestAttackStartsABattleWithAGroup(t *testing.T) {
	b := newBrawl(t)
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	b.into(920103)

	got := b.cmd("attack", "ruffian")
	assert.Contains(t, got, "The ruffian fights with a band of ruffians. Type attack ruffians.")
	assert.Nil(t, b.aria.Character.Aggro, "a member's name starts nothing")

	got = b.cmd("attack", "band")
	assert.Contains(t, got, "You go for the band of ruffians.")
	require.NotNil(t, b.aria.Character.Aggro)
	aim := b.aria.Character.Aggro.MobInstanceId
	assert.Contains(t, []int{pair[0].InstanceId, pair[1].InstanceId}, aim)

	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.fight()
	cur, ok := battle.Current(7)
	require.True(t, ok)
	assert.True(t, cur.Has(pair[0].InstanceId) && cur.Has(pair[1].InstanceId), "the battle is with the whole group")
}

func TestAttackByIdTakesTheWholeGroup(t *testing.T) {
	b := newBrawl(t)
	pair := spawnedHostiles(t, 920103)
	b.into(920103)
	got := b.cmd("attack", fmt.Sprintf("#%d", pair[1].InstanceId))
	assert.Contains(t, got, "You go for the band of ruffians.")
	require.NotNil(t, b.aria.Character.Aggro)
}

func TestTwoGroupsOfOneNameAreNumbered(t *testing.T) {
	b := newBrawl(t)
	six := spawnedHostiles(t, 920106)
	require.Len(t, six, 6)
	b.into(920106)
	assert.Equal(t, []string{
		`<ansi fg="mobname">A band of ruffians</ansi> (3).`,
		`<ansi fg="mobname">A second band of ruffians</ansi> (3).`,
	}, b.groupLines())

	b.cmd("attack", "ruffians#2")
	require.NotNil(t, b.aria.Character.Aggro)
	room := rooms.LoadRoom(920106)
	var groups []string
	for _, id := range room.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.Hostile && (len(groups) == 0 || groups[len(groups)-1] != m.SpawnGroup) {
			groups = append(groups, m.SpawnGroup)
		}
	}
	require.GreaterOrEqual(t, len(groups), 2)
	aimed := mobs.GetInstance(b.aria.Character.Aggro.MobInstanceId)
	assert.Equal(t, groups[1], aimed.SpawnGroup, "the second band")
}

func TestALoneMobIsAttackedByItsName(t *testing.T) {
	b := newBrawl(t)
	troll := spawnedHostiles(t, 920105)
	require.Len(t, troll, 1)
	b.into(920105)
	assert.Contains(t, b.cmd("attack", "cave troll"), "go for the cave troll.")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, troll[0].InstanceId, b.aria.Character.Aggro.MobInstanceId)
}

// TestNothingTypedChangesABattle: once a battle is joined, every attack
// form, a spell, a backstab, and a shot are refused, and the aim is
// unchanged (the owner's rule 5).
func TestNothingTypedChangesABattle(t *testing.T) {
	b := newBrawl(t)
	for _, m := range spawnedHostiles(t, 920106) {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000 // none falls: her aim holds
	}
	b.into(920106)
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.cmd("attack", "ruffians")
	b.fight()
	_, ok := battle.Current(7)
	require.True(t, ok)
	require.NotNil(t, b.aria.Character.Aggro)
	before := b.aria.Character.Aggro.MobInstanceId

	b.aria.Character.SpellBook["mm"] = 1
	b.aria.Character.SpellBook["heal"] = 1
	if b.aria.Character.Skills == nil {
		b.aria.Character.Skills = map[string]int{}
	}
	b.aria.Character.Skills["cast"] = 1
	b.aria.Character.Skills["skulduggery"] = 4
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 100, 100
	for _, c := range [][2]string{
		{"attack", ""}, {"attack", "ruffians"}, {"attack", "ruffians#2"}, {"attack", "ruffian"},
		{"cast", "mm ruffian"}, {"cast", "heal"}, {"backstab", "ruffian"}, {"shoot", "ruffian east"},
	} {
		got := b.cmd(c[0], c[1])
		assert.Contains(t, got, "The battle is under way", "%s %s", c[0], c[1])
		require.NotNil(t, b.aria.Character.Aggro)
		assert.Equal(t, before, b.aria.Character.Aggro.MobInstanceId, "%s %s changes nothing", c[0], c[1])
	}
	assert.Equal(t, 100, b.aria.Character.Mana, "no mana spent")
}

// TestSpellsAndShotsDontStartFights: out of a battle, a harmful spell at an
// enemy, or a shot, is refused with the attack pointer; a helpful spell
// still casts.
func TestSpellsAndShotsDontStartFights(t *testing.T) {
	b := newBrawl(t)
	spawnedHostiles(t, 920103)
	b.into(920103)
	b.aria.Character.SpellBook["mm"] = 1
	b.aria.Character.SpellBook["heal"] = 1
	if b.aria.Character.Skills == nil {
		b.aria.Character.Skills = map[string]int{}
	}
	b.aria.Character.Skills["cast"] = 1
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 100, 100

	got := b.cmd("cast", "mm ruffian")
	assert.Contains(t, got, "A harmful spell doesn't start a fight. Type attack ruffians to fight a band of ruffians.")
	assert.Nil(t, b.aria.Character.Aggro)
	assert.Equal(t, 100, b.aria.Character.Mana)

	b.cmd("cast", "heal")
	require.NotNil(t, b.aria.Character.Aggro, "a helpful spell is cast")
	assert.Equal(t, characters.SpellCast, b.aria.Character.Aggro.Type)
	b.aria.Character.Aggro = nil

	// A shot from the verge at the ruffians' alley isn't how a fight starts.
	sling := items.New(10014)
	require.NotZero(t, sling.ItemId)
	b.aria.Character.Equipment.Weapon = sling
	road := rooms.LoadRoom(920101)
	alley := rooms.LoadRoom(920103)
	road.Exits["north"] = road.Exits["east"]
	exit := road.Exits["north"]
	exit.RoomId = alley.RoomId
	road.Exits["north"] = exit
	t.Cleanup(func() { delete(road.Exits, "north") })
	alley.RemovePlayer(7)
	b.aria.Character.RoomId = road.RoomId
	road.AddPlayer(7)
	got = b.cmd("shoot", "ruffian north")
	assert.Contains(t, got, "A shot doesn't start a fight.")
	assert.Nil(t, b.aria.Character.Aggro)
}

func TestLookAtAGroup(t *testing.T) {
	b := newBrawl(t)
	spawnedHostiles(t, 920103)
	b.into(920103)
	got := b.cmd("look", "band")
	assert.Contains(t, got, "A band of ruffians, two strong, idle.")
	assert.Contains(t, got, "a ruffian (unhurt), a ruffian (unhurt)")
	assert.Contains(t, got, "Type scout ruffians to see how they stand, or attack ruffians to fight them.")
	assert.NotContains(t, b.cmd("look", "ruffian"), "strong,", "a member is looked at as before")

	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.cmd("attack", "ruffians")
	b.fight()
	assert.Contains(t, b.cmd("look", "ruffians"), "fighting you.")
}

func TestScoutAGroup(t *testing.T) {
	b := newBrawl(t)
	mixed := spawnedHostiles(t, 920104)
	room := b.into(920104)
	name := mixed[0].GroupName
	kind := strings.TrimPrefix(name, "a band of ")

	got := b.cmd("scout", "")
	assert.Contains(t, got, "Enemies here:")
	assert.Contains(t, got, mobparty.Capitalize(name)+" (2): a ")
	assert.Contains(t, got, "Type scout "+kind)

	round := util.GetRoundCount()
	got = b.cmd("scout", kind)
	assert.Contains(t, got, mobparty.Capitalize(name)+", as they stand (front row nearest you):")
	assert.Contains(t, got, "front [")
	assert.Contains(t, got, "(unhurt)")
	assert.Contains(t, got, "You aren't placed in your company's formation")
	assert.Equal(t, round, util.GetRoundCount(), "scouting spends no round")
	assert.Nil(t, b.aria.Character.Aggro, "and starts nothing")

	// Placed at the front, left: Aria can reach the front of columns 1 and 2.
	require.Contains(t, b.cmd("formation", "move me 1 1"), "Placed")
	got = b.cmd("scout", kind)
	assert.Contains(t, got, "* you can reach them from your place in the formation.")
	assert.Contains(t, got, "*")

	// In a fight, and on a waiting group, it works the same.
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.cmd("attack", kind)
	b.fight()
	assert.Contains(t, b.cmd("scout", kind), "as they stand")

	// In the dark, nothing.
	biome := room.Biome
	room.Biome = "cave"
	t.Cleanup(func() { room.Biome = biome })
	assert.Contains(t, b.cmd("scout", kind), "It's too dark to make them out.")
	assert.Contains(t, b.cmd("scout", "nobody"), "It's too dark")
}

func TestScoutFindsNoGroup(t *testing.T) {
	b := newBrawl(t)
	b.into(920102)
	assert.Contains(t, b.cmd("scout", ""), "You see no enemies here.")
	assert.Contains(t, b.cmd("scout", "ruffians"), `You see no group called "ruffians" here.`)
}

// TestALonePlayerWhoBrokeOffIsTurnedAgain (32c review M1, M2): a player
// with no company record is turned onto the next foe when theirs falls, in
// 29c's voice, and the room sees it. Phase 32d: `break` and a bare
// `attack` are refused in the battle and change nothing.
func TestALonePlayerWhoBrokeOffIsTurnedAgain(t *testing.T) {
	b := newBrawl(t)
	b.cmd("company", "dismiss all")
	rec, _ := module.registry.Get(7)
	module.registry.Remove(7)
	t.Cleanup(func() { module.registry.Put(rec) })
	for _, m := range spawnedHostiles(t, 920106) {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	room := b.into(920106)
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
	b.cmd("attack", "ruffians")
	b.fight()
	_, ok := battle.Current(7)
	require.True(t, ok)
	assert.Contains(t, b.cmd("break", ""), "Use retreat or flee to leave it.")
	b.fight()
	assert.Contains(t, b.cmd("attack", ""), "The battle is under way")
	require.NotNil(t, b.aria.Character.Aggro, "still fighting")

	target := b.aria.Character.Aggro.MobInstanceId
	room.RemoveMob(target)
	mobs.DestroyInstance(target)
	all := ""
	for i := 0; i < 3; i++ {
		b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 1000
		all += b.fight()
	}
	a := b.aria.Character.Aggro
	require.NotNil(t, a, "turned onto the next foe")
	assert.NotEqual(t, target, a.MobInstanceId)
	assert.Regexp(t, `You turn toward the (?:second|third) ruffian\.`, all)
	assert.NotContains(t, all, "You turn on")
}

// TestAThingIsNotHiddenByAGroup (32c review m4): a word that names a thing
// here (a room noun) looks at the thing, not at the group sharing the word.
func TestAThingIsNotHiddenByAGroup(t *testing.T) {
	b := newBrawl(t)
	spawnedHostiles(t, 920103)
	room := b.into(920103)
	if room.Nouns == nil {
		room.Nouns = map[string]string{}
	}
	room.Nouns["band"] = "A faded band of cloth is tied to a post."
	t.Cleanup(func() { delete(room.Nouns, "band") })
	got := b.cmd("look", "band")
	assert.Contains(t, got, "faded band of cloth")
	assert.NotContains(t, got, "two strong")
	assert.Contains(t, b.cmd("look", "ruffians"), "two strong", "the group's other words still work")
}

// TestPeacefulTagMatesAreNotAGroup (Phase 32d, the owner): the bandits
// here aren't hostile; without a spawn group their shared tag makes no
// "band", and each is attacked by its own name.
func TestPeacefulTagMatesAreNotAGroup(t *testing.T) {
	b := newBrawl(t)
	for _, m := range b.livingBandits() {
		m.SpawnGroup = ""
	}
	assert.Empty(t, b.groupLines(), "no band of bandits")
	assert.NotContains(t, b.cmd("scout", ""), "band of bandit")
	assert.Contains(t, b.cmd("attack", "bandit captain"), "You go for the bandit captain.")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, b.bandits["bandit captain"][0], b.aria.Character.Aggro.MobInstanceId)
}
