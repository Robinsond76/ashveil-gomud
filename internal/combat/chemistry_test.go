package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// fakeChemistry is a company provider for combat tests: user 4242 leads a
// company whose companion #1 is mob instance 4344.
type fakeChemistry struct{ bonus int }

func (fakeChemistry) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (fakeChemistry) InstanceFor(int, int) (int, bool)           { return 0, false }
func (fakeChemistry) LeaderAndKeyForInstance(instanceID int) (int, company.MemberKey, bool) {
	if instanceID == 4344 {
		return 4242, company.CompanionMemberKey(1), true
	}
	return 0, "", false
}
func (f fakeChemistry) ChemistryHitBonus(leaderUserID int, key company.MemberKey) int {
	if leaderUserID == 4242 {
		return f.bonus
	}
	return 0
}
func (fakeChemistry) ChemistryStanding(int, company.MemberKey) (company.ChemistryStandingView, bool) {
	return company.ChemistryStandingView{}, false
}

func withChemistry(t *testing.T, bonus int) {
	t.Helper()
	company.SetFormationProvider(fakeChemistry{bonus: bonus})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
}

func TestHitRollNoBonusNeverByBonus(t *testing.T) {
	loadTestData(t)
	for i := 0; i < 500; i++ {
		if _, byBonus := hitRoll(10, 10, 0, 0); byBonus {
			t.Fatal("no bonus can't decide a hit")
		}
	}
}

func TestHitRollBonusDecides(t *testing.T) {
	loadTestData(t)
	byBonus := 0
	for i := 0; i < 500; i++ {
		// The chance without the bonus is the floor (25); with it, 100.
		hit, decided := hitRoll(10, 10, -1000, 2000)
		if !hit {
			t.Fatal("a bonus to 100% must always hit")
		}
		if decided {
			byBonus++
		}
	}
	// Rolls 25..99 hit only because of the bonus: about 75%.
	if byBonus < 300 || byBonus > 450 {
		t.Fatalf("expected about 375 of 500 hits made by the bonus, got %d", byBonus)
	}
}

// chemistryFight runs AttackPlayerVsMob and AttackMobVsMob against an
// untouchably fast defender (a 25% floor without chemistry) and counts
// rounds that hit and rounds that said chemistry made a hit.
func chemistryFight(t *testing.T, rounds int, attack func(def *mobs.Mob) AttackResult) (hits, lines int) {
	t.Helper()
	for i := 0; i < rounds; i++ {
		def := &mobs.Mob{InstanceId: 4343, Character: *characters.New()}
		def.Character.RoomId = 90241
		def.Character.RaceId = 1
		def.Character.Health, def.Character.HealthMax.Value = 1000000, 1000000
		def.Character.Stats.Speed.ValueAdj = 100000
		res := attack(def)
		if res.Hit {
			hits++
		}
		n := 0
		for _, msg := range res.MessagesToSource {
			if msg == chemistryHitText {
				n++
			}
		}
		if n > 1 {
			t.Fatalf("the chemistry line shows at most once a round, got %d", n)
		}
		lines += n
	}
	return hits, lines
}

func chemistryRoom(t *testing.T) {
	t.Helper()
	loadTestData(t)
	races.LoadDataFiles()
	room := &rooms.Room{RoomId: 90241, Zone: `Test`, Biome: `city`, Tags: []string{rooms.TagLit}}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })
}

func TestAttackPlayerVsMobChemistryRaisesHits(t *testing.T) {
	chemistryRoom(t)
	user := users.NewUserRecord(4242, 4242)
	user.Character.RoomId = 90241
	user.Character.RaceId = 1
	user.Character.SetAggro(0, 4343, characters.DefaultAttack)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(4242) })
	attack := func(def *mobs.Mob) AttackResult { return AttackPlayerVsMob(user, def) }

	withChemistry(t, 0)
	plain, plainLines := chemistryFight(t, 400, attack)
	withChemistry(t, 1000)
	bonded, bondedLines := chemistryFight(t, 400, attack)

	if plainLines != 0 {
		t.Fatalf("no chemistry, no chemistry line: %d", plainLines)
	}
	if bonded < plain*2 || bondedLines == 0 {
		t.Fatalf("the leader's chemistry should raise hits: plain=%d bonded=%d lines=%d", plain, bonded, bondedLines)
	}
}

func TestAttackMobVsMobChemistryRaisesHits(t *testing.T) {
	chemistryRoom(t)
	companion := &mobs.Mob{InstanceId: 4344, Character: *characters.New()}
	companion.Character.RoomId = 90241
	companion.Character.RaceId = 1
	companion.Character.SetAggro(0, 4343, characters.DefaultAttack)
	stranger := &mobs.Mob{InstanceId: 4345, Character: companion.Character}
	companionAttack := func(def *mobs.Mob) AttackResult { return AttackMobVsMob(companion, def) }
	strangerAttack := func(def *mobs.Mob) AttackResult { return AttackMobVsMob(stranger, def) }

	withChemistry(t, 1000)
	plain, plainLines := chemistryFight(t, 400, strangerAttack)
	bonded, bondedLines := chemistryFight(t, 400, companionAttack)
	if plainLines != 0 {
		t.Fatalf("a mob outside the company gets no chemistry: %d lines", plainLines)
	}
	if bonded < plain*2 || bondedLines == 0 {
		t.Fatalf("a companion's chemistry should raise hits: plain=%d bonded=%d lines=%d", plain, bonded, bondedLines)
	}
}
