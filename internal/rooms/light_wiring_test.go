package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	testTorchBuff      = 9001
	testPartyLightBuff = 9002
	testNightVisBuff   = 9003
)

// setupLightWiring registers real buff specs for each light flag and a dark
// cave biome, so visibility runs through the same Buffs/users/mobs/parties
// lookups the game uses.
func setupLightWiring(t *testing.T) {
	t.Helper()
	mudlog.SetupLogger(nil, "low", "", false)
	for _, flag := range []string{FlagLightSource, FlagPartyLight, FlagNightVision} {
		buffs.SetTestFlag(flag)
	}
	specs := map[int]string{testTorchBuff: FlagLightSource, testPartyLightBuff: FlagPartyLight, testNightVisBuff: FlagNightVision}
	for id, flag := range specs {
		buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: id, Name: flag, TriggerCount: 1, Flags: []string{flag}})
	}
	saved := biomes
	biomes = map[string]*BiomeInfo{
		`default`: {BiomeId: `default`, Name: `Default`, Symbol: `•`, LitArea: true},
		`cave`:    {BiomeId: `cave`, Name: `Cave`, Symbol: `C`, DarkArea: true, Indoor: true},
	}
	users.ResetActiveUsers()
	t.Cleanup(func() {
		biomes = saved
		for id := range specs {
			buffs.RemoveTestBuffSpec(id)
		}
		users.ResetActiveUsers()
		ResetLightFixtures()
	})
}

func lightTestUser(t *testing.T, userId int, buffIds ...int) *users.UserRecord {
	t.Helper()
	u := users.NewUserRecord(userId, uint64(userId))
	for _, id := range buffIds {
		if !u.Character.Buffs.AddBuff(id, true) {
			t.Fatalf("buff %d not registered", id)
		}
	}
	users.SetTestUser(u)
	return u
}

func lightTestCompanion(t *testing.T, instanceId, leaderUserId int, buffIds ...int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{InstanceId: instanceId, Character: characters.Character{Buffs: buffs.New()}}
	m.Character.Charmed = characters.NewCharm(leaderUserId, -1, ``)
	for _, id := range buffIds {
		m.Character.Buffs.AddBuff(id, true)
	}
	mobs.SetTestInstance(m)
	t.Cleanup(func() { mobs.RemoveTestInstance(instanceId) })
	return m
}

func darkCave(roomId int) *Room {
	return &Room{RoomId: roomId, Zone: `Deep`, Biome: `cave`}
}

func TestCarriedLightIsPersonal(t *testing.T) {
	setupLightWiring(t)
	bearer := lightTestUser(t, 1, testTorchBuff)
	other := lightTestUser(t, 2)
	r := darkCave(500)
	r.SetTestOccupants([]int{1, 2}, nil)

	if got := r.VisibilityForUser(bearer); got != 1 {
		t.Fatalf("torch bearer in a dark cave should see dimly, got %d", got)
	}
	if got := r.VisibilityForUser(other); got != 0 {
		t.Fatalf("someone else's torch must not light a stranger, got %d", got)
	}
	if got := r.GetVisibility(); got != 0 {
		t.Fatalf("a carried torch must not raise the room's ambient light, got %d", got)
	}
}

func TestPartyLightCoversPartyMembersNotStrangers(t *testing.T) {
	setupLightWiring(t)
	lightTestUser(t, 1, testPartyLightBuff)
	member := lightTestUser(t, 2)
	stranger := lightTestUser(t, 3)
	p := parties.New(1)
	if p == nil || !p.InvitePlayer(2) || !p.AcceptInvite(2) {
		t.Fatal("failed to build party")
	}
	t.Cleanup(p.Disband)
	r := darkCave(501)
	r.SetTestOccupants([]int{1, 2, 3}, nil)

	if got := r.VisibilityForUser(member); got != 1 {
		t.Fatalf("party member should share the floating light, got %d", got)
	}
	if got := r.VisibilityForUser(stranger); got != 0 {
		t.Fatalf("a stranger must not share the floating light, got %d", got)
	}
}

func TestPartyLightBetweenLeaderAndCompanions(t *testing.T) {
	setupLightWiring(t)
	leader := lightTestUser(t, 1)
	caster := lightTestCompanion(t, 70, 1, testPartyLightBuff)
	follower := lightTestCompanion(t, 71, 1)
	outsider := lightTestCompanion(t, 72, 9)
	r := darkCave(502)
	r.SetTestOccupants([]int{1}, []int{caster.InstanceId, follower.InstanceId, outsider.InstanceId})

	if got := r.VisibilityForUser(leader); got != 1 {
		t.Fatalf("a companion's floating light should light its leader, got %d", got)
	}
	if got := r.VisibilityForMob(follower); got != 1 {
		t.Fatalf("a fellow companion should share the floating light, got %d", got)
	}
	if got := r.VisibilityForMob(outsider); got != 0 {
		t.Fatalf("another player's companion must not share it, got %d", got)
	}
}

func TestNightVisionAndFixtureWiring(t *testing.T) {
	setupLightWiring(t)
	seer := lightTestUser(t, 1, testNightVisBuff)
	plain := lightTestUser(t, 2)
	r := darkCave(503)
	r.SetTestOccupants([]int{1, 2}, nil)

	if got := r.VisibilityForUser(seer); got != 2 {
		t.Fatalf("night vision should see fully, got %d", got)
	}
	RegisterLightFixture(func(roomId int) bool { return roomId == 503 })
	if got := r.VisibilityForUser(plain); got != 1 {
		t.Fatalf("a fixture should light the room for everyone, got %d", got)
	}
}

// TestRoomLightConditionsFromRealRoom checks LightConditions on real rooms,
// not just the pure ambientLevel rules.
func TestRoomLightConditionsFromRealRoom(t *testing.T) {
	withTestBiomes(t) // forest (outdoor), house (lit, indoor)
	t.Cleanup(ResetLightFixtures)

	inn := &Room{RoomId: 600, Zone: `Wood`, Biome: `forest`, Tags: []string{TagIndoor}}
	c := inn.LightConditions()
	if !c.Indoor || !c.LitBiome {
		t.Fatalf("an indoor-tagged room in an outdoor biome is a furnished interior, got %+v", c)
	}
	if got := inn.GetVisibility(); got != 2 {
		t.Fatalf("an indoor-tagged inn room should be lit, got %d", got)
	}

	tagged := &Room{RoomId: 601, Zone: `Wood`, Biome: `forest`, Tags: []string{TagLit}}
	if !tagged.LightConditions().Fixture {
		t.Fatal("a lit-tagged room has a fixture")
	}
	RegisterLightFixture(func(roomId int) bool { return roomId == 602 })
	if !(&Room{RoomId: 602, Zone: `Wood`, Biome: `forest`}).LightConditions().Fixture {
		t.Fatal("a registered fixture provider marks its room")
	}
}
