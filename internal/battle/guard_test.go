package battle

import "testing"

func TestGuardCounts(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("no battle, no guards: %d", got)
	}
	if _, ok := SpendGuard(1, "companion:1"); ok {
		t.Error("no battle: nothing to spend")
	}
	Begin(1, 10, 5, "p", []int{100})
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("a battle starts with %d guards, got %d", MaxGuards, got)
	}
	if left, ok := SpendGuard(1, "companion:1"); !ok || left != 1 {
		t.Errorf("spend: %d %v", left, ok)
	}
	if left, ok := SpendGuard(1, "companion:1"); !ok || left != 0 {
		t.Errorf("spend the last: %d %v", left, ok)
	}
	if _, ok := SpendGuard(1, "companion:1"); ok {
		t.Error("none left")
	}
	if got := GuardsLeft(1, "leader"); got != MaxGuards {
		t.Errorf("each guardian has its own: %d", got)
	}

	// One back per GuardRefillRounds combat rounds, up to MaxGuards.
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("one round: %d", got)
	}
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("two rounds: %d", got)
	}
	TickGuards()
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 2 {
		t.Errorf("four rounds: %d", got)
	}
	TickGuards()
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("capped: %d", got)
	}

	// A charge doesn't carry over a full count: spent again, it takes the
	// whole wait.
	SpendGuard(1, "companion:1")
	TickGuards()
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("one round after spending from full: %d", got)
	}

	// Copies out: a caller's copy can't touch the battle.
	b, _ := Current(1)
	b.Guards["companion:1"] = Guard{Left: 9}
	if got := GuardsLeft(1, "companion:1"); got != 1 {
		t.Errorf("Current is a copy: %d", got)
	}

	// Gone with the battle; a new one starts full.
	End(1)
	if got := GuardsLeft(1, "companion:1"); got != 0 {
		t.Errorf("ended: %d", got)
	}
	Begin(1, 10, 6, "q", []int{101})
	if got := GuardsLeft(1, "companion:1"); got != MaxGuards {
		t.Errorf("a new battle: %d", got)
	}
}

func TestEnemyGuardsAreTheBattles(t *testing.T) {
	Reset()
	defer Reset()
	if _, ok := SpendEnemyGuard(7, 2); ok {
		t.Fatal("no battle, no guard")
	}
	Begin(7, 1, 1, "p", []int{1})
	SetCoordination(7, 3)
	if b, _ := Current(7); b.Coordination != 3 {
		t.Fatalf("tier %d", b.Coordination)
	}
	if left, ok := SpendEnemyGuard(7, 2); !ok || left != 1 {
		t.Fatalf("first: %d %v", left, ok)
	}
	TickGuards() // a company's refill never touches the enemy's
	if left, ok := SpendEnemyGuard(7, 2); !ok || left != 0 {
		t.Fatalf("second: %d %v", left, ok)
	}
	if _, ok := SpendEnemyGuard(7, 2); ok {
		t.Fatal("a third guard")
	}
	Begin(7, 1, 2, "q", []int{2})
	if b, _ := Current(7); b.Coordination != 0 || b.EnemyGuards != 0 {
		t.Fatal("a new battle starts fresh")
	}
}

// 33i2 review finding 2: a group fighting two players spends its guards
// from one count, and a later battle against it starts with them spent.
func TestEnemyGuardsAreTheGroups(t *testing.T) {
	Reset()
	defer Reset()
	Begin(1, 5, 1, "band", []int{9})
	Begin(2, 5, 1, "band", []int{9})
	Begin(3, 5, 1, "other", []int{8})
	if _, ok := SpendEnemyGuard(1, 1); !ok {
		t.Fatal("the band's one guard")
	}
	if _, ok := SpendEnemyGuard(2, 1); ok {
		t.Fatal("spent in the other battle against the same band")
	}
	if _, ok := SpendEnemyGuard(3, 1); !ok {
		t.Fatal("another group has its own")
	}
	End(2)
	Begin(2, 5, 3, "band", []int{9})
	if b, _ := Current(2); b.EnemyGuards != 1 {
		t.Fatalf("a new battle against the band starts spent: %d", b.EnemyGuards)
	}
}

// Phase 35b: a guardian has 2 guards, and 1 more each 10 levels, fixed
// for the battle the first time they are read.
func TestGuardsGrowWithLevel(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	for _, c := range []struct{ level, want int }{{1, 2}, {9, 2}, {10, 3}, {25, 4}, {30, 5}} {
		if got := MaxGuardsFor(c.level); got != c.want {
			t.Errorf("MaxGuardsFor(%d) = %d, want %d", c.level, got, c.want)
		}
	}
	CaptureGuards(1, "leader", 20) // no battle: nothing
	Begin(1, 10, 1, "p", []int{5})
	if got := GuardsLeft(1, "leader"); got != MaxGuards {
		t.Errorf("before capture, the default %d, got %d", MaxGuards, got)
	}
	CaptureGuards(1, "leader", 20)
	CaptureGuards(1, "leader", 40) // the first capture holds
	if got := GuardsLeft(1, "leader"); got != 4 {
		t.Errorf("a level-20 guardian has 4 guards, got %d", got)
	}
	for i := 0; i < 4; i++ {
		if _, ok := SpendGuard(1, "leader"); !ok {
			t.Fatalf("guard %d", i+1)
		}
	}
	if _, ok := SpendGuard(1, "leader"); ok {
		t.Error("a fifth guard")
	}
	for i := 0; i < 20; i++ {
		TickGuards()
	}
	if got := GuardsLeft(1, "leader"); got != 4 {
		t.Errorf("guards come back up to the captured 4, got %d", got)
	}
	b, _ := Current(1)
	if b.GuardMax["leader"] != 4 {
		t.Error("a copy carries the captured most")
	}
}
