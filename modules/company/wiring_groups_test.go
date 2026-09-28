package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
	assert.Contains(t, b.cmd("attack", kind), "You prepare to fight "+name+"!", "the survivor answers to its group's name")
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
	assert.Contains(t, got, "You prepare to fight a band of ruffians!")
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
	assert.Contains(t, got, "You prepare to fight a band of ruffians!")
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
	assert.Contains(t, b.cmd("attack", "cave troll"), "You prepare to fight cave troll!")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, troll[0].InstanceId, b.aria.Character.Aggro.MobInstanceId)
}

// TestNothingTypedChangesABattle: once a battle is joined, every attack
// form, a spell, a backstab, and a shot are refused, and the aim is
// unchanged (the owner's rule 5).
func TestNothingTypedChangesABattle(t *testing.T) {
	b := newBrawl(t)
	spawnedHostiles(t, 920106)
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
