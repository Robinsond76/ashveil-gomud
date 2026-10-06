package company

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
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
		mob.SpawnGroup = ""
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

	b.aimAt("bandit captain")
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
	b.aimAt("bandit captain")
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
	assert.Regexp(t, `turns? toward Brom`, strings.Join(seen, "\n"))
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
	b.aimAt("bandit captain")
	b.toughen()
	// The battle must outlast its first round: a lucky round once felled
	// every engaged bandit and ended it (a random failure, seen in Phase 44).
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value = 1000
		m.Character.Health = 1000
	}
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
	assert.Contains(t, got, "The battle is under way", "nothing typed acts on a battle (32c)")
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
		for i := range room.SpawnInfo {
			room.SpawnInfo[i].InstanceId, room.SpawnInfo[i].DespawnedRound = 0, 0
		}
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

	b.aimAt("bandit cutthroat")
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

// waitingBandit returns a living bandit outside Aria's battle.
func (b *brawl) waitingBandit() *mobs.Mob {
	b.t.Helper()
	cur, ok := battle.Current(7)
	require.True(b.t, ok)
	for _, m := range b.livingBandits() {
		if !cur.Has(m.InstanceId) {
			return m
		}
	}
	b.t.Fatal("no bandit is waiting")
	return nil
}

// TestSpellAtAWaitingGroupIsHeld: a harmful spell Aria casts at a group
// waiting its turn lands on no one, and a waiting group's harmful spell at
// Aria is held too, the caster keeping its place in line.
func TestSpellAtAWaitingGroupIsHeld(t *testing.T) {
	b := newBrawl(t)
	b.looseBandits()
	// Keep the active group alive while testing waiting-group admission.
	b.hold(nil)
	b.aimAt("bandit captain")
	b.toughen()
	b.hold(nil)
	b.fight()
	b.toughen()
	b.hold(nil)

	waiting := b.waitingBandit()
	health := waiting.Character.Health
	b.aria.Character.Mana = 100
	b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{waiting.InstanceId}})
	got := b.fight()
	assert.Contains(t, got, "Your spell has no foe in your battle.")
	assert.Equal(t, health, waiting.Character.Health, "the waiting bandit is untouched")

	b.toughen()
	b.hold(nil)
	caster := b.waitingBandit()
	caster.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})
	var mm bool
	events := b.listen()
	b.fight()
	held := false
	for _, e := range *events {
		if e.Kind == combatstream.CastComplete && e.Source.MobInstanceId == caster.InstanceId {
			mm = mm || e.Outcome == combatstream.OutcomeCast
			held = held || e.Outcome == combatstream.OutcomeHeld
		}
	}
	assert.False(t, mm, "the waiting bandit's spell isn't cast")
	assert.True(t, held, "the stream reports it held")
	require.NotNil(t, caster.Character.Aggro, "it keeps its place in line")
	assert.Equal(t, 7, caster.Character.Aggro.UserId)
}

// TestDownedPlayerIsNotDrawnIntoNewBattles: when a lone player falls, their
// battle ends, and the groups still aiming at them don't begin battle after
// battle while they lie there.
func TestDownedPlayerIsNotDrawnIntoNewBattles(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.cmd("company", "dismiss all")
	b.looseBandits()
	b.aimAt("bandit captain")
	for i := 0; i < 2; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		b.fight()
	}
	_, ok := battle.Current(7)
	require.True(t, ok)

	*got = nil
	for i := 0; i < 3; i++ {
		b.aria.Character.Health = -1 // down, not yet dead
		b.fight()
	}
	starts := 0
	for _, e := range *got {
		if e.Kind == combatstream.FightStart && e.Source.UserId == 7 {
			starts++
		}
	}
	assert.Zero(t, starts, "no new battle while she's down")
	_, ok = battle.Current(7)
	assert.False(t, ok)
}

// TestBackstabAtAWaitingGroupIsCalledOff: an attack other than a plain one
// (a backstab) at a group waiting its turn isn't left hanging: it's called
// off, and the player told why.
func TestBackstabAtAWaitingGroupIsCalledOff(t *testing.T) {
	b := newBrawl(t)
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	b.looseBandits()
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.toughen()

	waiting := b.waitingBandit()
	health := waiting.Character.Health
	b.aria.Character.SetAggro(0, waiting.InstanceId, characters.BackStab, 0)
	got := b.fight()
	assert.Contains(t, got, "That foe is waiting its turn. Finish your battle first.")
	assert.Equal(t, health, waiting.Character.Health)
	if a := b.aria.Character.Aggro; a != nil {
		assert.NotEqual(t, characters.BackStab, a.Type)
	}
}

// TestWaitingGroupsDontBlockFlight: fleeing a battle (the retreat order,
// Phase 33c), only the battle's group pursues; the groups waiting their
// turn don't. The battle's foe is slow, the waiting ones quick: a roll of
// 50 gets away only against the battle's foe (65%), not the quick (30%).
func TestWaitingGroupsDontBlockFlight(t *testing.T) {
	b := newBrawl(t)
	b.looseBandits()
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.toughen()
	cur, ok := battle.Current(7)
	require.True(t, ok)
	waiting := 0
	for _, m := range b.livingBandits() {
		if cur.Has(m.InstanceId) {
			m.Character.Stats.Speed.ValueAdj = 0
			// The battle's foe must outlast the retreat's rounds, or a quick
			// waiting group becomes the battle (30g4 lowered enemy HP).
			m.Character.HealthMax.Value = 1000
			m.Character.Health = 1000
			continue
		}
		m.Character.Stats.Speed.ValueAdj = 100000
		m.Character.SetAggro(7, 0, characters.DefaultAttack)
		waiting++
	}
	require.Positive(t, waiting)
	b.aria.Character.Stats.Speed.ValueAdj = 1

	t.Cleanup(hooks.UseRetreatRollForTest(func(int) int { return 50 }))
	b.unpin()
	require.Contains(t, b.cmd("flee", ""), "begins an ordered retreat")
	b.fight()
	b.unpin()
	got := b.fight()
	assert.NotContains(t, got, "cuts off your withdrawal")
	assert.Contains(t, got, "withdraws together")
}

// hostilesIn lists a room's living hostile mobs.
func hostilesIn(room *rooms.Room) []*mobs.Mob {
	var out []*mobs.Mob
	for _, id := range room.GetMobs() {
		if m := mobs.GetInstance(id); m != nil && m.Hostile && m.Character.Health > 0 {
			out = append(out, m)
		}
	}
	return out
}

// TestASurvivorIsNotReinforced: a spawned pair whittled down to one isn't
// topped up again; when the room respawns its fallen, the newcomer joins
// the survivor, a pair once more.
func TestASurvivorIsNotReinforced(t *testing.T) {
	newBrawl(t)
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	room := rooms.LoadRoom(920103)
	t.Cleanup(func() {
		for _, m := range hostilesIn(room) {
			room.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		}
		room.CleanupMobSpawns(true)
	})
	tracked := room.SpawnInfo[0].InstanceId
	var topUp, listed *mobs.Mob
	for _, m := range pair {
		if m.InstanceId == tracked {
			listed = m
		} else {
			topUp = m
		}
	}
	require.NotNil(t, topUp)
	require.NotNil(t, listed)

	// The top-up falls: the listed ruffian stands alone, and stays so.
	room.RemoveMob(topUp.InstanceId)
	mobs.DestroyInstance(topUp.InstanceId)
	room.Prepare(false)
	require.Len(t, hostilesIn(room), 1, "a survivor isn't topped up")

	// It falls too, and the room respawns it: a new group of one, topped
	// up. Then the listed one of the new pair falls and respawns, joining
	// the survivor.
	room.RemoveMob(listed.InstanceId)
	mobs.DestroyInstance(listed.InstanceId)
	room.Prepare(false) // notes the loss
	room.SpawnInfo[0].DespawnedRound = 0
	room.Prepare(false) // respawns: a new group of one, topped up
	fresh := hostilesIn(room)
	require.Len(t, fresh, 2)
	assert.Equal(t, fresh[0].SpawnGroup, fresh[1].SpawnGroup)

	var survivor *mobs.Mob
	for _, m := range fresh {
		if m.InstanceId == room.SpawnInfo[0].InstanceId {
			room.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		} else {
			survivor = m
		}
	}
	require.NotNil(t, survivor)
	room.Prepare(false)
	room.SpawnInfo[0].DespawnedRound = 0
	room.Prepare(false)
	again := hostilesIn(room)
	require.Len(t, again, 2, "the newcomer joins the survivor; no third")
	for _, m := range again {
		assert.Equal(t, survivor.SpawnGroup, m.SpawnGroup)
	}
}

// TestASpawnedPairIsOneBattle: a pair spawned from a room's list fights as
// one group: Aria, alone, fights both in a single battle.
func TestASpawnedPairIsOneBattle(t *testing.T) {
	b := newBrawl(t)
	got := b.listen()
	b.cmd("company", "dismiss all")
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	alley := rooms.LoadRoom(920103)
	b.road.RemovePlayer(7)
	b.aria.Character.RoomId = alley.RoomId
	alley.AddPlayer(7)
	t.Cleanup(func() { alley.RemovePlayer(7) })

	b.cmd("attack", "ruffians")
	var cur battle.Battle
	for i := 0; i < 5; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		b.fight()
		if c, ok := battle.Current(7); ok {
			cur = c
		}
	}
	require.NotNil(t, cur.Enemies)
	assert.True(t, cur.Has(pair[0].InstanceId) && cur.Has(pair[1].InstanceId), "both are in the one battle")
	starts := 0
	for _, e := range *got {
		if e.Kind == combatstream.FightStart {
			starts++
		}
	}
	assert.Equal(t, 1, starts)
}

// TestCastAtAWaitingGroupIsRefused: through the real `cast` commands, a
// harmful spell at a group waiting its turn is refused before any mana is
// spent, for Aria and for a waiting bandit casting at her.
func TestCastAtAWaitingGroupIsRefused(t *testing.T) {
	b := newBrawl(t)
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	b.looseBandits()
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	b.toughen()

	waiting := b.waitingBandit()
	b.aria.Character.SpellBook["mm"] = 1
	if b.aria.Character.Skills == nil {
		b.aria.Character.Skills = map[string]int{}
	}
	b.aria.Character.Skills["cast"] = 1
	b.aria.Character.ManaMax.Value = 100
	b.aria.Character.Mana = 100
	got := b.cmd("cast", "mm #"+strconv.Itoa(waiting.InstanceId))
	assert.Contains(t, got, "The battle is under way", "nothing is cast by hand in a battle (32c)")
	assert.Equal(t, 100, b.aria.Character.Mana, "no mana spent")
	if a := b.aria.Character.Aggro; a != nil {
		assert.NotEqual(t, characters.SpellCast, a.Type)
	}

	caster := b.waitingBandit()
	caster.Character.SpellBook["mm"] = 1
	caster.Character.ManaMax.Value = 100
	caster.Character.Mana = 100
	_, err := mobcommands.TryCommand("cast", "mm @7", caster.InstanceId)
	require.NoError(t, err)
	assert.Equal(t, 100, caster.Character.Mana, "the waiting bandit spends no mana")
	require.NotNil(t, caster.Character.Aggro, "it keeps its place in line")
	assert.Equal(t, characters.DefaultAttack, caster.Character.Aggro.Type)
	assert.Equal(t, 7, caster.Character.Aggro.UserId)
}

// TestRoomRegroupsStragglers: through Room.Prepare, a hostile mob that
// wandered in joins the room's idle group; a travel encounter's pair is
// left as it is; and a group in a battle takes no one.
func TestRoomRegroupsStragglers(t *testing.T) {
	newBrawl(t)
	pair := spawnedHostiles(t, 920103)
	require.Len(t, pair, 2)
	room := rooms.LoadRoom(920103)
	group := pair[0].SpawnGroup
	add := func(spawnGroup string) *mobs.Mob {
		m := mobs.NewMobById(9107, room.RoomId)
		require.NotNil(t, m)
		m.Hostile = true
		m.SpawnGroup = spawnGroup
		room.AddMob(m.InstanceId)
		t.Cleanup(func() {
			room.RemoveMob(m.InstanceId)
			mobs.DestroyInstance(m.InstanceId)
		})
		return m
	}

	// In a battle, the group takes no one.
	battle.Begin(99, room.RoomId, 1, "p", []int{pair[0].InstanceId, pair[1].InstanceId})
	rat := add("")
	room.Prepare(false)
	assert.Empty(t, rat.SpawnGroup, "a group in a battle takes no straggler")
	battle.Reset()

	// Idle again: the straggler joins; the encounter pair is left alone.
	enc := []*mobs.Mob{add("encounter:920103:1"), add("encounter:920103:1")}
	room.Prepare(false)
	assert.Equal(t, group, rat.SpawnGroup, "the straggler joins the idle group")
	assert.Zero(t, rat.MaxWander)
	for _, m := range enc {
		assert.Equal(t, "encounter:920103:1", m.SpawnGroup)
	}
}
