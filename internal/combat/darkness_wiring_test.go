package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// countHits runs AttackPlayerVsMob rounds times with the attacker standing in
// room, and counts rounds that landed at least one hit.
func countHits(t *testing.T, room *rooms.Room, rounds int) int {
	t.Helper()
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })

	user := users.NewUserRecord(4242, 4242)
	user.Character.RoomId = room.RoomId
	user.Character.RaceId = 1 // human
	user.Character.SetAggro(0, 4343, characters.DefaultAttack)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(4242) })

	hits := 0
	for i := 0; i < rounds; i++ {
		mob := &mobs.Mob{InstanceId: 4343, Character: *characters.New()}
		mob.Character.RoomId = room.RoomId
		mob.Character.RaceId = 1
		mob.Character.Health, mob.Character.HealthMax.Value = 1000000, 1000000
		if AttackPlayerVsMob(user, mob).Hit {
			hits++
		}
	}
	return hits
}

// TestAttackPlayerVsMobAppliesDarknessPenalty drives the real attack entry
// point: the same attacker lands markedly fewer hits in a pitch-black cave
// than in a lit room. Regression: the penalty was once passed to Hits as a
// positive modifier, which made darkness a +40 bonus.
func TestAttackPlayerVsMobAppliesDarknessPenalty(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	rooms.SetDarknessPenalties(rooms.DefaultDarkHitPenalty, rooms.DefaultDimHitPenalty)

	const rounds = 600
	dark := countHits(t, &rooms.Room{RoomId: 90002, Zone: `Test`, Biome: `cave`}, rounds)
	lit := countHits(t, &rooms.Room{RoomId: 90001, Zone: `Test`, Biome: `city`, Tags: []string{rooms.TagLit}}, rounds)

	if lit == 0 {
		t.Fatalf("expected some hits in a lit room, got 0 of %d", rounds)
	}
	if float64(dark) > float64(lit)*0.8 {
		t.Fatalf("darkness should reduce hits: lit=%d dark=%d of %d rounds", lit, dark, rounds)
	}
}
