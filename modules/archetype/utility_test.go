package archetype

import (
	"errors"
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// utilModule is the shipped-config module with a fixed roll and round.
func utilModule(t *testing.T, roll int) (*ArchetypeModule, *fakeStore, *uint64) {
	t.Helper()
	m, store := testModule(t)
	round := uint64(5000)
	m.roll = func() int { return roll }
	m.now = func() uint64 { return round }
	m.cast = func(*users.UserRecord, *rooms.Room, string) { t.Fatal("unexpected cast") }
	return m, store, &round
}

// trapRoom has a trapped chest (difficulty 5), a trapped north exit
// (difficulty 20) and an untrapped, unlocked south exit.
func trapRoom(t *testing.T, id int) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: id, Zone: "ArchetypeTest", Title: "Vault",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: id + 1, Lock: gamelock.Lock{Difficulty: 20, TrapBuffIds: []int{13}}},
			"south": {RoomId: id + 2},
		},
		Containers: map[string]rooms.Container{
			"chest": {Lock: gamelock.Lock{Difficulty: 5, TrapBuffIds: []int{13}}},
		},
	}
	rooms.SetTestRoom(r)
	t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	return r
}

// captureBuffs collects buff events queued while fn runs.
func captureBuffs(t *testing.T, fn func()) []events.Buff {
	t.Helper()
	events.ProcessEvents()
	var out []events.Buff
	id := events.RegisterListener(events.Buff{}, func(e events.Event) events.ListenerReturn {
		out = append(out, e.(events.Buff))
		return events.Continue
	})
	defer events.UnregisterListener(events.Buff{}, id)
	fn()
	events.ProcessEvents()
	return out
}

type rosterStub struct{ refs map[int][]survival.MemberRef }

func (r rosterStub) Roster(leader int) []survival.MemberRef { return r.refs[leader] }

// withCompanion gives leader a spawned companion (id 1) of an archetype at
// a character level, standing in roomID.
func withCompanion(t *testing.T, leader, instance, roomID, level int, archetype string) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{InstanceId: instance, Character: *characters.New()}
	mob.Character.Name = "Bran"
	mob.Character.RoomId = roomID
	mob.Character.Level = level
	mob.Character.Charm(leader, characters.CharmPermanent, "")
	mobs.SetTestInstance(mob)
	t.Cleanup(func() { mobs.RemoveTestInstance(instance) })
	survival.SetRosterProvider(rosterStub{refs: map[int][]survival.MemberRef{leader: {
		{Key: survival.LeaderMemberKey, Name: "Leader"},
		{Key: survival.CompanionMemberKey(1), Name: "Bran"},
	}}})
	company.SetFormationProvider(companyStub{leader: leader, companion: 1, instance: instance, archetype: archetype})
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
	})
	return mob
}

func TestAutoskillTogglesPersistAndRollBack(t *testing.T) {
	m, store, _ := utilModule(t, 50)
	assert.True(t, m.autoskillOn(60, "light"), "on by default")
	assert.Contains(t, m.autoskillList(60), "light")

	assert.Contains(t, m.setAutoskill(60, "light", false), "off")
	assert.False(t, m.autoskillOn(60, "light"))
	assert.False(t, store.saved.Autoskill[60]["light"])
	assert.True(t, m.autoskillOn(60, "traps"), "other utilities unaffected")

	assert.Contains(t, m.setAutoskill(60, "juggling", true), "no automatic skill")

	store.saveErr = errors.New("disk full")
	assert.Contains(t, m.setAutoskill(60, "light", true), "disk full")
	assert.False(t, m.autoskillOn(60, "light"), "rolled back")
	assert.Contains(t, m.setAutoskill(60, "traps", false), "disk full")
	assert.True(t, m.autoskillOn(60, "traps"), "rolled back to the default")
}

func TestAutoskillCommand(t *testing.T) {
	m, _, _ := utilModule(t, 50)
	u := newUser(61)
	_, err := m.autoskillCommand("traps off", u, nil, 0)
	require.NoError(t, err)
	assert.False(t, m.autoskillOn(61, "traps"))
}

func TestTrapSenseNeedsSomeoneWhoKnowsTraps(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	room := trapRoom(t, 97000)
	u := newUser(62)
	assert.Contains(t, m.sense(u, room, ""), "Nobody in your company")
	m.choose(u, "wizard", true)
	assert.Contains(t, m.sense(u, room, ""), "Nobody in your company", "wizards don't do traps")
}

func TestTrapSenseRollsPerLock(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	room := trapRoom(t, 97010)
	u := newUser(63)
	m.choose(u, "rogue", true)

	// Level 1: score 25 (+0 perception). Chest (5): 125-75 >= 0. North
	// (20): 125-150 < 0.
	text := m.sense(u, room, "")
	assert.Contains(t, text, "You find a trap on")
	assert.Contains(t, text, "chest")
	assert.NotContains(t, text, "north")

	assert.Contains(t, m.sense(u, room, ""), "wait", "cooldown")
}

func TestTrapSenseFailureAndTargets(t *testing.T) {
	m, _, _ := utilModule(t, 1)
	room := trapRoom(t, 97020)
	for i, tc := range []struct{ target, want string }{
		{"", "You find no traps."},
		{"chest", "You find no traps."},
		{"south", "You find no traps."},
		{"altar", "no such exit or container"},
	} {
		u := newUser(64 + i)
		m.choose(u, "rogue", true)
		assert.Contains(t, m.sense(u, room, tc.target), tc.want, "target %q", tc.target)
	}
}

func TestTrapDisarmSuccessPersistsAndExpires(t *testing.T) {
	m, store, round := utilModule(t, 100)
	room := trapRoom(t, 97030)
	u := newUser(70)
	m.choose(u, "rogue", true)

	text := m.disarm(u, room, "chest")
	assert.Contains(t, text, "You carefully disarm the trap")
	assert.Equal(t, uint64(5900), store.saved.Disarmed["97030-chest"])
	assert.False(t, m.TrapArmed("97030-chest"))
	assert.True(t, m.TrapArmed("97030-north"))
	assert.Contains(t, m.disarm(newRogue(t, m, 71), room, "chest"), "already disarmed")

	// A reload before expiry keeps it disarmed.
	reloaded := newModule()
	reloaded.store = store
	reloaded.now = func() uint64 { return 5100 }
	reloaded.load()
	assert.False(t, reloaded.TrapArmed("97030-chest"))

	// It re-arms at the expiry round.
	*round = 5900
	assert.True(t, m.TrapArmed("97030-chest"))
	assert.Empty(t, m.armedLocks(nil), "")
	_, still := m.registry.Disarmed["97030-chest"]
	assert.False(t, still, "expired disarms are pruned")

	// A reload after expiry prunes it at load.
	late := newModule()
	late.store = store
	late.now = func() uint64 { return 9999 }
	late.load()
	assert.Empty(t, late.registry.Disarmed)
}

func newRogue(t *testing.T, m *ArchetypeModule, id int) *users.UserRecord {
	t.Helper()
	u := newUser(id)
	m.choose(u, "rogue", true)
	return u
}

func TestTrapDisarmRefusals(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	room := trapRoom(t, 97040)
	assert.Contains(t, m.disarm(newRogue(t, m, 72), room, ""), "Usage")
	assert.Contains(t, m.disarm(newRogue(t, m, 73), room, "altar"), "no such")
	assert.Contains(t, m.disarm(newRogue(t, m, 74), room, "south"), "nothing to disarm")
	assert.Contains(t, m.disarm(newUser(75), room, "chest"), "Nobody in your company")
}

func TestTrapDisarmPlainFailure(t *testing.T) {
	m, store, _ := utilModule(t, 40) // 25+40-75 = -10: a miss inside the margin
	room := trapRoom(t, 97050)
	u := newRogue(t, m, 76)
	var text string
	buffs := captureBuffs(t, func() { text = m.disarm(u, room, "chest") })
	assert.Contains(t, text, "fail to disarm")
	assert.Empty(t, buffs)
	assert.True(t, m.TrapArmed("97050-chest"))
	assert.Empty(t, store.saved.Disarmed)
}

func TestTrapDisarmBackfireSpringsTrapOnPerformer(t *testing.T) {
	m, _, _ := utilModule(t, 1) // 25+1-75 = -49 < -25
	room := trapRoom(t, 97060)
	u := newRogue(t, m, 77)
	var text string
	buffs := captureBuffs(t, func() { text = m.disarm(u, room, "chest") })
	assert.Contains(t, text, "spring the trap")
	require.Len(t, buffs, 1)
	assert.Equal(t, 77, buffs[0].UserId)
	assert.Equal(t, 13, buffs[0].BuffId)
	assert.True(t, m.TrapArmed("97060-chest"))
}

func TestTrapDisarmSaveFailureRollsBack(t *testing.T) {
	m, store, _ := utilModule(t, 100)
	room := trapRoom(t, 97070)
	u := newRogue(t, m, 78)
	store.saveErr = errors.New("disk full")
	assert.Contains(t, m.disarm(u, room, "chest"), "disk full")
	assert.True(t, m.TrapArmed("97070-chest"))
}

func TestCompanionRogueIsBestMember(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	room := trapRoom(t, 97080)
	u := newUser(79)
	m.choose(u, "warrior", true)
	withCompanion(t, 79, 97801, 97080, 12, "rogue") // level 12 -> utility level 2

	best, ok := m.bestMember(u, utilityTraps, 97080)
	require.True(t, ok)
	assert.Equal(t, 1, best.CompanionID)
	assert.Equal(t, 2, best.Level)

	text := m.sense(u, room, "")
	assert.Contains(t, text, "Bran finds a trap on")

	// A companion elsewhere doesn't help.
	_, ok = m.bestMember(u, utilityTraps, 1)
	assert.False(t, ok)
}

func TestCompanionBackfireHitsCompanion(t *testing.T) {
	m, _, _ := utilModule(t, 1)
	room := trapRoom(t, 97090)
	u := newUser(80)
	withCompanion(t, 80, 97901, 97090, 1, "rogue")
	var text string
	buffs := captureBuffs(t, func() { text = m.disarm(u, room, "chest") })
	require.Len(t, buffs, 1)
	assert.Equal(t, 97901, buffs[0].MobInstanceId)
	assert.Contains(t, text, "Bran fumbles and springs the trap", "review 17b finding 4: verb agreement")
}

func TestTrapSeamUsesModule(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	archetypes.SetProvider(m)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	room := trapRoom(t, 97100)
	m.disarm(newRogue(t, m, 81), room, "chest")
	assert.False(t, archetypes.TrapArmed("97100-chest"))
	assert.True(t, archetypes.TrapArmed("97100-north"))
}

func TestTrappedLocksListsOnlyLockedTrappedOnes(t *testing.T) {
	room := trapRoom(t, 97110)
	ids := []string{}
	for _, l := range trappedLocks(room) {
		ids = append(ids, l.ID)
	}
	sort.Strings(ids)
	assert.Equal(t, []string{"97110-chest", "97110-north"}, ids)
}

// Review 17b finding 5: a mistyped target doesn't spend the sense cooldown;
// disarm keeps its own cooldown.
func TestTrapCooldowns(t *testing.T) {
	m, _, _ := utilModule(t, 1)
	room := trapRoom(t, 97120)
	u := newRogue(t, m, 82)
	assert.Contains(t, m.sense(u, room, "altar"), "no such")
	assert.Contains(t, m.sense(u, room, ""), "You find no traps.", "the typo cost no cooldown")
	assert.Contains(t, m.sense(u, room, ""), "wait")

	m.roll = func() int { return 40 } // a plain miss
	assert.Contains(t, m.disarm(u, room, "chest"), "fail to disarm")
	assert.Contains(t, m.disarm(u, room, "chest"), "wait", "disarm has its own cooldown")
}

// Review 17b finding 2: a downed companion can't act.
func TestDownedCompanionIsNotAMember(t *testing.T) {
	m, _, _ := utilModule(t, 100)
	room := trapRoom(t, 97130)
	u := newUser(83)
	bran := withCompanion(t, 83, 97930, 97130, 12, "rogue")
	bran.Character.Health = 0
	assert.Contains(t, m.sense(u, room, ""), "Nobody in your company")
	assert.Contains(t, m.disarm(u, room, "chest"), "Nobody in your company")
}

// Review 17b finding 10: permadeath clears the autoskill toggles too.
func TestPermadeathClearsToggles(t *testing.T) {
	m, store, _ := utilModule(t, 50)
	m.setAutoskill(84, "light", false)
	m.onPlayerDeath(events.PlayerDeath{UserId: 84, Permanent: true})
	assert.True(t, m.autoskillOn(84, "light"))
	_, persisted := store.saved.Autoskill[84]
	assert.False(t, persisted)
}
