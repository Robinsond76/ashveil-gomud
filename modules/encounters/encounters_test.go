package encounters

import (
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests go through the real user Go command and the real walking
// step seam, the real room manager and users, and the real spawn and
// death paths. Only the dice and the clock are scripted.

const (
	zoneName = "EncWood"
	startRm  = 96001
	woodRm   = 96002 // enabled, table "woods", the zone's chance
	deadRm   = 96003 // enabled with chance 0
	bareRm   = 96004 // no encounter block
	lairRm   = 96005 // enabled, boss table
	offRm    = 96006 // enabled: false
	userID   = 96701
)

var aliasesOnce sync.Once

func loadAliases(t *testing.T) {
	t.Helper()
	aliasesOnce.Do(func() {
		_, thisFile, _, _ := runtime.Caller(0)
		dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		keywords.LoadAliases()
	})
}

type world struct {
	m    *EncountersModule
	user *users.UserRecord
	now  time.Time
	fail func(roll int) // the next entry-chance roll's value; default 0 = spring
	roll int
}

func template(id int, name string, mutate func(*mobs.Mob)) {
	m := &mobs.Mob{MobId: mobs.MobId(id), Hostile: true}
	m.Character = *characters.New()
	m.Character.Name = name
	m.Character.Level = 1
	m.Character.Gold = 0
	if mutate != nil {
		mutate(m)
	}
	mobs.SetTestSpec(m)
}

func setup(t *testing.T) *world {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	loadAliases(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	battle.Reset()
	t.Cleanup(battle.Reset)
	t.Cleanup(parties.UseMemoryForTest())

	rooms.SetTestBiome(&rooms.BiomeInfo{BiomeId: "forest", Name: "Forest", Symbol: "f", LitArea: true})
	t.Cleanup(func() { rooms.RemoveTestBiome("forest") })
	template(96101, "wood wolf", nil)
	template(96102, "wood spider", nil)
	template(96103, "wood hexer", func(m *mobs.Mob) { m.Role = "healer" })
	template(96104, "wood ogre", func(m *mobs.Mob) { m.Solitary = true })
	t.Cleanup(func() {
		for _, id := range []mobs.MobId{96101, 96102, 96103, 96104} {
			mobs.RemoveTestSpec(id)
		}
	})
	chance0 := 0
	zone := &rooms.ZoneConfig{Name: zoneName, Encounters: encounters.ZoneConfig{
		Band: encounters.Band{Low: 8, High: 10},
		Tables: map[string][]encounters.Composition{
			"woods": {{ID: "wolves", Weight: 1, Text: "Wolves!", Members: []encounters.Member{{MobID: 96101, Count: 3}}}},
			"lair":  {{ID: "ogre-court", Boss: true, Weight: 1, Members: []encounters.Member{{MobID: 96104, Count: 1}, {MobID: 96101, Count: 2}}}},
		},
	}}
	t.Cleanup(rooms.SetTestZoneConfig(zone))
	mk := func(id int, setting *encounters.RoomSetting, exits map[string]exit.RoomExit) {
		r := &rooms.Room{RoomId: id, Zone: zoneName, Title: "Room", Biome: "forest", Exits: exits, Encounter: setting}
		rooms.SetTestRoom(r)
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	back := map[string]exit.RoomExit{"south": {RoomId: startRm}}
	mk(startRm, nil, map[string]exit.RoomExit{"north": {RoomId: woodRm}, "east": {RoomId: deadRm}, "west": {RoomId: bareRm}, "up": {RoomId: lairRm}, "down": {RoomId: offRm}})
	mk(woodRm, &encounters.RoomSetting{Enabled: true, Table: "woods"}, back)
	mk(deadRm, &encounters.RoomSetting{Enabled: true, Table: "woods", Chance: &chance0}, back)
	mk(bareRm, nil, back)
	mk(lairRm, &encounters.RoomSetting{Enabled: true, Table: "lair"}, back)
	mk(offRm, &encounters.RoomSetting{Enabled: false, Table: "woods"}, back)

	w := &world{now: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)}
	w.user = users.NewUserRecord(userID, userID)
	w.user.Character.Name = "Hero"
	w.user.Character.RoomId = startRm
	w.user.Character.ActionPoints = 1000
	w.user.Character.Health = 50
	w.user.Character.Validate()
	users.SetTestUser(w.user)

	w.m = newModule()
	w.m.clock = func() time.Time { return w.now }
	w.m.rng = func(n int) int { return w.roll % n } // 0 springs a chance roll; the weighted picks have one entry
	w.m.graces[userID] = encounters.Grace{}         // past grace, unless a test says otherwise
	walking.SuspendListeners(t)                     // the init() module's listeners use real dice
	remove := walking.AddStepListener(w.m.onStep)
	t.Cleanup(remove)
	removeArrival := walking.AddArrivalListener(w.m.onArrival)
	t.Cleanup(removeArrival)
	t.Cleanup(func() {
		for _, r := range w.m.active {
			r.Remove()
		}
	})
	return w
}

func (w *world) walk(t *testing.T, dir string) {
	t.Helper()
	from := w.user.Character.RoomId
	handled, err := usercommands.Go(dir, w.user, rooms.LoadRoom(from), 0)
	require.NoError(t, err)
	require.True(t, handled)
}

func (w *world) back(t *testing.T) {
	t.Helper()
	w.user.Character.RoomId = startRm
	w.user.Character.Aggro = nil
}

func TestStepIntoAnEnabledRoomSpringsAnEncounterOnTheCompany(t *testing.T) {
	w := setup(t)
	w.walk(t, "north")
	require.Equal(t, woodRm, w.user.Character.RoomId)
	require.Len(t, w.m.active, 1)
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	assert.Equal(t, userID, rec.Owner)
	require.Len(t, rec.Foes, 3)
	for _, id := range rec.Foes {
		foe := mobs.GetInstance(id)
		require.NotNil(t, foe)
		assert.Equal(t, userID, foe.EncounterOwner, "reserved for the company it sprang on")
		assert.Equal(t, rec.ID, foe.SpawnGroup)
		assert.True(t, foe.Hostile)
		assert.Zero(t, foe.MaxWander)
		assert.GreaterOrEqual(t, foe.Character.Level, 8, "levels come from the zone's band")
		assert.LessOrEqual(t, foe.Character.Level, 9, "2-3 foes top out one under the band's top")
		assert.False(t, foe.Boss)
		assert.Zero(t, foe.AmbushAdvantage, "a sudden appearance: neither side loses its opening round")
	}
	assert.Equal(t, 3, rec.standing())
	assert.Len(t, rooms.LoadRoom(woodRm).GetMobs(), 3, "they stand in the actual room")
}

func TestRoomsThatShouldNotRollNeverDo(t *testing.T) {
	w := setup(t)
	for _, dir := range []string{"east", "west", "down"} { // chance 0, no block, enabled: false
		for i := 0; i < 20; i++ {
			w.back(t)
			w.walk(t, dir)
		}
	}
	assert.Empty(t, w.m.active)

	// The zone has tables, but a room with no setting stays safe even at 100%.
	w.back(t)
	w.walk(t, "west")
	assert.Empty(t, w.m.active)
}

func TestOnlyOrdinaryStepsAndJourneyArrivalsRoll(t *testing.T) {
	w := setup(t)
	// Teleport, relocation and login are the room manager's, never a step.
	require.NoError(t, rooms.MoveToRoom(userID, woodRm))
	assert.Empty(t, w.m.active, "an admin or scripted relocation does not roll")
	// A look or scout in place does not either.
	_, _ = usercommands.Look("", w.user, rooms.LoadRoom(woodRm), 0)
	assert.Empty(t, w.m.active)
	// A journey's arrival does.
	w.back(t)
	require.NoError(t, rooms.MoveToRoom(userID, woodRm))
	walking.Arrived(userID, startRm, woodRm)
	assert.Len(t, w.m.active, 1, "a completed journey's arrival rolls like a step")
}

func TestGraceAfterABattleAndForFreshCharacters(t *testing.T) {
	w := setup(t)
	delete(w.m.graces, userID) // a fresh character
	w.walk(t, "north")
	assert.Empty(t, w.m.active, "first entry skipped")
	w.back(t)
	w.walk(t, "north")
	assert.Empty(t, w.m.active, "second entry skipped")
	w.back(t)
	w.walk(t, "north")
	assert.Empty(t, w.m.active, "two entries are not enough: 30 seconds have not passed")
	w.now = w.now.Add(31 * time.Second)
	w.back(t)
	w.walk(t, "north")
	assert.Len(t, w.m.active, 1, "both conditions met")
}

func TestBattleEndStartsASavedGrace(t *testing.T) {
	w := setup(t)
	store := &memStore{}
	w.m.store = store
	w.m.onBattleEnded(events.BattleEnded{UserId: userID, Outcome: "victory"})
	assert.Equal(t, encounters.GraceEntries, w.m.graces[userID].Entries)
	require.NotNil(t, store.saved)
	assert.Equal(t, encounters.NewGrace(w.now), store.saved.Graces[userID], "saved with the leader")

	// A restart reads it back, so crossing a zone or reconnecting cannot clear it.
	reborn := newModule()
	reborn.store = store
	reborn.load()
	assert.Equal(t, store.saved.Graces[userID], reborn.graces[userID])

	w.m.graces = reborn.graces
	w.walk(t, "north")
	assert.Empty(t, w.m.active, "the saved grace still suppresses")
}

func TestPurgeForgetsGraceAndClearsTheGroup(t *testing.T) {
	w := setup(t)
	w.walk(t, "north")
	require.Len(t, w.m.active, 1)
	foe := ""
	for id, r := range w.m.active {
		foe = id
		require.NotNil(t, mobs.GetInstance(r.Foes[0]))
	}
	store := &memStore{}
	w.m.store = store
	w.m.onUserPurged(events.UserPurged{UserId: userID})
	assert.Empty(t, w.m.active)
	assert.NotContains(t, w.m.graces, userID)
	assert.NotContains(t, store.saved.Graces, userID)
	assert.NotContains(t, w.m.active, foe)
}

func TestNoEncounterWhileFightingOrHoldingAGroup(t *testing.T) {
	w := setup(t)
	w.user.Character.Aggro = &characters.Aggro{MobInstanceId: 1}
	w.m.Entered(userID, startRm) // not in an enabled room anyway
	w.user.Character.RoomId = woodRm
	w.m.Entered(userID, woodRm)
	assert.Empty(t, w.m.active, "in a fight")
	w.user.Character.Aggro = nil
	w.user.Character.Health = 0
	w.m.Entered(userID, woodRm)
	assert.Empty(t, w.m.active, "downed")
	w.user.Character.Health = 50
	w.m.Entered(userID, woodRm)
	require.Len(t, w.m.active, 1)
	w.m.graces[userID] = encounters.Grace{}
	w.m.Entered(userID, woodRm)
	assert.Len(t, w.m.active, 1, "one unresolved group per leader")
}

func TestAPartyFollowerDoesNotRollTheLeaderDoes(t *testing.T) {
	w := setup(t)
	other := users.NewUserRecord(userID+1, userID+1)
	other.Character.Name = "Follower"
	other.Character.Health = 50
	other.Character.RoomId = woodRm
	users.SetTestUser(other)
	p := parties.New(userID)
	p.InvitePlayer(userID + 1)
	p.AcceptInvite(userID + 1)
	w.m.graces[userID+1] = encounters.Grace{}
	w.m.Entered(userID+1, woodRm)
	assert.Empty(t, w.m.active, "the follower's step is the leader's move")
	w.user.Character.RoomId = woodRm
	w.m.Entered(userID, woodRm)
	assert.Len(t, w.m.active, 1)
}

func TestRoomCapacity(t *testing.T) {
	w := setup(t)
	for i := 0; i < encounters.RoomGroupLimit; i++ {
		id := userID + 10 + i
		u := users.NewUserRecord(id, uint64(id))
		u.Character.Name = "Other"
		u.Character.Health = 50
		u.Character.RoomId = woodRm
		users.SetTestUser(u)
		w.m.graces[id] = encounters.Grace{}
		w.m.Entered(id, woodRm)
	}
	require.Len(t, w.m.active, encounters.RoomGroupLimit)
	w.user.Character.RoomId = woodRm
	w.m.Entered(userID, woodRm)
	assert.Len(t, w.m.active, encounters.RoomGroupLimit, "a full room skips the encounter without stockpiling one")
}

func TestBossGroupShape(t *testing.T) {
	w := setup(t)
	w.walk(t, "up")
	require.Len(t, w.m.active, 1)
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	require.True(t, rec.Boss)
	require.Len(t, rec.Foes, 3)
	boss := mobs.GetInstance(rec.Foes[0])
	assert.True(t, boss.Boss, "flagged so a hex resists it (38a)")
	assert.Equal(t, 8+encounters.BossLevelBonus, boss.Character.Level, "over the band's low by the boss bonus")
	assert.False(t, boss.Solitary, "the lone template leads a group here")
	ordinary := mobs.NewMobById(96101, woodRm, 8+encounters.BossLevelBonus)
	t.Cleanup(func() { mobs.DestroyInstance(ordinary.InstanceId) })
	assert.InDelta(t, float64(ordinary.Character.HealthMax.Value)*(1+encounters.BossHPBonus), float64(boss.Character.HealthMax.Value), 2, "the boss bonus over the HP of a foe of its level")
	assert.Equal(t, boss.Character.HealthMax.Value, boss.Character.Health)
	for _, id := range rec.Foes {
		foe := mobs.GetInstance(id)
		assert.Equal(t, 1, foe.Coordination, "a boss group runs no strategy (Rabble)")
		assert.True(t, foe.EncounterBoss)
		if foe != boss {
			assert.Equal(t, 8, foe.Character.Level, "escorts at the band's low")
			assert.False(t, foe.Boss)
		}
	}
}

func TestSpawnRollsBackWhenATemplateWillNotBuild(t *testing.T) {
	setup(t)
	before := len(rooms.LoadRoom(woodRm).GetMobs())
	_, err := enemyparty.SpawnEncounter(woodRm, userID, []encounters.Foe{{MobID: 96101, Level: 8}, {MobID: 99999, Level: 8}})
	require.Error(t, err)
	assert.Len(t, rooms.LoadRoom(woodRm).GetMobs(), before, "no half group is left in the room")
}

func TestOnlyTheCompanyAndItsAlliesMayFightIt(t *testing.T) {
	w := setup(t)
	w.walk(t, "north")
	require.Len(t, w.m.active, 1)
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	stranger := users.NewUserRecord(userID+50, userID+50)
	stranger.Character.Name = "Stranger"
	stranger.Character.Health = 50
	stranger.Character.RoomId = woodRm
	stranger.Character.ActionPoints = 100
	users.SetTestUser(stranger)
	room := rooms.LoadRoom(woodRm)
	room.SetTestOccupants([]int{userID, userID + 50}, rec.Foes)

	foe := mobs.GetInstance(rec.Foes[0])
	_, err := usercommands.Attack("#"+itoa(foe.InstanceId), stranger, room, 0)
	require.NoError(t, err)
	assert.Nil(t, stranger.Character.Aggro, "a nonparticipant cannot start the fight")

	// Its foes do not go looking for the stranger either.
	_, err = mobcommands.LookForTrouble("", foe, room)
	require.NoError(t, err)
	if foe.Character.Aggro != nil {
		assert.Equal(t, userID, foe.Character.Aggro.UserId, "it seeks only its company")
	}

	// An accepted party member may.
	p := parties.New(userID)
	p.InvitePlayer(userID + 50)
	p.AcceptInvite(userID + 50)
	assert.False(t, parties.ReservedFrom(foe.EncounterOwner, userID+50))
}

func TestFledOrAbandonedGroupsAreClearedOnceAndWonOnesRetired(t *testing.T) {
	w := setup(t)
	w.walk(t, "north")
	require.Len(t, w.m.active, 1)
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	w.m.settleLocked(false)
	assert.Len(t, w.m.active, 1, "the owner stands with the group: nothing is cleared")

	// The owner flees: a battle ending clears the survivors at once.
	w.user.Character.RoomId = startRm
	w.m.settleLocked(false)
	assert.Len(t, w.m.active, 1, "left alone, the timeout has not run")
	w.now = w.now.Add((encounters.AbandonSeconds + 1) * time.Second)
	w.m.settleLocked(false)
	assert.Empty(t, w.m.active, "an ownerless group expires")
	for _, id := range rec.Foes {
		assert.Nil(t, mobs.GetInstance(id), "gone from the world")
	}
	assert.Empty(t, rooms.LoadRoom(woodRm).GetMobs())

	// A flee clears at once.
	w.m.graces[userID] = encounters.Grace{}
	w.user.Character.RoomId = woodRm
	w.m.Entered(userID, woodRm)
	require.Len(t, w.m.active, 1)
	w.user.Character.RoomId = startRm
	w.m.settleLocked(true)
	assert.Empty(t, w.m.active)

	// A won group is retired without touching anything.
	w.m.graces[userID] = encounters.Grace{}
	w.user.Character.RoomId = woodRm
	w.m.Entered(userID, woodRm)
	require.Len(t, w.m.active, 1)
	for _, r := range w.m.active {
		for _, id := range r.Foes {
			mobs.GetInstance(id).Character.Health = 0
		}
	}
	w.m.settleLocked(false)
	assert.Empty(t, w.m.active)
}

func TestSurvivorsStayWhileAnyBattleStillInvolvesThem(t *testing.T) {
	w := setup(t)
	w.walk(t, "north")
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	// The owner is gone, but an ally's battle still fights a foe.
	w.user.Character.RoomId = startRm
	battle.Begin(userID+99, woodRm, 1, "g", []int{rec.Foes[0]})
	w.now = w.now.Add(10 * time.Minute)
	w.m.settleLocked(true)
	assert.Len(t, w.m.active, 1, "never deleted while an ally is still fighting")
	battle.Reset()
	w.m.settleLocked(true)
	assert.Empty(t, w.m.active)
}

func TestShippedEncounterContentIsValid(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	races.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	mobs.LoadDataFiles()
	for _, zone := range []string{"Dark Forest", "Catacombs"} {
		cfg := rooms.GetZoneConfig(zone)
		require.NotNil(t, cfg, zone)
		valid, diags := cfg.Encounters.Validate(lookup)
		assert.Empty(t, diags, zone)
		assert.True(t, valid.Band.Valid(), zone)
		assert.NotEmpty(t, valid.Tables, zone)
		for name, table := range cfg.Encounters.Tables {
			assert.Len(t, valid.Tables[name], len(table), "%s %s: nothing was disabled", zone, name)
		}
		enabled := 0
		for id := range cfg.RoomIds {
			room := rooms.LoadRoom(id)
			if room == nil || room.Encounter == nil {
				continue
			}
			enabled++
			assert.True(t, room.Encounter.Enabled)
			assert.NotEmpty(t, valid.Tables[room.Encounter.Table], "room %d names a real table", id)
		}
		assert.Positive(t, enabled, "%s has eligible rooms", zone)
	}
	// Phase 37: bosses are flagged (38a's hex resist reads it) and carry the
	// telegraphed threat the boss shape calls for (35d).
	for _, id := range []int{14, 25, 34, 37, 85} { // lich, abyssal creeper, ent, spider queen, forest ogre
		spec := mobs.GetMobSpec(mobs.MobId(id))
		require.NotNil(t, spec, id)
		assert.True(t, spec.Boss, spec.Character.Name)
	}
	for _, id := range []int{14, 85} { // the pilot lairs' bosses
		assert.Positive(t, mobs.GetMobSpec(mobs.MobId(id)).WindUps["crushing-blow"], "boss %d telegraphs a threat", id)
	}
}

type memStore struct{ saved *Registry }

func (s *memStore) Load(r *Registry) error {
	if s.saved == nil {
		*r = Registry{Graces: map[int]encounters.Grace{}, Bosses: map[int]map[string]time.Time{}}
		return nil
	}
	*r = copyRegistry(*s.saved)
	return nil
}

func (s *memStore) Save(r Registry) error {
	cp := copyRegistry(r)
	s.saved = &cp
	return nil
}

func copyRegistry(r Registry) Registry {
	cp := Registry{Graces: map[int]encounters.Grace{}, Bosses: map[int]map[string]time.Time{}}
	for k, v := range r.Graces {
		cp.Graces[k] = v
	}
	for k, comps := range r.Bosses {
		cp.Bosses[k] = map[string]time.Time{}
		for id, t := range comps {
			cp.Bosses[k][id] = t
		}
	}
	return cp
}

func itoa(n int) string { return strconv.Itoa(n) }

type countingStore struct {
	memStore
	saves int
}

func (s *countingStore) Save(r Registry) error {
	s.saves++
	return s.memStore.Save(r)
}

func TestEntriesSaveTheGraceOnlyWhenItChanges(t *testing.T) {
	w := setup(t)
	store := &countingStore{}
	w.m.store = store
	w.roll = 99 // never springs, so no battle ends to reset the grace
	for i := 0; i < 5; i++ {
		w.back(t)
		w.walk(t, "north")
	}
	assert.Zero(t, store.saves, "a spent grace consumes nothing, so ordinary steps write nothing")

	w.m.graces[userID] = encounters.NewGrace(w.now)
	w.back(t)
	w.walk(t, "north")
	assert.Equal(t, 1, store.saves, "consuming a grace entry is saved")
}

// Phase 40a2: a company working in place rolls at the room's chance plus a
// bonus, and learns whether a group sprang.
func TestAttemptAddsTheBonusAndReportsTheResult(t *testing.T) {
	w := setup(t)
	w.user.Character.RoomId = deadRm
	w.roll = 5 // a 5 in 100 springs only a chance above 5
	assert.False(t, w.m.Attempt(userID, deadRm, 0), "a 0% room never springs")
	assert.Empty(t, w.m.active)
	assert.True(t, w.m.Attempt(userID, deadRm, 10), "the bonus lifts 0% to 10%")
	assert.Len(t, w.m.active, 1)
	assert.False(t, w.m.Attempt(userID, deadRm, 100), "one unresolved group per leader")

	w.m.active = map[string]*record{}
	w.roll = 99
	assert.True(t, w.m.Attempt(userID, deadRm, 500), "the chance is capped at 100%")
	w.user.Character.RoomId = startRm
	assert.False(t, w.m.Attempt(userID, deadRm, 100), "the party must be in the room")
}

// Phase 37b: a company that beats the boss finds its lair quiet for
// BossRespawnSeconds of real time, saved with the leader, and the lair
// springs again after. A fled fight starts no cooldown.
func TestBossLairStaysQuietAfterTheBossFalls(t *testing.T) {
	w := setup(t)
	store := &memStore{}
	w.m.store = store
	w.walk(t, "up")
	require.Len(t, w.m.active, 1)
	var rec *record
	for _, r := range w.m.active {
		rec = r
	}
	// Fleeing leaves the boss standing: no cooldown, the lair may roll again.
	w.user.Character.RoomId = startRm
	w.m.settleLocked(true)
	assert.Empty(t, w.m.active)
	assert.Empty(t, w.m.bosses[userID], "a fled boss fight starts no cooldown")
	w.m.graces[userID] = encounters.Grace{}
	w.user.Character.RoomId = lairRm
	w.m.Entered(userID, lairRm)
	require.Len(t, w.m.active, 1, "the boss is still there to meet again")
	for _, r := range w.m.active {
		rec = r
	}

	// The boss falls (its escorts may stand): the lair goes quiet.
	boss := mobs.GetInstance(rec.Foes[0])
	boss.Character.Health = 0
	w.m.settleLocked(false)
	ready, ok := w.m.bosses[userID]["ogre-court"]
	require.True(t, ok)
	assert.Equal(t, w.now.Add(encounters.BossRespawnSeconds*time.Second), ready)
	require.NotNil(t, store.saved)
	assert.Equal(t, ready, store.saved.Bosses[userID]["ogre-court"], "saved with the leader")
	w.user.Character.RoomId = startRm // the escorts are cleared once the company leaves
	w.m.settleLocked(true)
	assert.Empty(t, w.m.active)

	// The lair is quiet however often the company comes back.
	for i := 0; i < 10; i++ {
		w.m.graces[userID] = encounters.Grace{}
		w.back(t)
		w.walk(t, "up")
	}
	assert.Empty(t, w.m.active, "no boss farming inside the cooldown")
	// Ordinary rooms are unaffected.
	w.back(t)
	w.m.graces[userID] = encounters.Grace{}
	w.walk(t, "north")
	assert.Len(t, w.m.active, 1)
	for _, r := range w.m.active {
		r.Remove()
	}
	w.m.active = map[string]*record{}

	// A restart keeps the cooldown, and it expires on the real clock.
	reborn := newModule()
	reborn.store = store
	reborn.clock = w.m.clock
	reborn.load()
	assert.Positive(t, reborn.bossCoolingLocked(userID, "ogre-court"))
	w.now = w.now.Add((encounters.BossRespawnSeconds + 1) * time.Second)
	assert.Zero(t, reborn.bossCoolingLocked(userID, "ogre-court"))
	w.back(t)
	w.m.graces[userID] = encounters.Grace{}
	w.walk(t, "up")
	require.Len(t, w.m.active, 1, "the lair wakes again")
}

func TestBossCooldownCoversThePartyAndPurgeForgetsIt(t *testing.T) {
	w := setup(t)
	other := users.NewUserRecord(userID+1, userID+1)
	other.Character.Name = "Follower"
	other.Character.Health = 50
	other.Character.RoomId = startRm
	users.SetTestUser(other)
	p := parties.New(userID)
	p.InvitePlayer(userID + 1)
	p.AcceptInvite(userID + 1)
	w.walk(t, "up")
	require.Len(t, w.m.active, 1)
	for _, r := range w.m.active {
		mobs.GetInstance(r.Foes[0]).Character.Health = 0
	}
	w.m.settleLocked(false)
	assert.Positive(t, w.m.bossCoolingLocked(userID, "ogre-court"))
	assert.Positive(t, w.m.bossCoolingLocked(userID+1, "ogre-court"), "a party cannot rotate its leader to farm the boss")
	assert.Equal(t, encounters.BossRespawnSeconds*time.Second, w.m.LairQuiet(userID+1, lairRm), "look and scout can say how long")
	assert.Zero(t, w.m.LairQuiet(userID+1, startRm), "an ordinary room has no lair to be quiet")
	p.Disband()
	delete(w.m.bosses, userID+1)
	w.m.onUserPurged(events.UserPurged{UserId: userID})
	assert.Zero(t, w.m.bossCoolingLocked(userID, "ogre-court"))
}

// 37b review: a member who beat the boss keeps the lair quiet for any party
// they join, so handing the lead to a fresh character cannot farm it.
func TestBossCooldownFollowsTheKillerIntoANewParty(t *testing.T) {
	w := setup(t)
	fresh := users.NewUserRecord(userID+1, userID+1)
	fresh.Character.Name = "Fresh"
	fresh.Character.Health = 50
	fresh.Character.RoomId = startRm
	users.SetTestUser(fresh)
	w.walk(t, "up")
	require.Len(t, w.m.active, 1)
	for _, r := range w.m.active {
		mobs.GetInstance(r.Foes[0]).Character.Health = 0
	}
	w.m.settleLocked(false)
	w.user.Character.RoomId = startRm
	w.m.settleLocked(true)
	require.Empty(t, w.m.active)
	require.Zero(t, w.m.bossCoolingLocked(userID+1, "ogre-court"), "the fresh character never fought it")

	p := parties.New(userID + 1)
	p.InvitePlayer(userID)
	p.AcceptInvite(userID)
	require.True(t, p.IsLeader(userID+1))
	assert.Positive(t, w.m.bossCoolingLocked(userID+1, "ogre-court"), "the killer's quiet follows them")
	for i := 0; i < 10; i++ {
		w.m.graces[userID+1] = encounters.Grace{}
		fresh.Character.RoomId = lairRm
		w.m.Entered(userID+1, lairRm)
	}
	assert.Empty(t, w.m.active, "the new leader cannot wake the lair while the killer rides along")
}
