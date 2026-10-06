package death

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// defeatEnv is a disposable world with the shipped brigand camp and the
// shipped defeat table, a leader on the Old King's Road with a pack, gold
// and a companion, and a foe to be beaten by.
type defeatEnv struct {
	t         *testing.T
	dataDir   string
	user      *users.UserRecord
	companion int
	pack      int // loose items carried before the fall
	run       func(command, rest string) string
}

func newDefeatEnv(t *testing.T, foeMob int) *defeatEnv {
	t.Helper()
	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	writeWiringWorld(t, dataDir)
	shipped := shippedWorld()
	copies := []string{
		"races/26-beast.yaml",
		"items/weapons-10000/10015-crude_cudgel.yaml",
		"rooms/brigand_camp/zone-config.yaml",
		"rooms/brigand_camp/91001.yaml",
		"rooms/brigand_camp/91002.yaml",
		"mobs/old_kings_road/86-road_brigand.yaml",
		"buffs/9301-bound.yaml",
	}
	flags, err := filepath.Glob(filepath.Join(shipped, "buffs-flags", "*.yaml"))
	require.NoError(t, err)
	for _, path := range flags {
		copies = append(copies, filepath.Join("buffs-flags", filepath.Base(path)))
	}
	for _, path := range copies {
		data, err := os.ReadFile(filepath.Join(shipped, path))
		require.NoError(t, err, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), data, 0600))
	}
	fixtures := map[string]string{
		"mobs/old_kings_road/9101-test_wolf.yaml": "mobid: 9101\nzone: Old Kings Road\nitemdropchance: 0\nhostile: true\ncharacter:\n  name: test wolf\n  raceid: 26\n  level: 3\n",
	}
	for path, data := range fixtures {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dataDir, path)), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, path), []byte(data), 0600))
	}
	races.LoadDataFiles()
	items.LoadDataFiles()
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NotEmpty(t, module.config().scenarios, "the shipped overlay carries the defeat table")
	// Another test's death in this round must not read as this one's.
	module.mu.Lock()
	module.returned = map[int]uint64{}
	module.mu.Unlock()

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
	})
	user := users.NewUserRecord(21, 1)
	user.Username = "ysabel"
	user.Password = "$2a$test"
	user.Character.Name = "Ysabel"
	user.Character.RaceId = 1
	user.Character.Alignment = 10
	user.Character.Level = 5
	user.Character.ActionPoints = 100
	user.Character.Validate()
	user.Character.Health = user.Character.HealthMax.Value
	user.Character.Gold = 200
	for i := 0; i < 6; i++ {
		user.Character.StoreItem(items.New(38))
	}
	user.Character.Equipment.Weapon = items.New(10015)
	user.Character.SetMiscData(domain.CheckpointKey, 2007)
	users.SetTestUser(user)
	require.NoError(t, rooms.MoveToRoom(user.UserId, 2002))
	t.Cleanup(func() {
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.RemovePlayer(user.UserId)
		}
	})
	freshEvents(t)
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg := e.(events.Message); msg.UserId == user.UserId {
			messages = append(messages, msg.Text)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	env := &defeatEnv{t: t, dataDir: dataDir, user: user}
	env.run = func(command, rest string) string {
		t.Helper()
		events.ProcessEvents()
		messages = nil
		handled, err := usercommands.TryCommand(command, rest, user.UserId, events.CmdSkipScripts)
		require.NoError(t, err)
		require.True(t, handled)
		events.ProcessEvents()
		return tagPattern.ReplaceAllString(strings.Join(messages, "\n"), "")
	}
	// Recruit a companion who stands with the leader.
	require.Contains(t, env.run("company", "summon training dummy"), "Companion summoned")
	var ok bool
	env.companion, ok = company.InstanceFor(user.UserId, 1)
	require.True(t, ok)
	env.pack = len(user.Character.Items)
	require.GreaterOrEqual(t, env.pack, 4)
	return env
}

// force makes the defeat table roll the scenario with this ID.
func forceScenario(t *testing.T, id string) {
	t.Helper()
	cfg := module.config()
	var before, total int
	found := false
	for _, s := range cfg.scenarios {
		w := max(1, s.Weight)
		if s.ID == id {
			found = true
			before = total
		}
		total += w
	}
	require.True(t, found, id)
	module.roll = func(n int) int { return min(before, n-1) }
	t.Cleanup(func() { module.roll = util.Rand })
}

// foe puts a hostile mob that has just landed the killing blow in the
// leader's room.
func (e *defeatEnv) foe(mobID int) *mobs.Mob {
	e.t.Helper()
	room := rooms.LoadRoom(e.user.Character.RoomId)
	foe := mobs.NewMobById(mobs.MobId(mobID), room.RoomId)
	require.NotNil(e.t, foe)
	foe.SpawnGroup = "test-pack"
	room.AddMob(foe.InstanceId)
	e.user.Character.KillerMobInstanceId = foe.InstanceId
	return foe
}

func (e *defeatEnv) packSize() int { return len(e.user.Character.Items) }

// defeatGuards kills capture guards as a fight would: each death is
// announced, then the body is gone.
func defeatGuards(room *rooms.Room, ids []int) {
	for _, id := range ids {
		events.AddToQueue(events.MobDeath{InstanceId: id, RoomId: room.RoomId})
	}
	events.ProcessEvents()
	vanishGuards(room, ids)
}

// vanishGuards removes guards without a fight, as a restart or an unloaded
// room does.
func vanishGuards(room *rooms.Room, ids []int) {
	for _, id := range ids {
		room.RemoveMob(id)
		mobs.DestroyInstance(id)
	}
}

func TestDefeatScenarioRescuedWakesHungryAtASettlementWithoutLosingALevel(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "wayfarer-rescue")
	env.foe(86)
	c := env.user.Character
	turn := util.GetTurnCount()

	out := env.run("suicide", "")
	assert.Contains(t, out, "carried you to")
	assert.Equal(t, 2007, c.RoomId, "the checkpoint's settlement: no settlement in the road's zone")
	assert.Equal(t, 5, c.Level, "a scenario costs no level")
	assert.Equal(t, env.pack, env.packSize(), "nothing lost")
	assert.Equal(t, 200, c.Gold)
	assert.Empty(t, rooms.LoadRoom(2002).Corpses, "no corpse where they fell")
	assert.False(t, module.Pending(env.user.UserId))
	assert.Nil(t, c.GetMiscData(domain.ScenarioKey), "the claim is cleared once they have woken")

	for _, member := range survival.CompanyNeeds(env.user.UserId) {
		assert.LessOrEqual(t, member.Needs.Hunger, 45, "%s wakes hungry", member.Name)
		assert.LessOrEqual(t, member.Needs.Fatigue, 20, "%s wakes exhausted", member.Name)
		assert.Equal(t, survival.BandLow, survival.BandFor(member.Needs.Hunger))
	}
	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
}

func TestDefeatScenarioCapturedHoldsThePackUntilTheGuardsFall(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-capture")
	env.foe(86)
	c := env.user.Character
	total := env.packSize()

	out := env.run("suicide", "")
	assert.Contains(t, out, "You wake bound in Brigands' Tent")
	assert.Contains(t, out, "50 gold is gone for good", "a quarter of 200")
	assert.Equal(t, 91001, c.RoomId)
	assert.Equal(t, 5, c.Level)
	assert.Empty(t, c.Items, "the pack is in the chest")
	assert.Len(t, c.Seized, total)
	assert.Zero(t, c.Gold)
	assert.Equal(t, 150, seizedGold(c))
	assert.Equal(t, 91001, seizedRoom(c))
	assert.NotZero(t, c.Equipment.Weapon.ItemId, "what they wear is theirs")
	mob := mobs.GetInstance(env.companion)
	require.NotNil(t, mob)
	assert.Equal(t, 91001, mob.Character.RoomId, "the company is bound with them")
	room := rooms.LoadRoom(91001)
	guards := module.guardsIn(room, domain.CaptorGroup(91001, env.user.UserId))
	require.Len(t, guards, 2)
	for _, id := range guards {
		assert.False(t, mobs.GetInstance(id).Hostile, "they guard; they do not attack")
	}

	// Bound: no leaving yet.
	assert.True(t, c.HasBuffFlag("no-go"))
	assert.Contains(t, env.run("go", "south"), "You can't do that!")
	assert.Equal(t, 91001, c.RoomId)

	// The chest stays shut while the guards stand.
	assert.Contains(t, env.run("reclaim", ""), "Defeat them first")
	assert.Empty(t, c.Items)

	// Saved with the character, so a restart keeps it.
	require.NoError(t, users.SaveUser(*env.user))
	data, err := os.ReadFile(filepath.Join(env.dataDir, "users", "21.yaml"))
	require.NoError(t, err)
	reloaded := users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	assert.Len(t, reloaded.Character.Seized, total)
	assert.Empty(t, reloaded.Character.Items)
	assert.Equal(t, 150, seizedGold(reloaded.Character))

	// The guards fall; the chest opens; the pack is whole again.
	defeatGuards(room, guards)
	out = env.run("reclaim", "")
	assert.Contains(t, out, fmt.Sprintf("take back your pack (%d items) and 150 gold", total))
	assert.Len(t, c.Items, total)
	assert.Empty(t, c.Seized)
	assert.Equal(t, 150, c.Gold)
	assert.Contains(t, env.run("reclaim", ""), "Nothing of yours is being held.")
	assert.Len(t, c.Items, total, "no item is duplicated")

	// And it can be escaped by walking out once the bonds are loose.
	c.CancelBuffsWithFlag(buffs.All)
	env.run("go", "south")
	assert.Equal(t, 91002, c.RoomId)
	env.run("go", "south")
	assert.Equal(t, 2002, c.RoomId, "the brigand camp opens onto the road")
}

func TestDefeatScenarioReclaimIsOnlyAtTheCaptureRoom(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-capture")
	env.foe(86)
	env.run("suicide", "")
	c := env.user.Character
	room := rooms.LoadRoom(91001)
	defeatGuards(room, module.guardsIn(room, domain.CaptorGroup(91001, env.user.UserId)))
	c.CancelBuffsWithFlag(buffs.All)
	env.run("go", "south")
	require.Equal(t, 91002, c.RoomId)
	assert.Contains(t, env.run("reclaim", ""), "locked in a chest in Brigands' Tent")
	assert.Empty(t, c.Items)
	assert.Len(t, c.Seized, env.pack, "still held, never lost by leaving")
}

// Regression (53 review): guards are not saved with the room, so a restart
// or an unloaded room used to leave the chest open without a fight. Only
// guards that fall in a fight count; the rest stand again.
func TestDefeatScenarioGuardsReturnUntilDefeated(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-capture")
	env.foe(86)
	env.run("suicide", "")
	c := env.user.Character
	room := rooms.LoadRoom(91001)
	group := domain.CaptorGroup(91001, env.user.UserId)
	guards := module.guardsIn(room, group)
	require.Len(t, guards, 2)
	assert.Equal(t, 2, domain.SeizedGuards(c))

	for _, id := range guards {
		assert.True(t, mobs.GetInstance(id).NeverBreak, "guards never flee or yield, so each one falls")
	}

	// A guard falling this round, its death not yet counted, is not lost:
	// reclaim neither opens the chest nor posts another.
	for _, id := range guards {
		mobs.GetInstance(id).Character.Health = 0
	}
	assert.Contains(t, env.run("reclaim", ""), "Defeat them first")
	assert.Empty(t, c.Items)
	assert.Len(t, room.GetMobs(), 3, "two guards and the companion; none posted")
	for _, id := range guards {
		mob := mobs.GetInstance(id)
		mob.Character.Health = mob.Character.HealthMax.Value
	}

	// One falls in a fight; the other is lost to a restart.
	defeatGuards(room, guards[:1])
	assert.Equal(t, 1, domain.SeizedGuards(c))
	vanishGuards(room, guards[1:])
	require.Empty(t, module.guardsIn(room, group))

	out := env.run("reclaim", "")
	assert.Contains(t, out, "Your captors stand over the chest again.")
	assert.Contains(t, out, "Defeat them first")
	assert.Empty(t, c.Items)
	back := module.guardsIn(room, group)
	require.Len(t, back, 1, "only the guard never beaten returns")

	// Walking back in after the room was emptied posts it as well.
	vanishGuards(room, back)
	c.CancelBuffsWithFlag(buffs.All)
	env.run("go", "south")
	assert.Contains(t, env.run("go", "north"), "Your captors stand over the chest again.")
	back = module.guardsIn(room, group)
	require.Len(t, back, 1)

	defeatGuards(room, back)
	assert.Zero(t, domain.SeizedGuards(c))
	assert.Contains(t, env.run("reclaim", ""), fmt.Sprintf("take back your pack (%d items)", env.pack))
	assert.Len(t, c.Items, env.pack)
	assert.Nil(t, c.GetMiscData(domain.SeizedByKey))
}

func TestDefeatScenarioResumesAfterAFailedWakeWithoutRerolling(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-capture")
	env.foe(86)
	c := env.user.Character
	total := env.packSize()

	// The first wake can't happen: the move fails, so they stay down and
	// the claim stays saved with the death's pending mark.
	realMove := module.moveToRoom
	module.moveToRoom = func(int, int) error { return os.ErrClosed }
	t.Cleanup(func() { module.moveToRoom = realMove })
	env.run("suicide", "")
	assert.True(t, module.Pending(env.user.UserId))
	assert.Equal(t, "brigand-capture", scenarioOf(c))
	assert.Equal(t, 5, c.Level, "no level taken, now or on any retry")
	assert.Equal(t, total, env.packSize(), "nothing taken from a company that never woke")
	assert.Equal(t, 200, c.Gold)

	// A restart reloads the character; the retry must finish the same
	// scenario even though the table would now roll something else.
	require.NoError(t, users.SaveUser(*env.user))
	data, err := os.ReadFile(filepath.Join(env.dataDir, "users", "21.yaml"))
	require.NoError(t, err)
	reloaded := users.UserRecord{}
	require.NoError(t, yaml.Unmarshal(data, &reloaded))
	assert.Equal(t, "brigand-capture", scenarioOf(reloaded.Character))

	module.moveToRoom = realMove
	module.roll = func(int) int { panic("a retry must never roll") }
	env.run("suicide", "")
	assert.False(t, module.Pending(env.user.UserId))
	assert.Equal(t, 91001, c.RoomId)
	assert.Len(t, c.Seized, total, "exactly once")
	assert.Empty(t, c.Items)
	assert.Equal(t, 5, c.Level)

	// A second suicide in the same round finds them already woken.
	env.run("suicide", "")
	assert.Len(t, c.Seized, total)
}

func TestDefeatScenarioLeftForDeadWoundsEveryoneAndSendsTheFoesAway(t *testing.T) {
	env := newDefeatEnv(t, 9101)
	forceScenario(t, "left-for-dead")
	foe := env.foe(9101)
	c := env.user.Character
	room := rooms.LoadRoom(2002)

	out := env.run("suicide", "")
	assert.Contains(t, out, "You wake where you fell")
	assert.Contains(t, out, "Each of you carries a lasting wound")
	assert.Equal(t, 2002, c.RoomId, "wakes where they fell")
	assert.Equal(t, 5, c.Level)
	require.Len(t, c.Wounds, 1)
	assert.False(t, c.Wounds[0].Light, "a lasting wound")
	assert.Equal(t, max(1, c.HealthMax.Value*20/100), c.Wounds[0].Points)
	mob := mobs.GetInstance(env.companion)
	require.NotNil(t, mob)
	require.Len(t, mob.Character.Wounds, 1, "the companion too")
	assert.Nil(t, mobs.GetInstance(foe.InstanceId), "the foe is gone")
	assert.NotContains(t, room.GetMobs(), foe.InstanceId)
	assert.Contains(t, room.GetMobs(rooms.FindCharmed), env.companion, "companions stay")
	assert.Equal(t, env.pack, env.packSize(), "left for dead is not a robbery")
	assert.Equal(t, 200, c.Gold)
}

// Regression (53 review): foes sent away from a room spawn used to come
// back the very next round, beside a half-health, wounded company. Any other
// hostile in the room goes with them.
func TestDefeatScenarioSentAwayFoesStayAwayForTheirRespawnTime(t *testing.T) {
	env := newDefeatEnv(t, 9101)
	forceScenario(t, "left-for-dead")
	room := rooms.LoadRoom(2002)
	saved := room.SpawnInfo
	t.Cleanup(func() { room.SpawnInfo = saved })
	room.SpawnInfo = append(append([]rooms.SpawnInfo{}, saved...), rooms.SpawnInfo{MobId: 9101, RespawnRate: "1 day"})
	room.Prepare(false)
	spawned := room.SpawnInfo[len(room.SpawnInfo)-1].InstanceId
	require.NotZero(t, spawned, "the room spawned its wolf")
	env.user.Character.KillerMobInstanceId = spawned
	bystander := mobs.NewMobById(9101, room.RoomId)
	require.NotNil(t, bystander)
	bystander.Hostile = true
	room.AddMob(bystander.InstanceId)

	env.run("suicide", "")
	require.Equal(t, 2002, env.user.Character.RoomId)
	assert.Nil(t, mobs.GetInstance(spawned), "the killer is gone")
	assert.Nil(t, mobs.GetInstance(bystander.InstanceId), "a hostile bystander is gone too")
	room.Prepare(false)
	assert.Zero(t, room.SpawnInfo[len(room.SpawnInfo)-1].InstanceId, "it does not respawn the next round")
	for _, id := range room.GetMobs() {
		if mob := mobs.GetInstance(id); mob != nil && id != env.companion {
			assert.False(t, mob.Hostile, "no hostile waits beside the woken company: %s", mob.Character.Name)
		}
	}
}

// Regression (53 review): a protected death (the death protection levels or
// perma-gear) never cost goods before scenarios, so it is never captured or
// robbed.
func TestDefeatProtectedDeathIsNeverCapturedOrRobbed(t *testing.T) {
	env := newDefeatEnv(t, 86)
	foe := env.foe(86)
	for n := 0; n < 8; n++ {
		module.roll = func(total int) int { return min(n, total-1) }
		require.True(t, module.ClaimDefeat(env.user.UserId, domain.Killer{MobID: 86, InstanceID: foe.InstanceId, Protected: true}))
		sc, ok := module.scenarioFor(env.user.Character)
		require.True(t, ok)
		assert.False(t, sc.Kind.TakesGoods(), "roll %d gave %s", n, sc.ID)
	}
	module.roll = util.Rand
	env.user.Character.SetMiscData(domain.ScenarioKey, nil)
}

func TestDefeatScenarioRobbedTakesGoldAndLooseGoodsOnly(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-robbery")
	env.foe(86)
	c := env.user.Character
	weapon := c.Equipment.Weapon

	out := env.run("suicide", "")
	assert.Contains(t, out, "40 gold is gone", "a fifth of 200")
	assert.Contains(t, out, "Taken:")
	assert.Equal(t, 2002, c.RoomId)
	assert.Equal(t, 160, c.Gold)
	assert.Equal(t, 5, c.Level)
	// A quarter of the pack, rounded up, is robbed (the table caps it at four).
	assert.Len(t, c.Items, env.pack-(env.pack*25+99)/100)
	assert.Equal(t, weapon, c.Equipment.Weapon, "equipped gear is safe")
	assert.Empty(t, c.Seized, "taken for good: nothing is held anywhere")
	assert.Empty(t, c.Wounds)
}

// A companion who fell in the fight is still on the roster, dead and
// raisable at a church, after a scenario wakes the company elsewhere.
func TestDefeatScenarioKeepsFallenCompanionsRaisable(t *testing.T) {
	env := newDefeatEnv(t, 86)
	forceScenario(t, "brigand-robbery")
	env.foe(86)
	companion := mobs.GetInstance(env.companion)
	require.NotNil(t, companion)
	companion.Character.Health = -1
	_, err := mobcommands.TryCommand("suicide", "", env.companion)
	require.NoError(t, err)
	events.ProcessEvents()
	require.Len(t, company.DeadCompanions(env.user.UserId), 1, "the companion fell in the fight")

	out := env.run("suicide", "")
	assert.Contains(t, out, "stripped of what the robbers could carry")
	assert.Contains(t, out, "Your fallen can still be raised at a church or shaman")
	dead := company.DeadCompanions(env.user.UserId)
	require.Len(t, dead, 1, "the scenario leaves the fallen on the roster")
	assert.Positive(t, dead[0].Remaining, "their rescue window still runs")
}

func TestDefeatFallsBackToTheChurchWhenNoScenarioFits(t *testing.T) {
	env := newDefeatEnv(t, 9101)
	env.foe(9101)
	// A wolf fits left-for-dead and rescued; with the table emptied the
	// church and its level loss stand.
	module.mu.Lock()
	module.cfg.scenarios = nil
	module.mu.Unlock()
	c := env.user.Character
	out := env.run("suicide", "")
	assert.Contains(t, out, "You lose a level (now level 4).")
	assert.Equal(t, 2007, c.RoomId)
	assert.Equal(t, 4, c.Level)
}

func TestDefeatKillerKindChoosesTheScenario(t *testing.T) {
	newDefeatEnv(t, 9101)
	cfg := module.config()
	zone := "Old Kings Road"
	wolfRace, wolfGroups := raceAndGroups(domain.Killer{MobID: 9101})
	assert.Equal(t, "beast", wolfRace)
	brigandRace, brigandGroups := raceAndGroups(domain.Killer{MobID: 86})
	assert.Equal(t, "human", brigandRace)
	kinds := func(race string, groups []string) map[domain.ScenarioKind]bool {
		out := map[domain.ScenarioKind]bool{}
		for _, s := range cfg.scenarios {
			if s.Fits(zone, race, groups) {
				out[s.Kind] = true
			}
		}
		return out
	}
	assert.Equal(t, map[domain.ScenarioKind]bool{domain.Rescued: true, domain.LeftForDead: true}, kinds(wolfRace, wolfGroups))
	assert.Equal(t, map[domain.ScenarioKind]bool{domain.Rescued: true, domain.Captured: true, domain.Robbed: true}, kinds(brigandRace, brigandGroups))
	assert.Equal(t, map[domain.ScenarioKind]bool{domain.Rescued: true}, kinds("", nil), "an unknown killer only rescues")
}
