package company

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 29b2 wiring: one battle at a time, and spawn groups, through the
// real round (shipped config, plugins.Load, real commands, DoCombat, idle
// mobs, queued mob commands).

// looseBandits makes every bandit hostile and a group of its own, as the
// owner's example describes (several groups in one room), so the battles
// can be watched one at a time. Spawning never makes lone groups; this
// test builds them by hand.
func (b *brawl) looseBandits() []int {
	var ids []int
	for _, mob := range b.livingBandits() {
		mob.Groups = nil
		mob.Hostile = true
		ids = append(ids, mob.InstanceId)
	}
	return ids
}

// fightTracker follows the fights on the stream: which are open, and the
// most open at once for one player.
type fightTracker struct {
	open    map[uint64]combatstream.Event // fight id -> its start
	maxOpen map[int]int                   // leader -> most fights open at once
	enemies map[uint64]map[int]bool       // fight id -> bandit ids that struck in it
}

func trackFights(events *[]combatstream.Event) fightTracker {
	ft := fightTracker{open: map[uint64]combatstream.Event{}, maxOpen: map[int]int{}, enemies: map[uint64]map[int]bool{}}
	for _, e := range *events {
		switch e.Kind {
		case combatstream.FightStart:
			ft.open[e.FightID] = e
			n := 0
			for _, s := range ft.open {
				if s.Source.UserId == e.Source.UserId {
					n++
				}
			}
			if n > ft.maxOpen[e.Source.UserId] {
				ft.maxOpen[e.Source.UserId] = n
			}
		case combatstream.FightEnd:
			delete(ft.open, e.FightID)
		}
	}
	return ft
}

// TestGroupsAreFoughtOneAtATime: five hostile groups in one room. The
// company fights one at a time, each its own fight and summary; a waiting
// group lands no blow on the company; the next battle begins as the last
// ends. The clock never moves.
func TestGroupsAreFoughtOneAtATime(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	bandits := map[int]bool{}
	for _, id := range b.looseBandits() {
		bandits[id] = true
	}
	company := b.companyInstances()

	b.cmd("attack", "bandit captain")
	seen := b.fightItOut(300)

	ft := trackFights(got)
	assert.Equal(t, 1, ft.maxOpen[7], "never two battles at once")
	assert.Empty(t, ft.open, "every battle ended")

	// Each blow between the company and a bandit is in the fight open for
	// that bandit's battle; a waiting bandit strikes no one.
	current := map[uint64]map[int]bool{} // open fight -> its bandits
	var open uint64
	fights := 0
	for _, e := range *got {
		switch e.Kind {
		case combatstream.FightStart:
			fights++
			open = e.FightID
			current[open] = map[int]bool{}
		case combatstream.Attack:
			src, tgt := e.Source.MobInstanceId, e.Target.MobInstanceId
			if bandits[src] && (e.Target.UserId == 7 || company[tgt]) {
				require.NotZero(t, e.FightID, "a bandit's blow on the company is in a battle")
				assert.Equal(t, open, e.FightID, "only the bandit in the current battle strikes")
			}
			if bandits[tgt] && (e.Source.UserId == 7 || company[src]) {
				assert.Equal(t, open, e.FightID, "the company strikes only its battle's foe")
			}
		}
	}
	assert.Equal(t, len(bandits), fights, "one battle for each group")
	assert.Equal(t, len(bandits), strings.Count(seen, summaryHeading), "a summary for each")

	assert.Equal(t, turn, util.GetTurnCount())
	assert.Equal(t, round, util.GetRoundCount())
}

// TestSecondPlayerTakesTheNextGroup: while Aria's company fights one group,
// the groups waiting on her turn on Brom when he walks in, and both battles
// run at once.
func TestSecondPlayerTakesTheNextGroup(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.looseBandits()
	b.cmd("attack", "bandit captain")
	b.toughen()
	b.fight()
	b.toughen()
	b.fight() // the other groups find Aria and wait

	brom := users.NewUserRecord(8, 2)
	brom.Username = "brom"
	brom.Password = "$2a$test"
	brom.Character.Name = "Brom"
	brom.Character.RaceId = 1
	brom.Character.Level = 3
	brom.Character.RoomId = b.road.RoomId
	brom.Character.Validate()
	brom.Character.HealthMax.Value = 1000
	brom.Character.Health = 1000
	users.SetTestUser(brom)
	b.road.AddPlayer(brom.UserId)
	t.Cleanup(func() { b.road.RemovePlayer(8) })

	var seen []string
	for i := 0; i < 10; i++ {
		if _, ok := battle.Current(8); ok {
			break
		}
		b.toughen()
		seen = append(seen, b.fight())
	}
	bromBattle, ok := battle.Current(8)
	require.True(t, ok, "Brom is fighting the next group")
	assert.Regexp(t, `turns? on Brom`, strings.Join(seen, "\n"))
	ariaBattle, ok := battle.Current(7)
	if ok {
		for id := range bromBattle.Enemies {
			assert.False(t, ariaBattle.Has(id), "each fights a group of their own")
		}
	}
	started := false
	for _, e := range *got {
		if e.Kind == combatstream.FightStart && e.Source.UserId == 8 {
			started = true
		}
	}
	assert.True(t, started, "Brom's battle is a fight on the stream")
}

// TestAttackOnAWaitingGroupIsRefused: in a battle, `attack` on a group
// waiting its turn is refused, and the player keeps their foe.
func TestAttackOnAWaitingGroupIsRefused(t *testing.T) {
	b := newBrawl(t)
	b.looseBandits()
	b.cmd("attack", "bandit captain")
	b.toughen()
	b.fight()
	cur, ok := battle.Current(7)
	require.True(t, ok)
	var waiting *mobs.Mob
	for _, m := range b.livingBandits() {
		if !cur.Has(m.InstanceId) {
			waiting = m
			break
		}
	}
	require.NotNil(t, waiting)
	before := *b.aria.Character.Aggro
	got := b.cmd("attack", "#"+strconv.Itoa(waiting.InstanceId))
	assert.Contains(t, got, "Finish that fight first.")
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, before.MobInstanceId, b.aria.Character.Aggro.MobInstanceId)
}

// spawnedHostiles prepares a room from its spawn list and returns its
// living hostile mobs.
func spawnedHostiles(t *testing.T, roomId int) []*mobs.Mob {
	t.Helper()
	room := rooms.LoadRoom(roomId)
	require.NotNil(t, room)
	room.Prepare(false)
	var out []*mobs.Mob
	for _, id := range room.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.Hostile {
			out = append(out, m)
		}
	}
	t.Cleanup(func() {
		for _, m := range out {
			room.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		}
		room.CleanupMobSpawns(true) // ready to spawn again on a repeated run
	})
	return out
}

// TestSpawnListsMakeGroups: spawning through Room.Prepare with real room
// and mob files. One hostile entry spawns a group of two; a rat and a
// ruffian spawn as one mixed group; a solitary troll stands alone; six
// entries make two groups of three. Grouped mobs don't wander.
func TestSpawnListsMakeGroups(t *testing.T) {
	newBrawl(t)

	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2, "one ruffian listed: two spawn")
	assert.NotEmpty(t, pair[0].SpawnGroup)
	assert.Equal(t, pair[0].SpawnGroup, pair[1].SpawnGroup)
	assert.Equal(t, "ruffian", pair[1].Character.Name)
	for _, m := range pair {
		assert.Zero(t, m.MaxWander, "a group stays together")
	}

	mixed := spawnedHostiles(t, 920104)
	require.Len(t, mixed, 2)
	assert.Equal(t, mixed[0].SpawnGroup, mixed[1].SpawnGroup, "a rat and a ruffian are one group")
	assert.ElementsMatch(t, []string{"big rat", "ruffian"}, []string{mixed[0].Character.Name, mixed[1].Character.Name})

	troll := spawnedHostiles(t, 920105)
	require.Len(t, troll, 1, "a solitary mob stands alone")
	assert.Empty(t, troll[0].SpawnGroup)

	six := spawnedHostiles(t, 920106)
	require.Len(t, six, 6)
	sizes := map[string]int{}
	for _, m := range six {
		sizes[m.SpawnGroup]++
	}
	assert.Len(t, sizes, 2)
	for g, n := range sizes {
		assert.Equal(t, 3, n, g)
	}

	// Preparing again doesn't add more.
	room := rooms.LoadRoom(920103)
	room.Prepare(false)
	hostiles := 0
	for _, id := range room.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.Hostile {
			hostiles++
		}
	}
	assert.Equal(t, 2, hostiles)
}

// TestSoloPlayerHasBattles: with no companion, a player still fights one
// group at a time, each a fight with a summary.
func TestSoloPlayerHasBattles(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.cmd("company", "dismiss all")
	require.Empty(t, b.companyInstances())
	b.looseBandits()

	b.cmd("attack", "bandit cutthroat")
	seen := b.fightItOut(400)

	ft := trackFights(got)
	assert.Equal(t, 1, ft.maxOpen[7], "one at a time")
	fights := 0
	for _, e := range *got {
		if e.Kind == combatstream.FightStart {
			fights++
		}
		if e.Kind == combatstream.Attack && e.Source.UserId == 7 {
			assert.NotZero(t, e.FightID, "a solo player's blows are in a battle")
		}
	}
	assert.Positive(t, fights)
	assert.Equal(t, fights, strings.Count(seen, summaryHeading))
}
